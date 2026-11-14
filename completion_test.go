package main

import (
	"io/fs"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testcase struct {
	lines    []string
	cursorX  int
	cursorY  int
	expected []string
}

func TestCompletion(t *testing.T) {
	scenarios, readErr := os.ReadDir("unit_tests")
	require.NoError(t, readErr)
	require.NotEmptyf(t, scenarios, "no tests found")

	only := os.Getenv("ONLY")
	if only != "" {
		t.Logf("filter for testcases: *%s*", only)
	}

	matched := 0

	for _, scenario := range scenarios {
		if scenario.IsDir() || !strings.Contains(scenario.Name(), only) {
			continue
		}

		matched++

		t.Run(scenario.Name(), func(t *testing.T) {
			fixture := getTestcase(t, scenario.Name())
			identifier, err := getIdentifierToComplete(
				defaultRegexpCursor,
				fixture.lines,
				fixture.cursorX,
				fixture.cursorY,
			)
			require.NoErrorf(t, err, "unable to get completion identifier")
			require.NotNilf(t, identifier, "invalid prompt (identifier = nil)")

			candidates, err := getCompletionCandidates(
				defaultRegexpCandidate, fixture.lines, identifier,
			)
			require.NoErrorf(t, err, "unable to get completion candidates")

			candidates = getUniqueCandidates(candidates)

			values := make([]string, 0, len(candidates))

			for _, candidate := range candidates {
				values = append(values, candidate.Value)
				t.Logf("candidate=%s y=%d x=%d",
					candidate.Value, candidate.Y, candidate.X)
			}

			assert.Equal(t, fixture.expected, values)
		})
	}

	require.Positivef(t, matched, "no fixtures match ONLY=%q", only)
}

func getTestcase(t *testing.T, filename string) testcase {
	t.Helper()

	fixtures := os.DirFS("unit_tests")
	data, err := fs.ReadFile(fixtures, filename)
	require.NoErrorf(t, err, "read scenario %s", filename)

	const headerLines = 2

	source := strings.Split(strings.TrimSpace(string(data)), "\n")
	require.GreaterOrEqualf(t, len(source), headerLines,
		"invalid scenario %s", filename)

	paneData, err := fs.ReadFile(fixtures, source[0])
	require.NoErrorf(t, err, "read pane %s", source[0])

	prompt := "$ " + source[1]
	lines := strings.Split(string(paneData), "\n")

	lines = append(lines, prompt)

	return testcase{
		lines:    lines,
		cursorX:  utf8.RuneCountInString(prompt),
		cursorY:  len(lines) - 1,
		expected: source[headerLines:],
	}
}
