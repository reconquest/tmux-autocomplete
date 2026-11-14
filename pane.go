package main

import (
	"regexp"
	"strings"

	"github.com/mattn/go-runewidth"
)

var reEscapeSequence = regexp.MustCompile(`\x1b\[([^m]+)m`)

// Pane holds captured source lines and the dimensions of their tmux pane.
type Pane struct {
	ID    string   `json:"id,omitempty"`
	Lines []string `json:"lines,omitempty"`

	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`
}

type paneLineLayout struct {
	positions []int
	height    int
}

// CapturePane captures source and dimensions using the same explicit pane ID.
func CapturePane(tmux *Tmux, paneID string, args ...string) (*Pane, error) {
	contents, err := tmux.CapturePane(
		append([]string{"-t", paneID}, args...)...)
	if err != nil {
		return nil, err
	}

	width, height, err := tmux.GetPaneSize("-t", paneID)
	if err != nil {
		return nil, err
	}

	return &Pane{
		ID:     paneID,
		Lines:  strings.Split(strings.TrimRight(contents, "\n"), "\n"),
		Width:  width,
		Height: height,
	}, nil
}

// GetBufferXY converts screen cells to rune coordinates, or returns (-1, -1).
//
//nolint:gocritic // Preserve the unnamed coordinate-return API.
//revive:disable-next-line:confusing-results Preserve unnamed coordinates.
func (pane *Pane) GetBufferXY(lines []string, screenX, screenY int) (int, int) {
	if pane.Width <= 0 || screenX < 0 || screenX > pane.Width || screenY < 0 {
		return -1, -1
	}

	for row, line := range lines {
		layout := pane.getLinePositions(line)
		if screenY >= layout.height {
			screenY -= layout.height

			continue
		}

		column := bufferColumn(
			line,
			layout.positions,
			screenY*pane.Width+screenX,
		)
		if column < 0 {
			return -1, -1
		}

		return column, row
	}

	return -1, -1
}

// GetScreenXY converts rune coordinates to cells, or returns (-1, -1).
// An exact-width line end retains the pending-wrap column equal to Width.
//
//nolint:gocritic // Preserve the unnamed coordinate-return API.
//revive:disable-next-line:confusing-results Preserve unnamed coordinates.
func (pane *Pane) GetScreenXY(lines []string, bufferX, bufferY int) (int, int) {
	if pane.Width <= 0 || bufferX < 0 || bufferY < 0 || bufferY >= len(lines) {
		return -1, -1
	}

	layout := pane.getLinePositions(lines[bufferY])
	if bufferX >= len(layout.positions) {
		return -1, -1
	}

	cell := layout.positions[bufferX]
	screenX, screenY := cell%pane.Width, cell/pane.Width

	if bufferX == len(layout.positions)-1 && cell > 0 && screenX == 0 {
		screenX = pane.Width
		screenY--
	}

	return screenX, pane.screenOffset(lines[:bufferY]) + screenY
}

func (pane *Pane) screenOffset(lines []string) int {
	offset := 0

	for _, line := range lines {
		layout := pane.getLinePositions(line)

		offset += layout.height
	}

	return offset
}

func bufferColumn(line string, positions []int, cell int) int {
	if cell > positions[len(positions)-1] {
		return -1
	}

	for index, symbol := range []rune(line) {
		width := runewidth.RuneWidth(symbol)
		if width > 0 && positions[index]+width > cell {
			return index
		}
	}

	return len(positions) - 1
}

func (pane *Pane) getLinePositions(line string) paneLineLayout {
	positions := make([]int, 0, len(line)+1)
	cells := 0

	for _, symbol := range line {
		width := runewidth.RuneWidth(symbol)
		column := cells % pane.Width

		if column > 0 && column+width > pane.Width {
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

	return paneLineLayout{positions: positions, height: height}
}

// GetPrintable strips ANSI styles and translates alternate-set line drawing.
// The captured source lines are not modified.
func (pane *Pane) GetPrintable() []string {
	printable := make([]string, 0, len(pane.Lines))
	for _, line := range pane.Lines {
		printable = append(printable, printableLine(line))
	}

	return printable
}

func printableLine(line string) string {
	line = reEscapeSequence.ReplaceAllLiteralString(line, "")

	symbols := make([]rune, 0, len(line))
	inGrid := false

	for _, symbol := range line {
		switch symbol {
		case '\x0e':
			inGrid = true

		case '\x0f':
			inGrid = false

		default:
			if inGrid {
				symbol = gridSymbol(symbol)
			}

			symbols = append(symbols, symbol)
		}
	}

	return string(symbols)
}

func gridSymbol(symbol rune) rune {
	const (
		source  = "lqwkxtunvmj"
		drawing = "┌─┬┐│├┤┼┴└┘"
	)

	index := strings.IndexRune(source, symbol)
	if index < 0 {
		return symbol
	}

	return []rune(drawing)[index]
}

// String returns the unmodified captured lines joined by newlines.
func (pane *Pane) String() string {
	return strings.Join(pane.Lines, "\n")
}
