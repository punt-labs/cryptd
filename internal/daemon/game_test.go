package daemon

import (
	"bufio"
	"context"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/interpreter"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/narrator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGame_SendDuringRunLoop(t *testing.T) {
	// Use a normal-mode server with real interpreter/narrator so RunLoop
	// doesn't panic on nil dependencies.
	scenarioDir := filepath.Join(repoRoot(t), "testdata", "scenarios")
	rules := interpreter.NewRules()
	tmpl := narrator.NewTemplate()
	interp := interpreter.NewSLM(nil, rules)
	narr := narrator.NewSLM(nil, tmpl)
	srv := NewServer(
		filepath.Join(t.TempDir(), "test.sock"),
		scenarioDir,
		WithInterpreter(interp),
		WithNarrator(narr),
	)

	// Create and start a game, send new_game so eng/state are initialized.
	g, err := srv.createAndStartGame()
	require.NoError(t, err)

	ctx := context.Background()
	newGameArgs := `{"scenario_id":"minimal","character_name":"Tester","character_class":"fighter"}`
	_, rpcErr := g.Send(ctx, "new_game", []byte(newGameArgs))
	require.Nil(t, rpcErr, "new_game failed: %+v", rpcErr)

	// Start RunLoop in a goroutine with a pipe we control.
	// The RunLoop blocks reading from the RPCRenderer scanner, so the game
	// goroutine is busy and cannot process Send commands.
	pipeR, pipeW := io.Pipe()
	defer func() { _ = pipeW.Close() }()

	scanner := bufio.NewScanner(pipeR)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	loopDone := make(chan struct{})
	go func() {
		defer close(loopDone)
		g.RunLoop(ctx, &RunLoopRequest{
			Scanner: scanner,
			Writer:  io.Discard,
			Interp:  interp,
			Narr:    narr,
		})
	}()

	// Give the RunLoop command time to be picked up by the game goroutine.
	time.Sleep(50 * time.Millisecond)

	// A concurrent Send should not hang — it should fail with context deadline
	// because the game goroutine is busy with RunLoop and not reading commands.
	sendCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	_, sendErr := g.Send(sendCtx, "look", nil)

	require.NotNil(t, sendErr, "Send should fail when RunLoop is active")
	assert.Contains(t, sendErr.Message, "context cancelled")

	// Clean up: close the pipe to make the scanner return EOF,
	// which causes RunLoop to exit.
	_ = pipeW.Close()
	// Cancel server context to ensure game goroutine exits cleanly.
	srv.cancel()
	<-loopDone
}

func TestGame_Stop(t *testing.T) {
	srv := testServer(t)

	g, err := srv.createAndStartGame()
	require.NoError(t, err)

	ctx := context.Background()
	newGameArgs := `{"scenario_id":"minimal","character_name":"Tester","character_class":"fighter"}`
	_, rpcErr := g.Send(ctx, "new_game", []byte(newGameArgs))
	require.Nil(t, rpcErr, "new_game failed: %+v", rpcErr)

	// Verify done channel blocks (game goroutine is alive).
	select {
	case <-g.done:
		t.Fatal("game goroutine should be running, but done channel is closed")
	default:
		// expected: game goroutine is alive
	}

	// Stop the game.
	g.Stop()

	// Verify done channel is closed (goroutine exited).
	select {
	case <-g.done:
		// expected: goroutine stopped
	default:
		t.Fatal("game goroutine should have stopped, but done channel is still open")
	}

	// Subsequent Send panics with "send on closed channel" because Stop
	// closes the commands channel. Verify via the done channel instead.
	select {
	case <-g.done:
		// expected: game goroutine terminated
	default:
		t.Fatal("expected done channel to be closed after Stop")
	}
}

func TestGame_StopDuringRunLoop(t *testing.T) {
	scenarioDir := filepath.Join(repoRoot(t), "testdata", "scenarios")
	rules := interpreter.NewRules()
	tmpl := narrator.NewTemplate()
	interp := interpreter.NewSLM(nil, rules)
	narr := narrator.NewSLM(nil, tmpl)

	ctx := context.Background()

	srv := NewServer(
		filepath.Join(t.TempDir(), "test.sock"),
		scenarioDir,
		WithInterpreter(interp),
		WithNarrator(narr),
	)

	g, err := srv.createAndStartGame()
	require.NoError(t, err)

	newGameArgs := `{"scenario_id":"minimal","character_name":"Tester","character_class":"fighter"}`
	_, rpcErr := g.Send(ctx, "new_game", []byte(newGameArgs))
	require.Nil(t, rpcErr, "new_game failed: %+v", rpcErr)

	// Start RunLoop with a pipe we control. The loop blocks on scanner.Scan().
	pipeR, pipeW := io.Pipe()

	scanner := bufio.NewScanner(pipeR)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	loopDone := make(chan struct{})
	go func() {
		defer close(loopDone)
		g.RunLoop(ctx, &RunLoopRequest{
			Scanner: scanner,
			Writer:  io.Discard,
			Interp:  interp,
			Narr:    narr,
		})
	}()

	// Give the RunLoop command time to be picked up by the game goroutine.
	time.Sleep(50 * time.Millisecond)

	// Close the pipe to make scanner.Scan() return false, which causes
	// RunLoop's game loop to exit.
	_ = pipeW.Close()

	// Wait for RunLoop to finish. This ensures the game goroutine is back
	// in its select loop, so Stop can safely close the commands channel
	// without racing with a concurrent send.
	select {
	case <-loopDone:
		// RunLoop returned
	case <-time.After(5 * time.Second):
		t.Fatal("RunLoop did not return within 5 seconds after pipe close")
	}

	// Now Stop should return immediately because the game goroutine is idle.
	stopDone := make(chan struct{})
	go func() {
		defer close(stopDone)
		g.Stop()
	}()

	select {
	case <-stopDone:
		// expected: Stop returned
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() did not return within 5 seconds — possible deadlock")
	}
}

func TestGame_PanicRecovery(t *testing.T) {
	srv := testServer(t)

	// Create and start a game, send new_game so eng/state are initialized.
	g, err := srv.createAndStartGame()
	require.NoError(t, err)

	ctx := context.Background()
	newGameArgs := `{"scenario_id":"minimal","character_name":"Tester","character_class":"fighter"}`
	_, rpcErr := g.Send(ctx, "new_game", []byte(newGameArgs))
	require.Nil(t, rpcErr, "new_game failed: %+v", rpcErr)

	// Send an Inspect that panics. The game goroutine's deferred recover()
	// catches it, logs it, and exits (closing the done channel).
	inspectErr := g.Inspect(ctx, func(eng *engine.Engine, state *model.GameState) {
		panic("deliberate test panic")
	})

	// Inspect should return an error because the game goroutine terminated.
	require.Error(t, inspectErr, "Inspect should return error after panic")
	assert.Contains(t, inspectErr.Error(), "game terminated")

	// Subsequent Send should also fail with "game terminated".
	_, sendErr := g.Send(ctx, "look", nil)
	require.NotNil(t, sendErr, "Send should fail after panic")
	assert.Contains(t, sendErr.Message, "game terminated")
}
