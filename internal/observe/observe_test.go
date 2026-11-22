package observe

import (
	"bytes"
	"log/slog"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrorKindOmitsPath(t *testing.T) {
	err := &os.PathError{Op: "mkdir", Path: "/private", Err: syscall.ENOTDIR}
	require.Equal(t, "not_a_directory", ErrorKind(err))
}

func TestLogPanicOmitsValue(t *testing.T) {
	var out bytes.Buffer
	log := slog.New(slog.NewTextHandler(&out, nil))
	func() {
		defer func() {
			LogPanic(log, "Test panic", recover())
		}()
		panic("private-panic-content")
	}()
	require.Contains(t, out.String(), "panic_type=string")
	require.Contains(t, out.String(), "function=github.com/punt-labs/cryptd/internal/observe.TestLogPanicOmitsValue")
	require.NotContains(t, out.String(), "private-panic-content")
}
