package main

import (
	"regexp"
	"strings"

	"github.com/mattn/go-runewidth"
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
	if pane.Width <= 0 || x < 0 || x > pane.Width || y < 0 {
		return -1, -1
	}

	for row, line := range lines {
		positions, height := pane.getLinePositions(line)
		if y < height {
			cell := y*pane.Width + x
			if cell > positions[len(positions)-1] {
				return -1, -1
			}

			for index, symbol := range []rune(line) {
				width := runewidth.RuneWidth(symbol)
				if width > 0 && positions[index]+width > cell {
					return index, row
				}
			}

			return len(positions) - 1, row
		}

		y -= height
	}

	return -1, -1
}

func (pane *Pane) GetScreenXY(lines []string, x, y int) (int, int) {
	if pane.Width <= 0 || x < 0 || y < 0 || y >= len(lines) {
		return -1, -1
	}

	offset := 0
	for row, line := range lines {
		positions, height := pane.getLinePositions(line)
		if row == y {
			if x >= len(positions) {
				return -1, -1
			}

			return positions[x] % pane.Width, offset + positions[x]/pane.Width
		}

		offset += height
	}

	return -1, -1
}

func (pane *Pane) getLinePositions(line string) ([]int, int) {
	positions := []int{}
	cells := 0
	for _, symbol := range line {
		width := runewidth.RuneWidth(symbol)
		if column := cells % pane.Width; column > 0 && column+width > pane.Width {
			cells += pane.Width - column
		}

		positions = append(positions, cells)
		cells += width
	}
	positions = append(positions, cells)

	height := 1
	if cells > 0 {
		height += (cells - 1) / pane.Width
	}

	return positions, height
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
