//go:build e2e

package e2e

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func installObservabilityBinaries(t *testing.T, root string) string {
	t.Helper()
	bin := t.TempDir()
	install := exec.CommandContext(t.Context(), "go", "install", "./cmd/cryptd", "./cmd/crypt", "./cmd/crypt-admin")
	install.Dir = root
	install.Env = append(os.Environ(), "GOBIN="+bin)
	out, err := install.CombinedOutput()
	require.NoError(t, err, "%s", out)
	return bin
}

func TestCrypt_AutoStartsDaemonOnDefaultSocket(t *testing.T) {
	root := repoRoot(t)
	bin := installObservabilityBinaries(t, root)
	home, err := os.MkdirTemp("", "cryptd-")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(home) })
	toolDir := filepath.Join(home, ".punt-labs", "cryptd")
	runDir := filepath.Join(toolDir, "run")
	socket := filepath.Join(runDir, "daemon.sock")
	pidPath := filepath.Join(runDir, "cryptd.pid")
	require.NoDirExists(t, runDir)

	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, filepath.Join(bin, "crypt"), "--plain")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(),
		"HOME="+home, "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"CRYPT_SCENARIO_DIR="+filepath.Join(root, "testdata", "scenarios"),
		"_CRYPTD_DAEMON=", "CRYPTD_API_KEY=", "CRYPTD_TELEMETRY=off",
	)
	cmd.Stdin = strings.NewReader("quit\n")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		pgid := -cmd.Process.Pid
		if err := syscall.Kill(pgid, syscall.SIGTERM); err != nil {
			assert.ErrorIs(t, err, syscall.ESRCH)
		}
		pidData, err := os.ReadFile(pidPath)
		if os.IsNotExist(err) {
			log, _ := os.ReadFile(filepath.Join(toolDir, "logs", "cryptd.log"))
			if bytes.Contains(log, []byte("Spawning detached daemon")) {
				for deadline := time.Now().Add(5 * time.Second); os.IsNotExist(err) && time.Now().Before(deadline); {
					time.Sleep(20 * time.Millisecond)
					pidData, err = os.ReadFile(pidPath)
				}
			}
		}
		if !os.IsNotExist(err) {
			require.NoError(t, err)
			pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
			require.NoError(t, err)
			require.Greater(t, pid, 1)
			if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
				assert.ErrorIs(t, err, syscall.ESRCH)
			}
			if !assert.Eventually(t, func() bool {
				_, err := os.Lstat(pidPath)
				return os.IsNotExist(err)
			}, 5*time.Second, 20*time.Millisecond) {
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
		}
		if !assert.Eventually(t, func() bool {
			_, err := os.Lstat(socket)
			return os.IsNotExist(err)
		}, 5*time.Second, 20*time.Millisecond) {
			_ = syscall.Kill(pgid, syscall.SIGKILL)
		}
	})
	require.NoError(t, cmd.Wait(), "%s", output.String())
	require.Contains(t, output.String(), "Farewell")
	info, err := os.Lstat(socket)
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&os.ModeSocket)
	require.NoFileExists(t, pidPath)
}

