.PHONY: vet test test-integration test-e2e lint markdownlint coverage check check-full build clean

# Quick quality gate (before every commit)
check: vet test lint markdownlint

# Full quality gate (before PR)
check-full: vet test test-integration coverage lint markdownlint build test-e2e

# Individual targets
vet:
	go vet ./...

test:
	go test -race -count=1 ./...

test-integration:
	go test -race -count=1 -tags integration ./...

test-e2e: build
	go test -tags e2e ./e2e/...

lint:
	staticcheck ./...

markdownlint:
	npx markdownlint-cli2 "**/*.md" "#node_modules"

coverage:
	go test -cover -coverprofile=coverage.out ./internal/engine/...
	@go tool cover -func=coverage.out | awk '/^total:/ { gsub(/%/, "", $$3); if ($$3+0 < 90) { print "FAIL: engine coverage " $$3 "% < 90%"; exit 1 } else { print "OK: engine coverage " $$3 "%" } }'
	@rm -f coverage.out

build:
	go build -o cryptd ./cmd/crypt

clean:
	rm -f cryptd coverage.out
