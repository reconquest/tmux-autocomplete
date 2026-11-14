package main

import (
	"strings"

	"github.com/mgutz/ansi"
)

func renderPane(pane *Pane, theme *Theme) string {
	text := reEscapeSequence.ReplaceAllStringFunc(
		pane.String(),
		func(sequence string) string {
			return decolorize(sequence, theme)
		},
	)

	return cursorSequence(0, 0) + ansi.ColorCode(theme.Fog.Text) + text
}

func decolorize(sequence string, theme *Theme) string {
	matches := reEscapeSequence.FindStringSubmatch(sequence)
	code := matches[1]

	switch code {
	case "49":
		return ansi.DefaultBG + ansi.ColorCode(theme.Fog.Text)

	case "0", "39":
		return ansi.Reset + ansi.ColorCode(theme.Fog.Text)

	case "1":
		return ""

	case "7":
		return ansi.ColorCode(theme.Fog.Background)

	default:
		return dimColor(code, theme)
	}
}

func dimColor(code string, theme *Theme) string {
	if len(code) == 1 {
		return ""
	}

	switch code[0] {
	case '3', '9':
		return ansi.ColorCode(theme.Fog.Text)

	case '4', '1':
		return ansi.ColorCode(theme.Fog.Background)

	default:
		return ansi.Reset
	}
}

func renderIdentifier(
	lines []string, pane *Pane, theme *Theme, identifier *Identifier,
) string {
	return cursorSequence(pane.GetScreenXY(lines, identifier.X, identifier.Y)) +
		ansi.ColorFunc(theme.Identifier)(identifier.Value)
}

func renderCandidates(
	lines []string, pane *Pane, theme *Theme, candidates []*Candidate,
) string {
	rendered := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if !candidate.Selected {
			rendered = append(
				rendered,
				renderCandidate(lines, pane, theme, candidate),
			)
		}
	}

	for _, candidate := range candidates {
		if candidate.Selected {
			rendered = append(
				rendered,
				renderCandidate(lines, pane, theme, candidate),
			)
		}
	}

	return strings.Join(rendered, "")
}

func renderCandidate(
	lines []string, pane *Pane, theme *Theme, candidate *Candidate,
) string {
	color := theme.Candidate.Normal
	if candidate.Selected {
		color = theme.Candidate.Selected
	}

	return cursorSequence(pane.GetScreenXY(lines, candidate.X, candidate.Y)) +
		ansi.ColorFunc(color)(candidate.Value)
}
