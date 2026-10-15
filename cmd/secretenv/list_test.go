package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestListNamesWithoutValues(t *testing.T) {
	value := "private-token-fixture"
	testConfig(t, Config{Projects: map[string]Project{"app": {Env: map[string]string{"TOKEN": value}}}})
	output, err := captureOutput(t, func() error { return listSecrets(nil) })
	if err != nil || !strings.Contains(output, `"app:TOKEN"`) || strings.Contains(output, value) {
		t.Fatalf("list: %s %v", output, err)
	}
}

func TestListHereMatchesFolderNamesAndPaths(t *testing.T) {
	root := t.TempDir()
	exactPath := filepath.Join(root, "exact", "my-app")
	testConfig(t, Config{
		Global: Project{Env: map[string]string{"GLOBAL_TOKEN": "private-fixture"}},
		Projects: map[string]Project{
			"my-app":        {Env: map[string]string{"FOLDER_TOKEN": "private-fixture"}},
			exactPath:       {Env: map[string]string{"PATH_TOKEN": "private-fixture"}},
			"display-label": {Env: map[string]string{"OTHER_TOKEN": "private-fixture"}},
		},
	})
	previousDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previousDirectory); err != nil {
			t.Error(err)
		}
	})
	for _, test := range []struct {
		folder    string
		reference string
	}{
		{filepath.Join(root, "work", "my-app"), "my-app:FOLDER_TOKEN"},
		{filepath.Join(root, "work", "my-app", "src"), "my-app:FOLDER_TOKEN"},
		{exactPath, exactPath + ":PATH_TOKEN"},
		{filepath.Join(exactPath, "src"), exactPath + ":PATH_TOKEN"},
		{filepath.Join(root, "my-app-other"), ""},
		{filepath.Join(root, "unrelated"), ""},
	} {
		t.Run(test.folder, func(t *testing.T) {
			if err := os.MkdirAll(test.folder, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Chdir(test.folder); err != nil {
				t.Fatal(err)
			}
			output, err := captureOutput(t, func() error { return listSecrets([]string{"--here"}) })
			if err != nil {
				t.Fatal(err)
			}
			var entries []struct {
				Scope      string   `json:"scope"`
				References []string `json:"references"`
			}
			if err := json.Unmarshal([]byte(output), &entries); err != nil {
				t.Fatal(err)
			}
			want := []string{"GLOBAL_TOKEN"}
			if test.reference != "" {
				want = append(want, test.reference)
			}
			var got []string
			for _, entry := range entries {
				got = append(got, entry.References...)
			}
			if len(entries) == 0 || entries[0].Scope != "global" || !reflect.DeepEqual(got, want) || strings.Contains(output, "private-fixture") {
				t.Fatalf("folder discovery: got %s; want references %q", output, want)
			}
		})
	}
	output, err := captureOutput(t, func() error { return listSecrets(nil) })
	if err != nil || !strings.Contains(output, "display-label:OTHER_TOKEN") {
		t.Fatal("bare list must still include secrets unrelated to the working directory")
	}
}

func TestListRejectsReplacedFlag(t *testing.T) {
	output, err := captureOutput(t, func() error { return listSecrets([]string{"--current"}) })
	if err == nil || output != "" || !strings.Contains(err.Error(), "--here") {
		t.Fatalf("expected replacement usage: output=%q error=%v", output, err)
	}
}
