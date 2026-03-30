.PHONY: help vet test test-integration test-e2e lint markdownlint coverage \
       check check-full build build-server build-client build-admin clean help \
       install uninstall man play play-unix \
       ollama-install ollama-start ollama-pull ollama-setup eval-slm \
       generate-dungeon validate-dungeon \
       eval-balance eval-balance-quick eval-balance-generated eval-balance-unix \
       demo demo-solo demo-exploration demo-inventory \
       demo-combat demo-save-load demo-unix-catacombs

SCENARIO_DIR = testdata/scenarios
DEMO_DIR     = testdata/demos
PREFIX       = /usr/local
BINDIR       = $(PREFIX)/bin
MANDIR       = $(PREFIX)/share/man/man1

##@ Default
all: build check                           ## Build binaries and run quality gate (default)

##@ Quality Gates
check: vet test lint markdownlint          ## Quick gate (before every commit)
check-full: vet test test-integration coverage lint markdownlint build test-e2e  ## Full gate (before PR)

##@ Testing
vet:                                       ## Run go vet
	go vet ./...

test:                                      ## Unit tests (race detector on)
	go test -race -count=1 ./...

test-integration:                          ## Integration tests (race detector on)
	go test -race -count=1 -tags integration ./...

test-e2e: build                            ## E2E tests (compiles binary first)
	go test -tags e2e ./e2e/...

coverage:                                  ## Engine coverage (fails below 90%)
	go test -cover -coverprofile=coverage.out ./internal/engine/...
	@go tool cover -func=coverage.out | awk '/^total:/ { gsub(/%/, "", $$3); if ($$3+0 < 90) { print "FAIL: engine coverage " $$3 "% < 90%"; exit 1 } else { print "OK: engine coverage " $$3 "%" } }'
	@rm -f coverage.out

lint:                                      ## Static analysis (staticcheck)
	staticcheck ./...

markdownlint:                              ## Lint markdown files
	npx markdownlint-cli2 "**/*.md" "#node_modules"

##@ Build
build: build-server build-client build-admin  ## Build all binaries

build-server:                              ## Build the cryptd server binary
	go build -o cryptd ./cmd/cryptd

build-client:                              ## Build the crypt client binary
	go build -o crypt ./cmd/crypt

build-admin:                               ## Build the crypt-admin authoring binary
	go build -o crypt-admin ./cmd/crypt-admin

clean:                                     ## Remove build artifacts
	rm -f cryptd crypt crypt-admin coverage.out

##@ Install
install: build                             ## Install binaries and man pages to PREFIX (/usr/local)
	install -d $(BINDIR) $(MANDIR)
	install -m 755 cryptd $(BINDIR)/cryptd
	install -m 755 crypt $(BINDIR)/crypt
	install -m 755 crypt-admin $(BINDIR)/crypt-admin
	install -m 644 man/man1/cryptd.1 $(MANDIR)/cryptd.1
	install -m 644 man/man1/crypt.1 $(MANDIR)/crypt.1
	install -m 644 man/man1/crypt-admin.1 $(MANDIR)/crypt-admin.1
	install -m 644 man/man1/eval-balance.1 $(MANDIR)/eval-balance.1

uninstall:                                 ## Remove installed binaries and man pages
	rm -f $(BINDIR)/cryptd $(BINDIR)/crypt $(BINDIR)/crypt-admin
	rm -f $(MANDIR)/cryptd.1 $(MANDIR)/crypt.1 $(MANDIR)/crypt-admin.1 $(MANDIR)/eval-balance.1

man:                                       ## View man pages locally (without installing)
	man man/man1/cryptd.1

##@ Play
play: build-server build-client            ## Play the game (client connects to server with default scenario)
	@CRYPT_SCENARIO_DIR=$(SCENARIO_DIR) ./cryptd serve -f --scenario minimal &\
	CRYPTD_PID=$$!; \
	sleep 0.5; \
	./crypt --scenario minimal; \
	kill $$CRYPTD_PID 2>/dev/null; wait $$CRYPTD_PID 2>/dev/null

play-unix: build-server build-client       ## Play the UNIX Catacombs scenario (9 rooms)
	@CRYPT_SCENARIO_DIR=scenarios ./cryptd serve -f --scenario unix-catacombs &\
	CRYPTD_PID=$$!; \
	sleep 0.5; \
	./crypt --scenario unix-catacombs; \
	kill $$CRYPTD_PID 2>/dev/null; wait $$CRYPTD_PID 2>/dev/null

play-mega: build-server build-client       ## Play the UNIX Megadungeon (53 rooms)
	@CRYPT_SCENARIO_DIR=scenarios ./cryptd serve -f --scenario unix-megadungeon &\
	CRYPTD_PID=$$!; \
	sleep 0.5; \
	./crypt --scenario unix-megadungeon; \
	kill $$CRYPTD_PID 2>/dev/null; wait $$CRYPTD_PID 2>/dev/null

