package main

import (
	"os"
	"testing"
)

func TestNonInteractiveStartupShowsAgentHelp(t *testing.T) {
	testConfig(t, Config{})
	if err := os.WriteFile(configPath(), []byte("invalid = ["), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TERM", "xterm-256color")
	input, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	originalInput := os.Stdin
	os.Stdin = input
	t.Cleanup(func() {
		os.Stdin = originalInput
		input.Close()
	})
	for _, test := range []struct {
		name string
		args []string
	}{
		{"startup", nil},
		{"menu", []string{"menu"}},
		{"edit", []string{"edit"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, err := captureStreams(t, func() error { return run(test.args) })
			if err != nil || stderr != "" || stdout != helpText {
				t.Fatalf("expected agent help without opening config or terminal: error=%v stderr=%q stdout=%q", err, stderr, stdout)
			}
		})
	}
}
