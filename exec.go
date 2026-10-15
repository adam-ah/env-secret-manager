package main

import (
	"errors"
	"fmt"
	"strings"
)

func resolveSecret(cfg Config, reference string) (string, string, error) {
	project, name := "", reference
	if separator := strings.LastIndex(reference, ":"); separator >= 0 {
		project, name = reference[:separator], reference[separator+1:]
		if project == "" {
			return "", "", errors.New("project name cannot be empty in PROJECT:KEY")
		}
	}
	if !validShellName(name) {
		return "", "", errors.New("invalid secret name")
	}
	env := cfg.Global.Env
	if project != "" {
		match, err := selectProject(cfg, project)
		if err != nil {
			return "", "", err
		}
		env = match.Project.Env
	}
	value, ok := env[name]
	if !ok {
		return "", "", errors.New("secret not found; use secretenv list to discover exact references")
	}
	if strings.ContainsRune(value, 0) {
		return "", "", errors.New("secret contains a NUL byte and cannot be used as an environment variable")
	}
	return name, value, nil
}

func execSecrets(args []string) error {
	separator := -1
	for i, arg := range args {
		if arg == "--" {
			separator = i
			break
		}
	}
	if separator < 1 || separator == len(args)-1 {
		return errors.New("usage: secretenv exec KEY [PROJECT:KEY ...] -- COMMAND [ARG ...]; secret values are never returned directly")
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	env := map[string]string{}
	for _, reference := range args[:separator] {
		name, value, err := resolveSecret(cfg, reference)
		if err != nil {
			return err
		}
		if _, exists := env[name]; exists {
			return errors.New("requested secrets have duplicate environment variable names; choose one reference per key")
		}
		env[name] = value
	}
	return executeSecretCommand(args[separator+1:], env)
}

type childExitError struct{ code int }

func (e *childExitError) Error() string {
	return fmt.Sprintf("child command exited with status %d", e.code)
}
