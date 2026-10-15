package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestGlobalScopeIsPermanentButSecretsCanBeEdited(t *testing.T) {
	testConfig(t, Config{Projects: map[string]Project{}})
	m := newWorkspace(Config{Projects: map[string]Project{}})
	if m.selector() != globalSelector || projectTitle(m.selector()) != "Global" {
		t.Fatal("Global is missing from an empty config")
	}
	m = uiType(m, "a")
	m = uiType(m, "OPENAI_KEY")
	m = uiKey(m, tea.KeyTab, "")
	m = uiType(m, "global-private-fixture")
	m = uiKey(m, tea.KeyCtrlS, "")
	cfg, err := loadConfig()
	if err != nil || cfg.Global.Env["OPENAI_KEY"] != "global-private-fixture" {
		t.Fatal("global secret not saved")
	}
	m.focus = 0
	m = uiType(m, "d")
	if m.confirmation != "" {
		t.Fatal("Global can be deleted")
	}
	m = uiType(m, "e")
	if m.form != "" {
		t.Fatal("Global can be renamed")
	}
	m.confirmAction = "delete-project"
	model, _ := m.performConfirmed()
	m = model.(workspace)
	if m.selector() != globalSelector || m.cfg.Global.Env["OPENAI_KEY"] != "global-private-fixture" {
		t.Fatal("Global was deleted")
	}
	m.focus = 1
	m = uiType(m, "d")
	m = uiType(m, "y")
	cfg, err = loadConfig()
	if err != nil || len(cfg.Global.Env) != 0 || m.selector() != globalSelector {
		t.Fatal("deleting a global secret removed Global or failed to save")
	}
}

func TestGlobalSecretsNeverAutoExport(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	testConfig(t, Config{Global: Project{Env: map[string]string{"OPENAI_KEY": "global-private-fixture"}}, Projects: map[string]Project{cwd: {Env: map[string]string{"LOCAL_TOKEN": "project-fixture"}}}})
	output, err := captureOutput(t, exportForCurrentDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output, "OPENAI_KEY") || strings.Contains(output, "global-private-fixture") {
		t.Fatal("Global entered shell exports")
	}
	if !strings.Contains(output, "LOCAL_TOKEN") {
		t.Fatal("project exports stopped working")
	}
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	_, value, err := resolveSecret(cfg, "OPENAI_KEY")
	if err != nil || value != "global-private-fixture" {
		t.Fatal("global retrieval requires a project")
	}
}

func TestDiscoveryReferences(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	testConfig(t, Config{Global: Project{Env: map[string]string{"OPENAI_KEY": "global-private-fixture"}}, Projects: map[string]Project{cwd: {Env: map[string]string{"API_TOKEN": "project-private-fixture"}}, "other-app": {Env: map[string]string{"TOKEN": "other-private-fixture"}}}})
	output, err := captureOutput(t, func() error { return listSecrets([]string{"--current"}) })
	if err != nil {
		t.Fatal(err)
	}
	var entries []struct {
		Project    string   `json:"project"`
		Scope      string   `json:"scope"`
		References []string `json:"references"`
	}
	if err := json.Unmarshal([]byte(output), &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Scope != "global" || len(entries[0].References) != 1 || entries[0].References[0] != "OPENAI_KEY" || entries[1].References[0] != cwd+":API_TOKEN" {
		t.Fatalf("unexpected discovery output: %s", output)
	}
	if strings.Contains(output, "private-fixture") || strings.Contains(output, "other-app") {
		t.Fatal("current discovery exposed values or unrelated projects")
	}
	output, err = captureOutput(t, func() error { return listSecrets([]string{"--current", "--project", "other-app"}) })
	if err == nil || output != "" {
		t.Fatal("conflicting discovery options accepted")
	}
}

func TestEditorRejectsNamesThatNeedShellQuoting(t *testing.T) {
	testConfig(t, Config{Projects: map[string]Project{}})
	for _, name := range []string{"my app", "bad'quote", "bad\"quote", "$(command)", "name;command", "name*", "~/code/app", "path\\app", "-option", "global", "Global", "name\nline"} {
		m := newWorkspace(Config{Projects: map[string]Project{}})
		model, _ := m.startForm("project")
		m = model.(workspace)
		m.fields[0].SetValue(name)
		model, _ = m.submitForm()
		m = model.(workspace)
		if m.formError == "" || len(m.cfg.Projects) != 0 {
			t.Fatalf("editor accepted unsafe name %q", name)
		}
	}
	for _, name := range []string{"my-app", "app_2", "app.v3", "/srv/code/my-app", "C:/code/my-app"} {
		if !validProjectName(name) {
			t.Fatalf("safe name rejected: %q", name)
		}
		if shellArgument(name+":API_TOKEN") != name+":API_TOKEN" {
			t.Fatalf("safe reference requires quoting: %q", name)
		}
	}
}
