package main

import (
	"errors"
	"fmt"
	"hash/crc32"
	"io"
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
	muDir       sync.Mutex

	// Connections keep track of references from one file to another.
	// key is a file path, where reference is pointing to.
	// value is a list of files that are pointing to the key.
	connections   Connections
	muConnections sync.Mutex

	mediaDirContents map[string][]Media
	muMedia          sync.Mutex
}

// NewGenerator creates a new Generator.
func NewGenerator() (*Generator, error) {
	ignore := &gitignore.GitIgnore{}
	ignoreFilepath := filepath.Join(cfg.InfoDirectory, cfg.IgnoreFile)
	if _, err := os.Stat(ignoreFilepath); err == nil {
		ignore, err = gitignore.CompileIgnoreFile(ignoreFilepath)
		if err != nil {
			return nil, fmt.Errorf("compiling ignore file: %w", err)
		}
	} else {
		log.Printf("Ignore file %q not found, ignoring", ignoreFilepath)
	}

	return &Generator{
		ignore:           ignore,
		contents:         Contents{},
		dirContents:      map[string][]File{},
		connections:      Connections{},
		mediaDirContents: map[string][]Media{},
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
		"crc32": func(path string) string {
			// calculate CRC32 checksum for a file
			file, err := os.Open(filepath.Join(cfg.OutputDirectory, path))
			if err != nil {
				log.Printf("Error opening file %q: %v", path, err)
				return ""
			}
			defer file.Close()

			hash := crc32.NewIEEE()
			if _, err := io.Copy(hash, file); err != nil {
				log.Printf("Error calculating CRC32 checksum for file %q: %v", path, err)
				return ""
			}

			return fmt.Sprintf("%x", hash.Sum32())
		},
		"div": func(a, b int) int {
			return a / b
		},
		"hasPrefix": strings.HasPrefix,
		"initials": func(name string) string {
			var initials string
			for _, s := range strings.Split(name, " ") {
				initials += strings.ToUpper(s[:1]) + " " // thin space
			}
			return strings.TrimSpace(initials)
		},
		"thumbStyle": func(media Media, width int) string {
			if media.ThumbPath == "" {
				return ""
			}

			return fmt.Sprintf(
				"background-size: %dpx %dpx; background-position: -%dpx -%dpx; width: %dpx; height: %dpx",
				media.ThumbTotalWidth*width/media.ThumbWidth,
				media.ThumbTotalHeight*width/media.ThumbWidth,
				media.ThumbXOffset*width/media.ThumbWidth,
				media.ThumbYOffset*width/media.ThumbWidth,
				width,
				media.ThumbHeight*width/media.ThumbWidth,
			)
		},
		"length": func(a time.Duration) string {
			// format duration as "1h 2m 3s"
			return fmt.Sprintf("%dh %dm", int(a.Hours()), int(a.Minutes())%60)
		},
		"either": func(args ...interface{}) bool {
			for _, arg := range args {
				switch v := arg.(type) {
				case bool:
					if v {
						return true
					}

				case string:
					if v != "" {
						return true
					}

				case int:
					if v != 0 {
						return true
					}

				case time.Duration:
					if v != 0 {
						return true
					}

				case []string:
					if len(v) != 0 {
						return true
					}
				}
			}
			return false
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

	g.copyStaticFiles()
	go g.walkInfoDirectory(files, errorsChan)
	go g.walkMediaDirectory()
	go g.processFiles(files, errorsChan, done)

FILE_PROCESSING:
	for {
		select {
		case err := <-errorsChan:
			close(done)
			return fmt.Errorf("walking info directory: %w", err)

		case <-done:
			log.Printf("Done processing files")
			close(done)
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

func (g *Generator) copyStaticFiles() {
	if cfg.StaticDirectory == "" {
		log.Printf("No static files directory specified, skipping")
		return
	}

	log.Printf("Copying static files from %q to %q", cfg.StaticDirectory, cfg.OutputDirectory)

	if err := os.MkdirAll(cfg.OutputDirectory, 0o755); err != nil {
		log.Fatalf("Error creating output directory %q: %v", cfg.OutputDirectory, err)
	}

	err := filepath.Walk(
		cfg.StaticDirectory,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			relPath := strings.TrimPrefix(path, cfg.StaticDirectory+string(filepath.Separator))
			return copyFile(path, filepath.Join(cfg.OutputDirectory, relPath))
		},
	)
	if err != nil {
		log.Fatalf("Error walking static directory %q: %v", cfg.StaticDirectory, err)
	}

	log.Printf("Done copying static files from %q to %q", cfg.StaticDirectory, cfg.OutputDirectory)
}

func (g *Generator) walkInfoDirectory(files chan<- string, errorsChan chan<- error) {
	defer close(files)

	infoDir, err := filepath.Abs(cfg.InfoDirectory)
	if err != nil {
		errorsChan <- fmt.Errorf("getting absolute path for %q: %w", cfg.InfoDirectory, err)
		return
	}

	log.Printf("Walking info directory %q", infoDir)

	err = filepath.Walk(
		infoDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath := strings.TrimPrefix(path, infoDir+string(filepath.Separator))

			if g.ignore.MatchesPath(relPath) {
				return nil
			}

			if info.IsDir() {
				g.addDir(relPath)
				return nil
			}

			files <- relPath
			return nil
		},
	)

	if err != nil {
		errorsChan <- err
	} else {
		log.Printf("Done walking info directory %q", cfg.InfoDirectory)
	}
}

// walkMediaDirectory scans the media directory for .thumbs.yml files,
// parses them and adds to g.mediaDirContents.
// mediaDirContents is a map where key is a directory path, and value is a list of media files in that directory.
// Information from .thumbs.yml used later in template to build links to thumbnails.
func (g *Generator) walkMediaDirectory() {
	if cfg.MediaDirectory == "" {
		log.Printf("No media files directory specified, skipping")
		return
	}

	mediaDir, err := filepath.Abs(cfg.MediaDirectory)
	if err != nil {
		log.Fatalf("Error getting absolute path for %q: %v", cfg.MediaDirectory, err)
	}

	log.Printf("Walking media directory %q", mediaDir)

	err = filepath.Walk(
		mediaDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath := strings.TrimPrefix(path, mediaDir+string(filepath.Separator))

			if info.IsDir() {
				return nil
			}

			if info.Name() != ".thumbs.yml" {
				return nil
			}

			media, err := parseMediaFile(path)
			if err != nil {
				return fmt.Errorf("parsing media file %q: %w", path, err)
			}

			g.addMedia(relPath, media)

			return nil
		},
	)

	if err != nil {
		log.Fatalf("Error walking media directory %q: %v", cfg.MediaDirectory, err)
	}

	log.Printf("Done walking media directory %q", cfg.MediaDirectory)
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
		g.addFile(file)
		return g.processYAMLFile(file)
	case ".md":
		g.addFile(file)
		return g.processMarkdownFile(file)
	case ".jpeg", ".jpg", ".png":
		g.addFile(file)
		return g.processImageFile(file)
	case ".mp4":
		g.addFile(file)
		return g.processVideoFile(file)
	default:
		if file == "_redirects" {
			return g.copyFileAsIs(file)
		}
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

func (g *Generator) copyFileAsIs(file string) error {
	return copyFile(
		filepath.Join(cfg.InfoDirectory, file),
		filepath.Join(cfg.OutputDirectory, file),
	)
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

	if _, ok := g.connections[to]; !ok {
		g.connections[to] = map[string]struct{}{}
	}

	g.connections[to][from] = struct{}{}
}

func (g *Generator) addMedia(path string, media []Media) {
	dir := filepath.Dir(path)
	if dir == "." {
		dir = ""
	}

	g.muMedia.Lock()
	g.mediaDirContents[dir] = media
	g.muMedia.Unlock()
}

func (g *Generator) addFile(path string) {
	dir := filepath.Dir(path)
	if dir == "." {
		dir = ""
	}

	g.muDir.Lock()
	g.dirContents[dir] = append(g.dirContents[dir], File{
		Name: removeFileExtention(filepath.Base(path)),
	})
	g.muDir.Unlock()
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

		// create directory
		if err := os.MkdirAll(filepath.Join(cfg.OutputDirectory, filepath.Dir(path)), 0o755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}

		f, err := os.Create(filepath.Join(cfg.OutputDirectory, path))
		if err != nil {
			return fmt.Errorf("creating file: %w", err)
		}

		pathWithoutExt := removeFileExtention(path)

		panels, breadcrumbs := g.buildPanels(pathWithoutExt, true)

		cnt := content
		cnt.Image = g.getImageForPath(pathWithoutExt)

		// add image to Characters
		for _, character := range cnt.Characters {
			character.Image = g.getImageForPath("People/" + character.Actor)
		}

		if err := g.templates.ExecuteTemplate(
			f,
			"index.gohtml",
			struct {
				CurrentPath string
				Dir         string
				Breadcrumbs Breadcrumbs
				Panels      Panels
				Content     *Content
				Timestamp   int64
			}{
				CurrentPath: pathWithoutExt,
				Dir:         filepath.Dir(pathWithoutExt),
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

func (g *Generator) getImageForPath(path string) *Media {
	dir := filepath.Dir(path)
	if dir == "." {
		dir = ""
	}

	base := filepath.Base(path)
	base = strings.ReplaceAll(base, ":", "") // file names can't contain colons

	dirContent, ok := g.mediaDirContents[dir]
	if !ok {
		return nil
	}

	for _, media := range dirContent {
		if strings.HasPrefix(media.Path, base) {
			return &media
		}
	}

	return nil
}

func (g *Generator) generateIndexes() error {
	for dir, files := range g.dirContents {
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

func copyFile(src, dst string) error {
	log.Printf("Copying file %q to %q", src, dst)
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}
