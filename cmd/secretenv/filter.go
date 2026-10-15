package main

import (
	"io"
	"sort"
	"strings"
)

type secretFilter struct {
	out     io.Writer
	secrets []string
	pending string
}

func newSecretFilter(out io.Writer, values []string) *secretFilter {
	seen := map[string]bool{}
	f := &secretFilter{out: out}
	for _, value := range values {
		if value != "" && !seen[value] {
			f.secrets = append(f.secrets, value)
			seen[value] = true
		}
	}
	sort.Slice(f.secrets, func(i, j int) bool { return len(f.secrets[i]) > len(f.secrets[j]) })
	return f
}

func (f *secretFilter) Write(p []byte) (int, error) {
	if len(f.secrets) == 0 {
		return f.out.Write(p)
	}
	f.pending += string(p)
	if err := f.drain(false); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (f *secretFilter) Flush() error { return f.drain(true) }

func (f *secretFilter) drain(final bool) error {
	var output strings.Builder
	for f.pending != "" {
		match := ""
		partial := false
		for _, value := range f.secrets {
			if strings.HasPrefix(f.pending, value) {
				if match == "" {
					match = value
				}
				continue
			}
			if !final && len(f.pending) < len(value) && strings.HasPrefix(value, f.pending) {
				partial = true
			}
		}
		if partial {
			break
		}
		if match != "" {
			output.WriteString("[REDACTED]")
			f.pending = f.pending[len(match):]
			continue
		}
		output.WriteByte(f.pending[0])
		f.pending = f.pending[1:]
	}
	if output.Len() == 0 {
		return nil
	}
	n, err := io.WriteString(f.out, output.String())
	if err == nil && n != output.Len() {
		return io.ErrShortWrite
	}
	return err
}
