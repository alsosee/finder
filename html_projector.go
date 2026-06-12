package main

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"html"
	"io"
	"io/fs"
	"log"
	"net/url"
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
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"

	"github.com/alsosee/finder/structs"
)

var errExecutingTemplate = errors.New("error executing template")

var caser = cases.Title(language.English, cases.NoLower)

//go:embed functions/*
var functionsFS embed.FS

// HTMLProjector renders the static website from a build graph.
type HTMLProjector struct {
	templates *template.Template

	config       structs.Config
	infoDir      string
	staticDir    string
	templatesDir string
	outputDir    string

	renderedPanelsCache map[string]string
	graph               *BuildGraph
	contents            structs.Contents

	muRenderedPanels sync.Mutex // protects writes to renderedPanelsCache
	crc32cache       map[string]string
	crc32mu          sync.Mutex
}

func NewHTMLProjector(config structs.Config, infoDir, staticDir, templatesDir, outputDir string) *HTMLProjector {
	return &HTMLProjector{
		config:              config,
		infoDir:             infoDir,
		staticDir:           staticDir,
		templatesDir:        templatesDir,
		outputDir:           outputDir,
		renderedPanelsCache: map[string]string{},
		crc32cache:          map[string]string{},
	}
}

func (g *HTMLProjector) fm() template.FuncMap {
	return template.FuncMap{
		"config":       func() structs.Config { return g.config },
		"join":         filepath.Join,
		"dir":          filepath.Dir,
		"base":         filepath.Base,
		"hasPrefix":    strings.HasPrefix,
		"trimPrefix":   strings.TrimPrefix,
		"strjoin":      strings.Join,
		"lower":        strings.ToLower,
		"isPerson":     structs.IsPerson,
		"personPrefix": structs.PersonPrefix,
		"debugPrint": func(v interface{}) string {
			var buf bytes.Buffer
			if err := yaml.NewEncoder(&buf).Encode(v); err != nil {
				return fmt.Sprintf("error encoding: %v", err)
			}
			return buf.String()
		},
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
		"content": func(path, caller string) *structs.Content {
			if g.contents == nil {
				return nil
			}
			path = g.canonicalContentPath(path)
			if c, ok := g.contents[path]; ok {
				return &c
			}
			return nil
		},
		// "connections" returns a list of connections for a given file path (no extension).
		"connections": func(path string) map[string][]structs.Connection {
			if g.graph == nil {
				return nil
			}
			if m, ok := g.graph.Connections[path]; ok {
				return m
			}

			return nil
		},
		"prev": func(id string) string {
			if g.graph == nil {
				return ""
			}
			if m, ok := g.graph.ChainPages[id]; ok {
				if prev, ok := m[false]; ok {
					return prev
				}
			}
			return ""
		},
		"next": func(id string) string {
			if g.graph == nil {
				return ""
			}
			if m, ok := g.graph.ChainPages[id]; ok {
				if next, ok := m[true]; ok {
					return next
				}
			}
			return ""
		},
		"crc32": g.crc32sum,
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

			// round down width to ceil number to avoid rounding errors
			// that can cause image to have 1px of the next image on the right
			width = float64(int(width))

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
		// "isPNG" currenty not used
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

			for _, episode := range content.Episodes {
				for _, character := range episode.Characters {
					if character.Name == characterName {
						return character
					}
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
		"type": func(c structs.Content) string {
			return c.Type()
		},
		"contentFieldName": func(field string) string {
			return structs.ContentFieldName(field)
		},
		"series": series,
		"isLast": func(i, total int) bool {
			return i == total-1
		},
		"escape": func(s string) string {
			return strings.ReplaceAll(s, `'`, `\'`)
		},
		"htmlEscape": html.EscapeString,
		"value":      newFileValue,
		"missing": func() []structs.Missing {
			if g.graph == nil {
				return nil
			}
			return g.graph.Missing
		},
		"missingAwardsLen": func(id string) int {
			if g.graph == nil {
				return 0
			}
			return len(g.graph.AwardsMissingContent[id])
		},
		"image":       g.getImageForPath,
		"formatTitle": g.formatTitle,
		"title":       caser.String,
		"awardYear":   awardYear,
		"prefix":      prefix,
		"columns": func() []structs.Column {
			return structs.ColumnsList
		},
		"column":        column,
		"chooseColumns": chooseColumns,
		"rootTypes":     func() map[string]string { return structs.RootTypes },
		"renderPanel":   g.renderPanel,
		"label": func(label string, list []string) string {
			if len(list) == 1 && strings.HasSuffix(label, "s") {
				return label[:len(label)-1]
			}
			return label
		},
		"fallback": func(args ...string) string {
			for _, arg := range args {
				if arg != "" {
					return arg
				}
			}
			return ""
		},
		"groupConnections": groupConnections,
		"escapeFileName":   structs.EscapeFileName,
		"canonicalPath":    g.canonicalContentPath,
	}
}

func (g *HTMLProjector) canonicalContentPath(path string) string {
	if g.contents == nil {
		return path
	}
	if _, ok := g.contents[path]; ok {
		return path
	}

	withoutColons := strings.ReplaceAll(path, ":", "")
	if withoutColons != path {
		if _, ok := g.contents[withoutColons]; ok {
			return withoutColons
		}
	}

	return path
}

func (g *HTMLProjector) Name() string {
	return "html"
}

func (g *HTMLProjector) Run(graph *BuildGraph) error {
	g.graph = graph
	g.contents = cloneContents(graph.Contents)

	t, err := template.New("").Funcs(g.fm()).ParseGlob(g.templatesDir + "/*")
	if err != nil {
		return fmt.Errorf("parsing templates: %w", err)
	}
	g.templates = t

	g.copyStaticFiles()
	g.copyFunctionsFiles()
	for _, file := range graph.PassthroughFiles {
		if err := g.copyFileAsIs(file); err != nil {
			return fmt.Errorf("copying passthrough file %q: %w", file, err)
		}
	}

	// Render Go templates
	if err := g.generateGoTemplates(); err != nil {
		return fmt.Errorf("generating go templates: %w", err)
	}

	if err := g.generateMissing(); err != nil {
		return fmt.Errorf("rendering missing: %w", err)
	}

	// Generate file templates
	if err := g.generateContentTemplates(); err != nil {
		return fmt.Errorf("generating content templates: %w", err)
	}

	// Generate index for each directory
	if err := g.generateIndexes(); err != nil {
		return fmt.Errorf("generating indexes: %w", err)
	}

	// Generate 404 page
	if err := g.generate404(); err != nil {
		return fmt.Errorf("generating 404 page: %w", err)
	}

	return nil
}

func (g *HTMLProjector) copyStaticFiles() {
	if g.staticDir == "" {
		log.Printf("No static files directory specified, skipping")
		return
	}

	log.Printf("Copying static files from %q to %q", g.staticDir, g.outputDir)

	if err := os.MkdirAll(g.outputDir, 0o755); err != nil {
		log.Fatalf("Error creating output directory %q: %v", g.outputDir, err)
	}

	err := filepath.Walk(
		g.staticDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return os.MkdirAll(filepath.Join(g.outputDir, strings.TrimPrefix(path, g.staticDir)), 0o755)
			}

			relPath := strings.TrimPrefix(path, g.staticDir+string(filepath.Separator))
			outPath := filepath.Join(g.outputDir, relPath)

			if strings.HasSuffix(path, ".gojs") {
				outPath = strings.TrimSuffix(outPath, ".gojs") + ".js"
				log.Printf("Processing GoJS file %q to %q", path, outPath)
				return g.processGoJSFile(path, outPath)
			}

			return copyFile(path, outPath)
		},
	)
	if err != nil {
		log.Fatalf("Error walking static directory %q: %v", g.staticDir, err)
	}

	log.Printf("Done copying static files from %q to %q", g.staticDir, g.outputDir)
}

