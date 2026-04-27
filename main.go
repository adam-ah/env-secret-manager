package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

const helpText = `
secretenv

Loads project-specific environment variables from a single global TOML file:

  ~/.secretenv/config.toml

It is designed for shell hooks. When you cd into a matching project folder,
it prints shell commands that export that project's env vars. When you leave,
it prints shell commands that unset all managed env vars.

It does NOT read:
  .env
  .envrc
  mise.toml
  project-local files
  secret-tool
  macOS Keychain
  AWS

It only reads:
  ~/.secretenv/config.toml

Commands:

  secretenv export
      Print shell code for the current directory.

  secretenv help
      Print this help.

Example ~/.secretenv/config.toml:

  [projects."screeners".env]
  APP_ENV = "local"
  SMTP_HOST = "localhost"
  SMTP_PORT = "1025"
  SMTP_USERNAME = "dev_user"
  SMTP_PASSWORD = "dev_password"
  CLOUDFLARE_TOKEN = "cf_dev_token"

  [projects."~/code/project-a".env]
  APP_ENV = "local"
  API_USERNAME = "project_a_user"
  API_PASSWORD = "project_a_password"

  [projects."/mnt/e/temp/project-b".env]
  APP_ENV = "local"
  DATABASE_URL = "postgres://user:pass@localhost:5432/project_b?sslmode=disable"

Matching rules:

  [projects."screeners"]
      Matches any folder named "screeners", case-insensitive:

        /tmp/screeners
        /tmp/SCREENERS
        /mnt/e/temp/screeners
        /my/lawd/screeners/subdir

  [projects."~/code/project-a"]
      Matches that absolute path and its children.

  [projects."/mnt/e/temp/project-b"]
      Matches that absolute path and its children.

Matching is case-insensitive everywhere, including Linux.

Zsh hook:

  Add this to ~/.zshrc:

  _secretenv_hook() {
  local out
  local exit_code

  command -v secretenv >/dev/null 2>&1 || return

  out="$(secretenv export 2>&1)"
  exit_code=$?

  if [[ $exit_code -ne 0 ]]; then
    echo "[secretenv] error:" >&2
    echo "$out" >&2
    return
  fi

  eval "$out"
  }

  autoload -Uz add-zsh-hook
  add-zsh-hook chpwd _secretenv_hook

  _secretenv_hook


Bash hook:

  Add this to ~/.bashrc:

    _secretenv_hook() {
      command -v secretenv >/dev/null 2>&1 || return
      eval "$(secretenv export)"
    }

    PROMPT_COMMAND="_secretenv_hook${PROMPT_COMMAND:+;$PROMPT_COMMAND}"

Install:

  mkdir -p ~/.secretenv
  nano ~/.secretenv/config.toml

  go install github.com/adam-ah/env-secret-manager@latest

  # or build from a clone:
  go build -o secretenv .
  mkdir -p ~/.local/bin
  mv secretenv ~/.local/bin/

Make sure ~/.local/bin is in PATH.

Security note:

  This avoids repo leaks because secrets are not inside project folders.
  But ~/.secretenv/config.toml is still plaintext. Protect it:

    chmod 700 ~/.secretenv
    chmod 600 ~/.secretenv/config.toml
`

type Config struct {
	Projects map[string]Project `toml:"projects"`
}

type Project struct {
	Env map[string]string `toml:"env"`
}

type Match struct {
	Selector    string
	ProjectPath string
	Project     Project
	Score       int
}

func main() {
	if len(os.Args) < 2 {
		fmt.Print(helpText)
		return
	}

	switch os.Args[1] {
	case "help", "-h", "--help":
		fmt.Print(helpText)
		return

	case "export":
		if err := exportForCurrentDirectory(); err != nil {
			fmt.Fprintf(os.Stderr, "secretenv: %v\n", err)
			os.Exit(1)
		}
		return

	default:
		fmt.Fprintf(os.Stderr, "secretenv: unknown command %q\n\n", os.Args[1])
		fmt.Print(helpText)
		os.Exit(2)
	}
}

