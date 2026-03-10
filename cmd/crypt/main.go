package main

import (
	"fmt"
	"os"

	"github.com/punt-labs/cryptd/internal/scenario"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: crypt <command> [args]")
		fmt.Fprintln(os.Stderr, "commands: validate")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "validate":
		runValidate(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func runValidate(args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: crypt validate <scenario-file>")
		os.Exit(1)
	}
	path := args[0]
	if _, err := scenario.Load(path); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("OK")
}
