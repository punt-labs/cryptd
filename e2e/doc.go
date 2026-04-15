//go:build e2e || acceptance

// Package e2e contains end-to-end and acceptance tests that build the
// cryptd server binary and drive it via subprocess (stdin/stdout in -t
// mode or Unix socket in daemon mode). Run with:
//
//	go test -tags e2e ./e2e/...
//	go test -tags acceptance ./e2e/...
package e2e
