package main

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gomarkdown/markdown"

	"github.com/alsosee/finder/structs"
)

type GraphBuilder struct {
	config           structs.Config
	scan             *ScanResult
	parser           *Parser
	infoDir          string
	openGraphEnabled bool

	contents             structs.Contents
	contentsByLower      map[string]string // lowercase path → canonical path
	dirContents          map[string][]structs.File
	connections          structs.Connections
	media                MediaCatalog
	chainPages           map[string]map[bool]string
	awardsMissingContent map[string][]structs.Award
	hashes               map[string]string
	awardPages           []string
	missingContent       map[string]*structs.Content
	diagnostics          []Diagnostic
	passthroughFiles     []string
	missingPages         []MissingPage
}

func NewGraphBuilder(config structs.Config, scan *ScanResult, parser *Parser, infoDir string, openGraphEnabled bool) *GraphBuilder {
	return &GraphBuilder{
		config:               config,
		scan:                 scan,
		parser:               parser,
		infoDir:              infoDir,
		openGraphEnabled:     openGraphEnabled,
		contents:             structs.Contents{},
		contentsByLower:      map[string]string{},
		dirContents:          map[string][]structs.File{},
		connections:          structs.Connections{},
		media:                MediaCatalog{},
		chainPages:           map[string]map[bool]string{},
		awardsMissingContent: map[string][]structs.Award{},
		hashes:               map[string]string{},
		missingContent:       map[string]*structs.Content{},
	}
}

func (b *GraphBuilder) Build() (*BuildGraph, error) {
	for _, dir := range b.scan.InfoDirs {
		b.addDir(dir)
	}
	for dir, media := range b.scan.Media {
		b.media[dir] = media
	}
	for _, file := range b.scan.InfoFiles {
		if err := b.processFile(file.Path); err != nil {
			return nil, fmt.Errorf("processing file %q: %w", file.Path, err)
		}
	}
	ReportDiagnostics(b.diagnostics)

	b.addAwards()

	missing := b.missing()
	b.addMissingFilesToPanels(missing)
	b.addMissingContent(missing)
	b.processPanels()

	return &BuildGraph{
		Config:               b.config,
		Contents:             b.contents,
		DirContents:          b.dirContents,
		Connections:          b.connections,
		Media:                b.media,
		Hashes:               b.hashes,
		MissingContent:       b.missingContent,
		Missing:              missing,
		AwardsMissingContent: b.awardsMissingContent,
		ChainPages:           b.chainPages,
		Diagnostics:          b.diagnostics,
		PassthroughFiles:     b.passthroughFiles,
		MissingPages:         b.missingPages,
		OpenGraphEnabled:     b.openGraphEnabled,
	}, nil
}

func (b *GraphBuilder) processFile(file string) error {
	if filepath.Base(file) == ".thumbs.yml" {
		return nil
	}

	switch filepath.Ext(file) {
	case ".yml", ".yaml":
		b.addFile(file)
		return b.processYAMLFile(file)
	case ".gomd":
		b.addFile(file)
		return b.processGoMarkdownFile(file)
	case ".md":
		b.addFile(file)
		return b.processMarkdownFile(file)
	case ".jpeg", ".jpg", ".png", ".mp4":
		b.addFile(file)
		return nil
	default:
		if file == "_redirects" {
			b.passthroughFiles = append(b.passthroughFiles, file)
			return nil
		}
		return fmt.Errorf("unknown file type: %q", file)
	}
}

func (b *GraphBuilder) processYAMLFile(file string) error {
	contentBytes, err := os.ReadFile(filepath.Join(b.infoDir, file))
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	b.addHash(file, contentBytes)

	content, diagnostics, err := b.parser.ParseContentYAML(file, contentBytes)
	if err != nil {
		return err
	}
	b.diagnostics = append(b.diagnostics, diagnostics...)

	content.Source = file
	content.GenerateID()
	content.AddMedia(b.media.ImageForPath)

	b.addContent(content)
	b.addConnections(content)

	return nil
}

func (b *GraphBuilder) processMarkdownFile(file string) error {
	contentBytes, err := os.ReadFile(filepath.Join(b.infoDir, file))
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	htmlBody := markdown.ToHTML(contentBytes, nil, nil)
	htmlBody = bytes.ReplaceAll(htmlBody, []byte("[ ] "), []byte(`<br><input type="checkbox" disabled> `))
	htmlBody = bytes.ReplaceAll(htmlBody, []byte("[x] "), []byte(`<br><input type="checkbox" disabled checked> `))
	htmlBody = bytes.ReplaceAll(htmlBody, []byte("<p><br>"), []byte("<p>"))

	b.addContent(structs.Content{
		Source: file,
		HTML:   string(htmlBody),
	})
	return nil
}

func (b *GraphBuilder) processGoMarkdownFile(file string) error {
	contentBytes, err := os.ReadFile(filepath.Join(b.infoDir, file))
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	b.addContent(structs.Content{
		Source: file,
		HTML:   string(contentBytes),
	})
	return nil
}

