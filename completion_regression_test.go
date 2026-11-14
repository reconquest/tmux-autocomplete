package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompletionCandidates(t *testing.T) {
	for _, scenario := range []struct {
		name       string
		lines      []string
		identifier *Identifier
		expected   []*Candidate
	}{
		{
			name:       "multibyte prefix excludes own word and nested variant",
			lines:      []string{"λ éx): éz):"},
			identifier: &Identifier{X: 2, Y: 0, Value: "é"},
			expected: []*Candidate{
				{Identifier: &Identifier{X: 7, Y: 0, Value: "éz):"}},
				{
					Identifier: &Identifier{X: 7, Y: 0, Value: "éz"},
					Parent:     "éz):",
				},
			},
		},
		{
			name:  "nested variants retain order and parent",
			lines: []string{"foo): bar baz]"},
			expected: []*Candidate{
				{Identifier: &Identifier{X: 0, Y: 0, Value: "foo):"}},
				{
					Identifier: &Identifier{X: 0, Y: 0, Value: "foo"},
					Parent:     "foo):",
				},
				{Identifier: &Identifier{X: 6, Y: 0, Value: "bar"}},
				{Identifier: &Identifier{X: 10, Y: 0, Value: "baz]"}},
				{
					Identifier: &Identifier{X: 10, Y: 0, Value: "baz"},
					Parent:     "baz]",
				},
			},
		},
		{
			name:  "empty trimmed variant is omitted",
			lines: []string{")]"},
			expected: []*Candidate{
				{Identifier: &Identifier{Value: ")]"}},
			},
		},
		{
			name:       "exact prefix and trimmed exact prefix are omitted",
			lines:      []string{"é é):"},
			identifier: &Identifier{X: 0, Y: 1, Value: "é"},
			expected: []*Candidate{
				{Identifier: &Identifier{X: 2, Y: 0, Value: "é):"}},
			},
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			candidates, err := getCompletionCandidates(
				`\S+`, scenario.lines, scenario.identifier,
			)
			require.NoError(t, err)
			assert.Equal(t, scenario.expected, candidates)
		})
	}
}

func TestSelectNextCandidate(t *testing.T) {
	for _, scenario := range []struct {
		name       string
		dirX       int
		dirY       int
		candidates []*Candidate
		expected   int
	}{
		{
			name: "right chooses nearest horizontal position",
			dirX: 1,
			candidates: []*Candidate{
				{Identifier: &Identifier{X: 2, Value: "own"}, Selected: true},
				{Identifier: &Identifier{X: 20, Value: "far"}},
				{Identifier: &Identifier{X: 5, Value: "near"}},
				{Identifier: &Identifier{X: 3, Y: 1, Value: "other row"}},
			},
			expected: 2,
		},
		{
			name: "right distance wins over shorter distant length",
			dirX: 1,
			candidates: []*Candidate{
				{Identifier: &Identifier{Value: "b"}, Selected: true},
				{Identifier: &Identifier{X: 2, Value: "bbbbb"}},
				{Identifier: &Identifier{X: 8, Value: "bbb"}},
			},
			expected: 1,
		},
		{
			name: "left distance wins over longer distant length",
			dirX: -1,
			candidates: []*Candidate{
				{
					Identifier: &Identifier{X: 10, Value: "bbbbb"},
					Selected:   true,
				},
				{Identifier: &Identifier{X: 8, Value: "b"}},
				{Identifier: &Identifier{X: 2, Value: "bbb"}},
			},
			expected: 1,
		},
		{
			name: "left chooses nearest horizontal position",
			dirX: -1,
			candidates: []*Candidate{
				{Identifier: &Identifier{X: 20, Value: "own"}, Selected: true},
				{Identifier: &Identifier{X: 2, Value: "far"}},
				{Identifier: &Identifier{X: 17, Value: "near"}},
				{Identifier: &Identifier{X: 19, Y: 1, Value: "other row"}},
			},
			expected: 2,
		},
		{
			name: "right chooses smallest larger nested rune length",
			dirX: 1,
			candidates: []*Candidate{
				{Identifier: &Identifier{Value: "é"}, Selected: true},
				{Identifier: &Identifier{Value: "é):"}},
				{Identifier: &Identifier{Value: "é)"}, Parent: "é):"},
				{Identifier: &Identifier{X: 1, Value: "next"}},
			},
			expected: 2,
		},
		{
			name: "left chooses largest smaller nested rune length",
			dirX: -1,
			candidates: []*Candidate{
				{Identifier: &Identifier{Value: "é):"}, Selected: true},
				{Identifier: &Identifier{Value: "é"}, Parent: "é):"},
				{Identifier: &Identifier{Value: "é)"}, Parent: "é):"},
			},
			expected: 2,
		},
		{
			name: "down prefers nearest row then column",
			dirY: 1,
			candidates: []*Candidate{
				{Identifier: &Identifier{X: 5, Y: 1}, Selected: true},
				{Identifier: &Identifier{X: 5, Y: 3}},
				{Identifier: &Identifier{X: 0, Y: 2}},
				{Identifier: &Identifier{X: 4, Y: 2}},
			},
			expected: 3,
		},
		{
			name: "up retains first candidate on distance tie",
			dirY: -1,
			candidates: []*Candidate{
				{Identifier: &Identifier{X: 5, Y: 3}, Selected: true},
				{Identifier: &Identifier{X: 4, Y: 2}},
				{Identifier: &Identifier{X: 6, Y: 2}},
				{Identifier: &Identifier{X: 5, Y: 1}},
			},
			expected: 1,
		},
		{
			name: "no next candidate retains selection",
			dirX: -1,
			candidates: []*Candidate{
				{Identifier: &Identifier{Value: "own"}, Selected: true},
				{Identifier: &Identifier{X: 5, Value: "next"}},
			},
			expected: 0,
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			selectNextCandidate(
				scenario.candidates,
				scenario.dirX,
				scenario.dirY,
			)

			for index, candidate := range scenario.candidates {
				assert.Equal(t, index == scenario.expected, candidate.Selected)
			}
		})
	}
}

