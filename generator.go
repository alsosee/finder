package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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

	contents   Contents
	muContents sync.Mutex

	// dirContents is a map where
	// key is a directory path,
	// value is a list of files and directories;
	// used to build Panels
	dirContents map[string][]File

	// Connections keep track of references from one file to another.
	// key is a file path, where reference is pointing to.
	// value is a list of files that are pointing to the key.
	connections   Connections
	muConnections sync.Mutex
}

// NewGenerator creates a new Generator.
func NewGenerator(cfg Config) (*Generator, error) {
	ignore := &gitignore.GitIgnore{}
	ignoreFilepath := filepath.Join(cfg.InfoDirectory, cfg.IgnoreFile)
	if _, err := os.Stat(ignoreFilepath); err == nil {
		ignore, err = gitignore.CompileIgnoreFile(ignoreFilepath)
		if err != nil {
			return nil, fmt.Errorf("compiling ignore file: %w", err)
		}
	}

	return &Generator{
		ignore:      ignore,
		contents:    Contents{},
		dirContents: map[string][]File{},
		connections: Connections{},
	}, nil
}

func (g *Generator) fm() template.FuncMap {
	return template.FuncMap{
		"join": func(dir, name string) string {
			return filepath.Join(dir, name)
		},
		"connections": func(path string) []string {
			g.muConnections.Lock()
			defer g.muConnections.Unlock()

			if m, ok := g.connections[path]; ok {
				var connections []string
				for k := range m {
					connections = append(connections, k)
				}
				return connections
			}
			return nil
		},
	}
}

// Run runs the generator.
func (g *Generator) Run() error {
	t, err := template.New("").Funcs(g.fm()).ParseGlob(cfg.TemplatesDirectory + "/*")
	if err != nil {
		return fmt.Errorf("parsing templates: %w", err)
	}
	g.templates = t

	defer measureTime()()

	// Go through all the files in the info directory
	var (
		files      = make(chan string)
		errorsChan = make(chan error)
		done       = make(chan struct{}, 1)
	)
	defer close(errorsChan)

	go g.walkInfoDirectory(files, errorsChan)
	go g.processFiles(files, errorsChan, done)

FILE_PROCESSING:
	for {
		select {
		case err := <-errorsChan:
			close(files)
			close(errorsChan)
			close(done)
			return fmt.Errorf("walking info directory: %w", err)

		case <-done:
			log.Printf("Done processing files")
			break FILE_PROCESSING
		}
	}

	// Generate file templates
	if err := g.generateContentTemplates(); err != nil {
		return fmt.Errorf("generating content templates: %w", err)
	}

	// Generate index for each directory
	if err := g.generateIndexes(); err != nil {
		return fmt.Errorf("generating indexes: %w", err)
	}

	return nil
}

func (g *Generator) walkInfoDirectory(files chan<- string, errorsChan chan<- error) {
	log.Printf("Walking info directory %q", cfg.InfoDirectory)

	err := filepath.Walk(
		cfg.InfoDirectory,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath := path[len(cfg.InfoDirectory):]

			if info.IsDir() {
				g.addDir(relPath)
				return nil
			}

			if g.ignore.MatchesPath(path) {
				return nil
			}

			g.addFile(relPath)
			files <- relPath
			return nil
		},
	)
	log.Printf("Done walking info directory %q", cfg.InfoDirectory)

	if err != nil {
		errorsChan <- err
	}

	close(files)
}

func (g *Generator) processFiles(files <-chan string, errorsChan chan<- error, done chan<- struct{}) {
	wg := sync.WaitGroup{}

	for i := 0; i < cfg.NumWorkers; i++ {
		for path := range files {
			wg.Add(1)
			go func(path string) {
				defer wg.Done()

				if err := g.processFile(path); err != nil {
					errorsChan <- fmt.Errorf("processing file %q: %w", path, err)
				}
			}(path)
		}
	}

	wg.Wait()

	done <- struct{}{}
}

// processFile processes a single file.
// For content files, like YAML and Markdown, it adds Content struct to g.contents.
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
		return fmt.Errorf("unknown file type: %q", file)
	}
}

func (g *Generator) processYAMLFile(file string) error {
	b, err := os.ReadFile(filepath.Join(cfg.InfoDirectory, file))
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	var content Content
	if err = yaml.Unmarshal(b, &content); err != nil {
		return fmt.Errorf("unmarshaling yaml: %w", err)
	}

	g.addContent(file, content)
	g.addConnections(file, content)

	return nil
}

func (g *Generator) processMarkdownFile(file string) error {
	b, err := os.ReadFile(filepath.Join(cfg.InfoDirectory, file))
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	htmlBody := markdown.ToHTML(b, nil, nil)

	g.addContent(file, Content{HTML: string(htmlBody)})
	return nil
}

func (g *Generator) processImageFile(_ string) error {
	return nil
}

func (g *Generator) processVideoFile(_ string) error {
	return nil
}

func (g *Generator) addContent(path string, content Content) {
	g.muContents.Lock()
	g.contents[removeFileExtention(path)] = content
	g.muContents.Unlock()
}

