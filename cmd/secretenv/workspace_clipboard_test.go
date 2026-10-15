package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCLIExamplesCanBePastedIntoShell(t *testing.T) {
	project := "~/code/it's $(printf injected)"
	name := "API_TOKEN"
	m := uiSize(newWorkspace(Config{Projects: map[string]Project{project: {Env: map[string]string{name: "private-canary"}}}}), 144, 40)
	m.projectIndex = 1
	shell, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh unavailable")
	}
	prefix := "secretenv() { printf '%s\\0' \"$@\"; }\n"
	for _, test := range []struct {
		command string
		want    []string
	}{
		{m.cliExample(), []string{"exec", project + ":" + name, "--", "curl", "--header", "Authorization: Bearer {API_TOKEN}", "https://api.example.com"}},
		{m.variableExample(), []string{"exec", project + ":" + name, "--", "your-program"}},
		{m.envCurlExample(), []string{"exec", project + ":" + name, "--", "sh", "-c", "exec curl --silent --show-error \\\n  --header \"Authorization: Bearer $API_TOKEN\" \\\n  https://api.example.com"}},
	} {
		output, err := exec.Command(shell, "-c", prefix+test.command).Output()
		if err != nil {
			t.Fatalf("example cannot be executed: %v", err)
		}
		got := strings.Split(strings.TrimSuffix(string(output), "\x00"), "\x00")
		if !reflect.DeepEqual(got, test.want) {
			t.Fatalf("arguments: got %q, want %q", got, test.want)
		}
		if strings.Contains(test.command, "private-canary") {
			t.Fatal("sample contains a secret value")
		}
		for _, line := range strings.Split(test.command, "\n") {
			if strings.HasSuffix(strings.TrimRight(line, " "), "\\") && !strings.HasSuffix(line, "\\") {
				t.Fatal("spaces after continuation")
			}
		}
	}
}

func TestCLIExamplesLiveBelowPanes(t *testing.T) {
	m := uiSize(newWorkspace(Config{Projects: map[string]Project{"app": {Env: map[string]string{"API_TOKEN": "private-canary"}}}}), 144, 40)
	m.projectIndex = 1
	view := m.View()
	for _, label := range []string{"INJECT AS AN ENV VARIABLE", "Available as $API_TOKEN in the child process", "CURL USING THE ENV VARIABLE", "INJECT INTO CURL"} {
		if !strings.Contains(view, label) {
			t.Fatal("missing bottom example:", label)
		}
	}
	if strings.ContainsAny(m.cliExampleView(140), "│╭╮╰╯") {
		t.Fatal("sample contains panel borders")
	}
	for _, key := range []rune{'c', 'v', 'u'} {
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
		if cmd == nil {
			t.Fatal("copy shortcut is not connected:", string(key))
		}
	}
}

func TestEnvCurlExampleUsesChildEnvironment(t *testing.T) {
	for _, program := range []string{"sh", "curl"} {
		if _, err := exec.LookPath(program); err != nil {
			t.Skip(program + " unavailable")
		}
	}
	for _, selector := range []string{globalSelector, "screeners"} {
		t.Run(selector, func(t *testing.T) {
			name := "CLOUDFLARE_API_TOKEN"
			secret := "demo-$value-$(echo-oops)-'quoted'-\"double\"-%s"
			cfg := Config{Projects: map[string]Project{}}
			setScopedProject(&cfg, selector, Project{Env: map[string]string{name: secret}})
			testConfig(t, cfg)
			t.Setenv(name, "wrong-parent-value")
			m := uiSize(newWorkspace(cfg), 144, 40)
			if selector != globalSelector {
				m.projectIndex = 1
			}
			received := make(chan string, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				header := r.Header.Get("Authorization")
				received <- header
				fmt.Fprint(w, header)
			}))
			defer server.Close()
			command := strings.ReplaceAll(m.envCurlExample(), "https://api.example.com", server.URL)
			prefix := "secretenv() { printf '%s\\0' \"$@\"; }\n"
			output, err := exec.Command("sh", "-c", prefix+command).Output()
			if err != nil {
				t.Fatal(err)
			}
			args := strings.Split(strings.TrimSuffix(string(output), "\x00"), "\x00")
			stdout, stderr, err := captureStreams(t, func() error { return run(args) })
			if err != nil || stdout != "Bearer [REDACTED]" || stderr != "" {
				t.Fatalf("env curl failed: error=%v stdout=%q stderr=%q", err, stdout, stderr)
			}
			select {
			case header := <-received:
				if header != "Bearer "+secret {
					t.Fatal("curl did not use the injected environment value literally")
				}
			default:
				t.Fatal("curl did not reach the local test server")
			}
		})
	}
}
