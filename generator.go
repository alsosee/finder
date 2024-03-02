package main

import (
	"bytes"
	"errors"
	"fmt"
	"hash/crc32"
	"html"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
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

	awardPages   []string
	muAwardPages sync.Mutex

	// awardsMissingContent used to temporary hold awards
	// that are for content that is not yet added.
	awardsMissingContent   map[string][]structs.Award
	muAwardsMissingContent sync.Mutex

	// chainPages used to keep track of next/prev pages in a series.
	chainPages   map[string]map[bool]string // from -> true(next)/false(prev) -> reference
	muChainPages sync.Mutex
}

// NewGenerator creates a new Generator.
func NewGenerator() (*Generator, error) {
	ignore, err := processIgnoreFile(cfg.IgnoreFile)
	if err != nil {
		return nil, fmt.Errorf("processing ignore file: %w", err)
	}

	return &Generator{
		ignore:               ignore,
		contents:             structs.Contents{},
		dirContents:          map[string][]structs.File{},
		connections:          structs.Connections{},
		mediaDirContents:     map[string][]structs.Media{},
		chainPages:           map[string]map[bool]string{},
		awardsMissingContent: map[string][]structs.Award{},
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
		"base":      filepath.Base,
		"hasPrefix": strings.HasPrefix,
		"strjoin":   strings.Join,
		"sum": func(ints ...int) int {
			var sum int
			for _, i := range ints {
				sum += i
			}
			return sum
		},
		"in": in,
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
		// "connections" returns a list of connections for a given file id
		"connections": func(id string) map[string][]string {
			g.muConnections.Lock()
			defer g.muConnections.Unlock()

			if m, ok := g.connections[id]; ok {
				return m
			}
			return nil
		},
		"prev": func(id string) string {
			g.muChainPages.Lock()
			defer g.muChainPages.Unlock()

			if m, ok := g.chainPages[id]; ok {
				if prev, ok := m[false]; ok {
					return prev
				}
			}
			return ""
		},
		"next": func(id string) string {
			g.muChainPages.Lock()
			defer g.muChainPages.Unlock()

			if m, ok := g.chainPages[id]; ok {
				if next, ok := m[true]; ok {
					return next
				}
			}
			return ""
		},
		"crc32": crc32sum,
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
		"thumbStylePx": func(media structs.Media, max float64, opt ...string) string {
			if media.ThumbPath == "" {
				return ""
			}

			var (
				backgroundWidth  = float64(media.ThumbTotalWidth) * max / float64(media.ThumbWidth)
				backgroundHeight = float64(media.ThumbTotalHeight) * max / float64(media.ThumbWidth)
				positionX        = float64(media.ThumbXOffset) * max / float64(media.ThumbWidth)
				positionY        = float64(media.ThumbYOffset) * max / float64(media.ThumbWidth)
				width            = max
				height           = float64(media.ThumbHeight) * max / float64(media.ThumbWidth)
			)

			p := ""
			if len(opt) > 0 {
				p = opt[0]
			}

			if media.Height > media.Width {
				backgroundWidth = float64(media.ThumbTotalWidth) * max / float64(media.ThumbHeight)
				backgroundHeight = float64(media.ThumbTotalHeight) * max / float64(media.ThumbHeight)
				positionX = float64(media.ThumbXOffset) * max / float64(media.ThumbHeight)
				positionY = float64(media.ThumbYOffset) * max / float64(media.ThumbHeight)
				width = float64(media.ThumbWidth) * max / float64(media.ThumbHeight)
				height = max
			}

			marginLeft := (max - width) / 2
			marginRight := max - width - marginLeft
			marginTop := (max - height) / 2
			marginBottom := max - height - marginTop

			style := fmt.Sprintf(
				"%sbackground-size: %.2fpx %.2fpx; %swidth: %.2fpx; %sheight: %.2fpx",
				p, backgroundWidth, backgroundHeight,
				p, width,
				p, height,
			)

			if marginLeft != 0 || marginRight != 0 {
				style += fmt.Sprintf("; %scomp-margin-left: %.2fpx; %scomp-margin-right: %.2fpx", p, marginLeft, p, marginRight)
			}

			if marginTop != 0 || marginBottom != 0 {
				style += fmt.Sprintf("; %scomp-margin-top: %.2fpx; %scomp-margin-bottom: %.2fpx", p, marginTop, p, marginBottom)
			}

			if positionX != 0 || positionY != 0 {
				style += fmt.Sprintf("; %sbackground-position: -%.2fpx -%.2fpx", p, positionX, positionY)
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
			width := float64(media.ThumbTotalWidth) * 100 / float64(media.ThumbWidth)
			height := float64(media.ThumbTotalHeight) * 100 / float64(media.ThumbHeight)

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
					"%sbackground-size: %.2f%% %.2f%%; %saspect-ratio: %d/%d;",
					p, width, height,
					p, arX, arY,
				)
			}

			return fmt.Sprintf(
				"%sbackground-size: %.2f%% %.2f%%; %sbackground-position: %.2f%% %.2f%%; %saspect-ratio: %d/%d;",
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
		"length": length,
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

				case []structs.Award:
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
		"characterByActor": func(content *structs.Content, characterName string) *structs.Character {
			// this function return a single character by actor or voice name
			// todo: support multiple characters with the same actor/voice
			if content == nil {
				return nil
			}
			for _, character := range content.Characters {
				if character.Actor == characterName {
					return character
				}
				if character.Voice == characterName {
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
		"htmlEscape": html.EscapeString,
		"missing":    g.missing,
		"missingAwardsLen": func(id string) int {
			g.muAwardsMissingContent.Lock()
			defer g.muAwardsMissingContent.Unlock()
			return len(g.awardsMissingContent[id])
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
		"awardYear":     awardYear,
		"prefix":        prefix,
		"chooseColumns": chooseColumns,
		"column":        column,
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

	g.addAwards()

	// Generate missing files
	if err := g.generateMissing(); err != nil {
		return fmt.Errorf("generating missing: %w", err)
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
			outPath := filepath.Join(cfg.OutputDirectory, relPath)

			if strings.HasSuffix(path, ".gojs") {
				outPath = strings.TrimSuffix(outPath, ".gojs") + ".js"
				log.Printf("Processing GoJS file %q to %q", path, outPath)
				return g.processGoJSFile(path, outPath)
			}

			return copyFile(path, outPath)
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

	id := removeFileExtention(file)

	g.addContent(
		id,
		structs.Content{
			ID:     id,
			Source: file,
			HTML:   string(htmlBody),
		},
	)
	return nil
}

func (g *Generator) processGoMarkdownFile(file string) error {
	b, err := os.ReadFile(filepath.Join(cfg.InfoDirectory, file))
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	id := removeFileExtention(file)
	g.addContent(id, structs.Content{
		ID:     id,
		Source: file,
		HTML:   string(b),
	})

	// conversion to HTML is done in generateGoTemplates()
	// after all the files are processed

	return nil
}

func (g *Generator) processGoJSFile(src, out string) error {
	// treat file as a Go template
	b, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	t, err := template.New("").Funcs(g.fm()).Parse(string(b))
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	outFile, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer outFile.Close()

	if err = t.Execute(outFile, nil); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

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
		g.addConnectionSingle(from, "People", character.Actor, "Actor", character.Name)
		g.addConnectionSingle(from, "People", character.Voice, "Voice", character.Name)
	}

	g.addConnectionList(from, "People", content.Authors, "Author")
	g.addConnectionList(from, "People", content.Writers, "Writer")
	g.addConnectionList(from, "People", content.Directors, "Director")
	g.addConnectionList(from, "People", content.Creators, "Creator")
	g.addConnectionList(from, "People", content.Producers, "Producer")
	g.addConnectionList(from, "People", content.Editors, "Editor")
	g.addConnectionList(from, "People", content.Artists, "Artist")
	g.addConnectionList(from, "People", content.Screenplay, "Screenplay")
	g.addConnectionList(from, "People", content.StoryBy, "Story")
	g.addConnectionList(from, "People", content.DialoguesBy, "Dialogues")
	g.addConnectionList(from, "People", content.Composers, "Composer")
	g.addConnectionList(from, "People", content.Hosts, "Host")
	g.addConnectionList(from, "People", content.Guests, "Guest")
	g.addConnectionList(from, "Companies", content.Distributors, "Distributor")
	g.addConnectionList(from, "Companies", content.Publishers, "Publisher")
	g.addConnectionList(from, "Companies", content.Production, "Production")
	g.addConnectionList(from, "", content.BasedOn, "Based on")

	g.addConnectionSingle(from, "People", content.Designer, "Designer")
	g.addConnectionSingle(from, "People", content.Cinematography, "Cinematography")
	g.addConnectionSingle(from, "People", content.Music, "Music")
	g.addConnectionSingle(from, "People", content.CoverArtist, "Cover artist")
	g.addConnectionSingle(from, "People", content.Colorist, "Colorist")
	g.addConnectionSingle(from, "Companies", content.Network, "Network")
	g.addConnectionSingle(from, "Companies", content.Developers, "Developers")
	g.addConnectionSingle(from, "", content.RemakeOf, "Remake")

	if content.Series != "" {
		g.addConnectionSingle(from, "", series(content), "Series")
	}

	if content.Previous != "" {
		g.addPrevious(from, content.Previous)
	}

	// Prepare for adding Awards
	if len(content.Categories) > 0 {
		g.addAwardPage(from)
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

func (g *Generator) addConnectionSingle(from, prefix string, item string, info ...string) {
	if item == "" {
		return
	}

	ref := item
	if prefix != "" {
		ref = prefix + "/" + item
	}

	g.addConnection(from, ref, info...)
}

func (g *Generator) addConnectionList(from, prefix string, list []string, info ...string) {
	for _, item := range list {
		ref := item
		if prefix != "" {
			ref = prefix + "/" + item
		}
		g.addConnection(from, ref, info...)
	}
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

				// extra fields to use in list view
				files[i].Columns.Add("Length", length(content.Length))
				files[i].Columns.Add("Directors", strings.Join(content.Directors, ", "))
				files[i].Columns.Add("Writers", strings.Join(content.Writers, ", "))
				files[i].Columns.Add("Distributors", strings.Join(content.Distributors, ", "))
				files[i].Columns.Add("Rating", content.Rating)
				files[i].Columns.Add("Released", content.Released)
				files[i].Columns.Add("Network", content.Network)
				files[i].Columns.Add("Creators", strings.Join(content.Creators, ", "))
				files[i].Columns.Add("Authors", strings.Join(content.Authors, ", "))
				files[i].Columns.Add("Hosts", strings.Join(content.Hosts, ", "))
				files[i].Columns.Add("Publishers", strings.Join(content.Publishers, ", "))
				files[i].Columns.Add("Screenplay", strings.Join(content.Screenplay, ", "))
				files[i].Columns.Add("StoryBy", strings.Join(content.StoryBy, ", "))
				files[i].Columns.Add("DialoguesBy", strings.Join(content.DialoguesBy, ", "))
				files[i].Columns.Add("Born", content.DOB)
				files[i].Columns.Add("Died", content.DOD)
			} else {
				files[i].Title = file.Name
			}
		}
		return files
	}

	return nil
}

func (g *Generator) generateContentTemplates() error {
	for id, content := range g.contents {
		path := filepath.Join(cfg.OutputDirectory, id+".html")
		panels, breadcrumbs := g.buildPanels(id, true)
		cnt := content

		err := g.executeTemplate(path, structs.PageData{
			CurrentPath: id,
			Dir:         filepath.Dir(id),
			Breadcrumbs: breadcrumbs,
			Panels:      panels,
			Content:     &cnt,
			Timestamp:   time.Now().Unix(),
		})
		if err != nil {
			return fmt.Errorf("executing template for %q: %w", id, err)
		}
	}

	return nil
}

func (g *Generator) generateGoTemplates() error {
	for id, content := range g.contents {
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
			return fmt.Errorf("executing template: %w", err)
		}

		htmlBody := markdown.ToHTML(buf.Bytes(), nil, nil)
		content.HTML = string(htmlBody)

		g.contents[id] = content
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
		panels, breadcrumbs := g.buildPanels(dir, false)

		err := g.executeTemplate(path, structs.PageData{
			CurrentPath: dir,
			Breadcrumbs: breadcrumbs,
			Panels:      panels,
			Content:     nil,
			Timestamp:   time.Now().Unix(),
			Connections: nil,
		})
		if err != nil {
			return fmt.Errorf("executing template for %q: %w", dir, err)
		}
	}

	return nil
}

func (g *Generator) generateMissing() error {
	missing := g.missing()

	// first, add files to all panels
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

		g.muDir.Lock()
		g.dirContents[filepath.Dir(id)] = append(g.dirContents[filepath.Dir(id)], file)
		g.muDir.Unlock()
	}

	// render all missing files
	for _, m := range missing {
		if len(m.From)+len(m.Awards) < 2 {
			continue
		}

		id := m.To
		image := g.getImageForPath(id)

		cnt := structs.Content{
			Name:   filepath.Base(id),
			Image:  image,
			Awards: m.Awards,
		}

		path := filepath.Join(cfg.OutputDirectory, id+".html")
		panels, breadcrumbs := g.buildPanels(id, true)

		err := g.executeTemplate(path, structs.PageData{
			CurrentPath: id,
			Dir:         filepath.Dir(id),
			Breadcrumbs: breadcrumbs,
			Panels:      panels,
			Content:     &cnt,
			Timestamp:   time.Now().Unix(),
		})
		if err != nil {
			return fmt.Errorf("executing template for %q: %w", id, err)
		}
	}

	return nil
}

func (g *Generator) missing() []structs.Missing {
	missing := map[string]map[string][]string{}

	// add missing content referenced by other files

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
		result = append(
			result,
			structs.Missing{
				To:     to,
				From:   from,
				Awards: g.awardsMissingContent[to],
			},
		)
	}

	// add missing content that got awards

	g.muAwardPages.Lock()
	for to, awards := range g.awardsMissingContent {
		if _, ok := g.contents[to]; !ok && len(awards) > 1 {
			result = append(
				result,
				structs.Missing{
					To:     to,
					From:   nil,
					Awards: awards,
				},
			)
		}
	}
	g.muAwardPages.Unlock()

	// Sort by number of references (descending),
	// so that the most referenced files are on top.
	// If the number of references is the same, sort by name.
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

func (g *Generator) executeTemplate(path string, pageData structs.PageData) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}

	if err := g.templates.ExecuteTemplate(f, "index.gohtml", pageData); err != nil {
		err2 := f.Close()
		if err2 != nil {
			err = errors.Join(err, err2)
		}
		return fmt.Errorf("executing template: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("closing file: %w", err)
	}

	return nil
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

			id := category.Winner.Reference
			if id == "" {
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
			if awadredContent, ok = g.contents[id]; !ok {
				g.muAwardsMissingContent.Lock()
				g.awardsMissingContent[id] = append(g.awardsMissingContent[id], award)
				g.muAwardsMissingContent.Unlock()
				continue
			}

			switch true {
			case category.Winner.Actor != "":
				// loop through all characters and find actor with the same name
				var found bool
				for _, character := range awadredContent.Characters {
					if character.Actor == category.Winner.Actor {
						character.Awards = append(character.Awards, award)
						found = true
						break
					}
				}
				if !found {
					log.Printf("No character found for %q", category.Winner.Actor)
				}
			case category.Winner.Cinematography != "":
				awadredContent.CinematographyAwards = append(awadredContent.CinematographyAwards, award)
			case category.Winner.Music != "":
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

			g.contents[id] = awadredContent
		}

		g.contents[awardPage] = content
	}
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

var crc32cache = map[string]string{}

// crc32 calculates CRC32 checksum for a file.
// It's used to add a get parameter to a static file URL,
// so that when the file is updated, the browser will download the new version.
func crc32sum(path string) string {
	if crc, ok := crc32cache[path]; ok {
		return crc
	}

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

	crc32cache[path] = fmt.Sprintf("%x", hash.Sum32())

	return crc32cache[path]
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

func in(needle string, slice ...string) bool {
	for _, s := range slice {
		if needle == s {
			return true
		}
	}
	return false
}

func length(a time.Duration) string {
	if a == 0 {
		return ""
	}

	// format duration as "1h 2m"
	return fmt.Sprintf("%dh %dm", int(a.Hours()), int(a.Minutes())%60)
}

func awardYear(c structs.Content) string {
	yearSt := removeFileExtention(filepath.Base(c.Source))

	if strings.Contains(c.Source, "/Oscar/") || strings.Contains(c.Source, "/BAFTA/") {
		// decrease year by 1 for Oscar awards
		// since they are awarded for the previous year
		// (e.g. 2023 Oscar awards are for movies released in 2022)
		// TODO: make it more generic
		year, err := strconv.Atoi(yearSt)
		if err != nil {
			log.Printf("Error parsing year from %q: %v", c.Source, err)
			return ""
		}
		yearSt = strconv.Itoa(year - 1)
	}

	return yearSt
}

// prefix returns a path prefix to a content referenced by the given content.
// For example, "Movies/Awards/Oscar/2023.yml" will return "Movies/2022"
func prefix(c structs.Content, year string) string {
	contentType := pathType(c.Source)

	if contentType == "Games" {
		contentType = "Games/Video"
	}

	return contentType + "/" + year
}

func chooseColumns(files []structs.File) []string {
	var total int
	columns := map[string]int{}
	for _, file := range files {
		if file.IsMissing {
			continue
		}
		total++
		for key := range file.Columns {
			columns[key]++
		}
	}

	// choose columns that are present in > half of all files
	chosenColumns := []string{}
	for key, count := range columns {
		if count > total/2 || key == "Died" {
			chosenColumns = append(chosenColumns, key)
		}
	}

	sort.Strings(chosenColumns)

	return chosenColumns
}

func column(file structs.File, column string) string {
	return file.Columns.Get(column)
}
