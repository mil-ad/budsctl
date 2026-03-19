package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// getConfiguredDefault reads the current configured default for the given key
// from pw-metadata. Returns empty string if none is set.
func getConfiguredDefault(key string) string {
	out, err := exec.Command("pw-metadata", "0", key).Output()
	if err != nil {
		return ""
	}
	// Output format: update: id:0 key:'...' value:'{"name":"..."}' type:'Spa:String:JSON'
	s := string(out)
	const prefix = "value:'"
	i := strings.Index(s, prefix)
	if i < 0 {
		return ""
	}
	s = s[i+len(prefix):]
	j := strings.Index(s, "' type:")
	if j < 0 {
		return ""
	}
	return s[:j]
}

// setConfiguredDefault sets the configured default sink/source by node name
// via pw-metadata. This tells WirePlumber the user's preferred device so it
// routes audio there as nodes appear, without disrupting existing streams.
func setConfiguredDefault(key string, nodeName string) error {
	value := fmt.Sprintf(`{"name":"%s"}`, nodeName)
	return exec.Command("pw-metadata", "0", key, value, "Spa:String:JSON").Run()
}

// previousAudio holds the pw-metadata values to restore on disconnect.
type previousAudio struct {
	sink   string // runtime default (default.audio.sink)
	source string // runtime default (default.audio.source)
}

// switchAudio saves the current runtime default sink/source, then sets the
// configured defaults to the Bluetooth device. Returns the previous runtime
// values so they can be restored on disconnect.
func switchAudio(mac string) *previousAudio {
	if _, err := exec.LookPath("pw-metadata"); err != nil {
		log.Printf("audio switch: pw-metadata not found, skipping (install pipewire)")
		return nil
	}

	// Save the runtime defaults — these reflect what the user is actually
	// listening on, which may differ from the configured (persistent) defaults.
	prev := &previousAudio{
		sink:   getConfiguredDefault("default.audio.sink"),
		source: getConfiguredDefault("default.audio.source"),
	}
	log.Printf("audio switch: saved previous sink=%s source=%s", prev.sink, prev.source)

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

	return prev
}

// restoreAudio sets the configured defaults back to what they were before
// switchAudio was called.
func restoreAudio(prev *previousAudio) {
	if prev == nil {
		return
	}
	if _, err := exec.LookPath("pw-metadata"); err != nil {
		return
	}

	if prev.sink != "" {
		if err := exec.Command("pw-metadata", "0", "default.configured.audio.sink", prev.sink, "Spa:String:JSON").Run(); err != nil {
			log.Printf("audio restore: failed to restore sink: %v", err)
		} else {
			log.Printf("audio restore: restored sink to %s", prev.sink)
		}
	}
	if prev.source != "" {
		if err := exec.Command("pw-metadata", "0", "default.configured.audio.source", prev.source, "Spa:String:JSON").Run(); err != nil {
			log.Printf("audio restore: failed to restore source: %v", err)
		} else {
			log.Printf("audio restore: restored source to %s", prev.source)
		}
	}
}
