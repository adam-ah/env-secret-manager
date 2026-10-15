package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
)

func listSecrets(args []string) error {
	flags := flag.NewFlagSet("list", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	project := flags.String("project", "", "exact configured name")
	here := flags.Bool("here", false, "Global plus secrets matching this folder's name/path")
	if err := flags.Parse(args); err != nil {
		return errors.New("usage: secretenv list [--here | --project NAME]")
	}
	if flags.NArg() != 0 {
		return errors.New("usage: secretenv list [--here | --project NAME]")
	}
	if *here && *project != "" {
		return errors.New("use either --here or --project, not both")
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	type entry struct {
		Project    string   `json:"project"`
		Scope      string   `json:"scope"`
		Secrets    []string `json:"secrets"`
		References []string `json:"references"`
	}
	makeEntry := func(selector string, project Project) entry {
		names := secretNames(project)
		references := make([]string, 0, len(names))
		for _, name := range names {
			if selector == globalSelector {
				references = append(references, name)
			} else {
				references = append(references, selector+":"+name)
			}
		}
		scope := "project"
		if selector == globalSelector {
			selector = "Global"
			scope = "global"
		}
		return entry{Project: selector, Scope: scope, Secrets: names, References: references}
	}
	entries := []entry{}
	if *project == "" {
		entries = append(entries, makeEntry(globalSelector, cfg.Global))
	}
	if *project != "" {
		match, err := selectProject(cfg, *project)
		if err != nil {
			return err
		}
		entries = append(entries, makeEntry(match.Selector, match.Project))
	} else if *here {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		if match := matchingProject(cfg, cleanPath(cwd)); match != nil {
			entries = append(entries, makeEntry(match.Selector, match.Project))
		}
	} else {
		for _, name := range projectNames(cfg) {
			entries = append(entries, makeEntry(name, cfg.Projects[name]))
		}
	}
	return json.NewEncoder(os.Stdout).Encode(entries)
}
