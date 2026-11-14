package main

import (
	"context"
	"os"
	"os/exec"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:gosmopolitan // Wide-glyph fixtures verify terminal cell coordinates.
func TestPaneBufferCoordinates(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		width int
		lines []string
		input [2]int
		want  [2]int
	}{
		{name: "empty line", width: 4, lines: []string{""}, want: [2]int{0, 0}},
		{name: "no lines", width: 4, want: [2]int{-1, -1}},
		{name: "zero width", lines: []string{"a"}, want: [2]int{-1, -1}},
		{name: "negative width", width: -1, want: [2]int{-1, -1}},
		{
			name:  "negative column",
			width: 4,
			lines: []string{"a"},
			input: [2]int{-1, 0},
			want:  [2]int{-1, -1},
		},
		{
			name:  "negative row",
			width: 4,
			lines: []string{"a"},
			input: [2]int{0, -1},
			want:  [2]int{-1, -1},
		},
		{
			name:  "column past width",
			width: 4,
			lines: []string{"abcdef"},
			input: [2]int{5, 0},
			want:  [2]int{-1, -1},
		},
		{
			name:  "column past end",
			width: 4,
			lines: []string{"a"},
			input: [2]int{2, 0},
			want:  [2]int{-1, -1},
		},
		{
			name:  "row past end",
			width: 4,
			lines: []string{"a"},
			input: [2]int{0, 1},
			want:  [2]int{-1, -1},
		},
		{
			name:  "pending wrap",
			width: 4,
			lines: []string{"abcd", "next"},
			input: [2]int{4, 0},
			want:  [2]int{4, 0},
		},
		{
			name:  "wrapped end",
			width: 4,
			lines: []string{"abcdefgh"},
			input: [2]int{4, 1},
			want:  [2]int{8, 0},
		},
		{
			name:  "wrapped row",
			width: 4,
			lines: []string{"abcdef", "z"},
			input: [2]int{1, 1},
			want:  [2]int{5, 0},
		},
		{
			name:  "following row",
			width: 4,
			lines: []string{"abcdef", "z"},
			input: [2]int{0, 2},
			want:  [2]int{0, 1},
		},
		{
			name:  "multibyte",
			width: 4,
			lines: []string{"é界z"},
			input: [2]int{3, 0},
			want:  [2]int{2, 0},
		},
		{
			name:  "wide glyph second cell",
			width: 4,
			lines: []string{"a界"},
			input: [2]int{2, 0},
			want:  [2]int{1, 0},
		},
		{
			name:  "wide edge padding",
			width: 4,
			lines: []string{"abc界"},
			input: [2]int{3, 0},
			want:  [2]int{3, 0},
		},
		{
			name:  "wide edge wrapped",
			width: 4,
			lines: []string{"abc界"},
			input: [2]int{1, 1},
			want:  [2]int{3, 0},
		},
		{
			name:  "combining",
			width: 4,
			lines: []string{"e\u0301b"},
			input: [2]int{1, 0},
			want:  [2]int{2, 0},
		},
		{
			name:  "combining end",
			width: 4,
			lines: []string{"abce\u0301"},
			input: [2]int{4, 0},
			want:  [2]int{5, 0},
		},
		{
			name:  "leading combining",
			width: 4,
			lines: []string{"\u0301a"},
			want:  [2]int{1, 0},
		},
		{
			name:  "only combining",
			width: 4,
			lines: []string{"\u0301"},
			want:  [2]int{1, 0},
		},
	}

	for _, testcase := range cases {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			pane := Pane{Width: testcase.width}
			column, row := pane.GetBufferXY(
				testcase.lines, testcase.input[0], testcase.input[1],
			)

			if got := [2]int{column, row}; got != testcase.want {
				t.Errorf("buffer coordinates = %v, want %v", got, testcase.want)
			}
		})
	}
}

