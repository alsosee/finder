package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/gomarkdown/markdown"
	gitignore "github.com/sabhiram/go-gitignore"
	"gopkg.in/yaml.v3"
)

// Generator is a struct that generates a static site.
type Generator struct {
	templates *template.Template
	ignore    *gitignore.GitIgnore

	fileLists   FileLists
	connections Connections
	contents    Contents
}

var fm = template.FuncMap{
	"join": func(dir, name string) string {
		return filepath.Join(dir, name)
	},
}

// NewGenerator creates a new Generator.
func NewGenerator() (*Generator, error) {
	t, err := template.New("").Funcs(fm).ParseGlob(cfg.TemplatesDirectory + "/*")
	if err != nil {
		return nil, fmt.Errorf("Error parsing templates: %v", err)
	}

	ignore := &gitignore.GitIgnore{}
	ignoreFilepath := filepath.Join(cfg.InfoDirectory, cfg.IgnoreFile)
	if _, err := os.Stat(ignoreFilepath); err == nil {
		ignore, err = gitignore.CompileIgnoreFile(ignoreFilepath)
		if err != nil {
			log.Fatalf("error reading ignore file: %v", err)
		}
	}

	return &Generator{
		templates:   t,
		ignore:      ignore,
		connections: Connections{},
		contents:    Contents{},
	}, nil
}

// Run runs the generator.
func (g *Generator) Run() error {
	defer measureTime()()

	var (
		files  = make(chan string)
		errors = make(chan error)
		done   = make(chan struct{}, 1)
	)
	defer close(errors)

	go g.walkInfoDirectory(files, errors)
	go g.processFiles(files, errors, done)

	for {
		select {
		case err := <-errors:
			close(files)
			close(done)
			return fmt.Errorf("Error walking info directory: %v", err)

		case <-done:
			log.Printf("Done processing files")
			return nil
		}
	}
}

func (g *Generator) walkInfoDirectory(files chan<- string, errors chan<- error) {
	log.Printf("Walking info directory %q", cfg.InfoDirectory)

	err := filepath.Walk(
		cfg.InfoDirectory,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if g.ignore.MatchesPath(path) {
				return nil
			}

			files <- path[len(cfg.InfoDirectory):]
			return nil
		},
	)
	log.Printf("Done walking info directory %q", cfg.InfoDirectory)

	if err != nil {
		errors <- err
	}

	close(files)
}

func (g *Generator) processFiles(files <-chan string, errors chan<- error, done chan<- struct{}) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Error processing files: %v", r)
		}
	}()

	for file := range files {
		log.Printf("Processing %s", file)

		err := g.processFile(file)
		if err != nil {
			errors <- fmt.Errorf("parsing file %q: %v", file, err)
			continue
		}
	}

	done <- struct{}{}
}

func (g *Generator) processFile(file string) error {
	switch filepath.Ext(file) {
	case ".yml", ".yaml":
		return g.processYAMLFile(file)
	case ".md":
		return g.processMarkdownFile(file)
	case ".jpeg", ".jpg", ".png":
		return g.processImageFile(file)
	case ".mp4":
		return g.processVideoFile(file)
	default:
		return fmt.Errorf("Unknown file type: %q", file)
	}
}

func (g *Generator) processYAMLFile(file string) error {
	b, err := os.ReadFile(filepath.Join(cfg.InfoDirectory, file))
	if err != nil {
		return fmt.Errorf("reading file: %v", err)
	}

	var content Content
	if err = yaml.Unmarshal(b, &content); err != nil {
		return fmt.Errorf("unmarshaling yaml: %v", err)
	}

	// change the file extension to .html
	file = file[:len(file)-len(filepath.Ext(file))] + ".html"
	outputFilepath := filepath.Join(cfg.OutputDirectory, file)

	if err := os.MkdirAll(filepath.Dir(outputFilepath), 0755); err != nil {
		return fmt.Errorf("creating output directory: %v", err)
	}

	f, err := os.Create(outputFilepath)
	if err != nil {
		return fmt.Errorf("creating output file: %v", err)
	}
	defer f.Close()

	return g.templates.ExecuteTemplate(
		f,
		"index.gohtml",
		struct {
			HXRequest   bool
			CurrentPath string
			Dirs        []Dir
			Panels      Panels
			Content     *Content
			Timestamp   int64
			Connections Connections
		}{
			HXRequest:   false,
			CurrentPath: "",
			Dirs:        buildDirs(filepath.Dir(file)),
			Panels:      g.buildPanels(filepath.Dir(file)),
			Content:     &content,
			Timestamp:   time.Now().Unix(),
			Connections: nil,
		},
	)
}

