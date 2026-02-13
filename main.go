package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: budsctl <daemon|status|toggle> [device]")
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "daemon":
		err = runDaemon()
	case "status":
		err = runStatus()
	case "toggle":
		device := ""
		if len(os.Args) > 2 {
			device = os.Args[2]
		}
		err = runToggle(device)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
