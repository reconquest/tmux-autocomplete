package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/reconquest/karma-go"
)

// Tmux executes tmux commands with optional literal standard input.
type Tmux struct {
	stdin io.Reader
}

// NewWindow runs a command assembled from caller-supplied shell-quoted
// arguments.
func (tmux *Tmux) NewWindow(args ...string) error {
	_, err := tmux.exec("new-window", strings.Join(args, " "))
	if err != nil {
		return err
	}

	return nil
}

// CapturePane returns pane text using the supplied tmux target options.
func (tmux *Tmux) CapturePane(args ...string) (string, error) {
	args = append([]string{"-p"}, args...)

	pane, err := tmux.exec("capture-pane", args...)
	if err != nil {
		return "", err
	}

	return pane, nil
}

// GetPaneSize returns the target pane width and height.
//
//nolint:gocritic // unnamedResult: preserve the unnamed width/height API.
//revive:disable-next-line:confusing-results Preserve unnamed results API.
func (tmux *Tmux) GetPaneSize(args ...string) (int, int, error) {
	var width int

	var height int

	err := tmux.Eval(
		map[string]any{
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

// Eval scans tmux format values into their corresponding destination pointers.
func (tmux *Tmux) Eval(values map[string]any, args ...string) error {
	format := make([]string, 0, len(values))
	binds := make([]any, 0, len(values))

	for key, bind := range values {
		format = append(format, "#{"+key+"}")
		binds = append(binds, bind)
	}

	args = append([]string{"-p"}, args...)
	args = append(args, strings.Join(format, "\t"))

	reply, err := tmux.exec("display-message", args...)
	if err != nil {
		return err
	}

	_, err = fmt.Sscan(reply, binds...)
	if err != nil {
		return karma.Format(err, "scan tmux format values")
	}

	return nil
}

// Paste loads literal text from stdin and pastes it into the supplied target.
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

	// #nosec G204 G702 -- Tmux argv and shell-quoted NewWindow are intentional.
	cmd := exec.CommandContext(context.Background(), "tmux", args...)

	cmd.Stdin = tmux.stdin

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), karma.
			Describe("command", command).
			Format(
				err,
				"exec tmux command",
			)
	}

	return string(output), nil
}

func (tmux *Tmux) withStdin(reader io.Reader) *Tmux {
	clone := *tmux

	clone.stdin = reader

	return &clone
}
