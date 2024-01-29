package main

import (
	"bytes"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/gomarkdown/markdown"
	gitignore "github.com/sabhiram/go-gitignore"
	"gopkg.in/yaml.v3"

	"github.com/alsosee/finder/structs"
)

// Generator is a struct that generates a static site.
type Generator struct {
	templates *template.Template
	ignore    *gitignore.GitIgnore

	contents   structs.Contents
	muContents sync.Mutex

	// dirContents is a map where
	// key is a directory path,
	// value is a list of files and directories;
	// used to build Panels
	dirContents map[string][]structs.File
	muDir       sync.Mutex

	// Connections keep track of references from one file to another.
	// key is a file path, where reference is pointing to.
	// value is a list of files that are pointing to the key.
	connections   structs.Connections
	muConnections sync.Mutex

	mediaDirContents map[string][]structs.Media
	muMedia          sync.Mutex
}

// NewGenerator creates a new Generator.
func NewGenerator() (*Generator, error) {
	ignore, err := processIgnoreFile(cfg.IgnoreFile)
	if err != nil {
		return nil, fmt.Errorf("processing ignore file: %w", err)
	}

	return &Generator{
		ignore:           ignore,
		contents:         structs.Contents{},
		dirContents:      map[string][]structs.File{},
		connections:      structs.Connections{},
		mediaDirContents: map[string][]structs.Media{},
	}, nil
}

func processIgnoreFile(ignoreFile string) (*gitignore.GitIgnore, error) {
	ignore := &gitignore.GitIgnore{}
	ignoreFilepath := filepath.Join(cfg.InfoDirectory, ignoreFile)
	if _, err := os.Stat(ignoreFilepath); err == nil {
		ignore, err = gitignore.CompileIgnoreFile(ignoreFilepath)
		if err != nil {
			return nil, fmt.Errorf("compiling ignore file: %w", err)
		}
	} else {
		log.Printf("Ignore file %q not found, ignoring", ignoreFilepath)
	}

	return ignore, nil
}

