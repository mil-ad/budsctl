package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func configPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		dir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(dir, "budsctl", "devices.json")
}

func loadConfig() ([]DeviceConfig, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var devices []DeviceConfig
	if err := json.Unmarshal(data, &devices); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return devices, nil
}

// resolveDevice picks a device address. If addr is non-empty, it is returned
// directly. Otherwise, the first device from the config is used.
func resolveDevice(cfg []DeviceConfig, addr string) (string, error) {
	if addr != "" {
		return addr, nil
	}
	if len(cfg) == 0 {
		return "", fmt.Errorf("no device specified and config is empty")
	}
	return cfg[0].Address, nil
}
