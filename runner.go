package main

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strings"
	"syscall"
)

func childEnvironment(parent []string, secrets map[string]string) []string {
	env := make([]string, 0, len(parent)+len(secrets))
	for _, entry := range parent {
		key, _, _ := strings.Cut(entry, "=")
		if _, replace := secrets[key]; !replace {
			env = append(env, entry)
		}
	}
	names := make([]string, 0, len(secrets))
	for name := range secrets {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		env = append(env, name+"="+secrets[name])
	}
	return env
}

func injectArguments(arguments []string, secrets map[string]string) []string {
	replacements := []string{}
	names := make([]string, 0, len(secrets))
	for name := range secrets {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		replacements = append(replacements, "{"+name+"}", secrets[name])
	}
	replace := strings.NewReplacer(replacements...)
	result := make([]string, len(arguments))
	for i, arg := range arguments {
		result[i] = replace.Replace(arg)
	}
	return result
}

func executeSecretCommand(arguments []string, secrets map[string]string) error {
	args := injectArguments(arguments, secrets)
	child := exec.Command(args[0], args[1:]...)
	child.Env = childEnvironment(os.Environ(), secrets)
	child.Stdin = os.Stdin
	values := make([]string, 0, len(secrets))
	for _, value := range secrets {
		values = append(values, value)
	}
	stdout := newSecretFilter(os.Stdout, values)
	stderr := newSecretFilter(os.Stderr, values)
	child.Stdout = stdout
	child.Stderr = stderr
	if err := child.Start(); err != nil {
		return errors.New("could not start child command; check the executable and arguments")
	}
	signals := make(chan os.Signal, 1)
	done := make(chan struct{})
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	go func() {
		for {
			select {
			case sig := <-signals:
				_ = child.Process.Signal(sig)
			case <-done:
				return
			}
		}
	}()
	err := child.Wait()
	close(done)
	outErr := stdout.Flush()
	errErr := stderr.Flush()
	if outErr != nil || errErr != nil {
		return errors.New("could not write filtered child output")
	}
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code := exitErr.ExitCode()
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			code = 128 + int(status.Signal())
		}
		if code < 1 {
			code = 1
		}
		return &childExitError{code: code}
	}
	return errors.New("could not complete child command or filter its output")
}