func (g *HTMLProjector) copyFunctionsFiles() {
	log.Printf("Copying functions files")

	// check if functions directory exists, if it does – exit
	if _, err := os.Stat("functions"); err == nil {
		log.Printf("Functions directory already exists, skipping")
		return
	}

	// unlike static files, functions directory has to be the directory where app is running
	// so it is not written to the HTML output directory
	if err := os.MkdirAll("functions", 0o755); err != nil {
		log.Fatalf("Error creating functions directory: %v", err)
	}

	// copy embedded functionsFS files to the functions directory
	err := fs.WalkDir(functionsFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return os.MkdirAll(path, 0o755)
		}

		outPath := filepath.Join(path)
		outFile, err := os.Create(outPath)
		if err != nil {
			return fmt.Errorf("creating file %q: %w", outPath, err)
		}
		defer func() { _ = outFile.Close() }()

		inFile, err := functionsFS.Open(path)
		if err != nil {
			return fmt.Errorf("opening file %q: %w", path, err)
		}

		_, err = io.Copy(outFile, inFile)
		if err != nil {
			return fmt.Errorf("copying file %q to %q: %w", path, outPath, err)
		}

		return nil
	})
	if err != nil {
		log.Fatalf("Error walking functions directory: %v", err)
	}
}

func (g *HTMLProjector) processGoJSFile(src, out string) error {
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
	defer func() { _ = outFile.Close() }()

	if err = t.Execute(outFile, nil); err != nil {
		return fmt.Errorf("%w for %q: %w", errExecutingTemplate, src, err)
	}

	return nil
}

