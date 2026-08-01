package main

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/alsosee/finder/structs"
)

func TestSitemapProjectorWritesCanonicalURLsAndLastModifiedDates(t *testing.T) {
	infoDir := t.TempDir()
	outputDir := t.TempDir()

	duneModified := time.Date(2026, 7, 20, 14, 30, 0, 0, time.UTC)
	peopleModified := time.Date(2026, 7, 21, 9, 15, 0, 0, time.FixedZone("offset", -4*60*60))

	mustWriteFile(t, filepath.Join(infoDir, "Movies", "2024", "Dune Part Two.yml"), "name: Dune Part Two\n")
	mustWriteFile(t, filepath.Join(infoDir, "People", "Alice & Bob.md"), "# Alice & Bob\n")
	mustSetModTime(t, filepath.Join(infoDir, "Movies", "2024", "Dune Part Two.yml"), duneModified)
	mustSetModTime(t, filepath.Join(infoDir, "People", "Alice & Bob.md"), peopleModified)

	graph := &BuildGraph{
		Config: structs.Config{URL: "https://alsosee.example"},
		Contents: structs.Contents{
			"Movies/2024/Dune Part Two": {
				Source: "Movies/2024/Dune Part Two.yml",
			},
			"People/Alice & Bob": {
				Source: "People/Alice & Bob.md",
			},
		},
		DirContents: map[string][]structs.File{
			"":            {},
			"Movies":      {},
			"Movies/2024": {},
		},
		MissingPages: []MissingPage{
			{ID: "Shows/Missing Show"},
		},
	}

	if err := (SitemapProjector{infoDir: infoDir, outputDir: outputDir}).Run(graph); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outputDir, "sitemap.xml"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.HasPrefix(string(data), xml.Header) {
		t.Fatalf("sitemap does not start with XML header: %q", string(data))
	}
	if !strings.Contains(string(data), `xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"`) {
		t.Fatalf("sitemap does not include sitemap namespace: %q", string(data))
	}

	var got sitemapURLSet
	if err := xml.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	entries := append([]sitemapURL{}, got.URLs...)

	want := []sitemapURL{
		{
			Loc:     "https://alsosee.example/",
			LastMod: "2026-07-21T13:15:00Z",
		},
		{
			Loc:     "https://alsosee.example/Movies/",
			LastMod: "2026-07-20T14:30:00Z",
		},
		{
			Loc:     "https://alsosee.example/Movies/2024/",
			LastMod: "2026-07-20T14:30:00Z",
		},
		{
			Loc:     "https://alsosee.example/Movies/2024/Dune%20Part%20Two",
			LastMod: "2026-07-20T14:30:00Z",
		},
		{
			Loc:     "https://alsosee.example/People/Alice%20&%20Bob",
			LastMod: "2026-07-21T13:15:00Z",
		},
		{
			Loc: "https://alsosee.example/Shows/Missing%20Show",
		},
	}
	if !reflect.DeepEqual(entries, want) {
		t.Fatalf("sitemap entries = %#v, want %#v", entries, want)
	}
}

func TestSitemapProjectorRequiresSiteURL(t *testing.T) {
	err := (SitemapProjector{outputDir: t.TempDir()}).Run(&BuildGraph{
		Config:      structs.Config{},
		DirContents: map[string][]structs.File{"": {}},
	})
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "site URL is required") {
		t.Fatalf("Run() error = %v, want site URL error", err)
	}
}

func TestSitemapURLForPathTrimsBaseAndAddsDirectorySlash(t *testing.T) {
	got := sitemapURLForPath("https://alsosee.example/", "Movies/Series: One", true)
	want := "https://alsosee.example/Movies/Series:%20One/"
	if got != want {
		t.Fatalf("sitemapURLForPath() = %q, want %q", got, want)
	}
}

func mustSetModTime(t *testing.T, path string, modTime time.Time) {
	t.Helper()
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}
}
