package main

import (
	"bytes"
	"errors"
	"fmt"
	"hash/crc32"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gomarkdown/markdown"

	"github.com/alsosee/finder/structs"
)

var errExecutingTemplate = errors.New("error executing template")

// Generator is a struct that generates a static site.
type Generator struct {

	// Connections keep track of references from one file to another.
	// key is a file path, where reference is pointing to.
	// value is a list of files that are pointing to the key.
	connections structs.Connections

	mediaDirContents map[string][]structs.Media

	// awardsMissingContent used to temporary hold awards
	// that are for content that is not yet added.
	awardsMissingContent map[string][]structs.Award

	// chainPages used to keep track of next/prev pages in a series.
	chainPages map[string]map[bool]string // from -> true(next)/false(prev) -> reference

	// hashes is map of CRC32 hashes for each file.
	// key is a file path, value is a hash.
	// Used by indexer to check if file was changed.
	hashes map[string]string

	awardPages []string

	muContents             sync.Mutex // protects writes to contents
	muDir                  sync.Mutex // protects writes to dirContents
	muConnections          sync.Mutex // protects writes to connections
	muMedia                sync.Mutex // protects writes to mediaDirContents
	muAwardPages           sync.Mutex // protects writes to awardPages
	muAwardsMissingContent sync.Mutex // protects writes to awardsMissingContent
	muChainPages           sync.Mutex // protects writes to chainPages
	muRenderedPanels       sync.Mutex // protects writes to renderedPanelsCache
	muHashes               sync.Mutex // protects writes to hashes
}

func (g *Generator) addContent(content structs.Content) {
	content.GenerateID()

	g.muContents.Lock()
	g.contents[content.SourceNoExtention] = content
	g.muContents.Unlock()
}

// addConnections adds a "connection" for a given content file.
func (g *Generator) addConnections(content structs.Content) {
	content.GenerateID()
	from := content.SourceNoExtention

	connections := content.Connections()
	for _, conn := range connections {
		switch conn.Meta {
		case structs.ConnectionPrevious:
			g.addPrevious(from, conn.To)
		case structs.ConnectionSeries:
			g.addConnection(from, series(content), conn)
		case structs.ConnectionNone:
			g.addConnection(from, conn.To, conn)
		default:
			g.addConnection(from, conn.To, conn)
		}
	}

	// Prepare for adding Awards
	if len(content.Categories) > 0 {
		g.addAwardPage(from)
	}
}

func (g *Generator) addConnection(from, to string, connection structs.Connection) {
	g.muConnections.Lock()
	defer g.muConnections.Unlock()

	if _, ok := g.connections[to]; !ok {
		g.connections[to] = map[string][]structs.Connection{}
	}

	if _, ok := g.connections[to][from]; !ok {
		g.connections[to][from] = []structs.Connection{}
	}

	g.connections[to][from] = append(g.connections[to][from], connection)
}

func (g *Generator) addPrevious(from, to string) {
	g.muChainPages.Lock()
	defer g.muChainPages.Unlock()

	if _, ok := g.chainPages[from]; !ok {
		g.chainPages[from] = map[bool]string{}
	}

	if _, ok := g.chainPages[to]; !ok {
		g.chainPages[to] = map[bool]string{}
	}

	g.chainPages[from][false] = to
	g.chainPages[to][true] = from
}

func (g *Generator) addAwardPage(id string) {
	g.muAwardPages.Lock()
	defer g.muAwardPages.Unlock()

	// track all pages that have awards
	// will be used to add Awards to content after all files are processed
	g.awardPages = append(g.awardPages, id)
}

func (g *Generator) addMedia(path string, media []structs.Media) {
	dir := filepath.Dir(path)
	if dir == "." {
		dir = ""
	}

	g.muMedia.Lock()
	g.mediaDirContents[dir] = media
	g.muMedia.Unlock()
}

func (g *Generator) addDirContents(path string, file structs.File) {
	dir := filepath.Dir(path)
	if dir == "." {
		dir = ""
	}

	g.muDir.Lock()
	g.dirContents[dir] = append(g.dirContents[dir], file)
	g.muDir.Unlock()
}

func (g *Generator) addFile(path string) {
	g.addDirContents(path, structs.File{
		Name:  removeFileExtention(filepath.Base(path)),
		Image: g.getImageForPath(removeFileExtention(path)),
	})
}

func (g *Generator) addHash(path string, b []byte) {
	g.muHashes.Lock()
	defer g.muHashes.Unlock()

	g.hashes[path] = fmt.Sprintf("%x", crc32.ChecksumIEEE(b))
}

func (g *Generator) addMissingContentHash(content *structs.Content) {
	g.muHashes.Lock()
	defer g.muHashes.Unlock()

	contentHash := g.generateMissingContentHash(content)
	g.hashes[content.Source] = contentHash
	g.missingContent[content.Source] = content
}

func (g *Generator) addDir(path string) {
	name := filepath.Base(path)
	if name == "." {
		return
	}

	g.addDirContents(path, structs.File{
		Name:     name,
		IsFolder: true,
	})
}

func (g *Generator) generateContentTemplates() error {

}