func exportForCurrentDirectory() error {
	cfgPath := filepath.Join(homeDir(), ".secretenv", "config.toml")

	var cfg Config
	if _, err := toml.DecodeFile(cfgPath, &cfg); err != nil {
		if os.IsNotExist(err) {
			clearActive()
			return nil
		}
		return fmt.Errorf("config error: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}
	cwd = cleanPath(cwd)

	match := matchingProject(cfg, cwd)

	allVars := allProjectVars(cfg)
	activeVars := strings.Fields(os.Getenv("SECRETENV_ACTIVE_VARS"))

	// Remove all currently or previously managed vars.
	for _, name := range union(allVars, activeVars) {
		if !validShellName(name) {
			fmt.Fprintf(os.Stderr, "secretenv: ignoring invalid env var name %q\n", name)
			continue
		}
		fmt.Printf("unset %s;\n", name)
	}

	if match == nil {
		clearActive()
		return nil
	}

	var exported []string

	names := make([]string, 0, len(match.Project.Env))
	for name := range match.Project.Env {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if !validShellName(name) {
			fmt.Fprintf(os.Stderr, "secretenv: ignoring invalid env var name %q\n", name)
			continue
		}

		value := match.Project.Env[name]
		fmt.Printf("export %s=%s;\n", name, shellQuote(value))
		exported = append(exported, name)
	}

	sort.Strings(exported)

	fmt.Printf("export SECRETENV_ACTIVE_VARS=%s;\n", shellQuote(strings.Join(exported, " ")))
	fmt.Printf("export SECRETENV_ACTIVE_PROJECT=%s;\n", shellQuote(match.Selector))

	return nil
}

func matchingProject(cfg Config, cwd string) *Match {
	var best *Match
	cwdCmp := pathCmp(cwd)

	for selector, project := range cfg.Projects {
		m := matchSelector(selector, project, cwdCmp)
		if m == nil {
			continue
		}

		if best == nil || m.Score > best.Score {
			best = m
		}
	}

	return best
}

func matchSelector(selector string, project Project, cwdCmp string) *Match {
	expanded := expandHome(selector)

	// Absolute path selector:
	//
	//   [projects."/mnt/e/temp/screeners"]
	//   [projects."~/code/screeners"]
	//
	// Matches that path and children.
	if filepath.IsAbs(expanded) {
		projectPath := cleanPath(expanded)
		projectCmp := pathCmp(projectPath)

		if cwdCmp == projectCmp || strings.HasPrefix(cwdCmp, projectCmp+pathSep()) {
			return &Match{
				Selector:    selector,
				ProjectPath: projectPath,
				Project:     project,
				Score:       10_000 + len(projectCmp),
			}
		}

		return nil
	}

	// Shorthand selector:
	//
	//   [projects."screeners"]
	//
	// Matches any folder segment named "screeners", case-insensitive:
	//
	//   /tmp/screeners
	//   /tmp/SCREENERS
	//   /mnt/e/temp/screeners/subdir
	short := pathCmp(filepath.Clean(selector))
	short = strings.Trim(short, string(os.PathSeparator))

	if short == "." || short == "" {
		return nil
	}

	cwdParts := splitPath(cwdCmp)
	shortParts := splitPath(short)

	if len(shortParts) == 0 || len(cwdParts) < len(shortParts) {
		return nil
	}

	for i := 0; i <= len(cwdParts)-len(shortParts); i++ {
		if equalSlice(cwdParts[i:i+len(shortParts)], shortParts) {
			// Prefer deeper / more specific matches.
			score := 1_000 + i*10 + len(short)
			return &Match{
				Selector:    selector,
				ProjectPath: strings.Join(cwdParts[:i+len(shortParts)], string(os.PathSeparator)),
				Project:     project,
				Score:       score,
			}
		}
	}

	return nil
}

func allProjectVars(cfg Config) []string {
	set := map[string]bool{}

	for _, project := range cfg.Projects {
		for name := range project.Env {
			set[name] = true
		}
	}

	var out []string
	for name := range set {
		out = append(out, name)
	}

	sort.Strings(out)
	return out
}

func union(a, b []string) []string {
	set := map[string]bool{}

	for _, x := range a {
		if x != "" {
			set[x] = true
		}
	}

	for _, x := range b {
		if x != "" {
			set[x] = true
		}
	}

	var out []string
	for x := range set {
		out = append(out, x)
	}

	sort.Strings(out)
	return out
}

func clearActive() {
	fmt.Println("export SECRETENV_ACTIVE_VARS='';")
	fmt.Println("export SECRETENV_ACTIVE_PROJECT='';")
}

func validShellName(s string) bool {
	if s == "" {
		return false
	}

	for i, r := range s {
		if r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
			continue
		}

		if i > 0 && r >= '0' && r <= '9' {
			continue
		}

		return false
	}

	return true
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func cleanPath(p string) string {
	abs, err := filepath.Abs(p)
	if err == nil {
		p = abs
	}

	p = filepath.Clean(p)

	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		p = resolved
	}

	return p
}

func expandHome(p string) string {
	if p == "~" {
		return homeDir()
	}

	if strings.HasPrefix(p, "~/") {
		return filepath.Join(homeDir(), p[2:])
	}

	return p
}

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return h
}

func pathCmp(p string) string {
	p = filepath.Clean(p)
	return strings.ToLower(p)
}

func pathSep() string {
	return string(os.PathSeparator)
}

func splitPath(p string) []string {
	p = strings.Trim(p, string(os.PathSeparator))
	if p == "" {
		return nil
	}
	return strings.Split(p, string(os.PathSeparator))
}

func equalSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