func (g *Generator) processMarkdownFile(file string) error {
	b, err := os.ReadFile(filepath.Join(cfg.InfoDirectory, file))
	if err != nil {
		return fmt.Errorf("reading file: %v", err)
	}

	htmlBody := markdown.ToHTML(b, nil, nil)
	content := Content{
		HTML: string(htmlBody),
	}

	// change the file extension to .html
	file = file[:len(file)-len(filepath.Ext(file))] + ".html"
	outputFilepath := filepath.Join(cfg.OutputDirectory, file)

	if err := os.MkdirAll(filepath.Dir(outputFilepath), 0755); err != nil {
		return fmt.Errorf("creating output directory: %v", err)
	}

	f, err := os.Create(outputFilepath)
	if err != nil {
		return fmt.Errorf("creating output file: %v", err)
	}
	defer f.Close()

	return g.templates.ExecuteTemplate(
		f,
		"index.gohtml",
		struct {
			HXRequest   bool
			CurrentPath string
			Dirs        []Dir
			Panels      Panels
			Content     *Content
			Timestamp   int64
			Connections Connections
		}{
			HXRequest:   false,
			CurrentPath: "",
			Dirs:        buildDirs(filepath.Dir(file)),
			Panels:      g.buildPanels(filepath.Dir(file)),
			Content:     &content,
			Timestamp:   time.Now().Unix(),
			Connections: nil,
		},
	)
}

func (g *Generator) processImageFile(file string) error {
	return nil
}

func (g *Generator) processVideoFile(file string) error {
	return nil
}

func (g *Generator) buildPanels(path string) Panels {
	dirs := buildDirs(path)
	panels := Panels{}
	for i := range dirs {
		// list files in the directory
		if panel, ok := g.fileLists[dirs[i].Path]; ok {
			panels = append(panels, panel)
		} else {
			panels = append(panels, Panel{
				Files: g.listFiles(dirs[i].Path),
			})
		}
	}

	return panels
}

func (g *Generator) listFiles(path string) (files []File) {
	realDir := filepath.Join(cfg.InfoDirectory, path)

	entries, err := os.ReadDir(realDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
	}

	for _, entry := range entries {
		name := entry.Name()

		if g.ignore.MatchesPath(name) {
			continue
		}

		// remove extension
		name = strings.TrimSuffix(name, filepath.Ext(name))
		if len(name) == 0 {
			name = entry.Name()
		}

		files = append(files, File{
			Name:            name,
			Dir:             path,
			IsFolder:        entry.IsDir(),
			IsInBreakcrumbs: false, // todo: entry.IsDir() && strings.HasPrefix(path, filepath.Join(dir, entry.Name())),
		})
	}

	return files
}

func buildDirs(path string) (dirs []Dir) {
	path = strings.TrimSuffix(path, string(filepath.Separator))

	var prevPath string
	for _, dir := range strings.Split(path, string(filepath.Separator)) {
		if len(dir) == 0 {
			path = "/"
			dir = "Home"
		} else {
			path = filepath.Join(prevPath, dir)
		}
		prevPath = path

		dirs = append(dirs, Dir{
			Name: dir,
			Path: path,
		})
	}
	return dirs
}

func measureTime() func() {
	start := time.Now()
	return func() {
		log.Printf("Elapsed: %v", time.Since(start))
	}
}
