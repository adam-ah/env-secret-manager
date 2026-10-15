package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallOnlyPrintsShellHookInstructions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/bash")
	for _, shell := range []string{"bash", "zsh"} {
		stdout, stderr, err := captureStreams(t, func() error { return run([]string{"install", shell}) })
		if err != nil || stderr != "" || !strings.Contains(stdout, "~/."+shell+"rc") || !strings.Contains(stdout, shellHook(shell)) {
			t.Fatalf("install %s: error=%v stdout=%q stderr=%q", shell, err, stdout, stderr)
		}
	}
	stdout, _, err := captureStreams(t, func() error { return run([]string{"install"}) })
	if err != nil || !strings.Contains(stdout, "~/.bashrc") {
		t.Fatal("install did not default to the user's shell")
	}
	entries, err := os.ReadDir(home)
	if err != nil || len(entries) != 0 {
		t.Fatal("install changed files")
	}
	for _, args := range [][]string{{"install", "fish"}, {"install", "bash", "extra"}, {"export"}} {
		stdout, _, err := captureStreams(t, func() error { return run(args) })
		if err == nil || stdout != "" {
			t.Fatalf("unexpected command result for %q: error=%v stdout=%q", args, err, stdout)
		}
	}
}

func TestShellHookChildProcess(t *testing.T) {
	if os.Getenv("SECRETENV_TEST_HOOK_CHILD") != "1" {
		return
	}
	if err := run([]string{"_hook"}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func TestShellHooksLoadAndClearProjectVariables(t *testing.T) {
	for _, shell := range []string{"bash", "zsh"} {
		t.Run(shell, func(t *testing.T) {
			program, err := exec.LookPath(shell)
			if err != nil {
				t.Skip(shell + " unavailable")
			}
			root := t.TempDir()
			project := filepath.Join(root, "my-app")
			if err := os.Mkdir(project, 0700); err != nil {
				t.Fatal(err)
			}
			testConfig(t, Config{
				Global:   Project{Env: map[string]string{"GLOBAL_ONLY": "global-private"}},
				Projects: map[string]Project{project: {Env: map[string]string{"SAMPLE_SECRET": "project-'quoted'-$literal"}}},
			})
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("SECRETENV_TEST_HOOK_CHILD", "1")
			preamble := "set -e\nunset GLOBAL_ONLY SAMPLE_SECRET\nsecretenv() { " + shellQuote(executable) + " -test.run '^TestShellHookChildProcess$'; }\ncd " + shellQuote(project) + "\n"
			checkPrompt := "[[ ${#chpwd_functions[@]} == 1 ]]\n"
			if shell == "bash" {
				preamble += "PROMPT_COMMAND=('printf existing-prompt')\n"
				checkPrompt = "[[ ${#PROMPT_COMMAND[@]} == 2 && ${PROMPT_COMMAND[0]} == _secretenv_hook && ${PROMPT_COMMAND[1]} == 'printf existing-prompt' ]]\n"
			}
			checks := "[[ $SAMPLE_SECRET == " + shellQuote("project-'quoted'-$literal") + " ]]\n[[ -z ${GLOBAL_ONLY+x} ]]\nprintf 'loaded\\n'\ncd " + shellQuote(root) + "\n_secretenv_hook\n[[ -z ${SAMPLE_SECRET+x} && -z ${GLOBAL_ONLY+x} && -z ${SECRETENV_ACTIVE_PROJECT-} ]]\nprintf 'cleared\\n'\n"
			command := preamble + shellHook(shell) + shellHook(shell) + checkPrompt + checks
			output, err := exec.Command(program, "-c", command).CombinedOutput()
			if err != nil || string(output) != "loaded\ncleared\n" {
				t.Fatalf("hook failed: error=%v output=%q", err, output)
			}
		})
	}
}
