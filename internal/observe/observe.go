// Package observe configures cryptd's shared diagnostics and usage capture.
// See DESIGN.md (DES-035) for the sink, schema, and failure-handling decisions.
package observe

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/punt-labs/punt-observe-go/logging"
	"github.com/punt-labs/punt-observe-go/telemetry"
)

const Tool = "cryptd"

type Operation string

const (
	Play          Operation = "cryptd.cli.play"
	Headless      Operation = "cryptd.cli.headless"
	Validate      Operation = "cryptd.cli.validate"
	ServeMCP      Operation = "cryptd.mcp.serve"
	SpawnDaemon   Operation = "cryptd.launcher.spawn"
	ServeDaemon   Operation = "cryptd.daemon.serve"
	NewGame       Operation = "cryptd.daemon.new_game"
	AdminGenerate Operation = "cryptd.admin.generate"
	AdminValidate Operation = "cryptd.admin.validate"
	AdminExport   Operation = "cryptd.admin.export"
)

const (
	succeeded telemetry.EnumMember = "ok"
	failed    telemetry.EnumMember = "failed"
)

func schemas() []telemetry.Schema {
	var result []telemetry.Schema
	for _, op := range [...]Operation{
		Play, Headless, Validate, ServeMCP, SpawnDaemon, ServeDaemon, NewGame,
		AdminGenerate, AdminValidate, AdminExport,
	} {
		result = append(result, telemetry.Schema{
			Name: string(op), Version: 1, Signal: telemetry.Span,
			Fields: map[string]telemetry.Field{
				"outcome": {Kind: telemetry.Enum, Members: []telemetry.EnumMember{succeeded, failed}, Required: true},
			},
		})
	}
	return result
}

// Runtime owns the logger and usage client for a single process.
type Runtime struct {
	Log     *slog.Logger
	capture *telemetry.Client
}

// Run opens diagnostics at the entry point and flushes usage before the caller exits.
// An empty operation configures diagnostics only, for argument errors.
func Run(role string, op Operation, run func(*Runtime) int) int {
	h, err := logging.New(Tool, role)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Fprintln(os.Stderr, "cryptd: timed out waiting for the diagnostic log rotation lock in ~/.punt-labs/cryptd/logs; retry after the lock is released")
		} else {
			fmt.Fprintf(os.Stderr, "cryptd: cannot initialize private diagnostic log (%s); check ~/.punt-labs/cryptd/logs\n", ErrorKind(err))
		}
		return 1
	}
	defer h.Close()
	log := slog.New(h).With("component", Tool+"."+role)
	runtime := &Runtime{Log: log}
	if op != "" {
		runtime.capture = telemetry.New(Tool, telemetry.Config{Diagnostics: log}, schemas()...)
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			runtime.capture.Shutdown(ctx)
		}()
	}
	started := time.Now()
	code := run(runtime)
	if op != "" {
		runtime.Record(op, started, code == 0)
	}
	return code
}

// ErrorKind classifies errors without retaining paths, addresses or payloads.
func ErrorKind(err error) string {
	var response interface{ HTTPStatusCode() int }
	if errors.As(err, &response) {
		return fmt.Sprintf("http_%d", response.HTTPStatusCode())
	}
	switch {
	case errors.Is(err, os.ErrPermission):
		return "permission_denied"
	case errors.Is(err, os.ErrNotExist):
		return "not_found"
	case errors.Is(err, syscall.ENOTDIR):
		return "not_a_directory"
	case errors.Is(err, syscall.EADDRINUSE):
		return "address_in_use"
	case errors.Is(err, syscall.ECONNREFUSED):
		return "connection_refused"
	case errors.Is(err, syscall.ENOSPC):
		return "disk_full"
	case os.IsTimeout(err) || errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.As(err, new(*json.SyntaxError)), errors.As(err, new(*json.UnmarshalTypeError)):
		return "invalid_json"
	}
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return fmt.Sprintf("errno_%d", errno)
	}
	return "other"
}

// LogPanic records the fault location without recording panic values or stack arguments.
func LogPanic(log *slog.Logger, message string, value any) {
	var pcs [8]uintptr
	frames := runtime.CallersFrames(pcs[:runtime.Callers(3, pcs[:])])
	for {
		frame, more := frames.Next()
		if frame.Function != "" && !strings.HasPrefix(frame.Function, "runtime.") {
			log.Error(message, "panic_type", fmt.Sprintf("%T", value), "function", frame.Function, "line", frame.Line)
			return
		}
		if !more {
			log.Error(message, "panic_type", fmt.Sprintf("%T", value))
			return
		}
	}
}

// Record captures only a declared operation's duration and outcome.
func (r *Runtime) Record(op Operation, started time.Time, ok bool) {
	outcome := failed
	if ok {
		outcome = succeeded
	}
	r.capture.Record(telemetry.Observation{
		Name: string(op), Version: 1, Started: started, Duration: time.Since(started),
		Attributes: []telemetry.Value{telemetry.EnumValue("outcome", outcome)},
	})
}

const LaunchIDEnv = "_CRYPTD_LAUNCH_ID"

func NewLaunchID() string {
	var id [16]byte
	rand.Read(id[:])
	return hex.EncodeToString(id[:])
}

// ChildLogger adds only a well-formed parent correlation ID from the environment.
func ChildLogger(log *slog.Logger) *slog.Logger {
	id := os.Getenv(LaunchIDEnv)
	if len(id) == 32 {
		if decoded, err := hex.DecodeString(id); err == nil {
			return log.With("launch_id", hex.EncodeToString(decoded))
		}
	}
	return log
}

// LoggerOrDiscard lets library callers opt out without writing to standard streams.
func LoggerOrDiscard(loggers []*slog.Logger) *slog.Logger {
	if len(loggers) > 0 && loggers[0] != nil {
		return loggers[0]
	}
	return slog.New(slog.DiscardHandler)
}
