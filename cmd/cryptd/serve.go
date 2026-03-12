package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/punt-labs/cryptd/internal/daemon"
	"github.com/punt-labs/cryptd/internal/inference"
	"github.com/punt-labs/cryptd/internal/interpreter"
	"github.com/punt-labs/cryptd/internal/narrator"
	"github.com/punt-labs/cryptd/internal/scenariodir"
)

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	socketPath := fs.String("socket", "", "Unix socket path (default ~/.crypt/daemon.sock)")
	listenAddr := fs.String("listen", "", "TCP listen address (e.g. :9000)")
	passthrough := fs.Bool("passthrough", false, "MCP passthrough mode (structured JSON, no interpreter/narrator)")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	socketExplicit := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "socket" {
			socketExplicit = true
		}
	})
	if socketExplicit && *listenAddr != "" {
		fmt.Fprintln(os.Stderr, "error: --socket and --listen are mutually exclusive")
		os.Exit(1)
	}

	absScenarioDir, err := filepath.Abs(scenariodir.Dir())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var opts []daemon.ServerOption

	if *passthrough {
		opts = append(opts, daemon.WithPassthrough())
	} else {
		interp, narr := resolveInterpreterNarrator()
		opts = append(opts, daemon.WithInterpreter(interp), daemon.WithNarrator(narr))
	}

	var srv *daemon.Server
	if *listenAddr != "" {
		srv = daemon.NewTCPServer(*listenAddr, absScenarioDir, opts...)
	} else {
		sock := *socketPath
		if sock == "" {
			var err error
			sock, err = daemon.DefaultSocketPath()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: cannot determine home directory: %v\nUse --socket or --listen to specify an address explicitly.\n", err)
				os.Exit(1)
			}
		}
		srv = daemon.NewServer(sock, absScenarioDir, opts...)
	}
	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// resolveInterpreterNarrator probes for an SLM (ollama) and returns the
// appropriate interpreter and narrator. Falls back to Rules + Template
// if no inference server is available.
func resolveInterpreterNarrator() (*interpreter.SLM, *narrator.SLM) {
	rules := interpreter.NewRules()
	tmpl := narrator.NewTemplate()

	rt := inference.Probe(context.Background(), inference.DefaultEndpoints(), time.Second)
	if rt != nil {
		client := inference.NewClient(rt.BaseURL, rt.Model, 5*time.Second)
		fmt.Fprintf(os.Stderr, "cryptd: using %s (model: %s)\n", rt.BaseURL, rt.Model)
		return interpreter.NewSLM(client, rules), narrator.NewSLM(client, tmpl)
	}

	fmt.Fprintln(os.Stderr, "cryptd: no inference server found — using rules+templates")
	return interpreter.NewSLM(nil, rules), narrator.NewSLM(nil, tmpl)
}