func TestCryptd_ReportsStartupRotationLockTimeout(t *testing.T) {
	root, home := repoRoot(t), t.TempDir()
	bin := installObservabilityBinaries(t, root)
	scenarioPath := filepath.Join(root, "testdata", "scenarios", "minimal.yaml")
	seed := exec.CommandContext(t.Context(), filepath.Join(bin, "crypt-admin"), "validate", scenarioPath)
	seed.Env = append(os.Environ(), "HOME="+home, "CRYPTD_TELEMETRY=off")
	out, err := seed.CombinedOutput()
	require.NoError(t, err, "%s", out)

	toolDir := filepath.Join(home, ".punt-labs", "cryptd")
	logPath := filepath.Join(toolDir, "logs", "cryptd.log")
	previousLog, err := os.ReadFile(logPath)
	require.NoError(t, err)
	require.NotEmpty(t, previousLog)
	locks, err := filepath.Glob(filepath.Join(filepath.Dir(logPath), "*.lock"))
	require.NoError(t, err)
	require.Len(t, locks, 1, "expected one rotation lock in the fresh logs directory")
	lock, err := os.OpenFile(locks[0], os.O_RDWR, 0)
	require.NoError(t, err, "open the rotation lock created by punt-observe-go")
	t.Cleanup(func() { require.NoError(t, lock.Close()) })
	require.NoError(t, syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB))

	for _, tc := range []struct {
		binary string
		args   []string
	}{
		{"cryptd", []string{"serve", "--socket", filepath.Join(home, "d.sock")}},
		{"crypt", []string{"mcp"}},
		{"crypt-admin", []string{"validate", scenarioPath}},
	} {
		t.Run(tc.binary, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, filepath.Join(bin, tc.binary), tc.args...)
			cmd.Env = append(os.Environ(), "HOME="+home, "_CRYPTD_DAEMON=", "CRYPTD_TELEMETRY=", "PUNT_TELEMETRY=")
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			require.NoError(t, ctx.Err(), "%s hung with the rotation lock held", tc.binary)
			var exit *exec.ExitError
			require.ErrorAs(t, err, &exit, "stderr: %s", stderr.String())
			require.Equal(t, 1, exit.ExitCode())
			require.Empty(t, stdout.String())
			require.Contains(t, stderr.String(), "timed out waiting for the diagnostic log rotation lock")
			require.Contains(t, stderr.String(), "~/.punt-labs/cryptd/logs")
		})
	}
	contents, err := os.ReadFile(logPath)
	require.NoError(t, err)
	require.Equal(t, previousLog, contents)
	require.NoFileExists(t, filepath.Join(toolDir, "telemetry.db"))
}

