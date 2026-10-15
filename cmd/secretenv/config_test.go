package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testConfig(t *testing.T, cfg Config) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	if err := saveConfig(cfg); err != nil {
		t.Fatal(err)
	}
}

func captureOutput(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = original; r.Close() }()
	result := fn()
	w.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(b), result
}

func TestConfigPermissionsAndParserErrors(t *testing.T) {
	testConfig(t, Config{Projects: map[string]Project{"app": {Env: map[string]string{"TOKEN": "keep-me"}}}})
	file, err := os.Stat(configPath())
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.Stat(filepath.Dir(configPath()))
	if err != nil {
		t.Fatal(err)
	}
	if file.Mode().Perm() != 0600 || dir.Mode().Perm() != 0700 {
		t.Fatalf("permissions: file=%v dir=%v", file.Mode(), dir.Mode())
	}
	if err := os.WriteFile(configPath(), []byte("TOKEN = accidentally-exposed-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = loadConfig()
	if err == nil || strings.Contains(err.Error(), "accidentally-exposed-secret") {
		t.Fatalf("unsafe parser error: %v", err)
	}
	_, err = captureOutput(t, exportForCurrentDirectory)
	if err == nil || strings.Contains(err.Error(), "accidentally-exposed-secret") {
		t.Fatalf("unsafe export parser error: %v", err)
	}
}
