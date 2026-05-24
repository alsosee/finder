package main

import (
	"fmt"
	"hash/crc32"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"

	"github.com/alsosee/finder/structs"
)

type SourceFile struct {
	Path string
	Hash string
}

type ScanResult struct {
	InfoFiles []SourceFile
	InfoDirs  []string
	Media     MediaCatalog
	Hashes    map[string]string
}

type MediaCatalog map[string][]structs.Media

func (m MediaCatalog) AddThumbsFile(path string, media []structs.Media) {
	dir := filepath.Dir(path)
	if dir == "." {
		dir = ""
	}
	m[dir] = media
}

func (m MediaCatalog) ImageForPath(path string) *structs.Media {
	dir := filepath.Dir(path)
	if dir == "." {
		dir = ""
	}

	base := structs.EscapeFileName(filepath.Base(path))
	for _, media := range m[dir] {
		mediaImage := media
		if removeFileExtention(media.Path) == base {
			return &mediaImage
		}
	}

	return nil
}

func (m MediaCatalog) PathsSharingThumb(path string) []string {
	dir := filepath.Dir(path)
	if dir == "." {
		dir = ""
	}

	image := m.ImageForPath(removeFileExtention(path))
	if image == nil || image.ThumbPath == "" {
		return nil
	}

	var result []string
	for _, media := range m[dir] {
		if media.ThumbPath == image.ThumbPath && media.Path != image.Path {
			result = append(result, filepath.Join(dir, removeFileExtention(media.Path)+".yml"))
		}
	}
	sort.Strings(result)
	return result
}

type Scanner struct {
	infoDir  string
	mediaDir string
	ignore   *gitignore.GitIgnore
}

func NewScanner(infoDir, mediaDir string, ignore *gitignore.GitIgnore) *Scanner {
	return &Scanner{
		infoDir:  infoDir,
		mediaDir: mediaDir,
		ignore:   ignore,
	}
}

func (s *Scanner) Scan() (*ScanResult, error) {
	result := &ScanResult{
		Media:  MediaCatalog{},
		Hashes: map[string]string{},
	}

	if err := s.scanInfo(result); err != nil {
		return nil, err
	}
	if err := s.scanMedia(result); err != nil {
		return nil, err
	}

	sort.Slice(result.InfoFiles, func(i, j int) bool {
		return result.InfoFiles[i].Path < result.InfoFiles[j].Path
	})
	sort.Strings(result.InfoDirs)

	return result, nil
}

func (s *Scanner) scanInfo(result *ScanResult) error {
	infoDir, err := filepath.Abs(s.infoDir)
	if err != nil {
		return fmt.Errorf("getting absolute path for %q: %w", s.infoDir, err)
	}
	mediaDir := ""
	if s.mediaDir != "" {
		mediaDir, _ = filepath.Abs(s.mediaDir)
	}
	sameInfoAndMedia := mediaDir != "" && mediaDir == infoDir

	log.Printf("Walking info directory %q", infoDir)

	err = filepath.WalkDir(infoDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath := strings.TrimPrefix(path, infoDir+string(filepath.Separator))
		if path == infoDir {
			relPath = ""
		}

		if s.ignore != nil && s.ignore.MatchesPath(relPath) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			if relPath != "" {
				result.InfoDirs = append(result.InfoDirs, relPath)
			}
			return nil
		}
		if sameInfoAndMedia && isMediaOnlyFile(relPath) {
			return nil
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %q: %w", relPath, err)
		}

		hash := fmt.Sprintf("%x", crc32.ChecksumIEEE(b))
		result.Hashes[relPath] = hash
		result.InfoFiles = append(result.InfoFiles, SourceFile{Path: relPath, Hash: hash})
		return nil
	})
	if err != nil {
		return fmt.Errorf("walking info directory: %w", err)
	}

	log.Printf("Done walking info directory %q", s.infoDir)
	return nil
}

func isMediaOnlyFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".mp4":
		return true
	default:
		return filepath.Base(path) == ".thumbs.yml"
	}
}

func (s *Scanner) scanMedia(result *ScanResult) error {
	if s.mediaDir == "" {
		log.Printf("No media files directory specified, skipping")
		return nil
	}

	mediaDir, err := filepath.Abs(s.mediaDir)
	if err != nil {
		return fmt.Errorf("getting absolute path for %q: %w", s.mediaDir, err)
	}

	log.Printf("Walking media directory %q", mediaDir)

	err = filepath.WalkDir(mediaDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != ".thumbs.yml" {
			return nil
		}

		relPath := strings.TrimPrefix(path, mediaDir+string(filepath.Separator))
		media, err := structs.ParseMediaFile(path)
		if err != nil {
			return fmt.Errorf("parsing media file %q: %w", path, err)
		}
		result.Media.AddThumbsFile(relPath, media)
		return nil
	})
	if err != nil {
		return fmt.Errorf("walking media directory: %w", err)
	}

	log.Printf("Done walking media directory %q", s.mediaDir)
	return nil
}

type BuildGraph struct {
	Config         structs.Config
	Contents       structs.Contents
	DirContents    map[string][]structs.File
	Connections    structs.Connections
	Media          MediaCatalog
	Hashes         map[string]string
	MissingContent map[string]*structs.Content
	AwardPages     []string
}

type GraphBuilder struct{}

func (GraphBuilder) Build(g *Generator) *BuildGraph {
	return &BuildGraph{
		Config:         g.config,
		Contents:       g.contents,
		DirContents:    g.dirContents,
		Connections:    g.connections,
		Media:          MediaCatalog(g.mediaDirContents),
		Hashes:         g.hashes,
		MissingContent: g.missingContent,
		AwardPages:     append([]string(nil), g.awardPages...),
	}
}

func NewBuildGraph(g *Generator) *BuildGraph {
	return GraphBuilder{}.Build(g)
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

type Projector interface {
	Name() string
	Run(*BuildGraph) error
}

func RunProjectors(graph *BuildGraph, projectors ...Projector) error {
	for _, projector := range projectors {
		log.Printf("Running %s projector", projector.Name())
		if err := projector.Run(graph); err != nil {
			return fmt.Errorf("running %s projector: %w", projector.Name(), err)
		}
	}
	return nil
}