func (b *GraphBuilder) addContent(content structs.Content) {
	content.GenerateID()
	b.contents[content.SourceNoExtention] = content
	b.contentsByLower[strings.ToLower(content.SourceNoExtention)] = content.SourceNoExtention
}

func (b *GraphBuilder) addConnections(content structs.Content) {
	content.GenerateID()
	from := content.SourceNoExtention

	for _, conn := range content.Connections() {
		switch conn.Meta {
		case structs.ConnectionPrevious:
			b.addPrevious(from, conn.To)
		case structs.ConnectionSeries:
			b.addConnection(from, series(content), conn)
		case structs.ConnectionNone:
			b.addConnection(from, conn.To, conn)
		default:
			b.addConnection(from, conn.To, conn)
		}
	}

	if len(content.Categories) > 0 {
		b.awardPages = append(b.awardPages, from)
	}
}

func (b *GraphBuilder) addConnection(from, to string, connection structs.Connection) {
	if _, ok := b.connections[to]; !ok {
		b.connections[to] = map[string][]structs.Connection{}
	}
	b.connections[to][from] = append(b.connections[to][from], connection)
}

func (b *GraphBuilder) addPrevious(from, to string) {
	if _, ok := b.chainPages[from]; !ok {
		b.chainPages[from] = map[bool]string{}
	}
	if _, ok := b.chainPages[to]; !ok {
		b.chainPages[to] = map[bool]string{}
	}

	b.chainPages[from][false] = to
	b.chainPages[to][true] = from
}

func (b *GraphBuilder) addDirContents(path string, file structs.File) {
	dir := filepath.Dir(path)
	if dir == "." {
		dir = ""
	}
	b.dirContents[dir] = append(b.dirContents[dir], file)
}

func (b *GraphBuilder) addFile(path string) {
	b.addDirContents(path, structs.File{
		Name:  removeFileExtention(filepath.Base(path)),
		Image: b.media.ImageForPath(removeFileExtention(path)),
	})
}

func (b *GraphBuilder) addHash(path string, contentBytes []byte) {
	b.hashes[path] = fmt.Sprintf("%x", crc32.ChecksumIEEE(contentBytes))
}

func (b *GraphBuilder) addMissingContentHash(content *structs.Content) {
	contentHash := b.generateMissingContentHash(content)
	b.hashes[content.Source] = contentHash
	b.missingContent[content.Source] = content
}

func (b *GraphBuilder) addDir(path string) {
	name := filepath.Base(path)
	if name == "." {
		return
	}

	b.addDirContents(path, structs.File{
		Name:     name,
		IsFolder: true,
	})
}

func (b *GraphBuilder) addAwards() {
	for _, awardPage := range b.awardPages {
		content := b.contents[awardPage]

		year := awardYear(content)
		p := prefix(content, year)

		for i, category := range content.Categories {
			switch {
			case category.Winner.Reference != "":
			case category.Winner.Movie != "":
				category.Winner.Reference = p + "/" + category.Winner.Movie
				category.Winner.Fallback = category.Winner.Movie
			case category.Winner.Game != "":
				category.Winner.Reference = p + "/" + category.Winner.Game
				category.Winner.Fallback = category.Winner.Game
			case category.Winner.Series != "":
				category.Winner.Reference = "Series/" + year + "/" + category.Winner.Series
				category.Winner.Fallback = category.Winner.Series
			case category.Winner.Person != "":
				category.Winner.Reference = filepath.Join("People", category.Winner.Person)
				category.Winner.Fallback = category.Winner.Person
			}
			category.Winner.Reference = b.canonicalContentPath(category.Winner.Reference)
			content.Categories[i] = category

			path := category.Winner.Reference
			if path == "" {
				log.Printf("Unknown winner reference in %q for %q", awardPage, category.Name)
				continue
			}

			award := structs.Award{
				Category:  category.Name,
				Reference: awardPage,
			}

			awardedContent, ok := b.contents[path]
			if !ok {
				path = b.canonicalMissingPath(path)
				b.awardsMissingContent[path] = append(b.awardsMissingContent[path], award)
				continue
			}

			switch true {
			case category.Winner.Actor != "":
				var found bool
				for _, character := range awardedContent.Characters {
					if character.Actor == category.Winner.Actor {
						character.Awards = append(character.Awards, &award)
						found = true
						break
					}
				}
				if !found {
					log.Printf("No character found for %q", category.Winner.Actor)
				}
			case len(category.Winner.Cinematography) > 0:
				awardedContent.CinematographyAwards = append(awardedContent.CinematographyAwards, award)
			case len(category.Winner.Music) > 0:
				awardedContent.MusicAwards = append(awardedContent.MusicAwards, award)
			case len(category.Winner.Editors) > 0:
				awardedContent.EditorsAwards = append(awardedContent.EditorsAwards, award)
			case len(category.Winner.Writers) > 0:
				awardedContent.WritersAwards = append(awardedContent.WritersAwards, award)
			case len(category.Winner.Directors) > 0:
				awardedContent.DirectorsAwards = append(awardedContent.DirectorsAwards, award)
			case len(category.Winner.Screenplay) > 0:
				awardedContent.ScreenplayAwards = append(awardedContent.ScreenplayAwards, award)
			default:
				awardedContent.Awards = append(awardedContent.Awards, award)
			}

			b.contents[path] = awardedContent
		}

		b.contents[awardPage] = content
	}
}

