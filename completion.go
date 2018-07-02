package main

import (
	"regexp"
	"strings"
)

type Candidate struct {
	*Identifier

	Selected bool
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
	args map[string]interface{},
	lines []string,
	x int,
	y int,
) (*Identifier, error) {
	textBeforeCursor := string([]rune(lines[y])[:x])

	matcher, err := regexp.Compile(
		`^.*?(` + args["--regexp"].(string) + `)$`,
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
	args map[string]interface{},
	lines []string,
	pane *Pane,
	identifier *Identifier,
) ([]*Candidate, error) {
	matcher, err := regexp.Compile(args["--regexp"].(string))
	if err != nil {
		return nil, err
	}

	var candidates []*Candidate

	for lineNumber, line := range lines {
		matches := matcher.FindAllStringIndex(line, -1)

		for _, match := range matches {
			value := line[match[0]:match[1]]

			if identifier != nil && !strings.HasPrefix(value, identifier.Value) {
				continue
			}

			if identifier != nil && value == identifier.Value {
				continue
			}

			var (
				x = len([]rune(line[:match[0]]))
				y = lineNumber
			)

			if identifier != nil && x == identifier.X && y == identifier.Y {
				continue
			}

			candidates = append(candidates, &Candidate{
				Identifier: &Identifier{
					X: x,
					Y: y,

					Value: value,
				},
			})
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
		if candidate.Y > y {
			continue
		}

		if candidate.Y == y {
			if candidate.X > x {
				continue
			}
		}

		if closest == nil {
			closest = candidate
			continue
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

func selectNextCandidate(
	candidates []*Candidate,
	dirX int,
	dirY int,
) {
	sign := func(value int) int {
		switch {
		case value > 0:
			return 1
		case value < 0:
			return -1
		default:
			return 0
		}
	}

	selected := getSelectedCandidate(candidates)
	if selected == nil {
		return
	}

	space := []*Candidate{}

	for _, candidate := range candidates {
		signX := sign(dirX)
		signY := sign(dirY)

		offsetX := sign(candidate.X - selected.X)
		offsetY := sign(candidate.Y - selected.Y)

		if dirY == 0 {
			if offsetY != 0 {
				continue
			}

			if signX != offsetX {
				continue
			}
		} else {
			if signY != offsetY {
				continue
			}
		}

		space = append(space, candidate)
	}

	if len(space) == 0 {
		return
	}

	closest := space[0]

	abs := func(x int) int {
		if x < 0 {
			x = -x
		}

		return x
	}

	for _, candidate := range space {
		distanceY := abs(selected.Y-candidate.Y) - abs(selected.Y-closest.Y)
		distanceX := abs(selected.X-candidate.X) - abs(selected.X-closest.X)

		if distanceY < 0 || distanceY == 0 && distanceX < 0 {
			closest = candidate
			continue
		}
	}

	closest.Selected = true
	selected.Selected = false
}

func getUniqueCandidates(candidates []*Candidate) []*Candidate {
	uniques := []*Candidate{}

	for _, candidate := range candidates {
		for _, unique := range uniques {
			if unique.Value == candidate.Value {
				goto skip
			}
		}

		uniques = append(uniques, candidate)

	skip:
	}

	return uniques
}
