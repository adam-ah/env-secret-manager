package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestRenderDocumentation(t *testing.T) {
	if os.Getenv("SECRETENV_RENDER_DOCS") != "1" {
		t.Skip("run ./render.sh to refresh documentation images")
	}
	profile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })
	cfg := Config{
		Global: Project{Env: map[string]string{"OPENAI_KEY": "demo-only-private", "DATABASE_URL": "demo-only-private"}},
		Projects: map[string]Project{
			"my-app":  {Env: map[string]string{"API_TOKEN": "demo-only-private", "DATABASE_URL": "demo-only-private", "WEBHOOK_SECRET": "demo-only-private"}},
			"website": {Env: map[string]string{"DEPLOY_TOKEN": "demo-only-private"}},
		},
	}
	m := uiSize(newWorkspace(cfg), 144, 38)
	m.projectIndex = 0
	m.focus = 1
	m.secretIndex = 1
	global := m.View()
	m.projectIndex = 1
	m.secretIndex = 0
	project := m.View()
	model, _ := m.startForm("secret")
	form := model.(workspace)
	form.fields[0].SetValue("SERVICE_TOKEN")
	form.fields[1].SetValue("demo-only-private")
	form.fieldIndex = 1
	form.fields[0].Blur()
	form.fields[1].Focus()
	m.page = 2
	m.resizeGuide()
	guide := m.View()
	file, err := os.Create("demo.cast")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(map[string]any{"version": 2, "width": 144, "height": 38, "title": "secretenv: Global, projects, editor and AI access", "env": map[string]string{"TERM": "xterm-256color"}}); err != nil {
		t.Fatal(err)
	}
	for _, frame := range []struct {
		time float64
		view string
	}{{0, global}, {4, project}, {8, form.View()}, {12, guide}, {20, global}} {
		if strings.Contains(frame.view, "demo-only-private") {
			t.Fatal("secret value in documentation frame")
		}
		output := fmt.Sprintf("\x1b[?25l\x1b[2J\x1b[H%s", strings.ReplaceAll(frame.view, "\n", "\r\n"))
		if err := encoder.Encode([]any{frame.time, "o", output}); err != nil {
			t.Fatal(err)
		}
	}
}
