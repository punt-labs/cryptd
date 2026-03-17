//go:build integration

package game_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/game"
	"github.com/punt-labs/cryptd/internal/interpreter"
	"github.com/punt-labs/cryptd/internal/lux"
	"github.com/punt-labs/cryptd/internal/narrator"
	"github.com/punt-labs/cryptd/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLuxDisplay_RealRoundTrip(t *testing.T) {
	sockPath := lux.DefaultSocketPath()
	if _, err := os.Stat(sockPath); err != nil {
		t.Skipf("No Lux display at %s — skipping real E2E test", sockPath)
	}

	client := lux.NewClient(
		lux.WithConnectTimeout(3*time.Second),
		lux.WithRecvTimeout(2*time.Second),
	)
	require.NoError(t, client.Connect())

	display := lux.NewDisplay(client, "cryptd-test", &lux.ShowOpts{
		FrameID:    "cryptd",
		FrameTitle: "Crypt",
		FrameSize:  [2]int{400, 600},
	})
	defer display.Close()

	luxRenderer := renderer.NewLux(display)
	eng := engine.New(loadScenario(t))
	interp := interpreter.NewRules()
	narr := narrator.NewTemplate()

	state := newState(t, eng)

	// Run with a short timeout — the test verifies show() succeeds,
	// not that we can play a full game. Context timeout is the exit.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := game.NewLoop(eng, interp, narr, luxRenderer).Run(ctx, &state)

	// Context timeout is expected — we didn't inject a quit event.
	if err != nil {
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	}

	// Verify no write errors from the display transport.
	assert.NoError(t, display.WriteErr())
}
