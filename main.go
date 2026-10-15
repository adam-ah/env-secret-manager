package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

var helpText = `secretenv — store secrets, inject them into commands, and redact their output.

  secretenv [menu|edit]                 Open editor; show help without a terminal
  secretenv list [--current|--project NAME]  List names and exact refs as JSON
  secretenv exec REF... -- PROGRAM [ARGS]   Inject secrets and run a program
  secretenv install [bash|zsh]           Show folder-based shell hook setup
  secretenv help|-h|--help               Show this help

REF: KEY (Global) or PROJECT:KEY. Find refs: secretenv list --current
Multiple secrets: exec REF REF -- PROGRAM. Agents: use list and exec only.

` + envExampleTitle + " · " + sampleExamples.envDescription() + "\n" +
	sampleExamples.variableCommand() + "\n" +
	envCurlExampleTitle + "\n" + sampleExamples.envCurlCommand() + "\n" +
	curlExampleTitle + "\n" + sampleExamples.curlCommand() + "\n"

type Config struct {
	Projects map[string]Project `toml:"projects"`
	Global   Project            `toml:"global"`
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
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "secretenv: %v\n", err)
		if childErr, ok := err.(*childExitError); ok {
			os.Exit(childErr.code)
		}
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return openMenu()
	}
	if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
		fmt.Print(helpText)
		return nil
	}
	switch args[0] {
	case "help", "-h", "--help":
		fmt.Print(helpText)
		return nil
	case "menu":
		return openMenu()
	case "edit":
		return editSecrets()
	case "install":
		return showInstall(args[1:])
	case "list":
		return listSecrets(args[1:])
	case "exec":
		return execSecrets(args[1:])
	case "_hook":
		if len(args) != 1 {
			return fmt.Errorf("internal shell hook takes no arguments")
		}
		return exportForCurrentDirectory()
	default:
		return fmt.Errorf("unknown command %q; run secretenv help", args[0])
	}
}

func exportForCurrentDirectory() error {
	cfgPath := configPath()

	var cfg Config
	if _, err := toml.DecodeFile(cfgPath, &cfg); err != nil {
		if os.IsNotExist(err) {
			clearActive()
			return nil
		}
		return fmt.Errorf("cannot parse config %s; check TOML syntax locally", cfgPath)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}
	cwd = cleanPath(cwd)

	match := matchingProject(cfg, cwd)

	allVars := allProjectVars(cfg)
	activeVars := strings.Fields(os.Getenv("SECRETENV_ACTIVE_VARS"))

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
