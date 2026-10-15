package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestSecretFilterAcrossEveryBoundary(t *testing.T) {
	for _, test := range []struct {
		input, want string
		values      []string
	}{
		{"before alpha-token and beta-token after", "before [REDACTED] and [REDACTED] after", []string{"alpha-token", "beta-token"}},
		{"abc ab a xyz", "[REDACTED] [REDACTED] [REDACTED] xyz", []string{"a", "ab", "abc"}},
		{"secretsecret partial-secr", "[REDACTED][REDACTED] partial-secr", []string{"secret", ""}},
		{"unicode 🔑token end", "unicode [REDACTED] end", []string{"🔑token"}},
		{"key=line1\nline2 done", "key=[REDACTED] done", []string{"line1\nline2"}},
		{"unchanged output", "unchanged output", []string{""}},
	} {
		for split := 0; split <= len(test.input); split++ {
			var output bytes.Buffer
			filter := newSecretFilter(&output, test.values)
			if _, err := filter.Write([]byte(test.input[:split])); err != nil {
				t.Fatal(err)
			}
			if _, err := filter.Write([]byte(test.input[split:])); err != nil {
				t.Fatal(err)
			}
			if err := filter.Flush(); err != nil {
				t.Fatal(err)
			}
			if output.String() != test.want {
				t.Fatalf("split %d: %q, want %q", split, output.String(), test.want)
			}
		}
	}
	var output bytes.Buffer
	filter := newSecretFilter(&output, []string{"private-value"})
	for _, b := range []byte("private-value") {
		filter.Write([]byte{b})
	}
	filter.Flush()
	if output.String() != "[REDACTED]" {
		t.Fatal("bytewise writes exposed a secret")
	}
}

func TestSecretFilterDoesNotDelayOrdinaryOutput(t *testing.T) {
	var output bytes.Buffer
	filter := newSecretFilter(&output, []string{"private-value"})
	filter.Write([]byte("Ready> "))
	if output.String() != "Ready> " {
		t.Fatal("ordinary prompt was buffered")
	}
	filter.Write([]byte("priva"))
	if strings.Contains(output.String(), "priva") {
		t.Fatal("partial secret was released early")
	}
	filter.Write([]byte("te-value"))
	filter.Flush()
	if output.String() != "Ready> [REDACTED]" {
		t.Fatal(output.String())
	}
}

type shortOutput struct{}

func (shortOutput) Write(p []byte) (int, error) { return 0, nil }
func TestSecretFilterWriteErrors(t *testing.T) {
	filter := newSecretFilter(shortOutput{}, []string{"secret"})
	if _, err := filter.Write([]byte("secret")); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("write error lost: %v", err)
	}
}