//nolint:gosmopolitan // Wide-glyph fixtures verify terminal cell coordinates.
func TestPaneScreenCoordinates(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		width int
		lines []string
		input [2]int
		want  [2]int
	}{
		{name: "empty line", width: 4, lines: []string{""}, want: [2]int{0, 0}},
		{name: "no lines", width: 4, want: [2]int{-1, -1}},
		{name: "zero width", lines: []string{"a"}, want: [2]int{-1, -1}},
		{name: "negative width", width: -1, want: [2]int{-1, -1}},
		{
			name:  "negative column",
			width: 4,
			lines: []string{"a"},
			input: [2]int{-1, 0},
			want:  [2]int{-1, -1},
		},
		{
			name:  "negative row",
			width: 4,
			lines: []string{"a"},
			input: [2]int{0, -1},
			want:  [2]int{-1, -1},
		},
		{
			name:  "column past end",
			width: 4,
			lines: []string{"é"},
			input: [2]int{2, 0},
			want:  [2]int{-1, -1},
		},
		{
			name:  "row past end",
			width: 4,
			lines: []string{"a"},
			input: [2]int{0, 1},
			want:  [2]int{-1, -1},
		},
		{
			name:  "pending wrap",
			width: 4,
			lines: []string{"abcd", "next"},
			input: [2]int{4, 0},
			want:  [2]int{4, 0},
		},
		{
			name:  "wrapped end",
			width: 4,
			lines: []string{"abcdefgh"},
			input: [2]int{8, 0},
			want:  [2]int{4, 1},
		},
		{
			name:  "wrapped rune",
			width: 4,
			lines: []string{"abcdef"},
			input: [2]int{4, 0},
			want:  [2]int{0, 1},
		},
		{
			name:  "following wrapped row",
			width: 4,
			lines: []string{"abcdef", "z"},
			input: [2]int{0, 1},
			want:  [2]int{0, 2},
		},
		{
			name:  "following full row",
			width: 4,
			lines: []string{"abcd", ""},
			input: [2]int{0, 1},
			want:  [2]int{0, 1},
		},
		{
			name:  "multibyte",
			width: 4,
			lines: []string{"é界z"},
			input: [2]int{2, 0},
			want:  [2]int{3, 0},
		},
		{
			name:  "wide edge",
			width: 4,
			lines: []string{"abc界"},
			input: [2]int{3, 0},
			want:  [2]int{0, 1},
		},
		{
			name:  "wide edge end",
			width: 4,
			lines: []string{"abc界"},
			input: [2]int{4, 0},
			want:  [2]int{2, 1},
		},
		{
			name:  "combining mark",
			width: 4,
			lines: []string{"e\u0301b"},
			input: [2]int{1, 0},
			want:  [2]int{1, 0},
		},
		{
			name:  "after combining",
			width: 4,
			lines: []string{"e\u0301b"},
			input: [2]int{2, 0},
			want:  [2]int{1, 0},
		},
		{
			name:  "combining pending wrap",
			width: 4,
			lines: []string{"abce\u0301"},
			input: [2]int{5, 0},
			want:  [2]int{4, 0},
		},
		{
			name:  "one cell width",
			width: 1,
			lines: []string{"ab"},
			input: [2]int{2, 0},
			want:  [2]int{1, 1},
		},
	}

	for _, testcase := range cases {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			pane := Pane{Width: testcase.width}
			column, row := pane.GetScreenXY(
				testcase.lines, testcase.input[0], testcase.input[1],
			)

			if got := [2]int{column, row}; got != testcase.want {
				t.Errorf("screen coordinates = %v, want %v", got, testcase.want)
			}
		})
	}
}

