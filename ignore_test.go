package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProcessIgnoreFileAcceptsWorkspaceRelativePath(t *testing.T) {
	workspace := t.TempDir()
	infoDir := filepath.Join(workspace, "info")
	if err := os.Mkdir(infoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ignorePath := filepath.Join(infoDir, ".ignore")
	if err := os.WriteFile(ignorePath, []byte(".DS_Store\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(workspace); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previousDir); err != nil {
			t.Errorf("restoring working directory: %v", err)
		}
	})

	ignore, err := processIgnoreFile(infoDir, filepath.Join("info", ".ignore"))
	if err != nil {
		t.Fatal(err)
	}
	if !ignore.MatchesPath(".DS_Store") {
		t.Error("workspace-relative ignore file was not loaded")
	}
}

func TestProcessIgnoreFileFallsBackToInfoDirectory(t *testing.T) {
	infoDir := t.TempDir()
	ignorePath := filepath.Join(infoDir, ".ignore")
	if err := os.WriteFile(ignorePath, []byte(".DS_Store\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	ignore, err := processIgnoreFile(infoDir, ".ignore")
	if err != nil {
		t.Fatal(err)
	}
	if !ignore.MatchesPath(".DS_Store") {
		t.Error("info-directory-relative ignore file was not loaded")
	}
}