func (g *Generator) generateGoTemplates() error {
	for path, content := range g.contents {
		if filepath.Ext(content.Source) != ".gomd" {
			continue
		}

		// render Go template
		t, err := g.templates.New("").Funcs(g.fm()).Parse(content.HTML)
		if err != nil {
			return fmt.Errorf("parsing template: %w", err)
		}

		var buf bytes.Buffer
		if err := t.Execute(&buf, nil); err != nil {
			return fmt.Errorf("%w for %q: %w", errExecutingTemplate, path, err)
		}

		htmlBody := markdown.ToHTML(buf.Bytes(), nil, nil)
		content.HTML = string(htmlBody)

		g.contents[path] = content
	}

	return nil
}

func (g *Generator) addMissingFilesToPanels(missing []structs.Missing) {
	// add files to all panels
	for _, m := range missing {
		if len(m.From)+len(m.Awards) < 2 {
			continue
		}

		id := m.To

		file := structs.File{
			Name:      filepath.Base(id),
			Title:     filepath.Base(id),
			Image:     g.getImageForPath(id),
			IsMissing: true,
		}

		g.addDirContents(id, file)

		// check if parent directory exists, usually a year
		dir := filepath.Dir(id)
		parentDir := filepath.Dir(dir)
		name := filepath.Base(dir)
		if parentDirContents, ok := g.dirContents[parentDir]; ok {
			found := false
			for _, f := range parentDirContents {
				if f.Name == name {
					found = true
				}
			}
			if !found {
				g.addDirContents(dir, structs.File{
					Name:      name,
					IsFolder:  true,
					IsMissing: true,
				})
			}
		}
	}
}

func (g *Generator) generateMissing(missing []structs.Missing) error {
	// create channel with PageData to render
	pagesDataChan := make(chan structs.PageData)

	// start 10 workers to render missing files
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for pd := range pagesDataChan {
				err := g.executeTemplate(pd.OutputPath, pd, "index.gohtml")
				if err != nil {
					log.Fatalf("Error executing template for %q: %v", pd.CurrentPath, err)
				}
			}
		}()
	}

	// render all missing files
	for _, m := range missing {
		if len(m.From)+len(m.Awards) < 2 {
			continue
		}

		id := m.To
		panels, breadcrumbs := g.buildPanels(id, true)
		content := g.generateContentForMissing(m)

		// Add hash for missing content to enable indexing
		g.addMissingContentHash(content)

		pagesDataChan <- structs.PageData{
			OutputPath:  filepath.Join(cfg.OutputDirectory, id+".html"),
			CurrentPath: id,
			Dir:         filepath.Dir(id),
			Breadcrumbs: breadcrumbs,
			Panels:      panels,
			Content:     content,
			Timestamp:   time.Now().Unix(),
		}
	}

	close(pagesDataChan)

	wg.Wait()
	return nil
}

func (g *Generator) generateContentForMissing(m structs.Missing) *structs.Content {
	content := &structs.Content{
		IsMissing: true,
		Source:    m.To + ".yml",
		Image:     g.getImageForPath(m.To),
		Awards:    m.Awards,
	}

	content.GenerateID()
	content.SetName(filepath.Base(m.To))

	return content
}

func (g *Generator) generateMissingContentHash(content *structs.Content) string {
	parts := []string{content.Source}

	// Add name
	name := content.GetName()
	if name != "" {
		parts = append(parts, name)
	}

	// Add all Media fields if Image exists
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

func (g *Generator) processPanels() {
	g.muDir.Lock()
	defer g.muDir.Unlock()

	for path, files := range g.dirContents {
		sort.Sort(structs.ByYearDesk(files))

		// update Title if content has it
		for i, file := range files {
			if content := g.contents[filepath.Join(path, file.Name)]; content.GetName() != "" {
				files[i].Title = content.GetName()

				for key, value := range content.Columns() {
					files[i].Columns.Add(key, value)
				}
			} else {
				files[i].Title = file.Name
			}
		}

		g.dirContents[path] = files
	}
}

func (g *Generator) addAwards() {
	for _, awardPage := range g.awardPages {
		content := g.contents[awardPage]

		year := awardYear(content)
		p := prefix(content, year)

		for i, category := range content.Categories {
			switch {
			case category.Winner.Reference != "":
				// reference is already set
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

			var (
				awadredContent structs.Content
				ok             bool
			)
			if awadredContent, ok = g.contents[path]; !ok {
				g.muAwardsMissingContent.Lock()
				g.awardsMissingContent[path] = append(g.awardsMissingContent[path], award)
				g.muAwardsMissingContent.Unlock()
				continue
			}

			switch true {
			case category.Winner.Actor != "":
				// loop through all characters and find actor with the same name
				var found bool
				for _, character := range awadredContent.Characters {
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
				awadredContent.CinematographyAwards = append(awadredContent.CinematographyAwards, award)
			case len(category.Winner.Music) > 0:
				awadredContent.MusicAwards = append(awadredContent.MusicAwards, award)
			case len(category.Winner.Editors) > 0:
				awadredContent.EditorsAwards = append(awadredContent.EditorsAwards, award)
			case len(category.Winner.Writers) > 0:
				awadredContent.WritersAwards = append(awadredContent.WritersAwards, award)
			case len(category.Winner.Directors) > 0:
				awadredContent.DirectorsAwards = append(awadredContent.DirectorsAwards, award)
			case len(category.Winner.Screenplay) > 0:
				awadredContent.ScreenplayAwards = append(awadredContent.ScreenplayAwards, award)
			default:
				awadredContent.Awards = append(awadredContent.Awards, award)
			}

			g.contents[path] = awadredContent
		}

		g.contents[awardPage] = content
	}
}
