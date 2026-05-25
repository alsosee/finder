package main

import (
	"path/filepath"
	"strings"

	"github.com/alsosee/finder/structs"
)

type BuildGraph struct {
	Config               structs.Config
	Contents             structs.Contents
	DirContents          map[string][]structs.File
	Connections          structs.Connections
	Media                MediaCatalog
	Hashes               map[string]string
	MissingContent       map[string]*structs.Content
	Missing              []structs.Missing
	AwardsMissingContent map[string][]structs.Award
	ChainPages           map[string]map[bool]string
	Diagnostics          []Diagnostic
	PassthroughFiles     []string
	MissingPages         []MissingPage
	OpenGraphEnabled     bool
}

type MissingPage struct {
	ID      string
	Content *structs.Content
}

func (g *BuildGraph) Document(path string) (*structs.Content, bool) {
	if content, ok := g.MissingContent[path]; ok {
		contentCopy := *content
		contentCopy.IsMissing = true
		contentCopy.Source = path
		contentCopy.SourceNoExtention = removeFileExtention(contentCopy.Source)
		contentCopy.GenerateID()
		contentCopy.AddMedia(g.Media.ImageForPath)
		return &contentCopy, true
	}

	id := removeFileExtention(path)
	content, ok := g.Contents[id]
	if !ok {
		return nil, false
	}
	contentCopy := content
	contentCopy.GenerateID()
	contentCopy.AddMedia(g.Media.ImageForPath)
	return &contentCopy, true
}

func (g *BuildGraph) FilesForPath(path string) []structs.File {
	return g.DirContents[path]
}

func (g *BuildGraph) Panels(path string, isFile bool) (structs.Panels, structs.Breadcrumbs) {
	panels := structs.Panels{}
	breadcrumbs := structs.Breadcrumbs{}

	dirs := strings.Split(path, string(filepath.Separator))
	if path != "" {
		dirs = append([]string{""}, dirs...)
	}

	cumulativePath := ""
	for _, dir := range dirs {
		cumulativePath = filepath.Join(cumulativePath, dir)

		if dir == "" {
			dir = g.Config.HomeLabel
		}

		breadcrumbs = append(breadcrumbs, structs.Dir{
			Name: dir,
			Path: cumulativePath,
		})

		if isFile && cumulativePath == path {
			break
		}

		panels = append(panels, structs.Panel{
			Dir:   cumulativePath,
			Files: g.FilesForPath(cumulativePath),
		})
	}

	return panels, breadcrumbs
}

func (g *BuildGraph) OpenGraphImage(id string) string {
	if !g.OpenGraphEnabled {
		return ""
	}
	return openGraphURL(g.Config.OpenGraphHost, id)
}