//nolint:gosmopolitan // Unicode fixtures verify terminal text preservation.
func TestPanePrintable(t *testing.T) {
	t.Parallel()

	pane := Pane{Lines: []string{
		"plain é界e\u0301",
		"\x1b[31mred\x1b[0m \x1b[0mreset",
		"\x0elqwkxtunvmj\x0f lqwkxtunvmj",
		"\x0e?界\x0eqq\x0fq\x0fq",
		"\x0el",
		"q",
		"",
	}}
	original := append([]string(nil), pane.Lines...)
	originalString := pane.String()
	want := []string{
		"plain é界e\u0301",
		"red reset",
		"┌─┬┐│├┤┼┴└┘ lqwkxtunvmj",
		"?界──qq",
		"┌",
		"q",
		"",
	}

	got := pane.GetPrintable()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("printable lines = %q, want %q", got, want)
	}

	if !reflect.DeepEqual(pane.Lines, original) {
		t.Errorf("source lines changed: %q", pane.Lines)
	}

	if pane.String() != originalString {
		t.Errorf("source string changed: %q", pane.String())
	}

	got[0] = "changed"

	if !reflect.DeepEqual(pane.Lines, original) {
		t.Errorf("printable lines alias source: %q", pane.Lines)
	}

	empty := (&Pane{}).GetPrintable()
	if empty == nil || len(empty) != 0 {
		t.Errorf("empty printable lines = %#v, want non-nil empty slice", empty)
	}
}

func TestPaneLocaleMatrix(t *testing.T) {
	t.Parallel()

	binary, err := os.Executable()
	require.NoError(t, err)

	for _, locale := range []string{"C.UTF-8", "ja_JP.UTF-8"} {
		for _, eastAsian := range []string{"", "0", "1"} {
			t.Run(locale+"/east-asian="+eastAsian, func(t *testing.T) {
				t.Parallel()

				ctx, cancel := context.WithTimeout(
					context.Background(),
					30*time.Second,
				)
				defer cancel()

				// #nosec G204 -- Execute only this test binary.
				command := exec.CommandContext(
					ctx,
					binary,
					"-test.run=^TestPane(BufferCoordinates|"+
						"ScreenCoordinates|Printable|LocalePrefix)$",
					"-test.count=1",
				)

				command.Env = append(os.Environ(),
					"LC_ALL="+locale,
					"LC_CTYPE="+locale,
					"LANG="+locale,
					"RUNEWIDTH_EASTASIAN="+eastAsian,
				)

				output, runErr := command.CombinedOutput()
				require.NoErrorf(t, runErr, "locale subprocess: %s", output)
			})
		}
	}
}

func TestPaneLocalePrefix(t *testing.T) {
	t.Parallel()

	for _, prefix := range []string{
		"é", "À", "⟦", "⟧", "⟨", "⟩", "⟪", "⟫", "⟬", "⟭", "⦅", "⦆",
	} {
		for _, testcase := range []struct {
			name   string
			width  int
			cursor [2]int
		}{
			{name: "unwrapped", width: 8, cursor: [2]int{4, 0}},
			{name: "pending wrap", width: 4, cursor: [2]int{4, 0}},
			{name: "wrapped", width: 3, cursor: [2]int{1, 1}},
		} {
			t.Run(prefix+"/"+testcase.name, func(t *testing.T) {
				t.Parallel()
				checkPanePrefix(t, prefix, testcase.width, testcase.cursor)
			})
		}
	}
}

func checkPanePrefix(t *testing.T, prefix string, width int, cursor [2]int) {
	t.Helper()

	pane := Pane{Width: width, Lines: []string{prefix + " he"}}
	lines := pane.GetPrintable()
	column, row := pane.GetBufferXY(lines, cursor[0], cursor[1])
	assert.Equal(t, [2]int{4, 0}, [2]int{column, row})

	identifier, err := getIdentifierToComplete(
		defaultRegexpCursor, lines, column, row,
	)
	require.NoError(t, err)
	require.Equal(t, &Identifier{X: 2, Y: 0, Value: "he"}, identifier)

	column, row = pane.GetScreenXY(lines, identifier.X, identifier.Y)
	assert.Equal(t, [2]int{2, 0}, [2]int{column, row})

	column, row = pane.GetBufferXY(lines, column, row)
	assert.Equal(t, [2]int{2, 0}, [2]int{column, row})

	column, row = pane.GetScreenXY(
		lines, identifier.X+identifier.Length(), identifier.Y,
	)
	assert.Equal(t, [2]int{column, row}, cursor)
}
