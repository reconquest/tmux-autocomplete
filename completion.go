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
	pane *Pane,
	x int,
	y int,
) (*Identifier, error) {
	lines := pane.Printable()

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
	pane *Pane,
	identifier *Identifier,
) ([]*Candidate, error) {
	lines := pane.Printable()

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

	distance := func(a, b *Candidate) int {
		metric := func(x1, y1, x2, y2 int) int {
			// we multiply y distance by 1.5 due font proportions
			return (x1-x2)*(x1-x2) + (y1-y2)*(y1-y2)*3/2
		}

		min := func(values ...int) int {
			min := values[0]
			for _, value := range values {
				if value < min {
					min = value
				}
			}

			return min
		}

		var (
			ax = a.X
			ay = a.Y
			bx = b.X
			by = b.Y

			al = a.Length()
			bl = b.Length()
		)

		return min(
			metric(ax, ay, bx, by),
			metric(ax+al, ay, bx+bl, by),
			metric(ax+al, ay, bx, by),
			metric(ax, ay, bx+al, by),
		)
	}

	selected := getSelectedCandidate(candidates)
	if selected == nil {
		return
	}

	cone := []*Candidate{}

	for _, candidate := range candidates {
		signX := sign(dirX)
		signY := sign(dirY)

		offsetX := sign(candidate.X - selected.X)
		offsetY := sign(candidate.Y - selected.Y)

		if dirX != 0 && signX != offsetX {
			continue
		}

		if dirY != 0 && signY != offsetY {
			continue
		}

		cone = append(cone, candidate)
	}

	if len(cone) == 0 {
		return
	}

	closest := cone[0]

	for _, candidate := range cone {
		if distance(selected, candidate) < distance(selected, closest) {
			closest = candidate
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
