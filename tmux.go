package main

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/reconquest/karma-go"
)

type Tmux struct {
	stdin io.Reader
}

func (tmux *Tmux) NewWindow(args ...string) error {
	_, err := tmux.exec("new-window", strings.Join(args, " "))
	if err != nil {
		return err
	}

	return nil
}

func (tmux *Tmux) CapturePane(args ...string) (string, error) {
	args = append([]string{"-p"}, args...)

	pane, err := tmux.exec("capture-pane", args...)
	if err != nil {
		return "", err
	}

	return pane, nil
}

func (tmux *Tmux) GetPaneSize(args ...string) (int, int, error) {
	var width int
	var height int

	err := tmux.Eval(
		map[string]interface{}{
			"pane_width":  &width,
			"pane_height": &height,
		},
		args...,
	)
	if err != nil {
		return 0, 0, err
	}

	return width, height, nil
}

func (tmux *Tmux) Eval(values map[string]interface{}, args ...string) error {
	format := []string{}
	binds := []interface{}{}

	for key, bind := range values {
		format = append(format, "#{"+key+"}")
		binds = append(binds, bind)
	}

	reply, err := tmux.exec(
		"display-message",
		"-p",
		strings.Join(format, "\t"),
	)
	if err != nil {
		return err
	}

	_, err = fmt.Sscan(reply, binds...)
	if err != nil {
		return err
	}

	return nil
}

func (tmux *Tmux) Paste(value string, args ...string) error {
	input := bytes.NewBufferString(value)

	_, err := tmux.withStdin(input).exec("load-buffer", "-")
	if err != nil {
		return err
	}

	_, err = tmux.exec(
		"paste-buffer",
		append([]string{"-d"}, args...)...,
	)
	if err != nil {
		return err
	}

	return nil
}

func (tmux *Tmux) exec(command string, args ...string) (string, error) {
	args = append([]string{command}, args...)

	cmd := exec.Command("tmux", args...)
	cmd.Stdin = tmux.stdin

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), karma.
			Describe("args", fmt.Sprintf("%q", args)).
			Format(
				err,
				"unable to exec tmux command",
			)
	}

	return string(output), nil
}

func (tmux *Tmux) withStdin(reader io.Reader) *Tmux {
	clone := *tmux

	clone.stdin = reader

	return &clone
}
