package main

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/reconquest/karma-go"
)

const trimRight = `)]"':`

// Candidate is a completion occurrence, optionally nested under a parent value.
type Candidate struct {
	*Identifier

	Selected bool
	Parent   string
}

// Identifier is text located at zero-based rune and line coordinates.
type Identifier struct {
	X int
	Y int

	Value string
}

// Length returns the identifier's rune count.
func (identifier *Identifier) Length() int {
	return utf8.RuneCountInString(identifier.Value)
}

//nolint:nilnil // Nil identifier denotes absence, not a regexp error.
func getIdentifierToComplete(
	regexpCursor string,
	lines []string,
	column int,
	row int,
) (*Identifier, error) {
	if row < 0 || row >= len(lines) || column < 0 {
		return nil, nil
	}

	symbols := []rune(lines[row])
	if column > len(symbols) {
		return nil, nil
	}

	textBeforeCursor := string(symbols[:column])

	matcher, err := regexp.Compile(`^.*?(` + regexpCursor + `)$`)
	if err != nil {
		return nil, karma.Format(err, "compile cursor regexp")
	}

	matches := matcher.FindStringSubmatch(textBeforeCursor)
	if len(matches) <= 1 {
		return nil, nil
	}

	return &Identifier{
		X:     column - utf8.RuneCountInString(matches[1]),
		Y:     row,
		Value: matches[1],
	}, nil
}

func getCompletionCandidates(
	regexpCandidate string,
	lines []string,
	identifier *Identifier,
) ([]*Candidate, error) {
	query := regexpCandidate
	if identifier != nil {
		query = regexp.QuoteMeta(identifier.Value) + regexpCandidate
	}

	matcher, err := regexp.Compile(query)
	if err != nil {
		return nil, karma.Format(err, "compile candidate regexp")
	}

	var candidates []*Candidate

	for row, line := range lines {
		candidates = append(candidates,
			getLineCandidates(matcher, line, row, identifier)...)
	}

	return candidates, nil
}

func getLineCandidates(
	matcher *regexp.Regexp,
	line string,
	row int,
	identifier *Identifier,
) []*Candidate {
	var candidates []*Candidate

	for _, match := range matcher.FindAllStringSubmatchIndex(line, -1) {
		column := utf8.RuneCountInString(line[:match[0]])
		variants := getCandidateVariants(line[match[0]:match[1]], column, row)

		for _, candidate := range variants {
			if isCompletionCandidate(candidate, identifier) {
				candidates = append(candidates, candidate)
			}
		}
	}

	return candidates
}

func getCandidateVariants(text string, column, row int) []*Candidate {
	candidates := []*Candidate{
		{Identifier: &Identifier{X: column, Y: row, Value: text}},
	}

	trimmed := strings.TrimRight(text, trimRight)
	if trimmed != "" && trimmed != text {
		candidates = append(candidates, &Candidate{
			Identifier: &Identifier{X: column, Y: row, Value: trimmed},
			Parent:     text,
		})
	}

	return candidates
}

func isCompletionCandidate(candidate *Candidate, identifier *Identifier) bool {
	if identifier == nil {
		return true
	}

	if candidate.X == identifier.X && candidate.Y == identifier.Y {
		return false
	}

	return candidate.Value != identifier.Value &&
		strings.HasPrefix(candidate.Value, identifier.Value)
}

func getSelectedCandidate(candidates []*Candidate) *Candidate {
	for _, candidate := range candidates {
		if candidate.Selected {
			return candidate
		}
	}

	return nil
}

func selectDefaultCandidate(candidates []*Candidate, column, row int) {
	var closest *Candidate

	for _, candidate := range candidates {
		if candidate.Parent == "" &&
			isCloserDefaultCandidate(candidate, closest, column, row) {
			closest = candidate
		}
	}

	if closest == nil {
		return
	}

	if selected := getSelectedCandidate(candidates); selected != nil {
		selected.Selected = false
	}

	closest.Selected = true
}

func isCloserDefaultCandidate(
	candidate, closest *Candidate,
	column, row int,
) bool {
	if closest == nil {
		return true
	}

	if candidate.Y > row || (candidate.Y == row && candidate.X > column) {
		return false
	}

	if candidate.Y < closest.Y {
		return false
	}

	return candidate.Y != closest.Y || candidate.X >= closest.X
}

func debugCandidate(candidate *Candidate) {
	selected := ""
	if candidate.Selected {
		selected = " (selected)"
	}

	debug.Printf(
		"y: %-3v x: %-3v length: %-3v value: %s%s",
		candidate.Y,
		candidate.X,
		candidate.Length(),
		candidate.Value,
		selected,
	)
}

func selectNextCandidate(candidates []*Candidate, dirX, dirY int) {
	debug.Printf("selecting next candidate")

	selected := getSelectedCandidate(candidates)
	if selected == nil {
		return
	}

	var closest *Candidate

	for _, candidate := range candidates {
		if !isNextCandidate(selected, dirX, dirY, candidate) {
			continue
		}

		if isCloserNextCandidate(selected, closest, candidate, dirX, dirY) {
			closest = candidate
		}
	}

	if closest == nil {
		return
	}

	closest.Selected = true
	selected.Selected = false
}

func isCloserNextCandidate(
	selected, closest, candidate *Candidate,
	dirX, dirY int,
) bool {
	if closest == nil {
		return true
	}

	distanceX := abs(selected.X-candidate.X) - abs(selected.X-closest.X)

	if dirY != 0 {
		distanceY := abs(selected.Y-candidate.Y) - abs(selected.Y-closest.Y)

		return distanceY < 0 || (distanceY == 0 && distanceX < 0)
	}

	if dirX == 0 {
		return false
	}

	if distanceX != 0 {
		return distanceX < 0
	}

	return isCloserNestedCandidate(selected, closest, candidate, dirX)
}

func isCloserNestedCandidate(
	selected, closest, candidate *Candidate,
	direction int,
) bool {
	if direction == 1 {
		return candidate.Length() > selected.Length() &&
			candidate.Length() < closest.Length()
	}

	return candidate.Length() < selected.Length() &&
		candidate.Length() > closest.Length()
}

func isNextCandidate(
	selected *Candidate,
	dirX, dirY int,
	candidate *Candidate,
) bool {
	debugCandidate(candidate)

	offsetY := sign(candidate.Y - selected.Y)
	if dirX == 0 {
		return sign(dirY) == offsetY
	}

	if offsetY != 0 {
		return false
	}

	signX := sign(dirX)
	offsetX := sign(candidate.X - selected.X)
	debug.Printf("signX: %d offsetX: %d", signX, offsetX)

	if signX == offsetX {
		return true
	}

	return offsetX == 0 &&
		sign(candidate.Length()-selected.Length()) == signX
}

func abs(x int) int {
	if x < 0 {
		x = -x
	}

	return x
}

func sign(value int) int {
	switch {
	case value > 0:
		return 1

	case value < 0:
		return -1

	default:
		return 0
	}
}

func getUniqueCandidates(candidates []*Candidate) []*Candidate {
	seen := make(map[[2]string]bool, len(candidates))
	uniques := []*Candidate{}

	for index := len(candidates) - 1; index >= 0; index-- {
		candidate := candidates[index]
		key := [2]string{candidate.Value, candidate.Parent}

		if seen[key] {
			continue
		}

		seen[key] = true

		uniques = append(uniques, candidate)
	}

	reversed := make([]*Candidate, 0, len(uniques))
	for index := len(uniques) - 1; index >= 0; index-- {
		reversed = append(reversed, uniques[index])
	}

	return reversed
}
