//go:build acceptance

package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/punt-labs/cryptd/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runScript runs cryptd serve -t with a script file and returns
// the JSON transcript entries.
func runScript(t *testing.T, bin, root, script string) []renderer.TranscriptEntry {
	t.Helper()
	scriptPath := filepath.Join(root, "testdata", "demos", script)
	cmd := exec.Command(bin, "serve", "-t",
		"--scenario", "minimal",
		"--script", scriptPath,
		"--json",
	)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CRYPT_SCENARIO_DIR=testdata/scenarios")

	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("cryptd serve -t exited %d; stderr: %s", ee.ExitCode(), ee.Stderr)
		}
		t.Fatalf("cryptd serve -t failed: %v", err)
	}

	var entries []renderer.TranscriptEntry
	require.NoError(t, json.Unmarshal(out, &entries))
	return entries
}

func TestAcceptance_FullRun(t *testing.T) {
	bin := serverBinary(t)
	root := repoRoot(t)
	entries := runScript(t, bin, root, "full-run.txt")

	// Should have multiple transcript entries and end with quit.
	require.True(t, len(entries) > 3, "expected >3 transcript entries, got %d", len(entries))

	// First entry is initial room description (no command).
	assert.Equal(t, "entrance", entries[0].Room)
	assert.Empty(t, entries[0].Command)

	// The "go south" command should move to goblin_lair.
	var movedSouth bool
	for _, e := range entries {
		if e.Command == "go south" && e.Room == "goblin_lair" {
			movedSouth = true
			break
		}
	}
	assert.True(t, movedSouth, "expected to reach goblin_lair via 'go south'")

	// Quit should be the last command.
	last := entries[len(entries)-1]
	assert.Equal(t, "quit", last.Command)
	assert.Contains(t, last.Response, "Farewell")
}

func TestAcceptance_CombatWalkthrough(t *testing.T) {
	bin := serverBinary(t)
	root := repoRoot(t)
	entries := runScript(t, bin, root, "combat-walkthrough.txt")

	require.True(t, len(entries) > 5, "expected >5 transcript entries")

	// Should reach goblin_lair and see combat start.
	var reachedGoblinLair bool
	var combatStarted bool
	for _, e := range entries {
		if e.Command == "south" && e.Room == "goblin_lair" {
			reachedGoblinLair = true
		}
		if strings.Contains(e.Response, "Combat begins") {
			combatStarted = true
		}
	}
	assert.True(t, reachedGoblinLair, "expected to reach goblin_lair via 'south'")
	assert.True(t, combatStarted, "expected combat start message in transcript")

	// Should see attack commands in transcript.
	var sawAttack bool
	for _, e := range entries {
		if e.Command == "attack" {
			sawAttack = true
			break
		}
	}
	assert.True(t, sawAttack, "expected attack commands in transcript")
}

func TestAcceptance_PickUpItem(t *testing.T) {
	bin := serverBinary(t)
	root := repoRoot(t)
	entries := runScript(t, bin, root, "pick-up-item.txt")

	require.True(t, len(entries) > 3)

	// Should see "pick up" response.
	var pickedUp bool
	for _, e := range entries {
		if e.Command == "take short sword" {
			assert.Contains(t, e.Response, "Short Sword")
			pickedUp = true
			break
		}
	}
	assert.True(t, pickedUp, "expected to pick up Short Sword")

	// Should see equip response.
	var equipped bool
	for _, e := range entries {
		if e.Command == "equip short sword" {
			assert.Contains(t, e.Response, "Short Sword")
			equipped = true
			break
		}
	}
	assert.True(t, equipped, "expected to equip Short Sword")
}

func TestAcceptance_Combat(t *testing.T) {
	bin := serverBinary(t)
	root := repoRoot(t)
	entries := runScript(t, bin, root, "combat.txt")

	require.True(t, len(entries) > 5)

	// Should see combat victory.
	var victory bool
	for _, e := range entries {
		if strings.Contains(e.Response, "victorious") || strings.Contains(e.Response, "defeat") {
			victory = true
			break
		}
	}
	assert.True(t, victory, "expected combat victory in transcript")
}

func TestAcceptance_SaveAndReload(t *testing.T) {
	bin := serverBinary(t)
	root := repoRoot(t)

	// Run in a temp dir so saves don't pollute the repo.
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(root, "testdata", "demos", "save-and-reload.txt")
	cmd := exec.Command(bin, "serve", "-t",
		"--scenario", "minimal",
		"--script", scriptPath,
		"--json",
	)
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(),
		"CRYPT_SCENARIO_DIR="+filepath.Join(root, "testdata", "scenarios"),
	)

	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("cryptd serve -t exited %d; stderr: %s", ee.ExitCode(), ee.Stderr)
		}
		t.Fatalf("cryptd serve -t failed: %v", err)
	}

	var entries []renderer.TranscriptEntry
	require.NoError(t, json.Unmarshal(out, &entries))
	require.True(t, len(entries) > 2)

	// Should see save confirmation.
	var saved bool
	for _, e := range entries {
		if e.Command == "save test-slot" {
			assert.Contains(t, e.Response, "test-slot")
			saved = true
			break
		}
	}
	assert.True(t, saved, "expected save confirmation")

	// Should see load confirmation.
	var loaded bool
	for _, e := range entries {
		if e.Command == "load test-slot" {
			assert.Contains(t, e.Response, "test-slot")
			loaded = true
			break
		}
	}
	assert.True(t, loaded, "expected load confirmation")
}