func (g *Generator) addConnections(path string, content Content) {
	for _, ref := range content.References {
		g.addConnection(removeFileExtention(path), ref.Path)
	}
}

func (g *Generator) addConnection(from, to string) {
	g.muConnections.Lock()
	defer g.muConnections.Unlock()

	log.Printf("Adding connection from %q to %q", from, to)

	if _, ok := g.connections[to]; !ok {
		g.connections[to] = map[string]struct{}{}
	}

	g.connections[to][from] = struct{}{}
}

func (g *Generator) addFile(path string) {
	dir := filepath.Dir(path)
	if dir == "." {
		dir = ""
	}

	g.dirContents[dir] = append(g.dirContents[dir], File{
		Name:     removeFileExtention(filepath.Base(path)),
		IsFolder: false,
	})
}

func (g *Generator) addDir(path string) {
	dir := filepath.Dir(path)
	if dir == "." {
		dir = ""
	}

	name := filepath.Base(path)
	if name == "." {
		return
	}

	g.dirContents[dir] = append(g.dirContents[dir], File{
		Name:     name,
		IsFolder: true,
	})
}

func (g *Generator) getFilesForPath(path string) []File {
	if files, ok := g.dirContents[path]; ok {
		sort.Sort(ByNameFolderOnTop(files))
		return files
	}

	return nil
}

func (g *Generator) generateContentTemplates() error {
	for path, content := range g.contents {
		// replace extension with .html
		path = path[:len(path)-len(filepath.Ext(path))] + ".html"

		log.Printf("Generating %q", path)

		// create directory
		if err := os.MkdirAll(filepath.Join(cfg.OutputDirectory, filepath.Dir(path)), 0o755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}

		f, err := os.Create(filepath.Join(cfg.OutputDirectory, path))
		if err != nil {
			return fmt.Errorf("creating file: %w", err)
		}

		panels, breadcrumbs := g.buildPanels(removeFileExtention(path), true)

		cnt := content

		if err := g.templates.ExecuteTemplate(
			f,
			"index.gohtml",
			struct {
				CurrentPath string
				Breadcrumbs Breadcrumbs
				Panels      Panels
				Content     *Content
				Timestamp   int64
			}{
				CurrentPath: removeFileExtention(path),
				Breadcrumbs: breadcrumbs,
				Panels:      panels,
				Content:     &cnt,
				Timestamp:   time.Now().Unix(),
			},
		); err != nil {
			err2 := f.Close()
			if err2 != nil {
				err = errors.Join(err, err2)
			}
			return fmt.Errorf("executing template: %w", err)
		}

		if err := f.Close(); err != nil {
			return fmt.Errorf("closing file: %w", err)
		}
	}

	return nil
}

func (g *Generator) generateIndexes() error {
	for dir, files := range g.dirContents {
		log.Printf("Generating index for %q", dir)
		sort.Sort(ByNameFolderOnTop(files))

		path := filepath.Join(cfg.OutputDirectory, dir, "index.html")

		// create directory
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}

		f, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("creating file: %w", err)
		}

		panels, breadcrumbs := g.buildPanels(dir, false)

		if err := g.templates.ExecuteTemplate(
			f,
			"index.gohtml",
			struct {
				CurrentPath string
				Breadcrumbs []Dir
				Panels      Panels
				Content     *Content
				Timestamp   int64
				Connections Connections
			}{
				CurrentPath: dir,
				Breadcrumbs: breadcrumbs,
				Panels:      panels,
				Content:     nil,
				Timestamp:   time.Now().Unix(),
				Connections: nil,
			},
		); err != nil {
			err2 := f.Close()
			if err2 != nil {
				err = errors.Join(err, err2)
			}
			return fmt.Errorf("executing template: %w", err)
		}

		if err := f.Close(); err != nil {
			return fmt.Errorf("closing file: %w", err)
		}
	}

	return nil
}

func (g *Generator) buildPanels(path string, isFile bool) (Panels, Breadcrumbs) {
	panels := Panels{}
	breadcrumbs := Breadcrumbs{}

	dirs := strings.Split(path, string(filepath.Separator))
	if path != "" {
		dirs = append([]string{""}, dirs...)
	}

	cumulativePath := ""
	for _, dir := range dirs {
		cumulativePath = filepath.Join(cumulativePath, dir)

		if dir == "" {
			dir = "Home"
		}

		breadcrumbs = append(breadcrumbs, Dir{
			Name: dir,
			Path: cumulativePath,
		})

		// if it's a file, don't add last panel
		// (it is the file itself, which will be rendered
		if isFile && cumulativePath == path {
			break
		}

		panels = append(panels, Panel{
			Dir:   cumulativePath,
			Files: g.getFilesForPath(cumulativePath),
		})
	}

	return panels, breadcrumbs
}

func measureTime() func() {
	start := time.Now()
	return func() {
		log.Printf("Elapsed: %v", time.Since(start))
	}
}

func removeFileExtention(path string) string {
	withoutExt := path[:len(path)-len(filepath.Ext(path))]
	if withoutExt != "" {
		return withoutExt
	}
	return path
}
