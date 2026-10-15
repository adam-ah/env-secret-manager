package main

import (
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
