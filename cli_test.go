package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docopt/docopt-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIArguments(t *testing.T) {
	parser := docopt.Parser{HelpHandler: docopt.NoHelpHandler}
	options, err := parser.ParseArgs(cliUsage(), []string{}, "")
	require.NoError(t, err)

	args, err := bindArguments(options)
	require.NoError(t, err)
	assert.Equal(t, defaultRegexpCursor, args.ValueCursorRegexp)
	assert.Equal(t, defaultRegexpCandidate, args.ValueCandidateRegexp)
	assert.Equal(t, defaultThemePath, args.ValueThemePath)
	assert.Equal(t, "light", args.ValueTheme)
	assert.False(t, args.FlagNoPrefix)
	assert.False(t, args.FlagWorker)

	options, err = parser.ParseArgs(cliUsage(), []string{
		"-W", "%7", "12", "4", "--no-prefix", "--exec", "a program",
		"--theme-path", "", "--theme", "dark", "--debug", "",
		"--regexp-cursor", "[ab]", "--regexp-candidate", "[cd]",
	}, "")
	require.NoError(t, err)

	args, err = bindArguments(options)
	require.NoError(t, err)
	assert.Equal(t, arguments{
		ValuePane: "%7", ValueCursorX: "12", ValueCursorY: "4",
		FlagWorker: true, FlagNoPrefix: true, ValueProgram: "a program",
		ValueTheme: "dark", ValueCursorRegexp: "[ab]",
		ValueCandidateRegexp: "[cd]",
	}, args)
	assert.NotNil(t, options["--debug"])
	assert.NotNil(t, options["--theme-path"])
}

func TestCLIWorkerQuoting(t *testing.T) {
	values := []string{
		"", "plain", "two words", "a'b", "\"quoted\"", "$HOME", "`echo bad`",
		"$(echo bad)", "semi;colon", "line\nbreak", "back\\slash", "λ",
	}
	path := filepath.Join(t.TempDir(), "worker's $logs")

	command := append([]string{"printf", "%s\\0"}, values...)
	quoted := quoteWorkerCommand(command, path)
	// #nosec G204 -- Fixed hostile fixtures exercise shell quoting.
	cmd := exec.CommandContext(
		context.Background(), "sh", "-c", strings.Join(quoted, " "),
	)
	output, err := cmd.Output()
	require.NoError(t, err)
	assert.Equal(t, strings.Join(values, "\x00")+"\x00", string(output))

	// #nosec G304 -- Read the test's temporary redirection target.
	logs, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Empty(t, logs)
}

func TestCLIWorkerFlags(t *testing.T) {
	options := map[string]any{
		"--theme-path": nil,
		"--debug":      "",
		"--exec":       "some program",
		"--no-prefix":  true,
		"--help":       false,
		"<pane>":       nil,
	}
	command := workerCommand("the binary", options, "the themes")
	require.NotEmpty(t, command)
	assert.Equal(t, "the binary", command[0])
	assert.Len(t, command, 8)

	flags := make(map[string]string)

	for index := 1; index < len(command); index++ {
		flag := command[index]
		if flag == "--no-prefix" {
			flags[flag] = "true"

			continue
		}

		index++
		require.Less(t, index, len(command))

		flags[flag] = command[index]
	}

	assert.Equal(t, map[string]string{
		"--theme-path": "the themes", "--debug": "",
		"--exec": "some program", "--no-prefix": "true",
	}, flags)
}

func TestCLIPanicReport(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("TMPDIR", directory)

	report := panicReport{
		CursorX: 12, CursorY: 4,
		Pane:  &Pane{ID: "%7", Width: 80, Height: 24},
		Lines: []string{"source contents"},
	}
	command := &command{report: &report}
	command.recoverPanic("regression reason")
	assert.Equal(t, exitPanic, command.exitCode)

	paths, err := filepath.Glob(filepath.Join(directory, "*.log"))
	require.NoError(t, err)
	require.Len(t, paths, 1)

	info, err := os.Stat(paths[0])
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(reportFileMode), info.Mode().Perm())

	data, err := os.ReadFile(paths[0])
	require.NoError(t, err)
	assert.Contains(t, string(data), "regression reason")
	assert.Contains(t, string(data), "goroutine ")
	assert.Contains(t, string(data), "source contents")

	start := bytes.Index(data, []byte("\n{"))
	require.NotEqual(t, -1, start)

	encoded := data[start+1:]

	var decoded panicReport

	err = json.Unmarshal(encoded, &decoded)
	require.NoError(t, err)
	assert.Equal(t, report, decoded)
	assert.JSONEq(t,
		`{"cursor_x":12,"cursor_y":4,"lines":["source contents"],`+
			`"pane":{"id":"%7","width":80,"height":24}}`, string(encoded),
	)
}

func TestCLIReportWriteFailure(t *testing.T) {
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing"))

	report := &panicReport{}
	err := report.write("regression reason")
	require.Error(t, err)
	assert.ErrorContains(t, err, "write bug report")
}

func TestCLIDebugPrivacy(t *testing.T) {
	original := debug

	t.Cleanup(func() { debug = original })

	file, err := openDebug(filepath.Join(t.TempDir(), "debug.log"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })

	info, err := file.Stat()
	require.NoError(t, err)
	assert.Zero(t, info.Mode().Perm()&0o077)
}

func TestCLICursorSequence(t *testing.T) {
	assert.Equal(t, "\x1b[1;1H", cursorSequence(0, 0))
	assert.Equal(t, "\x1b[5;13H", cursorSequence(12, 4))
}

func TestCLISourceAndDisplayDimensions(t *testing.T) {
	source := &Pane{Width: 8, Height: 4}
	worker := &worker{
		args:    &arguments{ValueCursorRegexp: defaultRegexpCursor},
		pane:    source,
		lines:   []string{"hello helm", "hel"},
		cursorX: 3,
		cursorY: 2,
	}
	err := worker.findIdentifier()
	require.NoError(t, err)
	require.NotNil(t, worker.identifier)
	assert.Equal(t, "hel", worker.identifier.Value)
	assert.Equal(t, 0, worker.identifier.X)
	assert.Equal(t, 1, worker.identifier.Y)

	display := *source

	display.Width, display.Height = 40, 12
	assert.Equal(t, "\x1b[2;1Hhel",
		renderIdentifier(worker.lines, &display, &Theme{}, worker.identifier),
	)
	assert.Equal(t, 8, source.Width)
	assert.Equal(t, 4, source.Height)

	err = worker.findIdentifier()
	require.NoError(t, err)
	assert.Equal(t, "hel", worker.identifier.Value)
	assert.Equal(t, 1, worker.identifier.Y)
}
