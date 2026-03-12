.PHONY: help vet test test-integration test-e2e lint markdownlint coverage \
       check check-full build clean help \
       ollama-install ollama-start ollama-pull ollama-setup eval-slm \
       demo demo-solo demo-lux demo-exploration demo-inventory \
       demo-combat demo-save-load demo-unix-catacombs

SCENARIO_DIR = testdata/scenarios
DEMO_DIR     = testdata/demos

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
build:                                     ## Build the cryptd binary
	go build -o cryptd ./cmd/crypt

clean:                                     ## Remove build artifacts
	rm -f cryptd coverage.out

##@ Demos — Scripted Playthroughs
demo: demo-exploration demo-inventory demo-combat demo-save-load demo-unix-catacombs  ## Run all CLI demos

demo-exploration: build                    ## Explore, take items, navigate, help, quit
	CRYPT_SCENARIO_DIR=$(SCENARIO_DIR) ./cryptd autoplay --scenario minimal --script $(DEMO_DIR)/full-run.txt

demo-inventory: build                      ## Take, examine, equip, unequip, drop
	CRYPT_SCENARIO_DIR=$(SCENARIO_DIR) ./cryptd autoplay --scenario minimal --script $(DEMO_DIR)/pick-up-item.txt

demo-combat: build                         ## Equip sword, fight goblin, gain XP
	CRYPT_SCENARIO_DIR=$(SCENARIO_DIR) ./cryptd autoplay --scenario minimal --script $(DEMO_DIR)/combat-walkthrough.txt

demo-save-load: build                      ## Save to slot, reload
	CRYPT_SCENARIO_DIR=$(SCENARIO_DIR) ./cryptd autoplay --scenario minimal --script $(DEMO_DIR)/save-and-reload.txt

demo-unix-catacombs: build                 ## Full 9-room UNIX-themed dungeon crawl
	CRYPT_SCENARIO_DIR=$(SCENARIO_DIR) ./cryptd autoplay --scenario unix-catacombs --script $(DEMO_DIR)/unix-catacombs.txt

##@ Demos — Advanced
demo-solo: ollama-setup build              ## Interactive solo mode with SLM narration
	CRYPT_SCENARIO_DIR=$(SCENARIO_DIR) ./cryptd solo --scenario minimal

demo-lux: build                            ## Lux JSON-lines wire format (show/update protocol)
	@echo '{"type":"input","payload":"look"}' > /tmp/cryptd-lux-input.jsonl
	@echo '{"type":"input","payload":"take short sword"}' >> /tmp/cryptd-lux-input.jsonl
	@echo '{"type":"input","payload":"equip short sword"}' >> /tmp/cryptd-lux-input.jsonl
	@echo '{"type":"input","payload":"go south"}' >> /tmp/cryptd-lux-input.jsonl
	@echo '{"type":"input","payload":"attack"}' >> /tmp/cryptd-lux-input.jsonl
	@echo '{"type":"input","payload":"attack"}' >> /tmp/cryptd-lux-input.jsonl
	@echo '{"type":"input","payload":"attack"}' >> /tmp/cryptd-lux-input.jsonl
	@echo '{"type":"input","payload":"attack"}' >> /tmp/cryptd-lux-input.jsonl
	@echo '{"type":"quit"}' >> /tmp/cryptd-lux-input.jsonl
	CRYPT_SCENARIO_DIR=$(SCENARIO_DIR) ./cryptd solo --scenario minimal --lux < /tmp/cryptd-lux-input.jsonl 2>/dev/null
	@rm -f /tmp/cryptd-lux-input.jsonl

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
	ollama pull $(SLM_MODEL)

ollama-setup: ollama-pull                  ## Install ollama, start server, pull model

eval-slm: ollama-setup build              ## Run SLM accuracy eval (65+ inputs, needs ollama)
	go run ./cmd/eval-slm --model $(SLM_MODEL)

##@ Help
help:                                      ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } \
		/^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""
