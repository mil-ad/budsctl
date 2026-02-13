package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/godbus/dbus/v5"
)

func socketPath() string {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = "/tmp"
	}
	return filepath.Join(dir, "budsctl.sock")
}

type daemon struct {
	bz           *bluez
	activeDevice string // MAC address of the active device
	mu           sync.Mutex
}

func (d *daemon) handleRequest(req IPCRequest) IPCResponse {
	d.mu.Lock()
	defer d.mu.Unlock()

	switch req.Command {
	case "status":
		if d.activeDevice == "" {
			return IPCResponse{State: string(StateDisabled)}
		}
		state := d.bz.resolveState(d.activeDevice)
		return IPCResponse{State: string(state), Device: d.activeDevice}

	case "toggle":
		addr := req.Device
		if addr == "" {
			return IPCResponse{Error: "device address is required"}
		}

		// Switching devices: disconnect+block the old one first.
		if d.activeDevice != "" && d.activeDevice != addr {
			prev := d.activeDevice
			if d.bz.resolveState(prev) == StateConnected {
				log.Printf("switching from %s to %s, disconnecting old device", prev, addr)
				d.bz.disconnect(prev)
				d.bz.setBlocked(prev, true)
			}
		}
		d.activeDevice = addr

		newState, err := d.bz.toggle(addr)
		if err != nil {
			return IPCResponse{Error: err.Error()}
		}
		return IPCResponse{State: string(newState), Device: addr}

	default:
		return IPCResponse{Error: fmt.Sprintf("unknown command: %q", req.Command)}
	}
}

func (d *daemon) handleConn(conn net.Conn) {
	defer conn.Close()

	var req IPCRequest
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		resp := IPCResponse{Error: "invalid request: " + err.Error()}
		json.NewEncoder(conn).Encode(resp)
		return
	}

	resp := d.handleRequest(req)
	json.NewEncoder(conn).Encode(resp)
}

func (d *daemon) watchSignals(sigCh chan *dbus.Signal) {
	for sig := range sigCh {
		if sig.Name != propsSignal {
			continue
		}
		// Body: [interface_name string, changed_props map[string]Variant, invalidated []string]
		if len(sig.Body) < 2 {
			continue
		}
		iface, ok := sig.Body[0].(string)
		if !ok || iface != deviceIface {
			continue
		}
		changed, ok := sig.Body[1].(map[string]dbus.Variant)
		if !ok {
			continue
		}
		connVar, ok := changed["Connected"]
		if !ok {
			continue
		}
		connected, ok := connVar.Value().(bool)
		if !ok || connected {
			continue
		}
		// Connected flipped to false — check if it's the active device.
		mac := macFromPath(sig.Path)
		d.mu.Lock()
		active := d.activeDevice
		d.mu.Unlock()
		if mac == "" || mac != active {
			continue
		}
		log.Printf("active device %s disconnected, auto-blocking", mac)
		if err := d.bz.setBlocked(mac, true); err != nil {
			log.Printf("auto-block failed: %v", err)
		}
	}
}

func runDaemon() error {
	bz, err := newBluez()
	if err != nil {
		return err
	}
	defer bz.close()

	sock := socketPath()
	os.Remove(sock) // remove stale socket
	ln, err := net.Listen("unix", sock)
	if err != nil {
		return fmt.Errorf("listen %s: %w", sock, err)
	}
	os.Chmod(sock, 0700)
	defer os.Remove(sock)
	defer ln.Close()

	d := &daemon{bz: bz}

	// Signal watcher goroutine.
	dbusSignals := bz.subscribePropertyChanges()
	go d.watchSignals(dbusSignals)

	// Graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("shutting down")
		ln.Close()
	}()

	log.Printf("listening on %s", sock)
	for {
		conn, err := ln.Accept()
		if err != nil {
			// Listener closed by shutdown goroutine.
			return nil
		}
		go d.handleConn(conn)
	}
}
