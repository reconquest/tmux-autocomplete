package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testcase struct {
	path     string
	lines    []string
	cursorX  int
	cursorY  int
	expected []string
}

func TestCompletion(t *testing.T) {
	test := assert.New(t)

	scenarios, err := filepath.Glob("unit_tests/*")
	if err != nil {
		panic(err)
	}

	only := os.Getenv("ONLY")
	if only != "" {
		log.Printf("filter for testcases: *%s*", only)
	}

	testcases := []testcase{}
	for _, scenario := range scenarios {
		// ignore dirs
		if isFileExists(scenario) {
			if strings.Contains(scenario, only) {
				testcases = append(testcases, getTestcase(scenario))
			}
		}
	}

	if len(scenarios) == 0 {
		panic("no tests found")
	}

	for _, testcase := range testcases {
		id, err := getIdentifierToComplete(
			defaultRegexpCursor,
			testcase.lines,
			testcase.cursorX,
			testcase.cursorY,
		)
		if err != nil {
			test.Errorf(err, "unable to get completion identifier: %s", testcase.path)
			continue
		}

		if !test.NotNilf(id, "invalid prompt (identifier = nil) in %s", testcase.path) {
			continue
		}

		candidates, err := getCompletionCandidates(defaultRegexpCandidate, testcase.lines, id)
		if err != nil {
			test.Errorf(err, "unable to get completion candidates: %s", testcase.path)
			continue
		}

		candidates = getUniqueCandidates(candidates)

		values := []string{}
		for _, candidate := range candidates {
			values = append(values, candidate.Value)
		}

		test.EqualValues(testcase.expected, values, "%s", testcase.path)
	}
}

func getTestcase(filename string) testcase {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	src := strings.Split(strings.TrimSpace(string(data)), "\n")

	pane := src[0]
	prompt := src[1]
	candidates := src[2:]

	// no cache because zfs already has arc
	paneData, err := ioutil.ReadFile("unit_tests/" + pane)
	if err != nil {
		panic(err)
	}

	lines := strings.Split(string(paneData), "\n")

	prefix := "$ "
	lines = append(lines, prefix+prompt)
	cursorY := len(lines) - 1
	cursorX := len(prefix + prompt)

	return testcase{
		path:     filename,
		lines:    lines,
		cursorX:  cursorX,
		cursorY:  cursorY,
		expected: candidates,
	}
}

func isFileExists(path string) bool {
	stat, err := os.Stat(path)
	return !os.IsNotExist(err) && !stat.IsDir()
}
