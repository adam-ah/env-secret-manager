package main

import (
	"os"
	"testing"
)

func TestResolveSecret(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{Global: Project{Env: map[string]string{"TOKEN": "global-value", "OPENAI_KEY": "global-openai", "NUL": "bad\x00value"}}, Projects: map[string]Project{
		"app":         {Env: map[string]string{"TOKEN": "raw-value\nwith-newline", "EMPTY": ""}},
		cwd:           {Env: map[string]string{"TOKEN": "directory-value"}},
		"C:/code/app": {Env: map[string]string{"TOKEN": "windows-path-value"}},
	}}
	for _, test := range []struct {
		reference, want string
		fails           bool
	}{
		{"TOKEN", "global-value", false},
		{"app:TOKEN", "raw-value\nwith-newline", false},
		{"app:EMPTY", "", false},
		{cwd + ":TOKEN", "directory-value", false},
		{"C:/code/app:TOKEN", "windows-path-value", false},
		{":TOKEN", "", true},
		{"MISSING", "", true},
		{"app:MISSING", "", true},
		{"missing:TOKEN", "", true},
		{"NUL", "", true},
	} {
		t.Run(test.reference, func(t *testing.T) {
			_, value, err := resolveSecret(cfg, test.reference)
			if (err != nil) != test.fails || value != test.want {
				t.Fatalf("resolve: value %q error %v", value, err)
			}
		})
	}
}

func TestBareRetrievalNeverPrintsValues(t *testing.T) {
	testConfig(t, Config{Global: Project{Env: map[string]string{"OPENAI_KEY": "private-fixture"}}})
	for _, args := range [][]string{
		{"get", "OPENAI_KEY"},
		{"exec", "OPENAI_KEY"},
		{"exec", "OPENAI_KEY", "--"},
		{"exec", "--", "program"},
	} {
		output, err := captureOutput(t, func() error { return run(args) })
		if err == nil || output != "" {
			t.Fatalf("unsafe retrieval for %q: output %q error %v", args, output, err)
		}
	}
}
