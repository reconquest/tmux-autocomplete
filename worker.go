package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/nsf/termbox-go"
	"github.com/reconquest/executil-go"
	"github.com/reconquest/karma-go"
)

type worker struct {
	args       *arguments
	tmux       *Tmux
	pane       *Pane
	lines      []string
	cursorX    int
	cursorY    int
	identifier *Identifier
	candidates []*Candidate
}

func captureWorker(args *arguments, tmux *Tmux) (*worker, error) {
	worker := &worker{args: args, tmux: tmux}

	_, err := fmt.Sscan(args.ValueCursorX, &worker.cursorX)
	if err != nil {
		return nil, karma.Format(err, "parse cursor x")
	}

	_, err = fmt.Sscan(args.ValueCursorY, &worker.cursorY)
	if err != nil {
		return nil, karma.Format(err, "parse cursor y")
	}

	worker.pane, err = CapturePane(tmux, args.ValuePane, "-eJ")
	if err != nil {
		return nil, karma.Format(err, "capture completion pane")
	}

	worker.lines = worker.pane.GetPrintable()

	return worker, nil
}

func (worker *worker) complete(theme *Theme) error {
	err := worker.findIdentifier()
	if err != nil {
		return err
	}

	if !worker.args.FlagNoPrefix && worker.identifier == nil {
		return nil
	}

	_, err = fmt.Fprint(
		os.Stdout,
		cursorSequence(worker.cursorX, worker.cursorY),
	)
	if err != nil {
		return karma.Describe("x", worker.cursorX).
			Describe("y", worker.cursorY).
			Format(err, "move cursor")
	}

	err = worker.findCandidates()
	if err != nil {
		return err
	}

	switch len(worker.candidates) {
	case 0:
		return nil

	case 1:
		return worker.useCurrentCandidate()

	default:
		return worker.selectCandidate(theme)
	}
}

func (worker *worker) findIdentifier() error {
	if worker.args.FlagNoPrefix {
		return nil
	}

	column, row := worker.pane.GetBufferXY(
		worker.lines, worker.cursorX, worker.cursorY,
	)

	identifier, err := getIdentifierToComplete(
		worker.args.ValueCursorRegexp, worker.lines, column, row,
	)
	if err != nil {
		return karma.Format(err, "find completion identifier")
	}

	worker.identifier = identifier

	return nil
}

func (worker *worker) findCandidates() error {
	candidates, err := getCompletionCandidates(
		worker.args.ValueCandidateRegexp, worker.lines, worker.identifier,
	)
	if err != nil {
		return karma.Format(err, "find completion candidates")
	}

	if len(candidates) == 0 {
		return nil
	}

	worker.candidates = getUniqueCandidates(candidates)
	if worker.identifier == nil {
		last := worker.candidates[len(worker.candidates)-1]

		worker.identifier = last.Identifier
	}

	selectDefaultCandidate(
		worker.candidates, worker.identifier.X, worker.identifier.Y,
	)

	return nil
}

func (worker *worker) selectCandidate(theme *Theme) error {
	err := termbox.Init()
	if err != nil {
		return karma.Format(err, "initialize terminal")
	}

	displayPane := *worker.pane

	displayPane.Width, displayPane.Height = termbox.Size()

	for {
		done, eventErr := worker.pollCandidate(&displayPane, theme)
		if eventErr != nil {
			return eventErr
		}

		if done {
			return nil
		}
	}
}

func (worker *worker) pollCandidate(
	displayPane *Pane, theme *Theme,
) (bool, error) {
	err := worker.render(displayPane, theme)
	if err != nil {
		return false, err
	}

	return worker.handleEvent(termbox.PollEvent(), displayPane)
}

func (worker *worker) render(displayPane *Pane, theme *Theme) error {
	text := renderPane(displayPane, theme) +
		renderIdentifier(worker.lines, displayPane, theme, worker.identifier) +
		renderCandidates(worker.lines, displayPane, theme, worker.candidates)

	_, err := fmt.Fprint(os.Stdout, text)
	if err != nil {
		return karma.Format(err, "render completion")
	}

	return nil
}

func (worker *worker) handleEvent(
	event termbox.Event, displayPane *Pane,
) (bool, error) {
	switch event.Type {
	case termbox.EventKey:
		return worker.handleKey(event)

	case termbox.EventResize:
		displayPane.Width, displayPane.Height = event.Width, event.Height

		err := termbox.Sync()
		if err != nil {
			return false, karma.Describe("width", event.Width).
				Describe("height", event.Height).
				Format(err, "sync resized terminal")
		}

	case termbox.EventError:
		return false, karma.Format(event.Err, "read terminal event")

	case termbox.EventMouse, termbox.EventInterrupt,
		termbox.EventRaw, termbox.EventNone:

	default:
		return false, nil
	}

	return false, nil
}

func (worker *worker) handleKey(event termbox.Event) (bool, error) {
	switch {
	case event.Ch == 'k', event.Key == termbox.KeyArrowUp:
		selectNextCandidate(worker.candidates, 0, -1)

	case event.Ch == 'j', event.Key == termbox.KeyArrowDown:
		selectNextCandidate(worker.candidates, 0, 1)

	case event.Ch == 'h', event.Key == termbox.KeyArrowLeft:
		selectNextCandidate(worker.candidates, -1, 0)

	case event.Ch == 'l', event.Key == termbox.KeyArrowRight:
		selectNextCandidate(worker.candidates, 1, 0)

	case event.Key == termbox.KeyEnter:
		return true, worker.useCurrentCandidate()

	case event.Key == termbox.KeyCtrlC:
		return true, nil

	default:
		return false, nil
	}

	return false, nil
}

func (worker *worker) useCurrentCandidate() error {
	selected := getSelectedCandidate(worker.candidates)
	if selected == nil {
		return nil
	}

	text := selected.Value
	if worker.args.ValueProgram != "" {
		return executeCandidate(worker.args.ValueProgram, text)
	}

	if !worker.args.FlagNoPrefix {
		text = string([]rune(text)[worker.identifier.Length():])
	}

	err := worker.tmux.Paste(text, "-t", worker.pane.ID)
	if err != nil {
		return karma.Describe("pane", worker.pane.ID).
			Format(err, "paste completion")
	}

	return nil
}

func executeCandidate(program, text string) error {
	// #nosec G204 -- The --exec option runs a program without a shell.
	cmd := exec.CommandContext(context.Background(), program, text)

	_, _, err := executil.Run(cmd)
	if err != nil {
		return karma.Describe("program", program).
			Format(err, "execute completion program")
	}

	return nil
}
