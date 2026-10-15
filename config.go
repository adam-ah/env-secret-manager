package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

func configPath() string { return filepath.Join(homeDir(), ".secretenv", "config.toml") }

func loadConfig() (Config, error) {
	cfg := Config{Projects: map[string]Project{}, Global: Project{Env: map[string]string{}}}
	if _, err := toml.DecodeFile(configPath(), &cfg); err != nil && !os.IsNotExist(err) {

		return cfg, fmt.Errorf("cannot read config %s; check permissions and TOML syntax locally", configPath())
	}
	return cfg, nil
}

const globalSelector = "@global"

func workspaceProjects(cfg Config) []string {
	return append([]string{globalSelector}, projectNames(cfg)...)
}

func scopedProject(cfg Config, selector string) Project {
	if selector == globalSelector {
		return cfg.Global
	}
	return cfg.Projects[selector]
}

func setScopedProject(cfg *Config, selector string, project Project) {
	if selector == globalSelector {
		cfg.Global = project
	} else {
		cfg.Projects[selector] = project
	}
}

func validProjectName(s string) bool {
	if s == "" || s == "." || s == ".." || s[0] == '-' || strings.EqualFold(s, "global") {
		return false
	}
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._-/:", r) {
			continue
		}
		return false
	}
	return true
}

func shellArgument(s string) string {
	var out strings.Builder
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._-/:", r)) {
			out.WriteByte('\\')
		}
		out.WriteRune(r)
	}
	return out.String()
}

func saveConfig(cfg Config) error {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return fmt.Errorf("cannot encode config")
	}
	dir := filepath.Dir(configPath())
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".config-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(buf.Bytes()); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), configPath())
}

func projectNames(cfg Config) []string {
	names := make([]string, 0, len(cfg.Projects))
	for name := range cfg.Projects {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func secretNames(p Project) []string {
	names := make([]string, 0, len(p.Env))
	for name := range p.Env {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func selectProject(cfg Config, selector string) (*Match, error) {
	if selector != "" {
		if strings.EqualFold(selector, "global") {
			return &Match{Selector: globalSelector, Project: cfg.Global}, nil
		}
		p, ok := cfg.Projects[selector]
		if !ok {
			return nil, fmt.Errorf("project not found; run secretenv list for names and references")
		}
		return &Match{Selector: selector, Project: p}, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	match := matchingProject(cfg, cleanPath(cwd))
	if match == nil {
		return nil, fmt.Errorf("no matching project; use --project with an exact selector")
	}
	return match, nil
}