func TestCryptd_RecordsDiagnosticsAndTelemetry(t *testing.T) {
	root := repoRoot(t)
	home, err := os.MkdirTemp("", "cryptd-")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(home) })
	cwd := t.TempDir()
	bin := installObservabilityBinaries(t, root)

	cmd := func(program string, args ...string) *exec.Cmd {
		c := exec.CommandContext(t.Context(), filepath.Join(bin, program), args...)
		c.Dir = cwd
		c.Env = append(os.Environ(),
			"HOME="+home, "CRYPT_SCENARIO_DIR="+filepath.Join(root, "testdata", "scenarios"),
			"CRYPTD_API_KEY=", "CRYPTD_TELEMETRY=", "PUNT_TELEMETRY=",
		)
		return c
	}
	toolDir := filepath.Join(home, ".punt-labs", "cryptd")
	logDir := filepath.Join(toolDir, "logs")
	logPath := filepath.Join(logDir, "cryptd.log")
	dbPath := filepath.Join(toolDir, "telemetry.db")
	runDir := filepath.Join(toolDir, "run")
	socket := filepath.Join(runDir, "daemon.sock")
	pidPath := filepath.Join(runDir, "cryptd.pid")
	legacy := filepath.Join(home, ".crypt")
	require.NoError(t, os.Mkdir(legacy, 0o700))
	for _, name := range []string{"daemon.sock", "cryptd.pid"} {
		require.NoError(t, os.WriteFile(filepath.Join(legacy, name), []byte("legacy fixture"), 0o600))
	}
	logText := func() string {
		b, _ := os.ReadFile(logPath)
		return string(b)
	}

	launch := cmd("cryptd", "serve")
	out, err := launch.CombinedOutput()
	require.NoError(t, err, "%s", out)
	pidMatch := regexp.MustCompile(`spawned daemon \(PID ([0-9]+)\)`).FindSubmatch(out)
	require.Len(t, pidMatch, 2)
	pid, err := strconv.Atoi(string(pidMatch[1]))
	require.NoError(t, err)
	child, err := os.FindProcess(pid)
	require.NoError(t, err)
	childStopped := false
	t.Cleanup(func() {
		if !childStopped {
			_ = child.Kill()
		}
	})
	require.Eventually(t, func() bool {
		return strings.Contains(logText(), "Game daemon is listening")
	}, 10*time.Second, 20*time.Millisecond)

	pidData, err := os.ReadFile(pidPath)
	require.NoError(t, err)
	require.Equal(t, strconv.Itoa(pid)+"\n", string(pidData))
	info, err := os.Stat(runDir)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o700), info.Mode().Perm())

	const private = "private-fixture@example.invalid"
	conn, err := net.DialTimeout("unix", socket, time.Second)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, conn.SetDeadline(time.Now().Add(5*time.Second)))
	responses := bufio.NewScanner(conn)
	send := func(line string, wantCode int) {
		t.Helper()
		_, err := fmt.Fprintln(conn, line)
		require.NoError(t, err)
		require.True(t, responses.Scan(), "missing daemon response: %v", responses.Err())
		var resp struct {
			Error *struct{ Code int }
		}
		require.NoError(t, json.Unmarshal(responses.Bytes(), &resp))
		if wantCode == 0 {
			require.Nil(t, resp.Error, "%s", responses.Bytes())
		} else {
			require.NotNil(t, resp.Error)
			require.Equal(t, wantCode, resp.Error.Code)
		}
	}
	send(private, -32700)
	send(`{"jsonrpc":"2.0","id":1,"method":"session.init","params":{"mode":"passthrough","session_id":"`+private+`"}}`, 0)
	send(`{"jsonrpc":"2.0","id":2,"method":"game.new","params":{"scenario_id":"missing","character_name":"`+private+`","character_class":"fighter"}}`, -32602)
	send(`{"jsonrpc":"2.0","id":3,"method":"game.new","params":{"scenario_id":"minimal","character_name":"`+private+`","character_class":"fighter"}}`, 0)

	mcp := cmd("crypt", "mcp")
	mcp.Stdin = strings.NewReader(private + "\n" +
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"smoke","version":"1"}}}` + "\n")
	mcp.Stderr = nil // Equivalent to an MCP host discarding stderr.
	out, err = mcp.Output()
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	require.Len(t, lines, 2)
	for _, line := range lines {
		require.True(t, json.Valid([]byte(line)), "non-protocol MCP stdout: %q", line)
	}
	require.Contains(t, lines[0], `"code":-32700`)
	require.Contains(t, lines[1], `"serverInfo"`)

	for _, sig := range []os.Signal{os.Interrupt, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			bridge := exec.CommandContext(ctx, mcp.Path, mcp.Args[1:]...)
			bridge.Dir, bridge.Env = mcp.Dir, mcp.Env
			stdin, err := bridge.StdinPipe()
			require.NoError(t, err)
			stdout, err := bridge.StdoutPipe()
			require.NoError(t, err)
			var stderr bytes.Buffer
			bridge.Stderr = &stderr
			require.NoError(t, bridge.Start())
			defer func() {
				if bridge.ProcessState == nil {
					_ = bridge.Process.Kill()
					_ = bridge.Wait()
				}
			}()
			_, err = fmt.Fprintln(stdin, `{"jsonrpc":"2.0","id":1,"method":"ping"}`)
			require.NoError(t, err)
			response := bufio.NewScanner(stdout)
			require.True(t, response.Scan(), "missing MCP ping reply: %v", response.Err())
			require.JSONEq(t, `{"jsonrpc":"2.0","id":1,"result":{}}`, response.Text())
			require.NoError(t, bridge.Process.Signal(sig))
			require.NoError(t, bridge.Wait(), "stderr: %s", stderr.String())
			require.Empty(t, stderr.String())
		})
	}

	play := cmd("crypt", "--plain")
	play.Stdin = strings.NewReader("quit\n")
	out, err = play.CombinedOutput()
	require.NoError(t, err, "%s", out)
	require.Contains(t, string(out), "Farewell")

	script := filepath.Join(home, "play.txt")
	require.NoError(t, os.WriteFile(script, []byte("quit\n"), 0o600))
	out, err = cmd("cryptd", "serve", "-t", "--scenario", "minimal", "--name", private, "--script", script, "--json").Output()
	require.NoError(t, err)
	require.True(t, json.Valid(out))

	scenarioPath := filepath.Join(root, "testdata", "scenarios", "minimal.yaml")
	out, err = cmd("crypt-admin", "validate", scenarioPath).CombinedOutput()
	require.NoError(t, err, "%s", out)
	require.Equal(t, "OK\n", string(out))
	_, err = cmd("crypt-admin", "validate", private).Output()
	require.Error(t, err)

	for _, name := range []string{"CRYPTD_TELEMETRY", "PUNT_TELEMETRY"} {
		offHome := t.TempDir()
		off := cmd("crypt-admin", "validate", scenarioPath)
		off.Env = append(off.Env, "HOME="+offHome, name+"=off")
		out, err = off.CombinedOutput()
		require.NoError(t, err, "%s", out)
		require.NoFileExists(t, filepath.Join(offHome, ".punt-labs", "cryptd", "telemetry.db"))
		require.FileExists(t, filepath.Join(offHome, ".punt-labs", "cryptd", "logs", "cryptd.log"))
	}

	require.NoError(t, child.Signal(os.Interrupt))
	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	require.NoError(t, err)
	defer db.Close()
	waitForDaemonRows := func(want int) {
		t.Helper()
		require.Eventually(t, func() bool {
			var n int
			err := db.QueryRow("SELECT count(*) FROM signals WHERE name = 'cryptd.daemon.serve'").Scan(&n)
			return err == nil && n == want
		}, 10*time.Second, 20*time.Millisecond)
	}
	waitForDaemonRows(1)
	childStopped = true
	require.NoFileExists(t, pidPath)
	require.NoFileExists(t, socket)

	badSocket := filepath.Join(home, "not-a-directory", "d.sock")
	require.NoError(t, os.WriteFile(filepath.Dir(badSocket), nil, 0o600))
	out, err = cmd("cryptd", "serve", "--socket", badSocket).CombinedOutput()
	require.NoError(t, err, "%s", out) // Spawn success is not listener readiness.
	waitForDaemonRows(2)
	require.NoFileExists(t, pidPath)

	log := logText()
	require.Contains(t, log, "[WARNING] cryptd.daemon: Rejected malformed JSON-RPC request")
	require.Contains(t, log, "[WARNING] cryptd.mcp: Rejected MCP request")
	require.Equal(t, 3, strings.Count(log, "[INFO] cryptd.mcp: MCP bridge stopped outcome=ok"))
	require.NotContains(t, log, "MCP bridge stopped after a transport failure")
	require.Regexp(t, `\[ERROR\] cryptd.daemon: Game daemon stopped after a listener failure[^\n]*error_kind=not_a_directory`, log)
	require.Contains(t, log, "Finished scenario administration command")
	require.NotContains(t, log, private)
	require.NotContains(t, log, home)
	require.NoFileExists(t, filepath.Join(home, ".crypt", "cryptd.log"))
	for _, name := range []string{"daemon.sock", "cryptd.pid"} {
		data, err := os.ReadFile(filepath.Join(legacy, name))
		require.NoError(t, err)
		require.Equal(t, "legacy fixture", string(data))
	}
	launches := regexp.MustCompile(`Spawning detached daemon.*launch_id=([0-9a-f]{32})`).FindAllStringSubmatch(log, -1)
	require.Len(t, launches, 2, "%s", log)
	for i, stages := range [][]string{
		{"Starting game daemon", "Game daemon is listening", "Game daemon stopped"},
		{"Starting game daemon", "Game daemon stopped after a listener failure"},
	} {
		for _, stage := range stages {
			require.Regexp(t, stage+`[^\n]*launch_id=`+launches[i][1], log)
		}
	}

	for path, mode := range map[string]os.FileMode{logDir: 0o700, logPath: 0o600, dbPath: 0o600} {
		info, err := os.Stat(path)
		require.NoError(t, err)
		require.Equal(t, mode, info.Mode().Perm(), "%s", path)
	}

	rows, err := db.Query("SELECT signal, name, schema_version, end_ns - start_ns, attributes FROM signals")
	require.NoError(t, err)
	defer rows.Close()
	outcomes := make(map[string][]string)
	for rows.Next() {
		var signal, name, attrs string
		var version int
		var duration int64
		require.NoError(t, rows.Scan(&signal, &name, &version, &duration, &attrs))
		require.Equal(t, "span", signal)
		require.Equal(t, 1, version)
		require.GreaterOrEqual(t, duration, int64(0))
		require.NotContains(t, attrs, private)
		var fields map[string]string
		require.NoError(t, json.Unmarshal([]byte(attrs), &fields))
		require.Len(t, fields, 1)
		outcomes[name] = append(outcomes[name], fields["outcome"])
	}
	require.NoError(t, rows.Err())
	expected := map[string][]string{
		"cryptd.daemon.serve":    {"failed", "ok"},
		"cryptd.daemon.new_game": {"failed", "ok"},
		"cryptd.launcher.spawn":  {"ok", "ok"},
		"cryptd.cli.play":        {"ok"},
		"cryptd.cli.headless":    {"ok"},
		"cryptd.mcp.serve":       {"ok", "ok", "ok"},
		"cryptd.admin.validate":  {"failed", "ok"},
	}
	require.Len(t, outcomes, len(expected))
	for operation, want := range expected {
		require.ElementsMatch(t, want, outcomes[operation], operation)
	}
}
