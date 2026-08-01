package main

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type SitemapProjector struct {
	infoDir   string
	outputDir string
}

func (p SitemapProjector) Name() string {
	return "sitemap"
}

func (p SitemapProjector) Run(graph *BuildGraph) error {
	if strings.TrimSpace(graph.Config.URL) == "" {
		return fmt.Errorf("site URL is required to generate sitemap")
	}

	outPath := filepath.Join(p.outputDir, "sitemap.xml")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("creating sitemap output dir: %w", err)
	}

	data, err := xml.MarshalIndent(newSitemapURLSet(sitemapEntries(graph, p.infoDir)), "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling sitemap xml: %w", err)
	}

	data = append([]byte(xml.Header), data...)
	data = append(data, '\n')

	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("writing sitemap %q: %w", outPath, err)
	}

	return nil
}

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type sitemapPath struct {
	isDir   bool
	lastMod time.Time
}

func sitemapEntries(graph *BuildGraph, infoDir string) []sitemapURL {
	paths := map[string]sitemapPath{}

	for dir := range graph.DirContents {
		paths[dir] = sitemapPath{isDir: true}
	}
	for id, content := range graph.Contents {
		lastMod, _ := fileModTime(infoDir, content.Source)
		paths[id] = sitemapPath{lastMod: lastMod}
	}
	for _, missingPage := range graph.MissingPages {
		source := ""
		if missingPage.Content != nil {
			source = missingPage.Content.Source
		}
		lastMod, _ := fileModTime(infoDir, source)
		paths[missingPage.ID] = sitemapPath{lastMod: lastMod}
	}

	addDirectoryLastMod(paths)

	keys := make([]string, 0, len(paths))
	for path := range paths {
		keys = append(keys, path)
	}
	sort.Strings(keys)

	entries := make([]sitemapURL, 0, len(keys))
	for _, path := range keys {
		entry := sitemapURL{
			Loc: sitemapURLForPath(graph.Config.URL, path, paths[path].isDir),
		}
		if !paths[path].lastMod.IsZero() {
			entry.LastMod = paths[path].lastMod.UTC().Format(time.RFC3339)
		}
		entries = append(entries, entry)
	}

	return entries
}

func addDirectoryLastMod(paths map[string]sitemapPath) {
	for path, entry := range paths {
		if entry.isDir || entry.lastMod.IsZero() {
			continue
		}

		dir := filepath.Dir(path)
		if dir == "." {
			dir = ""
		}

		for {
			dirEntry, ok := paths[dir]
			if ok && dirEntry.isDir && entry.lastMod.After(dirEntry.lastMod) {
				dirEntry.lastMod = entry.lastMod
				paths[dir] = dirEntry
			}
			if dir == "" {
				break
			}
			dir = filepath.Dir(dir)
			if dir == "." {
				dir = ""
			}
		}
	}
}

func fileModTime(infoDir, source string) (time.Time, bool) {
	if source == "" {
		return time.Time{}, false
	}

	info, err := os.Stat(filepath.Join(infoDir, source))
	if err != nil {
		return time.Time{}, false
	}

	return info.ModTime(), true
}

func sitemapURLForPath(baseURL, path string, isDir bool) string {
	baseURL = strings.TrimRight(baseURL, "/")

	if path == "" {
		return baseURL + "/"
	}

	u := url.URL{Path: "/" + filepath.ToSlash(path)}
	loc := baseURL + u.EscapedPath()
	if isDir {
		loc += "/"
	}

	return loc
}

func newSitemapURLSet(urls []sitemapURL) sitemapURLSet {
	return sitemapURLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}
}
