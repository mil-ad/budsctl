package main

import (
	"fmt"
	"os"
)

const helpText = `budsctl - Bluetooth device manager

Usage:
  budsctl <command> [arguments]

Commands:
  daemon          Start the background daemon
  status          Show the current device state
  toggle <MAC>    Connect or disconnect a Bluetooth device

The daemon must be running for status and toggle to work.
On connect, audio is automatically switched to the Bluetooth device
if wireplumber (wpctl) is installed.`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, helpText)
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "-h", "--help", "help":
		fmt.Println(helpText)
		return
	case "daemon":
		err = runDaemon()
	case "status":
		err = runStatus()
	case "toggle":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: budsctl toggle <MAC>")
			os.Exit(1)
		}
		err = runToggle(os.Args[2])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		fmt.Fprintln(os.Stderr, helpText)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
