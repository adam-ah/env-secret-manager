package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func uiKey(m workspace, kind tea.KeyType, value string) workspace {
	model, _ := m.Update(tea.KeyMsg{Type: kind, Runes: []rune(value)})
	return model.(workspace)
}
func uiType(m workspace, value string) workspace { return uiKey(m, tea.KeyRunes, value) }
func uiSize(m workspace, w, h int) workspace {
	model, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return model.(workspace)
}
func TestWorkspaceEditingAndConfirmation(t *testing.T) {
	testConfig(t, Config{Projects: map[string]Project{}})
	m := newWorkspace(Config{Projects: map[string]Project{}})
	m = uiType(m, "n")
	m = uiType(m, "test-app")
	m = uiKey(m, tea.KeyCtrlS, "")
	if m.selector() != "test-app" || m.form != "" {
		t.Fatal("project was not created")
	}
	m = uiType(m, "a")
	m = uiType(m, "API_TOKEN")
	m = uiKey(m, tea.KeyTab, "")
	m = uiType(m, "masked-canary-78132")
	if strings.Contains(m.View(), "masked-canary-78132") {
		t.Fatal("editor rendered a secret")
	}
	m = uiKey(m, tea.KeyCtrlS, "")
	cfg, err := loadConfig()
	if err != nil || cfg.Projects["test-app"].Env["API_TOKEN"] != "masked-canary-78132" {
		t.Fatal("secret was not saved")
	}
	m = uiType(m, "e")
	m = uiKey(m, tea.KeyTab, "")
	m = uiType(m, "replacement-canary-78132")
	m = uiKey(m, tea.KeyCtrlS, "")
	if m.confirmation == "" || m.confirmYes {
		t.Fatal("replacement must require confirmation, defaulting to cancel")
	}
	m = uiKey(m, tea.KeyEnter, "")
	cfg, _ = loadConfig()
	if cfg.Projects["test-app"].Env["API_TOKEN"] != "masked-canary-78132" {
		t.Fatal("cancel replaced a secret")
	}
	m = uiKey(m, tea.KeyCtrlS, "")
	m = uiType(m, "y")
	cfg, _ = loadConfig()
	if cfg.Projects["test-app"].Env["API_TOKEN"] != "replacement-canary-78132" || m.form != "" || len(m.fields) != 0 {
		t.Fatal("replacement was not saved or editor retained its input")
	}
	m = uiType(m, "d")
	m = uiKey(m, tea.KeyEnter, "")
	if m.selectedSecret() != "API_TOKEN" {
		t.Fatal("cancel deleted a secret")
	}
	m = uiType(m, "d")
	m = uiType(m, "y")
	cfg, _ = loadConfig()
	if len(cfg.Projects["test-app"].Env) != 0 {
		t.Fatal("confirmed deletion was not saved")
	}
}
func TestWorkspaceSearchAndCancel(t *testing.T) {
	m := newWorkspace(Config{Projects: map[string]Project{"app": {Env: map[string]string{"API_TOKEN": "canary", "DATABASE_URL": "other"}}}})
	m.projectIndex = 1
	m = uiType(m, "/")
	m = uiType(m, "database")
	if len(m.names()) != 1 || m.selectedSecret() != "DATABASE_URL" {
		t.Fatal("search did not filter names")
	}
	m = uiKey(m, tea.KeyEsc, "")
	if len(m.names()) != 2 || m.searching {
		t.Fatal("escape did not clear the filter")
	}
	m = uiType(m, "a")
	m = uiType(m, "UNSAVED")
	m = uiKey(m, tea.KeyTab, "")
	m = uiType(m, "temporary-secret")
	m = uiKey(m, tea.KeyEsc, "")
	if m.form != "" || len(m.fields) != 0 || len(m.cfg.Projects["app"].Env) != 2 {
		t.Fatal("cancel retained input or changed config")
	}
}
func TestWorkspaceLayoutAndSecretPrivacy(t *testing.T) {
	cfg := Config{Projects: map[string]Project{"~/code/my-app": {Env: map[string]string{"API_TOKEN": "never-render-this-canary", "DATABASE_URL": "also-private"}}}}
	for _, size := range [][2]int{{64, 24}, {80, 24}, {100, 32}, {144, 24}, {144, 40}} {
		for _, empty := range []bool{false, true} {
			active := cfg
			if empty {
				active = Config{Projects: map[string]Project{}}
			}
			m := uiSize(newWorkspace(active), size[0], size[1])
			if !empty {
				m.projectIndex = 1
			}
			for page := 0; page < 4; page++ {
				m.page = page
				m.resizeGuide()
				view := m.View()
				if strings.Contains(view, "never-render-this-canary") || strings.Contains(view, "also-private") {
					t.Fatal("secret leaked into workspace")
				}
				if lipgloss.Width(view) > size[0] || lipgloss.Height(view) > size[1] {
					t.Fatalf("layout overflow at %v page %d: %dx%d", size, page, lipgloss.Width(view), lipgloss.Height(view))
				}
				if !strings.Contains(strings.ToLower(ansi.Strip(view)), "quit") {
					t.Fatalf("footer clipped at %v page %d", size, page)
				}
			}
			m.page = 0
			model, _ := m.startForm("secret")
			m = model.(workspace)
			if !strings.Contains(ansi.Strip(m.View()), "Save") {
				t.Fatalf("save button clipped at %v", size)
			}
		}
	}
}
