package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
)

func ipcCall(req IPCRequest) (IPCResponse, error) {
	conn, err := net.Dial("unix", socketPath())
	if err != nil {
		return IPCResponse{}, fmt.Errorf("connect to daemon: %w (is `budsctl daemon` running?)", err)
	}
	defer conn.Close()

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return IPCResponse{}, fmt.Errorf("send request: %w", err)
	}

	var resp IPCResponse
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return IPCResponse{}, fmt.Errorf("read response: %w", err)
	}
	return resp, nil
}

func runStatus() error {
	resp, err := ipcCall(IPCRequest{Command: "status"})
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(resp)
}

func runToggle(device string) error {
	resp, err := ipcCall(IPCRequest{Command: "toggle", Device: device})
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}
	return json.NewEncoder(os.Stdout).Encode(resp)
}
