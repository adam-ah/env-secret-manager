package main

import (
	"os"
	"strings"

	"github.com/atotto/clipboard"
	osc52 "github.com/aymanbagabas/go-osc52/v2"
	tea "github.com/charmbracelet/bubbletea"
)

type copiedCLIExample struct {
	err      error
	terminal bool
}

func (m workspace) secretReference() string {
	if m.selector() == globalSelector {
		return m.selectedSecret()
	}
	return m.selector() + ":" + m.selectedSecret()
}

func (m workspace) examples() secretExamples {
	return secretExamples{reference: m.secretReference(), name: m.selectedSecret()}
}

func (m workspace) cliExample() string {
	return m.examples().curlCommand()
}

func (m workspace) variableExample() string {
	return m.examples().variableCommand()
}

func (m workspace) envCurlExample() string {
	return m.examples().envCurlCommand()
}

func (m workspace) cliExampleView(w int) string {
	lines := []string{canvasText(envExampleTitle, accent, true) + canvasText("  ·  v Copy command", muted, false)}
	lines = append(lines, canvasText(fit(safeLabel(m.examples().envDescription()), w), muted, false))
	for _, line := range strings.Split(m.variableExample(), "\n") {
		lines = append(lines, canvasText(fit(line, w), ink, false))
	}
	if m.height >= 32 {
		lines = append(lines, "", canvasText(envCurlExampleTitle, accent, true)+canvasText("  ·  u Copy env curl", muted, false))
		for _, line := range strings.Split(m.envCurlExample(), "\n") {
			lines = append(lines, canvasText(fit(line, w), ink, false))
		}
	} else {
		lines = append(lines, canvasText("u Copy curl using the child env variable", muted, false))
	}
	if m.height >= 28 {
		lines = append(lines, "")
	}
	lines = append(lines, canvasText(curlExampleTitle, accent, true)+canvasText("  ·  c Copy curl command", muted, false))
	for _, line := range strings.Split(m.cliExample(), "\n") {
		lines = append(lines, canvasText(fit(line, w), ink, false))
	}
	return strings.Join(lines, "\n")
}

func copyCLIExample(command string) tea.Cmd {
	return func() tea.Msg {
		if err := clipboard.WriteAll(command); err == nil {
			return copiedCLIExample{}
		}
		sequence := osc52.New(command)
		if os.Getenv("TMUX") != "" {
			sequence = sequence.Tmux()
		}
		_, err := sequence.WriteTo(os.Stdout)
		return copiedCLIExample{err: err, terminal: true}
	}
}
