package main

import (
	"fmt"
	"log"
	"os/exec"
)

// setConfiguredDefault sets the configured default sink/source by node name
// via pw-metadata. This tells WirePlumber the user's preferred device so it
// routes audio there as nodes appear, without disrupting existing streams.
func setConfiguredDefault(key string, nodeName string) error {
	value := fmt.Sprintf(`{"name":"%s"}`, nodeName)
	return exec.Command("pw-metadata", "0", key, value, "Spa:String:JSON").Run()
}

// switchAudio sets the configured default sink/source to the Bluetooth device
// immediately on connect. The WirePlumber-managed node names follow a
// predictable pattern: bluez_output.<MAC> / bluez_input.<MAC>.
// By setting this before WirePlumber creates its stable nodes, it will route
// audio directly to the BT device without an intermediate hop through the
// previous default.
func switchAudio(mac string) {
	if _, err := exec.LookPath("pw-metadata"); err != nil {
		log.Printf("audio switch: pw-metadata not found, skipping (install pipewire)")
		return
	}

	sinkName := "bluez_output." + mac
	if err := setConfiguredDefault("default.configured.audio.sink", sinkName); err != nil {
		log.Printf("audio switch: failed to set configured sink %q: %v", sinkName, err)
	} else {
		log.Printf("audio switch: set configured sink to %q for %s", sinkName, mac)
	}

	// Source is best-effort (A2DP profile doesn't expose one).
	sourceName := "bluez_input." + mac
	if err := setConfiguredDefault("default.configured.audio.source", sourceName); err != nil {
		log.Printf("audio switch: failed to set configured source %q: %v", sourceName, err)
	} else {
		log.Printf("audio switch: set configured source to %q for %s", sourceName, mac)
	}
}
