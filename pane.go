package main

import (
	"regexp"
	"strings"
)

var reEscapeSequence = regexp.MustCompile(`\x1b\[([^m]+)m`)

type Pane struct {
	ID    string
	Lines []string

	Width  int
	Height int
}

func CapturePane(tmux *Tmux, id string, args ...string) (*Pane, error) {
	contents, err := tmux.CapturePane(append([]string{"-t", id}, args...)...)
	if err != nil {
		return nil, err
	}

	width, height, err := tmux.GetPaneSize("-t", id)
	if err != nil {
		return nil, err
	}

	return &Pane{
		ID:     id,
		Lines:  strings.Split(strings.TrimRight(contents, "\n"), "\n"),
		Width:  width,
		Height: height,
	}, nil
}

func (pane *Pane) GetBufferXY(lines []string, x, y int) (int, int) {
	for row, line := range lines {
		offset := (len([]rune(line)) - 1) / pane.Width

		if row+offset >= y {
			x = x + (y-row)*pane.Width
			y = row
			break
		}

		y -= offset
	}

	return x, y
}

func (pane *Pane) GetScreenXY(lines []string, x, y int) (int, int) {
	offset := 0

	for row, line := range lines {
		if row == y {
			return x % pane.Width, y + x/pane.Width + offset
		} else {
			offset += (len([]rune(line)) - 1) / pane.Width
		}
	}

	return x, y
}

func (pane *Pane) GetPrintable() []string {
	printable := []string{}

	for _, line := range pane.Lines {
		line = reEscapeSequence.ReplaceAllLiteralString(line, ``)

		printable = append(printable, line)
	}

	return printable
}

func (pane *Pane) String() string {
	return strings.Join(pane.Lines, "\n")
}
