package main

import (
	"regexp"
	"strings"
)

var trimRight = `)]"':`

type Candidate struct {
	*Identifier

	Selected bool
	Parent   string
}

type Identifier struct {
	X int
	Y int

	Value string
}

func (identifier *Identifier) Length() int {
	return len([]rune(identifier.Value))
}

func getIdentifierToComplete(
	regexpCursor string,
	lines []string,
	x int,
	y int,
) (*Identifier, error) {
	textBeforeCursor := string([]rune(lines[y])[:x])

	matcher, err := regexp.Compile(
		`^.*?(` + regexpCursor + `)$`,
	)
	if err != nil {
		return nil, err
	}

	matches := matcher.FindStringSubmatch(textBeforeCursor)

	if len(matches) < 2 {
		return nil, nil
	}

	return &Identifier{
		X: x - len(matches[1]),
		Y: y,

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
		return nil, err
	}

	var candidates []*Candidate

	type unit struct {
		value string
		start int
	}

	for lineNumber, line := range lines {
		matches := matcher.FindAllStringSubmatchIndex(line, -1)

		for _, match := range matches {
			var (
				start, end = match[0], match[1]
				text       = line[start:end]
			)

			units := []unit{
				{text, start},
			}

			trimmed := strings.TrimRight(text, trimRight)

			if len(trimmed) > 0 && trimmed != text {
				units = append(
					units,
					unit{trimmed, start},
				)
			}

			for number, unit := range units {
				if identifier != nil && !strings.HasPrefix(unit.value, identifier.Value) {
					continue
				}

				if identifier != nil && unit.value == identifier.Value {
					continue
				}

				var (
					x = len([]rune(line[:unit.start]))
					y = lineNumber
				)

				if identifier != nil && x == identifier.X && y == identifier.Y {
					continue
				}

				parent := ""
				if number > 0 {
					parent = text
				}
				candidates = append(candidates, &Candidate{
					Identifier: &Identifier{
						X: x,
						Y: y,

						Value: unit.value,
					},
					Parent: parent,
				})
			}
		}
	}

	return candidates, nil
}

func getSelectedCandidate(candidates []*Candidate) *Candidate {
	for _, candidate := range candidates {
		if candidate.Selected {
			return candidate
		}
	}

	return nil
}

func selectDefaultCandidate(
	candidates []*Candidate,
	x int,
	y int,
) {
	if len(candidates) == 0 {
		return
	}

	var closest *Candidate

	for _, candidate := range candidates {
		if candidate.Parent != "" {
			continue
		}
		if closest == nil {
			closest = candidate
			continue
		}

		if candidate.Y > y {
			continue
		}

		if candidate.Y == y {
			if candidate.X > x {
				continue
			}
		}

		if y-candidate.Y > y-closest.Y {
			continue
		}

		if candidate.Y == closest.Y {
			if candidate.X < closest.X {
				continue
			}
		}

		closest = candidate
	}

	if selected := getSelectedCandidate(candidates); selected != nil {
		selected.Selected = false
	}

	closest.Selected = true
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

// the following code has many if-conditions that can be omitted, but please do
// not refactor it, it has been made consciously in order to reduce cognitive
// load
func selectNextCandidate(
	candidates []*Candidate,
	dirX int,
	dirY int,
) {
	debug.Printf("selecting next candidate")

	selected := getSelectedCandidate(candidates)
	if selected == nil {
		return
	}

	next := []*Candidate{}

	for _, candidate := range candidates {
		if isNextCandidate(selected, dirX, dirY, candidate) {
			next = append(next, candidate)
		}
	}

	if len(next) == 0 {
		return
	}

	closest := next[0]

	for _, candidate := range next {
		distanceX := abs(selected.X-candidate.X) - abs(selected.X-closest.X)

		switch {
		case dirY != 0:
			distanceY := abs(selected.Y-candidate.Y) - abs(selected.Y-closest.Y)
			if distanceY < 0 {
				closest = candidate
				continue
			}

			if distanceY == 0 && distanceX < 0 {
				closest = candidate
				continue
			}

		case dirX != 0:
			if distanceX < 0 {
				closest = candidate
				continue
			}

			if dirX == 1 {
				// looking for size greater than selected but the difference
				// should be as small as possible
				if candidate.Length() > selected.Length() && candidate.Length() < closest.Length() {
					closest = candidate
				}
			} else {
				// looking for size less than selected but the difference
				// should be as small as possible (near to selected as possible)
				if candidate.Length() < selected.Length() && candidate.Length() > closest.Length() {
					closest = candidate
				}
			}
		}
	}

	closest.Selected = true
	selected.Selected = false
}

func isNextCandidate(selected *Candidate, dirX, dirY int, candidate *Candidate) bool {
	debugCandidate(candidate)

	signX := sign(dirX)
	signY := sign(dirY)

	offsetX := sign(candidate.X - selected.X)
	offsetY := sign(candidate.Y - selected.Y)

	if dirX != 0 {
		// case: move horizontally
		if offsetY != 0 {
			// not interested in vertical moves in horizontal mode
			return false
		}

		debug.Printf("signX: %d offsetX: %d", signX, offsetX)
		if signX == offsetX {
			return true
		}

		// wrong direction

		if offsetX == 0 {
			// that case is possible when we have two items on the same
			// X coordinate but with different length of identifier
			if signX > 0 {
				// move right
				if candidate.Length() > selected.Length() {
					// not interested in less size, candidate length should be
					// greater than slected
					return true
				}
			} else {
				// move left
				if candidate.Length() < selected.Length() {
					return true
				}
			}
		}

		return false
	}
	// case: move vertically
	if signY == offsetY {
		return true
	}

	return false
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
	uniques := []*Candidate{}

mainLoop:
	for _, candidate := range candidates {
		for _, unique := range uniques {
			if unique.Value == candidate.Value && unique.Parent == candidate.Parent {
				continue mainLoop
			}
		}

		uniques = append(uniques, candidate)
	}

	return uniques
}
