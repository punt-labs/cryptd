package main

import (
	"fmt"
	"os"

	"github.com/punt-labs/cryptd/internal/observe"
	"github.com/punt-labs/cryptd/internal/scenario"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: cryptd <command> [args]")
		fmt.Fprintln(os.Stderr, "commands: serve, validate")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		os.Exit(runServe(os.Args[2:]))
	case "validate":
		os.Exit(observe.Run("cli", observe.Validate, func(obs *observe.Runtime) int {
			obs.Log.Warn("Used deprecated scenario validation command")
			return runValidate(os.Args[2:])
		}))
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func runValidate(args []string) int {
	fmt.Fprintln(os.Stderr, "warning: 'cryptd validate' is deprecated, use 'crypt-admin validate' instead")
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: cryptd validate <scenario-file>")
		return 1
	}
	path := args[0]
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		fmt.Fprintln(os.Stderr, "error: directory-format scenarios require 'crypt-admin validate'")
		return 1
	}
	if _, err := scenario.Load(path); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	fmt.Println("OK")
	return 0
}
