package main

// DeviceState represents the current state of a Bluetooth device.
type DeviceState string

const (
	StateConnected  DeviceState = "connected"
	StateConnecting DeviceState = "connecting"
	StateBlocked    DeviceState = "blocked"
	StateDisabled   DeviceState = "disabled"
)

// IPCRequest is sent from the CLI client to the daemon.
type IPCRequest struct {
	Command string `json:"command"`          // "status" | "toggle"
	Device  string `json:"device,omitempty"` // MAC address, optional
}

// IPCResponse is sent from the daemon back to the CLI client.
type IPCResponse struct {
	State  string `json:"state,omitempty"`  // "connected", "connecting", "blocked", "disabled"
	Device string `json:"device,omitempty"` // MAC address of active device
	Error  string `json:"error,omitempty"`
}

