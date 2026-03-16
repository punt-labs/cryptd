package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/punt-labs/cryptd/internal/scenario"
	"github.com/punt-labs/cryptd/internal/scengen"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "generate":
		runGenerate(os.Args[2:])
	case "validate":
		runValidate(os.Args[2:])
	case "export":
		runExport(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: crypt-admin <command> [args]")
	fmt.Fprintln(os.Stderr, "commands: generate, validate, export")
}

// requireFlagValue checks that a flag has a value argument following it.
func requireFlagValue(flag string, args []string, i int) string {
	if i >= len(args) {
		fmt.Fprintf(os.Stderr, "error: %s requires a value\n", flag)
		os.Exit(1)
	}
	if strings.HasPrefix(args[i], "--") {
		fmt.Fprintf(os.Stderr, "error: %s requires a value, got flag %s\n", flag, args[i])
		os.Exit(1)
	}
	return args[i]
}

func runGenerate(args []string) {
	var (
		topology = "tree"
		source   = ""
		title    = ""
		output   = ""
		dbPath   = ""
	)

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--topology":
			i++
			topology = requireFlagValue("--topology", args, i)
		case "--source":
			i++
			source = requireFlagValue("--source", args, i)
		case "--title":
			i++
			title = requireFlagValue("--title", args, i)
		case "--output":
			i++
			output = requireFlagValue("--output", args, i)
		case "--db":
			i++
			dbPath = requireFlagValue("--db", args, i)
		default:
			fmt.Fprintf(os.Stderr, "unknown flag: %s\n", args[i])
			fmt.Fprintln(os.Stderr, "usage: crypt-admin generate --topology tree --source <path> --title <title> --output <dir>")
			os.Exit(1)
		}
	}

	if source == "" || title == "" || output == "" {
		fmt.Fprintln(os.Stderr, "usage: crypt-admin generate --topology tree --source <path> --title <title> --output <dir>")
		fmt.Fprintln(os.Stderr, "required: --source, --title, --output")
		os.Exit(1)
	}

	var topoSource scengen.TopologySource
	switch topology {
	case "tree":
		topoSource = &scengen.TreeSource{Root: source}
	default:
		fmt.Fprintf(os.Stderr, "unknown topology: %s (supported: tree)\n", topology)
		os.Exit(1)
	}

	// Generate graph.
	g, err := scengen.GenerateGraph(topoSource)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating graph: %v\n", err)
		os.Exit(1)
	}

	// Store title and death in graph metadata for SQLite round-trip.
	g.Meta["title"] = title
	g.Meta["death"] = "respawn"

	// Optionally persist to SQLite.
	if dbPath != "" {
		store, err := scengen.OpenStore(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error opening database: %v\n", err)
			os.Exit(1)
		}
		defer store.Close()
		if err := store.CreateSchema(); err != nil {
			fmt.Fprintf(os.Stderr, "error creating schema: %v\n", err)
			os.Exit(1)
		}
		if err := store.SaveGraph(g); err != nil {
			fmt.Fprintf(os.Stderr, "error saving graph: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "saved graph to %s\n", dbPath)
	}

	// Run visitors and export.
	content := scengen.NewScenarioContent()
	content.Title = title
	content.Death = "respawn"
	visitor := &scengen.DescriptionVisitor{}
	if err := visitor.Visit(g, content); err != nil {
		fmt.Fprintf(os.Stderr, "error running visitor: %v\n", err)
		os.Exit(1)
	}

	if err := scengen.WriteYAMLDir(g, content, output); err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		os.Exit(1)
	}

	// Self-validate the output.
	if _, err := scenario.LoadDir(output); err != nil {
		fmt.Fprintf(os.Stderr, "warning: generated scenario failed self-validation: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("generated %d rooms → %s\n", len(g.Nodes), output)
}

func runValidate(args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: crypt-admin validate <path>")
		os.Exit(1)
	}
	path := args[0]

	// Check if path is a directory (directory format).
	info, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if info.IsDir() {
		if _, err := scenario.LoadDir(path); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	} else {
		if _, err := scenario.Load(path); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}
	fmt.Println("OK")
}

func runExport(args []string) {
	var dbPath, output string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--db":
			i++
			dbPath = requireFlagValue("--db", args, i)
		case "--output":
			i++
			output = requireFlagValue("--output", args, i)
		default:
			fmt.Fprintf(os.Stderr, "unknown flag: %s\n", args[i])
			os.Exit(1)
		}
	}

	if dbPath == "" || output == "" {
		fmt.Fprintln(os.Stderr, "usage: crypt-admin export --db <path> --output <dir>")
		os.Exit(1)
	}

	store, err := scengen.OpenStore(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening database: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	g, err := store.LoadGraph()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading graph: %v\n", err)
		os.Exit(1)
	}

	content := scengen.NewScenarioContent()
	content.Title = g.Meta["title"]
	if content.Title == "" {
		content.Title = "Exported Scenario"
	}
	content.Death = g.Meta["death"]
	if content.Death == "" {
		content.Death = "respawn"
	}
	visitor := &scengen.DescriptionVisitor{}
	if err := visitor.Visit(g, content); err != nil {
		fmt.Fprintf(os.Stderr, "error running visitor: %v\n", err)
		os.Exit(1)
	}

	if err := scengen.WriteYAMLDir(g, content, output); err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		os.Exit(1)
	}

	// Self-validate the exported scenario (mirrors runGenerate).
	if _, err := scenario.LoadDir(output); err != nil {
		fmt.Fprintf(os.Stderr, "warning: exported scenario failed self-validation: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("exported %d rooms → %s\n", len(g.Nodes), output)
}