##@ Demos — Scripted Playthroughs
demo: demo-exploration demo-inventory demo-combat demo-save-load demo-unix-catacombs  ## Run all CLI demos

demo-exploration: build-server              ## Explore, take items, navigate, help, quit
	CRYPT_SCENARIO_DIR=$(SCENARIO_DIR) ./cryptd serve -t --scenario minimal --script $(DEMO_DIR)/full-run.txt

demo-inventory: build-server               ## Take, examine, equip, unequip, drop
	CRYPT_SCENARIO_DIR=$(SCENARIO_DIR) ./cryptd serve -t --scenario minimal --script $(DEMO_DIR)/pick-up-item.txt

demo-combat: build-server                  ## Equip sword, fight goblin, gain XP
	CRYPT_SCENARIO_DIR=$(SCENARIO_DIR) ./cryptd serve -t --scenario minimal --script $(DEMO_DIR)/combat-walkthrough.txt

demo-save-load: build-server               ## Save to slot, reload
	CRYPT_SCENARIO_DIR=$(SCENARIO_DIR) ./cryptd serve -t --scenario minimal --script $(DEMO_DIR)/save-and-reload.txt

demo-unix-catacombs: build-server          ## Full 9-room UNIX-themed dungeon crawl
	CRYPT_SCENARIO_DIR=$(SCENARIO_DIR) ./cryptd serve -t --scenario unix-catacombs --script $(DEMO_DIR)/unix-catacombs.txt

##@ Demos — Advanced
demo-solo: build-server                    ## Interactive solo mode (rules+templates, cryptd serve -t)
	CRYPT_SCENARIO_DIR=$(SCENARIO_DIR) ./cryptd serve -t --scenario minimal

##@ Ollama
SLM_MODEL = gemma3:1b
OLLAMA    = $(shell command -v ollama 2>/dev/null)

ollama-install:                            ## Install ollama via Homebrew
	@if command -v ollama >/dev/null 2>&1; then echo "ollama already installed"; else brew install ollama; fi

ollama-start: ollama-install               ## Start ollama server (background)
	@if pgrep -x ollama > /dev/null 2>&1; then echo "ollama already running"; else \
		GIN_MODE=release ollama serve > /dev/null 2>&1 & \
		printf "waiting for ollama"; \
		for i in 1 2 3 4 5 6 7 8 9 10; do \
			if ollama list > /dev/null 2>&1; then echo " ready"; break; fi; \
			printf "."; sleep 1; \
		done; \
	fi

ollama-pull: ollama-start                  ## Pull the preferred SLM model
	@ollama pull $(SLM_MODEL) > /dev/null 2>&1 && echo "model $(SLM_MODEL) ready"

ollama-setup: ollama-pull                  ## Install ollama, start server, pull model

eval-slm: ollama-setup build              ## Run SLM accuracy eval (65+ inputs, needs ollama)
	go run ./cmd/eval-slm --model $(SLM_MODEL)

##@ Scenario Generation
GENERATED_DIR = .tmp/generated-scenario
TREE_SOURCE   = .tmp/dungeon-tree

generate-dungeon: build-admin              ## Generate a scenario from .tmp/dungeon-tree/
	./crypt-admin generate --topology tree --source $(TREE_SOURCE) --title "Generated Dungeon" --output $(GENERATED_DIR)/

validate-dungeon: build-admin              ## Validate the generated scenario
	./crypt-admin validate $(GENERATED_DIR)/

##@ Balance
eval-balance:                              ## Run monkey balance eval (1000 sessions, all classes, minimal)
	CRYPT_SCENARIO_DIR=$(SCENARIO_DIR) go run ./cmd/eval-balance --scenario minimal --class all --players 1000 --max-moves 200

eval-balance-quick:                        ## Quick balance check (100 sessions, fighter only)
	CRYPT_SCENARIO_DIR=$(SCENARIO_DIR) go run ./cmd/eval-balance --scenario minimal --class fighter --players 100 --max-moves 50

eval-balance-generated:                    ## Balance eval on generated scenario (200 sessions)
	CRYPT_SCENARIO_DIR=.tmp go run ./cmd/eval-balance --scenario generated-scenario --class all --players 200 --max-moves 300

eval-balance-unix:                         ## Balance eval on unix-catacombs (500 sessions)
	CRYPT_SCENARIO_DIR=scenarios go run ./cmd/eval-balance --scenario unix-catacombs --class all --players 500 --max-moves 200

##@ Help
help:                                      ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } \
		/^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""
