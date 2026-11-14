package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func fakeTmux(t *testing.T) string {
	t.Helper()

	const script = `#!/bin/sh
printf '%s\000' "$@" >>"$TMUX_ARGS"
/bin/cat >>"$TMUX_INPUT"
case "$1" in
display-message)
	case "$5" in
	'#{pane_width}'*) printf '80\t24' ;;
	'#{pane_height}'*) printf '24\t80' ;;
	*) printf '%s' "$TMUX_REPLY" ;;
	esac
	;;
*) printf '%s' "$TMUX_REPLY" ;;
esac
exit "${TMUX_STATUS:-0}"
`

	directory := t.TempDir()
	// #nosec G306 -- Executable fake tmux is private to the test directory.
	require.NoError(t, os.WriteFile(
		filepath.Join(directory, "tmux"), []byte(script), 0o700,
	))
	t.Setenv("PATH", directory)
	t.Setenv("TMUX_ARGS", filepath.Join(directory, "args"))
	t.Setenv("TMUX_INPUT", filepath.Join(directory, "input"))
	t.Setenv("TMUX_REPLY", "")
	t.Setenv("TMUX_STATUS", "0")

	return directory
}

func tmuxArgs(t *testing.T, directory string) []string {
	t.Helper()

	// #nosec G304 -- Read only the argv fixture in the test-owned directory.
	data, err := os.ReadFile(filepath.Join(directory, "args"))
	require.NoError(t, err)

	return strings.Split(strings.TrimSuffix(string(data), "\x00"), "\x00")
}

func TestTmuxArguments(t *testing.T) {
	directory := fakeTmux(t)
	tmux := &Tmux{}
	target := "session:window.1; $(touch never)"

	t.Setenv("TMUX_REPLY", "pane text\n")

	pane, err := tmux.CapturePane("-t", target)
	require.NoError(t, err)
	require.Equal(t, "pane text\n", pane)
	require.Equal(t, []string{"capture-pane", "-p", "-t", target},
		tmuxArgs(t, directory))
}

func TestTmuxNewWindow(t *testing.T) {
	directory := fakeTmux(t)
	tmux := &Tmux{}
	args := []string{"'a program'", "'$(literal)'", "2>'a fifo'"}
	require.NoError(t, tmux.NewWindow(args...))
	require.Equal(t, []string{"new-window", strings.Join(args, " ")},
		tmuxArgs(t, directory))
}

func TestTmuxPaste(t *testing.T) {
	directory := fakeTmux(t)
	tmux := &Tmux{}
	value := "$(touch never); 'quoted'\n\x00literal"
	require.NoError(t, tmux.Paste(value, "-t", "%42"))
	require.Equal(t, []string{
		"load-buffer", "-", "paste-buffer", "-d", "-t", "%42",
	}, tmuxArgs(t, directory))
	require.Nil(t, tmux.stdin)

	// #nosec G304 -- Read only the stdin fixture in the test-owned directory.
	input, err := os.ReadFile(filepath.Join(directory, "input"))
	require.NoError(t, err)
	require.Equal(t, value, string(input))
}

func TestTmuxPaneSize(t *testing.T) {
	directory := fakeTmux(t)
	tmux := &Tmux{}

	width, height, err := tmux.GetPaneSize("-t", "%42")
	require.NoError(t, err)
	require.Equal(t, 80, width)
	require.Equal(t, 24, height)

	args := tmuxArgs(t, directory)
	require.Equal(t, []string{"display-message", "-p", "-t", "%42"},
		args[:4])
	require.ElementsMatch(t, []string{"#{pane_width}", "#{pane_height}"},
		strings.Split(args[4], "\t"))
}

func TestTmuxErrors(t *testing.T) {
	directory := fakeTmux(t)
	tmux := &Tmux{}

	t.Setenv("TMUX_REPLY", "not a number")

	var value int

	err := tmux.Eval(map[string]any{"invalid": &value})
	require.ErrorContains(t, err, "scan tmux format values")

	t.Setenv("TMUX_STATUS", "1")

	output, err := tmux.exec("capture-pane")
	require.ErrorContains(t, err, "exec tmux command")
	require.Equal(t, "not a number", output)

	require.NoError(t, os.Remove(filepath.Join(directory, "args")))
	require.Error(t, tmux.Paste("literal", "-t", "%42"))
	require.Equal(t, []string{"load-buffer", "-"}, tmuxArgs(t, directory))
}