func (g *Generator) fm() template.FuncMap {
	return template.FuncMap{
		"config":    cfg.GetString,
		"join":      filepath.Join,
		"dir":       filepath.Dir,
		"hasPrefix": strings.HasPrefix,
		"strjoin":   strings.Join,
		// "content" returns a Content struct for a given file path (without extension)
		// It is used to render references.
		"content": func(id string) *structs.Content {
			g.muContents.Lock()
			defer g.muContents.Unlock()

			if c, ok := g.contents[id]; ok {
				return &c
			}
			return nil
		},
		// "connections" returns a list of connections for a given file path (without extension)
		"connections": func(path string) map[string][]string {
			g.muConnections.Lock()
			defer g.muConnections.Unlock()

			if m, ok := g.connections[path]; ok {
				return m
			}
			return nil
		},
		// "crc32" calculates CRC32 checksum for a file.
		// It's used to add a get parameter to a static file URL,
		// so that when the file is updated, the browser will download the new version.
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
		"initials": func(name string) string {
			if name == "" {
				return ""
			}
			var initials string
			for _, s := range strings.Split(name, " ") {
				initials += strings.ToUpper(s[:1]) + " " // thin space
			}
			return strings.TrimSpace(initials)
		},
		// "thumbStylePx" returns CSS styles for a thumbnail image,
		// where background-size is in pixels.
		// It's used for non-responsive images, and more reliable than "thumbStylePct".
		"thumbStylePx": func(media structs.Media, max int, opt ...string) string {
			if media.ThumbPath == "" {
				return ""
			}

			var (
				backgroundWidth  = media.ThumbTotalWidth * max / media.ThumbWidth
				backgroundHeight = media.ThumbTotalHeight * max / media.ThumbWidth
				positionX        = media.ThumbXOffset * max / media.ThumbWidth
				positionY        = media.ThumbYOffset * max / media.ThumbWidth
				width            = max
				height           = media.ThumbHeight * max / media.ThumbWidth
			)

			p := ""
			if len(opt) > 0 {
				p = opt[0]
			}

			if media.Height > media.Width {
				backgroundWidth = media.ThumbTotalWidth * max / media.ThumbHeight
				backgroundHeight = media.ThumbTotalHeight * max / media.ThumbHeight
				positionX = media.ThumbXOffset * max / media.ThumbHeight
				positionY = media.ThumbYOffset * max / media.ThumbHeight
				width = media.ThumbWidth * max / media.ThumbHeight
				height = max
			}

			marginLeft := (max - width) / 2
			marginRight := max - width - marginLeft

			style := fmt.Sprintf(
				"%sbackground-size: %dpx %dpx; %swidth: %dpx; %sheight: %dpx; %scomp-margin-left: %dpx; %scomp-margin-right: %dpx",
				p, backgroundWidth, backgroundHeight,
				p, width,
				p, height,
				p, marginLeft,
				p, marginRight,
			)

			if positionX != 0 || positionY != 0 {
				style += fmt.Sprintf("; %sbackground-position: -%dpx -%dpx", p, positionX, positionY)
			}

			return style
		},
		// "thumbStylePct" returns CSS styles for a thumbnail image,
		// where background-size is in percents. It's used for responsive images.
		// It can be used when last image in the sprite has the same width as the current one,
		// which is the case for most people/characters images.
		// Also, it doesn't add "comp-margin-left" and "comp-margin-right" styles,
		// which are used to center the image in lists.
		"thumbStylePct": func(media structs.Media, prefix ...string) string {
			if media.ThumbPath == "" {
				return ""
			}

			p := ""
			if len(prefix) > 0 {
				p = prefix[0]
			}

			// assume than image width is 100%
			// how much bigger the whole sprite is?
			width := media.ThumbTotalWidth * 100 / media.ThumbWidth
			height := media.ThumbTotalHeight * 100 / media.ThumbHeight

			positionX := 0.0
			positionY := 0.0
			if media.ThumbTotalWidth != media.ThumbWidth {
				// position 100% is the right edge of the image
				// assuming here that last image in the sprite has the same width as the current one
				positionX = float64(media.ThumbXOffset) * 100 / float64(media.ThumbTotalWidth-media.ThumbWidth)
			}
			if media.ThumbTotalHeight != media.ThumbHeight {
				positionY = float64(media.ThumbYOffset) * 100 / float64(media.ThumbTotalHeight-media.ThumbHeight)
			}

			arX := media.ThumbWidth
			arY := media.ThumbHeight
			if arX == arY {
				arX = 1
				arY = 1
			}

			if positionX == 0 && positionY == 0 {
				return fmt.Sprintf(
					"%sbackground-size: %d%% %d%%; %saspect-ratio: %d/%d;",
					p, width, height,
					p, arX, arY,
				)
			}

			return fmt.Sprintf(
				"%sbackground-size: %d%% %d%%; %sbackground-position: %.2f%% %.2f%%; %saspect-ratio: %d/%d;",
				p, width, height,
				p, positionX, positionY,
				p, arX, arY,
			)
		},
		"isPNG": func(path string) bool {
			return strings.HasSuffix(path, ".png")
		},
		// "isJPG" is used to add "jpg" class to links that have JPG image thumbnails
		// (to add a shadow and border radius to them)
		"isJPG": func(path string) bool {
			return strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg")
		},
		"length": func(a time.Duration) string {
			// format duration as "1h 2m 3s"
			return fmt.Sprintf("%dh %dm", int(a.Hours()), int(a.Minutes())%60)
		},
		// "either" returns true if any of the arguments is true-ish
		// (bool true, string not empty, int not 0, time.Duration not 0, []string not empty, []Reference not empty)
		// it's useful for checking if "either" of the fields is set in the template
		// to avoid rendering empty HTML tags (e.g. ".labels" paragraph)
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

				case []structs.Reference:
					if len(v) != 0 {
						return true
					}
				}
			}
			return false
		},
		"character": func(content structs.Content, characterName string) *structs.Character {
			for _, character := range content.Characters {
				if character.Name == characterName {
					return character
				}
			}
			return nil
		},
		// "dict" used to pass multiple key-value pairs to a template
		// (e.g. {{ template "something" dict "Key1" "value1" "Key2" "value2" }})
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("invalid dict call")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
		// "type" return a type of the content in singular form
		// (e.g. "person" for "People", "book" for "Books", etc.)
		// it used to add an additional context to reference link
		// when current page and the reference have the same name
		"type": func(c structs.Content) string {
			// get first part of the Source path
			// (e.g. "People" or "Book")
			root := pathType(c.Source)
			switch root {
			case "People":
				return "person"
			case "Books":
				return "book"
			case "Movies":
				return "movie"
			case "Games":
				return "game"
			default:
				return strings.ToLower(root)
			}
		},
		"series": series,
		"isLast": func(i, total int) bool {
			return i == total-1
		},
		"escape": func(s string) string {
			return strings.ReplaceAll(s, `'`, `\'`)
		},
		"missing": func() []structs.Missing {
			missing := map[string]map[string][]string{}

			g.muConnections.Lock()
			g.muContents.Lock()
			for to, from := range g.connections {
				if _, ok := g.contents[to]; !ok && len(from) > 1 {
					missing[to] = from
				}
			}
			g.muContents.Unlock()
			g.muConnections.Unlock()

			result := []structs.Missing{}
			for to, from := range missing {
				result = append(result, structs.Missing{To: to, From: from})
			}

			// sort by number of references (descending)
			// so that the most referenced files are on top
			sort.Slice(result, func(i, j int) bool {
				return len(result[i].From) > len(result[j].From)
			})

			return result
		},
		"title": func(b structs.Breadcrumbs) string {
			b = b[1:] // skip the first element (it's always "Home")
			if len(b) == 0 {
				return "Also, see"
			}

			var dirs []string
			for _, dir := range b {
				dirs = append(dirs, dir.Name)
			}

			slices.Reverse(dirs)

			return strings.Join(dirs, " \\ ")
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

	g.walkMediaDirectory()
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

	// Render Go templates
	if err := g.generateGoTemplates(); err != nil {
		return fmt.Errorf("generating go templates: %w", err)
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

			media, err := structs.ParseMediaFile(path)
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
	case ".gomd":
		g.addFile(file)
		return g.processGoMarkdownFile(file)
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

	var content structs.Content
	if err = yaml.Unmarshal(b, &content); err != nil {
		return fmt.Errorf("unmarshaling yaml: %w", err)
	}

	id := removeFileExtention(file)

	content.Source = file
	content.Image = g.getImageForPath(id)

	// add image to Characters
	for _, character := range content.Characters {
		character.Image = g.getImageForPath(filepath.Join(id, "Characters", character.Name))
		character.ActorImage = g.getImageForPath("People/" + character.Actor)
	}

	g.addContent(id, content)
	g.addConnections(id, content)

	return nil
}

func (g *Generator) processMarkdownFile(file string) error {
	b, err := os.ReadFile(filepath.Join(cfg.InfoDirectory, file))
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	htmlBody := markdown.ToHTML(b, nil, nil)

	// replace [ ] and [x] with checkboxes and break lines with <br> at the end of the line with checkbox
	// except for the first line
	htmlBody = bytes.ReplaceAll(htmlBody, []byte("[ ] "), []byte(`<br><input type="checkbox" disabled> `))
	htmlBody = bytes.ReplaceAll(htmlBody, []byte("[x] "), []byte(`<br><input type="checkbox" disabled checked> `))
	htmlBody = bytes.ReplaceAll(htmlBody, []byte("<p><br>"), []byte("<p>"))

	g.addContent(file, structs.Content{
		Source: file,
		HTML:   string(htmlBody),
	})
	return nil
}

func (g *Generator) processGoMarkdownFile(file string) error {
	b, err := os.ReadFile(filepath.Join(cfg.InfoDirectory, file))
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	g.addContent(file, structs.Content{
		Source: file,
		HTML:   string(b),
	})

	// conversion to HTML is done in generateGoTemplates()
	// after all the files are processed

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

func (g *Generator) addContent(id string, content structs.Content) {
	g.muContents.Lock()
	g.contents[id] = content
	g.muContents.Unlock()
}

func (g *Generator) addConnections(from string, content structs.Content) {
	for _, ref := range content.References {
		g.addConnection(from, ref.Path)
	}

	// Add connections for other less obvious references
	// (maybe it would be better to define these connections in some config.yml file,
	// or use Go struct field tags, but for now it's fine)

	for _, character := range content.Characters {
		if character.Actor != "" {
			g.addConnection(from, "People/"+character.Actor, "Actor", character.Name)
		}
		if character.Voice != "" {
			g.addConnection(from, "People/"+character.Voice, "Voice", character.Name)
		}
	}

	if content.Author != "" {
		g.addConnection(from, "People/"+content.Author, "Author")
	}

	for _, author := range content.Authors {
		g.addConnection(from, "People/"+author, "Author")
	}

	if content.Designer != "" {
		g.addConnection(from, "People/"+content.Designer, "Designer")
	}

	for _, writer := range content.Writers {
		g.addConnection(from, "People/"+writer, "Writer")
	}

	for _, director := range content.Directors {
		g.addConnection(from, "People/"+director, "Director")
	}

	for _, producer := range content.Producers {
		g.addConnection(from, "People/"+producer, "Producer")
	}

	for _, ref := range content.BasedOn {
		g.addConnection(from, ref, "Based on")
	}

	if content.Cinematography != "" {
		g.addConnection(from, "People/"+content.Cinematography, "Cinematography")
	}

	if content.Editor != "" {
		g.addConnection(from, "People/"+content.Editor, "Editor")
	}

	if content.Music != "" {
		g.addConnection(from, "People/"+content.Music, "Music")
	}

	for _, artist := range content.Artists {
		g.addConnection(from, "People/"+artist, "Artist")
	}

	if content.CoverArtist != "" {
		g.addConnection(from, "People/"+content.CoverArtist, "Cover Artist")
	}

	if content.Colorist != "" {
		g.addConnection(from, "People/"+content.Colorist, "Colorist")
	}

	if content.Series != "" {
		g.addConnection(from, series(content), "Series")
	}

	if content.Distributor != "" {
		g.addConnection(from, "Companies/"+content.Distributor, "Distributor")
	}

	if content.Publisher != "" {
		g.addConnection(from, "Companies/"+content.Publisher, "Publisher")
	}

	for _, production := range content.Production {
		g.addConnection(from, "Companies/"+production, "Production")
	}

	if content.Developers != "" {
		g.addConnection(from, "Companies/"+content.Developers, "Developers")
	}
}

func (g *Generator) addConnection(from, to string, info ...string) {
	g.muConnections.Lock()
	defer g.muConnections.Unlock()

	if _, ok := g.connections[to]; !ok {
		g.connections[to] = map[string][]string{}
	}

	if _, ok := g.connections[to][from]; !ok {
		g.connections[to][from] = info
		return
	}

	g.connections[to][from] = append(g.connections[to][from], info...)
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

func (g *Generator) addFile(path string) {
	dir := filepath.Dir(path)
	if dir == "." {
		dir = ""
	}

	g.muDir.Lock()
	g.dirContents[dir] = append(g.dirContents[dir], structs.File{
		Name:  removeFileExtention(filepath.Base(path)),
		Image: g.getImageForPath(removeFileExtention(path)),
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

	g.muDir.Lock()
	g.dirContents[dir] = append(g.dirContents[dir], structs.File{
		Name:     name,
		IsFolder: true,
	})
	g.muDir.Unlock()
}

func (g *Generator) getFilesForPath(path string) []structs.File {
	if files, ok := g.dirContents[path]; ok {
		sort.Sort(structs.ByYearDesk(files))

		// update Title if content has it
		for i, file := range files {
			if content := g.contents[filepath.Join(path, file.Name)]; content.Name != "" {
				files[i].Title = content.Name
			} else {
				files[i].Title = file.Name
			}
		}
		return files
	}

	return nil
}

func (g *Generator) generateContentTemplates() error {
	for path, content := range g.contents {
		id := removeFileExtention(path)
		path = id + ".html" // replace extension with .html

		// create directory
		if err := os.MkdirAll(filepath.Join(cfg.OutputDirectory, filepath.Dir(path)), 0o755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}

		f, err := os.Create(filepath.Join(cfg.OutputDirectory, path))
		if err != nil {
			return fmt.Errorf("creating file: %w", err)
		}

		panels, breadcrumbs := g.buildPanels(id, true)

		cnt := content

		if err := g.templates.ExecuteTemplate(
			f,
			"index.gohtml",
			struct {
				CurrentPath string
				Dir         string
				Breadcrumbs structs.Breadcrumbs
				Panels      structs.Panels
				Content     *structs.Content
				Timestamp   int64
			}{
				CurrentPath: id,
				Dir:         filepath.Dir(id),
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
			return fmt.Errorf("executing template for %q: %w", id, err)
		}

		if err := f.Close(); err != nil {
			return fmt.Errorf("closing file: %w", err)
		}
	}

	return nil
}

func (g *Generator) generateGoTemplates() error {
	for path, content := range g.contents {
		if filepath.Ext(path) != ".gomd" {
			continue
		}

		// render Go template
		t, err := g.templates.New("").Funcs(g.fm()).Parse(content.HTML)
		if err != nil {
			return fmt.Errorf("parsing template: %w", err)
		}

		var buf bytes.Buffer
		if err := t.Execute(&buf, nil); err != nil {
			return fmt.Errorf("executing template: %w", err)
		}

		htmlBody := markdown.ToHTML(buf.Bytes(), nil, nil)
		content.HTML = string(htmlBody)

		g.contents[path] = content
	}

	return nil
}

func (g *Generator) getImageForPath(path string) *structs.Media {
	dir := filepath.Dir(path)
	if dir == "." {
		dir = ""
	}

	base := filepath.Base(path)

	dirContent, ok := g.mediaDirContents[dir]
	if !ok {
		return nil
	}

	for _, media := range dirContent {
		if removeFileExtention(media.Path) == base {
			return &media
		}
	}

	return nil
}

func (g *Generator) generateIndexes() error {
	for dir, files := range g.dirContents {
		sort.Sort(structs.ByNameFolderOnTop(files))

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
				Breadcrumbs []structs.Dir
				Panels      structs.Panels
				Content     *structs.Content
				Timestamp   int64
				Connections structs.Connections
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

func (g *Generator) buildPanels(path string, isFile bool) (structs.Panels, structs.Breadcrumbs) {
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
			dir = "Home"
		}

		breadcrumbs = append(breadcrumbs, structs.Dir{
			Name: dir,
			Path: cumulativePath,
		})

		// if it's a file, don't add last panel
		// (it is the file itself, which will be rendered)
		if isFile && cumulativePath == path {
			break
		}

		panels = append(panels, structs.Panel{
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

func pathType(path string) string {
	return strings.Split(path, string(filepath.Separator))[0]
}

// series generates path to a series page
// for Movies: /Movies/Series/<Series name>
// for Video Games: /Games/Video/Series/<Series name>
// Since most of the content arranged in a folders by year,
// series page is 2 levels up from the current page.
func series(c structs.Content) string {
	return filepath.Join(
		filepath.Dir(filepath.Dir(c.Source)),
		"Series",
		c.Series,
	)
}
