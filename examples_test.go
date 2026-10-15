package main

import (
	"os"
	"strings"
	"testing"
)

func TestPublishedExamplesStayConsistent(t *testing.T) {
	if lines := strings.Count(helpText, "\n"); lines > 24 {
		t.Fatalf("help exceeds one screen: %d lines", lines)
	}
	for _, path := range []string{"docs/usage.md"} {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, command := range []string{sampleExamples.variableCommand(), sampleExamples.envCurlCommand(), sampleExamples.curlCommand()} {
			if !strings.Contains(string(body), "```sh\n"+command+"\n```") || !strings.Contains(helpText, command) {
				t.Fatalf("%s and CLI help must use the same complete, copyable example: %s", path, command)
			}
		}
	}
	for _, selector := range []string{globalSelector, "go-secrets"} {
		cfg := Config{Projects: map[string]Project{}}
		setScopedProject(&cfg, selector, Project{Env: map[string]string{"SAMPLE_SECRET": "private-canary"}})
		m := uiSize(newWorkspace(cfg), 144, 40)
		if selector != globalSelector {
			m.projectIndex = 1
		}
		m.page = 2
		guide := m.guideContent()
		for _, view := range []string{m.cliExampleView(140), guide} {
			if !strings.Contains(view, "Available as $SAMPLE_SECRET in the child process") || strings.Contains(view, "private-canary") || strings.Contains(view, "$go-secrets:") {
				t.Fatal("UI help does not explain the child variable consistently")
			}
		}
	}
}
