package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/punt-labs/cryptd/internal/observe"
	"github.com/punt-labs/cryptd/internal/scenario"
	"github.com/punt-labs/cryptd/internal/scengen"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var operation observe.Operation
	var run func([]string) int
	switch os.Args[1] {
	case "generate":
		operation, run = observe.AdminGenerate, runGenerate
	case "validate":
		operation, run = observe.AdminValidate, runValidate
	case "export":
		operation, run = observe.AdminExport, runExport
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
	os.Exit(observe.Run("admin", operation, func(obs *observe.Runtime) int {
		code := run(os.Args[2:])
		obs.Log.Info("Finished scenario administration command", "command", os.Args[1], "ok", code == 0)
		return code
	}))
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: crypt-admin <command> [args]")
	fmt.Fprintln(os.Stderr, "commands: generate, validate, export")
}

// requireFlagValue checks that a flag has a value argument following it.
func requireFlagValue(flag string, args []string, i int) (string, bool) {
	if i >= len(args) {
		fmt.Fprintf(os.Stderr, "error: %s requires a value\n", flag)
		return "", false
	}
	if strings.HasPrefix(args[i], "--") {
		fmt.Fprintf(os.Stderr, "error: %s requires a value, got flag %s\n", flag, args[i])
		return "", false
	}
	return args[i], true
}

func runGenerate(args []string) int {
	var (
		topology = "tree"
		source   = ""
		title    = ""
		output   = ""
		dbPath   = ""
	)

	for i := 0; i < len(args); i++ {
		flag := args[i]
		var target *string
		switch flag {
		case "--topology":
			target = &topology
		case "--source":
			target = &source
		case "--title":
			target = &title
		case "--output":
			target = &output
		case "--db":
			target = &dbPath
		default:
			fmt.Fprintf(os.Stderr, "unknown flag: %s\n", args[i])
			fmt.Fprintln(os.Stderr, "usage: crypt-admin generate --topology tree --source <path> --title <title> --output <dir>")
			return 1
		}
		i++
		value, ok := requireFlagValue(flag, args, i)
		if !ok {
			return 1
		}
		*target = value
	}

	if source == "" || title == "" || output == "" {
		fmt.Fprintln(os.Stderr, "usage: crypt-admin generate --topology tree --source <path> --title <title> --output <dir>")
		fmt.Fprintln(os.Stderr, "required: --source, --title, --output")
		return 1
	}

	var topoSource scengen.TopologySource
	switch topology {
	case "tree":
		topoSource = &scengen.TreeSource{Root: source}
	default:
		fmt.Fprintf(os.Stderr, "unknown topology: %s (supported: tree)\n", topology)
		return 1
	}

	// Generate graph.
	g, err := scengen.GenerateGraph(topoSource)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating graph: %v\n", err)
		return 1
	}

	// Store title and death in graph metadata for SQLite round-trip.
	g.Meta["title"] = title
	g.Meta["death"] = "respawn"

	// Optionally persist to SQLite.
	if dbPath != "" {
		store, err := scengen.OpenStore(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error opening database: %v\n", err)
			return 1
		}
		defer store.Close()
		if err := store.CreateSchema(); err != nil {
			fmt.Fprintf(os.Stderr, "error creating schema: %v\n", err)
			return 1
		}
		if err := store.SaveGraph(g); err != nil {
			fmt.Fprintf(os.Stderr, "error saving graph: %v\n", err)
			return 1
		}
		fmt.Fprintf(os.Stderr, "saved graph to %s\n", dbPath)
	}

	// Run visitors in order: description → enemies → items.
	// ItemVisitor depends on EnemyVisitor (places potions near enemies).
	content := scengen.NewScenarioContent()
	content.Title = title
	content.Death = "respawn"
	genSeed := time.Now().UnixNano()
	rng := rand.New(rand.NewSource(genSeed))
	fmt.Fprintf(os.Stderr, "  seed: %d\n", genSeed)
	visitors := []scengen.Visitor{
		&scengen.DescriptionVisitor{},
		&scengen.EnemyVisitor{Rng: rng, SpawnRate: -1},
		&scengen.ItemVisitor{Rng: rng, IncludeSpells: true},
	}
	for _, vis := range visitors {
		if err := vis.Visit(g, content); err != nil {
			fmt.Fprintf(os.Stderr, "error running visitor %s: %v\n", vis.Name(), err)
			return 1
		}
	}

	fmt.Fprintf(os.Stderr, "  rooms: %d, enemies: %d rooms, total XP: %d, items: %d\n",
		len(g.Nodes), scengen.EnemyCount(content), scengen.TotalXP(content), len(content.Items))

	if err := scengen.WriteYAMLDir(g, content, output); err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		return 1
	}

	// Self-validate the output.
	if _, err := scenario.LoadDir(output); err != nil {
		fmt.Fprintf(os.Stderr, "warning: generated scenario failed self-validation: %v\n", err)
		return 1
	}

	fmt.Printf("generated %d rooms → %s\n", len(g.Nodes), output)
	return 0
}

func runValidate(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: crypt-admin validate <path>")
		return 1
	}
	path := args[0]

	// Check if path is a directory (directory format).
	info, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	if info.IsDir() {
		if _, err := scenario.LoadDir(path); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
	} else {
		if _, err := scenario.Load(path); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
	}
	fmt.Println("OK")
	return 0
}

func runExport(args []string) int {
	var dbPath, output string

	for i := 0; i < len(args); i++ {
		flag := args[i]
		var target *string
		switch flag {
		case "--db":
			target = &dbPath
		case "--output":
			target = &output
		default:
			fmt.Fprintf(os.Stderr, "unknown flag: %s\n", args[i])
			return 1
		}
		i++
		value, ok := requireFlagValue(flag, args, i)
		if !ok {
			return 1
		}
		*target = value
	}

	if dbPath == "" || output == "" {
		fmt.Fprintln(os.Stderr, "usage: crypt-admin export --db <path> --output <dir>")
		return 1
	}

	store, err := scengen.OpenStore(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening database: %v\n", err)
		return 1
	}
	defer store.Close()

	g, err := store.LoadGraph()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading graph: %v\n", err)
		return 1
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
	exportRng := rand.New(rand.NewSource(time.Now().UnixNano()))
	visitors := []scengen.Visitor{
		&scengen.DescriptionVisitor{},
		&scengen.EnemyVisitor{Rng: exportRng, SpawnRate: -1},
		&scengen.ItemVisitor{Rng: exportRng, IncludeSpells: true},
	}
	for _, vis := range visitors {
		if err := vis.Visit(g, content); err != nil {
			fmt.Fprintf(os.Stderr, "error running visitor %s: %v\n", vis.Name(), err)
			return 1
		}
	}

	if err := scengen.WriteYAMLDir(g, content, output); err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		return 1
	}

	// Self-validate the exported scenario (mirrors runGenerate).
	if _, err := scenario.LoadDir(output); err != nil {
		fmt.Fprintf(os.Stderr, "warning: exported scenario failed self-validation: %v\n", err)
		return 1
	}

	fmt.Printf("exported %d rooms → %s\n", len(g.Nodes), output)
	return 0
}