func TestSelectDefaultCandidate(t *testing.T) {
	candidates := []*Candidate{
		{Identifier: &Identifier{X: 0, Y: 0}},
		{Identifier: &Identifier{X: 2, Y: 1}},
		{Identifier: &Identifier{X: 4, Y: 1}},
		{Identifier: &Identifier{X: 5, Y: 1}, Parent: "parent"},
		{Identifier: &Identifier{X: 8, Y: 1}, Selected: true},
		{Identifier: &Identifier{X: 0, Y: 2}},
	}
	selectDefaultCandidate(candidates, 6, 1)

	for index, candidate := range candidates {
		assert.Equal(t, index == 2, candidate.Selected)
	}
}

func TestUniqueCandidates(t *testing.T) {
	candidates := []*Candidate{
		{Identifier: &Identifier{X: 0, Value: "é"}},
		{Identifier: &Identifier{X: 1, Value: "other"}},
		{Identifier: &Identifier{X: 2, Value: "é"}, Parent: "é)"},
		{Identifier: &Identifier{X: 3, Value: "é"}},
	}
	assert.Equal(t, candidates[1:], getUniqueCandidates(candidates))
}

func TestIdentifierAbsenceAndRegexpErrors(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		column int
		row    int
	}{
		{name: "no matching prefix", column: 1},
		{name: "negative column", column: -1},
		{name: "negative row", row: -1},
		{name: "past end of line", column: 2},
		{name: "past last line", row: 1},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			identifier, err := getIdentifierToComplete(
				`[a-z]+`, []string{"λ"}, scenario.column, scenario.row,
			)
			require.NoError(t, err)
			assert.Nil(t, identifier)
		})
	}

	identifier, err := getIdentifierToComplete("[", []string{"a"}, 1, 0)
	require.Error(t, err)
	assert.Nil(t, identifier)

	candidates, err := getCompletionCandidates("[", []string{"a"}, nil)
	require.Error(t, err)
	assert.Nil(t, candidates)

	identifier, err = getIdentifierToComplete(`\S+`, []string{"λ éx"}, 3, 0)
	require.NoError(t, err)
	assert.Equal(t, &Identifier{X: 2, Y: 0, Value: "é"}, identifier)
}
