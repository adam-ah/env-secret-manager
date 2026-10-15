package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
)

func captureStreams(t *testing.T, fn func() error) (string, string, error) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = outW, errW
	var stdout, stderr bytes.Buffer
	outDone, errDone := make(chan error, 1), make(chan error, 1)
	go func() { _, err := io.Copy(&stdout, outR); outDone <- err }()
	go func() { _, err := io.Copy(&stderr, errR); errDone <- err }()
	result := fn()
	outW.Close()
	errW.Close()
	os.Stdout, os.Stderr = oldOut, oldErr
	if err := <-outDone; err != nil {
		t.Fatal(err)
	}
	if err := <-errDone; err != nil {
		t.Fatal(err)
	}
	outR.Close()
	errR.Close()
	return stdout.String(), stderr.String(), result
}

func TestSecretChildProcess(t *testing.T) {
	if os.Getenv("SECRETENV_TEST_CHILD") != "1" {
		return
	}
	if os.Getenv("PARENT_TEST_VALUE") != "inherited" {
		os.Exit(91)
	}
	if os.Getenv("OPENAI_KEY") != "alpha-token-78132" || os.Getenv("API_TOKEN") != "beta-token-98421" {
		os.Exit(92)
	}
	if value, exists := os.LookupEnv("EMPTY"); !exists || value != "" {
		os.Exit(94)
	}
	cwd, err := os.Getwd()
	if err != nil || cwd != os.Getenv("PARENT_TEST_CWD") {
		os.Exit(93)
	}
	for _, value := range []string{os.Getenv("OPENAI_KEY"), os.Getenv("API_TOKEN")} {
		fmt.Fprint(os.Stdout, "out ", value[:4])
		fmt.Fprintln(os.Stdout, value[4:])
		fmt.Fprint(os.Stderr, "err ", value[:4])
		fmt.Fprintln(os.Stderr, value[4:])
	}
	for i, arg := range os.Args {
		if arg == "--" {
			for _, value := range os.Args[i+1:] {
				fmt.Fprintln(os.Stdout, "arg "+value)
			}
			break
		}
	}
	if code := os.Getenv("SECRETENV_CHILD_EXIT"); code != "" {
		n, _ := strconv.Atoi(code)
		os.Exit(n)
	}
	os.Exit(0)
}

func TestExecInjectsMultipleSecretsAndFiltersBothStreams(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	testConfig(t, Config{Global: Project{Env: map[string]string{"OPENAI_KEY": "alpha-token-78132", "EMPTY": ""}}, Projects: map[string]Project{"app": {Env: map[string]string{"API_TOKEN": "beta-token-98421"}}}})
	t.Setenv("SECRETENV_TEST_CHILD", "1")
	t.Setenv("PARENT_TEST_VALUE", "inherited")
	t.Setenv("PARENT_TEST_CWD", cwd)
	t.Setenv("OPENAI_KEY", "old-parent-value")
	args := []string{"OPENAI_KEY", "app:API_TOKEN", "EMPTY", "--", executable, "-test.run=^TestSecretChildProcess$", "--", "header={API_TOKEN}", "both={OPENAI_KEY}:{API_TOKEN}"}
	stdout, stderr, err := captureStreams(t, func() error { return execSecrets(args) })
	if err != nil {
		t.Fatalf("exec: %v stdout=%q stderr=%q", err, stdout, stderr)
	}
	for _, output := range []string{stdout, stderr} {
		if strings.Contains(output, "alpha-token-78132") || strings.Contains(output, "beta-token-98421") {
			t.Fatal("child output exposed a secret")
		}
		if strings.Count(output, "[REDACTED]") < 2 {
			t.Fatalf("both secrets were not filtered: %q", output)
		}
	}
	if !strings.Contains(stdout, "arg header=[REDACTED]") || !strings.Contains(stdout, "arg both=[REDACTED]:[REDACTED]") {
		t.Fatalf("argument injection failed: %q", stdout)
	}
	if os.Getenv("OPENAI_KEY") != "old-parent-value" {
		t.Fatal("parent environment was changed")
	}
	t.Setenv("SECRETENV_CHILD_EXIT", "37")
	stdout, stderr, err = captureStreams(t, func() error { return execSecrets(args) })
	var childErr *childExitError
	if !errors.As(err, &childErr) || childErr.code != 37 {
		t.Fatalf("exit status lost: %v", err)
	}
	if strings.Contains(stdout+stderr, "alpha-token-78132") {
		t.Fatal("failed child exposed a secret")
	}
}

func TestExecValidatesAllReferencesBeforeLaunching(t *testing.T) {
	testConfig(t, Config{Global: Project{Env: map[string]string{"TOKEN": "global-fixture"}}, Projects: map[string]Project{"app": {Env: map[string]string{"TOKEN": "project-fixture"}}}})
	for _, args := range [][]string{
		{"TOKEN", "app:TOKEN", "--", "echo", "should-not-run"},
		{"TOKEN", "MISSING", "--", "echo", "should-not-run"},
		{"missing:TOKEN", "--", "echo", "should-not-run"},
	} {
		stdout, stderr, err := captureStreams(t, func() error { return execSecrets(args) })
		if err == nil || stdout != "" || stderr != "" {
			t.Fatalf("invalid invocation ran: %q %q %v", stdout, stderr, err)
		}
	}
}

func TestArgumentInjectionIsNotRecursive(t *testing.T) {
	got := injectArguments([]string{"tool", "{A}-{B}"}, map[string]string{"A": "{B}", "B": "second"})
	if got[1] != "{B}-second" {
		t.Fatalf("values were interpreted as templates: %q", got)
	}
}

func TestExecStartErrorsDoNotExposeInjectedArguments(t *testing.T) {
	value := "not-an-executable-private-fixture-76139"
	testConfig(t, Config{Global: Project{Env: map[string]string{"TOKEN": value}}})
	stdout, stderr, err := captureStreams(t, func() error { return execSecrets([]string{"TOKEN", "--", "{TOKEN}"}) })
	if err == nil || stdout != "" || stderr != "" || strings.Contains(err.Error(), value) {
		t.Fatalf("unsafe start error: stdout=%q stderr=%q error=%v", stdout, stderr, err)
	}
}
