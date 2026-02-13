package main

import (
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	busName        = "org.bluez"
	adapterPath    = "/org/bluez/hci0"
	adapterIface   = "org.bluez.Adapter1"
	deviceIface    = "org.bluez.Device1"
	propsIface     = "org.freedesktop.DBus.Properties"
	propsSignal    = "org.freedesktop.DBus.Properties.PropertiesChanged"
)

// deviceObjectPath converts a MAC address like "AA:BB:CC:DD:EE:FF" to
// "/org/bluez/hci0/dev_AA_BB_CC_DD_EE_FF".
func deviceObjectPath(addr string) dbus.ObjectPath {
	escaped := strings.ReplaceAll(addr, ":", "_")
	return dbus.ObjectPath(adapterPath + "/dev_" + escaped)
}

// macFromPath extracts a MAC address from a BlueZ device object path.
func macFromPath(path dbus.ObjectPath) string {
	s := string(path)
	prefix := adapterPath + "/dev_"
	if !strings.HasPrefix(s, prefix) {
		return ""
	}
	return strings.ReplaceAll(s[len(prefix):], "_", ":")
}

// bluez wraps a system D-Bus connection for BlueZ operations.
type bluez struct {
	conn *dbus.Conn
}

func newBluez() (*bluez, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, fmt.Errorf("connect to system bus: %w", err)
	}
	// Quick check that BlueZ is on the bus.
	var names []string
	if err := conn.BusObject().Call("org.freedesktop.DBus.ListNames", 0).Store(&names); err != nil {
		conn.Close()
		return nil, fmt.Errorf("list bus names: %w", err)
	}
	found := false
	for _, n := range names {
		if n == busName {
			found = true
			break
		}
	}
	if !found {
		conn.Close()
		return nil, fmt.Errorf("org.bluez not found on system bus — is bluetooth.service running?")
	}
	return &bluez{conn: conn}, nil
}

func (b *bluez) close() {
	b.conn.Close()
}

// --- property helpers ---

func (b *bluez) getProp(path dbus.ObjectPath, iface, prop string) (dbus.Variant, error) {
	obj := b.conn.Object(busName, path)
	var v dbus.Variant
	err := obj.Call(propsIface+".Get", 0, iface, prop).Store(&v)
	return v, err
}

func (b *bluez) setProp(path dbus.ObjectPath, iface, prop string, val interface{}) error {
	obj := b.conn.Object(busName, path)
	return obj.Call(propsIface+".Set", 0, iface, prop, dbus.MakeVariant(val)).Err
}

func (b *bluez) getBool(path dbus.ObjectPath, iface, prop string) (bool, error) {
	v, err := b.getProp(path, iface, prop)
	if err != nil {
		return false, err
	}
	val, ok := v.Value().(bool)
	if !ok {
		return false, fmt.Errorf("property %s is not bool", prop)
	}
	return val, nil
}

// --- adapter ---

func (b *bluez) adapterPowered() (bool, error) {
	return b.getBool(adapterPath, adapterIface, "Powered")
}

func (b *bluez) setAdapterPowered(on bool) error {
	return b.setProp(adapterPath, adapterIface, "Powered", on)
}

// --- device ---

func (b *bluez) deviceConnected(addr string) (bool, error) {
	return b.getBool(deviceObjectPath(addr), deviceIface, "Connected")
}

func (b *bluez) deviceBlocked(addr string) (bool, error) {
	return b.getBool(deviceObjectPath(addr), deviceIface, "Blocked")
}

func (b *bluez) devicePaired(addr string) (bool, error) {
	return b.getBool(deviceObjectPath(addr), deviceIface, "Paired")
}

func (b *bluez) setBlocked(addr string, blocked bool) error {
	return b.setProp(deviceObjectPath(addr), deviceIface, "Blocked", blocked)
}

func (b *bluez) connect(addr string) error {
	obj := b.conn.Object(busName, deviceObjectPath(addr))
	return obj.Call(deviceIface+".Connect", 0).Err
}

func (b *bluez) disconnect(addr string) error {
	obj := b.conn.Object(busName, deviceObjectPath(addr))
	return obj.Call(deviceIface+".Disconnect", 0).Err
}

// --- state resolution ---

func (b *bluez) resolveState(addr string) DeviceState {
	powered, err := b.adapterPowered()
	if err != nil || !powered {
		return StateDisabled
	}
	paired, err := b.devicePaired(addr)
	if err != nil || !paired {
		return StateDisabled
	}
	connected, _ := b.deviceConnected(addr)
	if connected {
		return StateConnected
	}
	blocked, _ := b.deviceBlocked(addr)
	if !blocked {
		return StateConnecting
	}
	return StateIdle
}

// --- toggle ---

func (b *bluez) toggle(addr string) (DeviceState, error) {
	state := b.resolveState(addr)
	switch state {
	case StateConnected:
		if err := b.disconnect(addr); err != nil {
			return state, fmt.Errorf("disconnect: %w", err)
		}
		if err := b.setBlocked(addr, true); err != nil {
			return state, fmt.Errorf("block: %w", err)
		}
		return StateIdle, nil

	case StateConnecting:
		if err := b.setBlocked(addr, true); err != nil {
			return state, fmt.Errorf("block: %w", err)
		}
		return StateIdle, nil

	case StateIdle:
		if err := b.setBlocked(addr, false); err != nil {
			return state, fmt.Errorf("unblock: %w", err)
		}
		if err := b.connect(addr); err != nil {
			return state, fmt.Errorf("connect: %w", err)
		}
		return StateConnected, nil

	case StateDisabled:
		if err := b.setAdapterPowered(true); err != nil {
			return state, fmt.Errorf("power on: %w", err)
		}
		if err := b.setBlocked(addr, false); err != nil {
			return state, fmt.Errorf("unblock: %w", err)
		}
		if err := b.connect(addr); err != nil {
			return state, fmt.Errorf("connect: %w", err)
		}
		return StateConnected, nil
	}
	return state, fmt.Errorf("unexpected state %q", state)
}

// --- signal subscription ---

func (b *bluez) subscribePropertyChanges() chan *dbus.Signal {
	b.conn.BusObject().Call(
		"org.freedesktop.DBus.AddMatch", 0,
		"type='signal',interface='"+propsIface+"',member='PropertiesChanged',path_namespace='/org/bluez'",
	)
	ch := make(chan *dbus.Signal, 16)
	b.conn.Signal(ch)
	return ch
}