func (g *HTMLProjector) copyFileAsIs(file string) error {
	return copyFile(
		filepath.Join(g.infoDir, file),
		filepath.Join(g.outputDir, file),
	)
}

// renderPanel renders a panel and caches the result
// it is an optimisation to not render the same panel multiple times.
// drawback is that some content still has to be changed dynamically,
// that is why markInPathLinks function is used
func (g *HTMLProjector) renderPanel(panel structs.Panel, index int, isLast bool, path string) string {
	g.muRenderedPanels.Lock()
	defer g.muRenderedPanels.Unlock()

	if rendered, ok := g.renderedPanelsCache[panel.Dir]; ok {
		return markInPathLinks(rendered, panel, path, isLast)
	}

	rendered, err := g.renderPanelImpl(panel, index)
	if err != nil {
		log.Fatalf("rendering panel %q: %v", path, err)
	}

	g.renderedPanelsCache[panel.Dir] = rendered

	return markInPathLinks(rendered, panel, path, isLast)
}

func (g *HTMLProjector) renderPanelImpl(panel structs.Panel, index int) (string, error) {
	var b bytes.Buffer
	err := g.templates.Lookup("panel.gohtml").Execute(&b, struct {
		Panel structs.Panel
		Index int
	}{
		Panel: panel,
		Index: index,
	})

	if err != nil {
		return "", fmt.Errorf("executing panel template: %w", err)
	}

	return b.String(), nil
}

func markInPathLinks(s string, panel structs.Panel, path string, isLast bool) string {
	if isLast {
		s = strings.Replace(
			s,
			`onclick="panelClick(event)"`,
			`onclick="panelClick(event)" id="_"`,
			1,
		)
	}

	for _, file := range panel.Files {
		filePath := filepath.Join(panel.Dir, file.Name)
		if file.IsFolder && strings.HasPrefix(path, filePath) {
			// add "in-path" class to folder link
			return strings.Replace(
				s,
				`" href="/`+filePath+`/"`,
				` in-path" href="/`+filePath+`/"`,
				1,
			)
		}

		if !file.IsFolder && path == filePath {
			// add "in-path" and "active" classes to file link
			return strings.Replace(
				s,
				`" href="/`+filePath+`"`,
				` active in-path" href="/`+filePath+`"`,
				1,
			)
		}
	}

	return s
}

func (g *HTMLProjector) generateContentTemplates() error {
	for id, content := range g.contents {
		path := filepath.Join(g.outputDir, id+".html")
		panels, breadcrumbs := g.graph.Panels(id, true)
		cnt := content

		err := g.executeTemplate(path, structs.PageData{
			CurrentPath:    id,
			Dir:            filepath.Dir(id),
			Breadcrumbs:    breadcrumbs,
			Panels:         panels,
			Content:        &cnt,
			Timestamp:      time.Now().Unix(),
			OpenGraphImage: g.graph.OpenGraphImage(id),
		}, "index.gohtml")
		if err != nil {
			return fmt.Errorf("%w for %q: %w", errExecutingTemplate, id, err)
		}
	}

	return nil
}

