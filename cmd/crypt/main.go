package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: crypt <command> [args]")
		fmt.Fprintln(os.Stderr, "commands: connect, solo, headless, autoplay")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "connect":
		runConnect(os.Args[2:])
	case "solo":
		runSolo(os.Args[2:])
	case "headless":
		runHeadless(os.Args[2:])
	case "autoplay":
		runAutoplay(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
