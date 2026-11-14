package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/reconquest/karma-go"
)

func start(args map[string]any, themePath string, tmux *Tmux) error {
	var pane, cursorX, cursorY string

	err := tmux.Eval(map[string]any{
		"pane_id": &pane, "cursor_x": &cursorX, "cursor_y": &cursorY,
	})
	if err != nil {
		return karma.Format(err, "get current pane/cursor")
	}

	logsPipe, err := mkfifo()
	if err != nil {
		return karma.Format(err, "make fifo")
	}
	defer removeLogged(logsPipe)

	cmd := workerCommand(os.Args[0], args, themePath)

	cmd = append(cmd, pane, cursorX, cursorY, "-W")
	cmd = quoteWorkerCommand(cmd, logsPipe)

	err = tmux.NewWindow(cmd...)
	if err != nil {
		return karma.Format(err, "create new tmux window")
	}

	return readWorkerLogs(logsPipe)
}

func workerCommand(
	executable string, args map[string]any, themePath string,
) []string {
	cmd := []string{executable}

	for flag, value := range args {
		switch flag {
		case "--theme-path":
			cmd = append(cmd, flag, themePath)

		case "--regexp":
			cmd = append(cmd, flag, fmt.Sprint(value))

		default:
			cmd = appendWorkerFlag(cmd, flag, value)
		}
	}

	return cmd
}

func appendWorkerFlag(cmd []string, flag string, value any) []string {
	switch typed := value.(type) {
	case string:
		return append(cmd, flag, typed)

	case bool:
		if typed {
			return append(cmd, flag)
		}

	case nil:
		return cmd

	default:
		panic(fmt.Sprintf(
			"unexpected type of flag %s: %#v (%T)", flag, value, value,
		))
	}

	return cmd
}

func quoteWorkerCommand(cmd []string, logsPipe string) []string {
	for index, arg := range cmd {
		cmd[index] = shellQuote(arg)
	}

	return append(cmd, "2>"+shellQuote(logsPipe))
}

func readWorkerLogs(logsPipe string) error {
	// #nosec G304 -- Read the FIFO created for this worker.
	file, err := os.Open(logsPipe)
	if err != nil {
		return karma.Describe("path", logsPipe).Format(err, "open worker logs")
	}
	defer closeLogged(file)

	file.Fd()

	logs, err := io.ReadAll(file)
	if err != nil {
		return karma.Describe("path", logsPipe).Format(err, "read worker logs")
	}

	if len(logs) > 0 {
		return errors.New(string(logs))
	}

	return nil
}

func removeLogged(path string) {
	err := os.Remove(path)
	if err != nil {
		log.Print(
			karma.Describe("path", path).Format(err, "remove worker logs"),
		)
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
