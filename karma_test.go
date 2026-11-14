package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/nsf/termbox-go"
	"github.com/reconquest/karma-go"
	"github.com/stretchr/testify/require"
)

func TestKarmaFileErrors(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "missing", "file")
	for _, scenario := range []struct {
		name      string
		operation string
		call      func(string) error
	}{
		{"debug", "open debug log", func(path string) error {
			_, err := openDebug(path)

			return err
		}},
		{"worker logs", "open worker logs", readWorkerLogs},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			err := scenario.call(path)
			require.ErrorIs(t, err, os.ErrNotExist)

			var cause *os.PathError

			require.ErrorAs(t, err, &cause)
			require.Equal(t, path, cause.Path)
			require.ErrorIs(t, err, cause)

			var wrapped karma.Karma

			require.ErrorAs(t, err, &wrapped)
			require.Equal(t, scenario.operation, wrapped.Message)
			require.Equal(t, []karma.KeyValue{{Key: "path", Value: path}},
				wrapped.Context.GetKeyValues())
		})
	}
}

func TestKarmaTerminalErrorIdentity(t *testing.T) {
	t.Parallel()

	cause := errors.New("terminal unavailable")
	worker := &worker{}
	done, err := worker.handleEvent(
		termbox.Event{Type: termbox.EventError, Err: cause}, &Pane{},
	)
	require.False(t, done)
	require.ErrorIs(t, err, cause)

	var wrapped karma.Karma

	require.ErrorAs(t, err, &wrapped)
	require.Equal(t, "read terminal event", wrapped.Message)
	require.Empty(t, wrapped.Context.GetKeyValues())
}

func TestKarmaTmuxContextPrivacy(t *testing.T) {
	fakeTmux(t)
	t.Setenv("TMUX_STATUS", "7")
	t.Setenv("TMUX_REPLY", "private command output")

	tmux := &Tmux{}
	output, err := tmux.exec("new-window", "private shell command")
	require.Equal(t, "private command output", output)

	var cause *exec.ExitError

	require.ErrorAs(t, err, &cause)
	require.Equal(t, 7, cause.ExitCode())
	require.ErrorIs(t, err, cause)

	var wrapped karma.Karma

	require.ErrorAs(t, err, &wrapped)
	require.Equal(t, "exec tmux command", wrapped.Message)
	require.Equal(t, []karma.KeyValue{{Key: "command", Value: "new-window"}},
		wrapped.Context.GetKeyValues())
	require.NotContains(t, err.Error(), "private")
}

func TestKarmaPaneContext(t *testing.T) {
	fakeTmux(t)
	t.Setenv("TMUX_STATUS", "1")

	pane, err := CapturePane(&Tmux{}, "%42")
	require.Nil(t, pane)

	var cause *exec.ExitError

	require.ErrorAs(t, err, &cause)
	require.ErrorIs(t, err, cause)

	var wrapped karma.Karma

	require.ErrorAs(t, err, &wrapped)
	require.Equal(t, "capture pane", wrapped.Message)
	require.Equal(t, []karma.KeyValue{{Key: "pane", Value: "%42"}},
		wrapped.Context.GetKeyValues())
}