func (g *HTMLProjector) generateGoTemplates() error {
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

func (g *HTMLProjector) getImageForPath(path string) *structs.Media {
	if g.graph == nil {
		return nil
	}
	return g.graph.Media.ImageForPath(path)
}

func (g *HTMLProjector) formatTitle(b structs.Breadcrumbs) string {
	b = b[1:] // skip the first element (it's always "Home")
	if len(b) == 0 {
		return g.config.Title
	}

	var dirs []string
	for _, dir := range b {
		dirs = append(dirs, dir.Name)
	}

	slices.Reverse(dirs)

	return strings.Join(dirs, " \\ ")
}

func (g *HTMLProjector) generateIndexes() error {
	for dir := range g.graph.DirContents {
		path := filepath.Join(g.outputDir, dir, "index.html")
		panels, breadcrumbs := g.graph.Panels(dir, false)

		err := g.executeTemplate(path, structs.PageData{
			CurrentPath: dir,
			Breadcrumbs: breadcrumbs,
			Panels:      panels,
			Content:     nil,
			Timestamp:   time.Now().Unix(),
			Connections: nil,
		}, "index.gohtml")
		if err != nil {
			return fmt.Errorf("%w for %q: %w", errExecutingTemplate, dir, err)
		}
	}

	return nil
}

func (g *HTMLProjector) generateMissing() error {
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
	for _, missingPage := range g.graph.MissingPages {
		id := missingPage.ID
		panels, breadcrumbs := g.graph.Panels(id, true)
		pagesDataChan <- structs.PageData{
			OutputPath:     filepath.Join(g.outputDir, id+".html"),
			CurrentPath:    id,
			Dir:            filepath.Dir(id),
			Breadcrumbs:    breadcrumbs,
			Panels:         panels,
			Content:        missingPage.Content,
			Timestamp:      time.Now().Unix(),
			OpenGraphImage: g.graph.OpenGraphImage(id),
		}
	}

	close(pagesDataChan)

	wg.Wait()
	return nil
}

func (g *HTMLProjector) generate404() error {
	outputPath := filepath.Join(g.outputDir, "404.html")

	return g.executeTemplate(outputPath, structs.PageData{
		CurrentPath: "404",
		Dir:         "",
		Breadcrumbs: structs.Breadcrumbs{
			{Name: g.config.HomeLabel},
			{Name: g.config.NotFoundHeader, IsCurrent: true},
		},
		Panels:    nil, // no panels on 404 page
		Timestamp: time.Now().Unix(),
	}, "404.gohtml")
}

func (g *HTMLProjector) executeTemplate(path string, pageData structs.PageData, templateName string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}

	if err := g.templates.ExecuteTemplate(f, templateName, pageData); err != nil {
		err2 := f.Close()
		if err2 != nil {
			err = errors.Join(err, err2)
		}
		return fmt.Errorf("executing template for %q: %w", path, err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("closing file: %w", err)
	}

	return nil
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

	if a < time.Hour {
		// format duration as "2m"
		return fmt.Sprintf("%dm", int(a.Minutes()))
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
		columnInfo := lookupColumnInfo(key)
		if count > total/2 || (columnInfo != nil && columnInfo.AlwaysShow) {
			chosenColumns = append(chosenColumns, key)
		}
	}

	sort.Strings(chosenColumns)

	return chosenColumns
}

func lookupColumnInfo(columnTitle string) *structs.Column {
	for _, column := range structs.ColumnsList {
		if column.Title == columnTitle || strings.ToLower(columnTitle) == column.Name {
			return &column
		}
	}

	log.Printf("Column %s not found", columnTitle)
	return nil
}

func column(file structs.File, column string) string {
	return file.Columns.Get(column)
}

func groupConnections(connections map[string][]structs.Connection) []structs.ConnectionLine {
	result := []structs.ConnectionLine{}

	// - Groups:
	//   - Label1: Director
	// 	 - Label2: Actor
	//     InfoGroup:
	//     - Role1
	//     - Role2
	//   Parents:
	//   - Parent1
	//   - Parent2

	for from, conns := range connections {
		line := structs.ConnectionLine{
			From:    from,
			Groups:  []structs.ConnectionLineItem{},
			Parents: []string{},
		}

		labelGroups := map[string]structs.ConnectionLineItem{}

		// group by label, combine info
		for _, conn := range conns {
			group, exists := labelGroups[conn.Label]
			if !exists {
				group = structs.ConnectionLineItem{
					Label: conn.Label,
				}
			}

			if conn.Info != "" && !slices.Contains(group.Info, conn.Info) {
				group.Info = append(group.Info, conn.Info)
			}
			if conn.Parent != "" && !slices.Contains(line.Parents, conn.Parent) {
				line.Parents = append(line.Parents, conn.Parent)
			}

			labelGroups[conn.Label] = group
		}

		// flatten the map to a slice
		list := make([]structs.ConnectionLineItem, 0, len(labelGroups))
		for _, conn := range labelGroups {
			list = append(list, conn)
		}

		// sort by length of info, or alphabetically if lengths are equal
		sort.Slice(list, func(i, j int) bool {
			if len(list[i].Info) != len(list[j].Info) {
				return len(list[i].Info) < len(list[j].Info)
			}

			return list[i].Label < list[j].Label
		})

		// lowercase all labels except first one
		for i := 1; i < len(list); i++ {
			list[i].Label = strings.ToLower(list[i].Label)
		}

		line.Groups = list

		result = append(result, line)
	}

	// sort by "From" field
	sort.Slice(result, func(i, j int) bool {
		return result[i].From < result[j].From
	})

	return result
}

func newFileValue(content structs.Content, dir string) string {
	b, err := yaml.Marshal(content)
	if err != nil {
		log.Fatalf("Error marshaling content: %v", err)
		return ""
	}

	// todo: add more placeholders depending on dir

	return url.PathEscape(string(b))
}