func (b *GraphBuilder) canonicalContentPath(path string) string {
	if _, ok := b.contents[path]; ok {
		return path
	}

	withoutColons := strings.ReplaceAll(path, ":", "")
	if withoutColons != path {
		if _, ok := b.contents[withoutColons]; ok {
			return withoutColons
		}
	}

	if canonical, ok := b.contentsByLower[strings.ToLower(path)]; ok {
		return canonical
	}

	return path
}

// canonicalMissingPath returns an existing key from awardsMissingContent that
// matches path case-insensitively, or path itself if no match exists.
func (b *GraphBuilder) canonicalMissingPath(path string) string {
	lower := strings.ToLower(path)
	for key := range b.awardsMissingContent {
		if strings.ToLower(key) == lower {
			return key
		}
	}
	return path
}

func (b *GraphBuilder) missing() []structs.Missing {
	missing := map[string]map[string][]structs.Connection{}

	for to, from := range b.connections {
		if _, ok := b.contents[to]; !ok && len(from) > 1 {
			missing[to] = from
		}
	}

	result := []structs.Missing{}
	for to, from := range missing {
		result = append(result, structs.Missing{
			To:     to,
			From:   from,
			Awards: b.awardsMissingContent[to],
		})
	}

	for to, awards := range b.awardsMissingContent {
		if _, ok := b.contents[to]; !ok && len(awards) > 1 {
			result = append(result, structs.Missing{
				To:     to,
				From:   nil,
				Awards: awards,
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		ilen := len(result[i].From) + len(result[i].Awards)
		jlen := len(result[j].From) + len(result[j].Awards)
		if ilen == jlen {
			return result[i].To < result[j].To
		}
		return ilen > jlen
	})

	return result
}

func (b *GraphBuilder) addMissingFilesToPanels(missing []structs.Missing) {
	for _, m := range missing {
		if len(m.From)+len(m.Awards) < 2 {
			continue
		}

		id := m.To
		b.addDirContents(id, structs.File{
			Name:      filepath.Base(id),
			Title:     filepath.Base(id),
			Image:     b.media.ImageForPath(id),
			IsMissing: true,
		})

		dir := filepath.Dir(id)
		parentDir := filepath.Dir(dir)
		name := filepath.Base(dir)
		if parentDirContents, ok := b.dirContents[parentDir]; ok {
			found := false
			for _, f := range parentDirContents {
				if f.Name == name {
					found = true
				}
			}
			if !found {
				b.addDirContents(dir, structs.File{
					Name:      name,
					IsFolder:  true,
					IsMissing: true,
				})
			}
		}
	}
}

func (b *GraphBuilder) addMissingContent(missing []structs.Missing) {
	for _, m := range missing {
		if len(m.From)+len(m.Awards) < 2 {
			continue
		}

		content := b.generateContentForMissing(m)
		b.addMissingContentHash(content)
		b.missingPages = append(b.missingPages, MissingPage{
			ID:      m.To,
			Content: content,
		})
	}
}

func (b *GraphBuilder) generateContentForMissing(m structs.Missing) *structs.Content {
	content := &structs.Content{
		IsMissing: true,
		Source:    m.To + ".yml",
		Image:     b.media.ImageForPath(m.To),
		Awards:    m.Awards,
	}

	content.GenerateID()
	content.SetName(filepath.Base(m.To))
	return content
}

func (b *GraphBuilder) generateMissingContentHash(content *structs.Content) string {
	parts := []string{content.Source}

	if name := content.GetName(); name != "" {
		parts = append(parts, name)
	}

	if content.Image != nil {
		parts = append(parts,
			content.Image.Path,
			content.Image.ThumbPath,
			fmt.Sprintf("%d", content.Image.Width),
			fmt.Sprintf("%d", content.Image.Height),
			fmt.Sprintf("%d", content.Image.ThumbXOffset),
			fmt.Sprintf("%d", content.Image.ThumbYOffset),
			fmt.Sprintf("%d", content.Image.ThumbWidth),
			fmt.Sprintf("%d", content.Image.ThumbHeight),
			fmt.Sprintf("%d", content.Image.ThumbTotalWidth),
			fmt.Sprintf("%d", content.Image.ThumbTotalHeight),
		)
	}

	hashData := strings.Join(parts, "|")
	hash := crc32.ChecksumIEEE([]byte(hashData))
	return fmt.Sprintf("%x", hash)
}

func (b *GraphBuilder) processPanels() {
	for path, files := range b.dirContents {
		sort.Sort(structs.ByYearDesk(files))

		for i, file := range files {
			if content := b.contents[filepath.Join(path, file.Name)]; content.GetName() != "" {
				files[i].Title = content.GetName()
				for key, value := range content.Columns() {
					files[i].Columns.Add(key, value)
				}
			} else {
				files[i].Title = file.Name
			}
		}

		b.dirContents[path] = files
	}
}
