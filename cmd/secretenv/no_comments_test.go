package main

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoCommentsInGoCode(t *testing.T) {
	positions := token.NewFileSet()
	err := filepath.WalkDir(repositoryPath(), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(positions, path, nil, parser.ParseComments)
		if err != nil {
			t.Errorf("parse %s: %v", path, err)
			return nil
		}
		for _, group := range file.Comments {
			for _, comment := range group.List {
				t.Errorf("%s: comments are not allowed in Go source", positions.Position(comment.Pos()))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
