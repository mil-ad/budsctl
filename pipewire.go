package main

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var nodeLineRe = regexp.MustCompile(`^\s*\*?\s*(\d+)\.\s+(.+)`)

// findBluezNodeID parses `wpctl status` output to find a PipeWire node ID
// for a Bluetooth device matching the given MAC address.
// nodeType is "Sinks" or "Sources".
func findBluezNodeID(mac string, nodeType string) (int, error) {
	out, err := exec.Command("wpctl", "status").Output()
	if err != nil {
		return 0, fmt.Errorf("wpctl status: %w", err)
	}

	// Find candidate node IDs in the right section.
	inAudio := false
	inSection := false
	var candidates []int

	for _, line := range strings.Split(string(out), "\n") {
		trimmed := strings.TrimSpace(line)

		// Detect the Audio top-level section.
		if trimmed == "Audio" {
			inAudio = true
			continue
		}
		// A new top-level section (e.g. "Video") ends Audio.
		if inAudio && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "│") && !strings.HasPrefix(line, "├") && !strings.HasPrefix(line, "└") && trimmed != "" {
			// Check if this is a non-indented, non-tree line — new top-level section.
			if !strings.ContainsAny(line[:1], " │├└") {
				inAudio = false
			}
		}
		if !inAudio {
			continue
		}

		// Detect subsection headers like "Sinks:" or "Sources:".
		if strings.HasSuffix(trimmed, ":") {
			sectionName := strings.TrimSuffix(trimmed, ":")
			inSection = sectionName == nodeType
			continue
		}

		if !inSection {
			continue
		}

		m := nodeLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		id, _ := strconv.Atoi(m[1])
		candidates = append(candidates, id)
	}

	// Check each candidate with `wpctl inspect` to confirm it belongs to our device.
	escapedMAC := strings.ReplaceAll(strings.ToUpper(mac), ":", "_")
	for _, id := range candidates {
		out, err := exec.Command("wpctl", "inspect", strconv.Itoa(id)).Output()
		if err != nil {
			continue
		}
		if strings.Contains(string(out), escapedMAC) {
			return id, nil
		}
	}

	return 0, fmt.Errorf("no %s node found for %s", nodeType, mac)
}

// setDefaultNode runs `wpctl set-default <id>`.
func setDefaultNode(id int) error {
	return exec.Command("wpctl", "set-default", strconv.Itoa(id)).Run()
}

// switchAudio polls for the Bluetooth device's PipeWire sink/source to appear,
// then sets them as defaults. Called in a goroutine after a BT connection.
func switchAudio(mac string) {
	if _, err := exec.LookPath("wpctl"); err != nil {
		log.Printf("audio switch: wpctl not found, skipping (install wireplumber)")
		return
	}

	var sinkID int
	var err error

	// Poll up to 5 seconds for the sink to appear.
	for range 10 {
		sinkID, err = findBluezNodeID(mac, "Sinks")
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		log.Printf("audio switch: sink not found for %s: %v", mac, err)
		return
	}

	if err := setDefaultNode(sinkID); err != nil {
		log.Printf("audio switch: failed to set default sink %d: %v", sinkID, err)
	} else {
		log.Printf("audio switch: set default sink to %d for %s", sinkID, mac)
	}

	// Source is best-effort (A2DP doesn't expose one).
	sourceID, err := findBluezNodeID(mac, "Sources")
	if err != nil {
		return
	}
	if err := setDefaultNode(sourceID); err != nil {
		log.Printf("audio switch: failed to set default source %d: %v", sourceID, err)
	} else {
		log.Printf("audio switch: set default source to %d for %s", sourceID, mac)
	}
}
