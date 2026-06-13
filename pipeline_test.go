package main

import (
	"os"
	"path/filepath"
	"testing"

	gitignore "github.com/sabhiram/go-gitignore"
)

func TestScannerSkipsMediaOnlyFilesWhenInfoAndMediaAreSameDirectory(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "Movie.yml"), "name: Movie\n")
	mustWriteFile(t, filepath.Join(dir, "Movie.jpg"), "not really an image")
	mustWriteFile(t, filepath.Join(dir, ".thumbs.yml"), "[]\n")

	scan, err := NewScanner(dir, dir, &gitignore.GitIgnore{}).Scan()
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(scan.InfoFiles) != 1 {
		t.Fatalf("got files %#v, expected one YAML file", scan.InfoFiles)
	}
	if scan.InfoFiles[0].Path != "Movie.yml" {
		t.Fatalf("got file %q, expected Movie.yml", scan.InfoFiles[0].Path)
	}
}

func TestSchemaMetadataValidateYAMLReportsUnknownNestedField(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "_finder", "schema.yml"), `
content:
  type: object
  properties:
    name:
      type: string
    characters:
      type: array
      items:
        type: character
character:
  type: object
  properties:
    name:
      type: string
`)

	meta, err := LoadSchemaMetadata(dir)
	if err != nil {
		t.Fatalf("LoadSchemaMetadata() error = %v", err)
	}

	_, diagnostics, err := NewParser(meta).ParseContentYAML("Movie.yml", []byte("name: Movie\ncharacters:\n  - name: Alice\n    typo: yes\n"))
	if err != nil {
		t.Fatalf("ParseContentYAML() error = %v", err)
	}

	if len(diagnostics) != 1 {
		t.Fatalf("got diagnostics %#v, expected one", diagnostics)
	}
	if diagnostics[0].Path != "characters[0].typo" {
		t.Fatalf("got path %q, expected characters[0].typo", diagnostics[0].Path)
	}
	if diagnostics[0].Type != "character" {
		t.Fatalf("got type %q, expected character", diagnostics[0].Type)
	}
	if diagnostics[0].Field != "typo" {
		t.Fatalf("got field %q, expected typo", diagnostics[0].Field)
	}
	if diagnostics[0].Line != 4 {
		t.Fatalf("got line %d, expected 4", diagnostics[0].Line)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
