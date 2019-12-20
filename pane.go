package main

import (
	"regexp"
	"strings"
)

var reEscapeSequence = regexp.MustCompile(`\x1b\[([^m]+)m`)

type Pane struct {
	ID    string   `json:"id,omitempty"`
	Lines []string `json:"lines,omitempty"`

	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`
}

func CapturePane(tmux *Tmux, id string, args ...string) (*Pane, error) {
	contents, err := tmux.CapturePane(append([]string{"-t", id}, args...)...)
	if err != nil {
		return nil, err
	}

	width, height, err := tmux.GetPaneSize()
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

		inGrid := false
		symbols := []rune(line)
		for i := 0; i < len(symbols); i++ {
			symbol := symbols[i]
			if symbol == '\x0e' {
				inGrid = true
				symbols = append(symbols[:i], symbols[i+1:]...)
				i--
				continue
			}

			if symbol == '\x0f' {
				inGrid = false
				symbols = append(symbols[:i], symbols[i+1:]...)
				i--
				continue
			}

			if inGrid {
				switch symbol {
				case 'l':
					symbols[i] = '┌'
				case 'q':
					symbols[i] = '─'
				case 'w':
					symbols[i] = '┬'
				case 'k':
					symbols[i] = '┐'
				case 'x':
					symbols[i] = '│'
				case 't':
					symbols[i] = '├'
				case 'u':
					symbols[i] = '┤'
				case 'n':
					symbols[i] = '┼'
				case 'v':
					symbols[i] = '┴'
				case 'm':
					symbols[i] = '└'
				case 'j':
					symbols[i] = '┘'
				}
			}
		}

		printable = append(printable, string(symbols))
	}

	return printable
}

func (pane *Pane) String() string {
	return strings.Join(pane.Lines, "\n")
}
