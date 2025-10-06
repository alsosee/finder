package processors

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"hash/crc32"
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

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"

	"github.com/alsosee/finder/structs"
)

var ErrExecutingTemplate = errors.New("error executing template")
var caser = cases.Title(language.English, cases.NoLower)

//go:embed functions/*
var functionsFS embed.FS

type HTMLGenerator struct {
	TemplatesDirectory string
	StaticDirectory    string
	OutputDirectory    string

	// todo:
	muContents sync.Mutex

	// dirContents is a map where
	// key is a directory path,
	// value is a list of files and directories;
	// used to build Panels
	dirContents map[string][]structs.File

	contents               map[string]structs.Content // path without extension -> Content struct
	mediaDirContents       map[string][]structs.Media // directory -> list of media files in that directory
	muAwardsMissingContent sync.Mutex
	awardsMissingContent   map[string][]structs.Award // content ID -> list of awards that reference this content
	muChainPages           sync.Mutex
	chainPages             map[string]map[bool]string // content ID -> map[isNext]pageID
	muConnections          sync.Mutex
	connections            map[string]map[string][]structs.Connection // content ID -> map[referenced content ID]connections
	muAwardPages           sync.Mutex
	awardPages             map[string][]structs.Award // content ID -> list of awards that reference this content
	muRenderedPanels       sync.Mutex
	renderedPanelsCache    map[string]string // panel directory -> rendered HTML

	templates *template.Template
}

var _ structs.Processor = (*HTMLGenerator)(nil)

func (h *HTMLGenerator) Init() error {
	var err error

	t, err := template.New("").Funcs(h.fm()).ParseGlob(h.TemplatesDirectory + "/*")
	if err != nil {
		return fmt.Errorf("parsing templates: %w", err)
	}
	h.templates = t

	h.copyStaticFiles()
	h.copyFunctionsFiles()

	return nil
}

func (h *HTMLGenerator) ProcessFiles(contents structs.Contents) error {
	for id, content := range contents {
		path := filepath.Join(h.OutputDirectory, id+".html")
		panels, breadcrumbs := h.buildPanels(id, true)
		cnt := content

		err := h.executeTemplate(path, structs.PageData{
			CurrentPath: id,
			Dir:         filepath.Dir(id),
			Breadcrumbs: breadcrumbs,
			Panels:      panels,
			Content:     &cnt,
			Timestamp:   time.Now().Unix(),
		}, "index.gohtml")
		if err != nil {
			return fmt.Errorf("%w for %q: %w", ErrExecutingTemplate, id, err)
		}
	}

	return nil
}

func (h *HTMLGenerator) ProcessDirectories(dirs map[string][]structs.File) error {
	for dir := range dirs {
		path := filepath.Join(h.OutputDirectory, dir, "index.html")
		panels, breadcrumbs := h.buildPanels(dir, false)

		err := h.executeTemplate(path, structs.PageData{
			CurrentPath: dir,
			Breadcrumbs: breadcrumbs,
			Panels:      panels,
			Content:     nil,
			Timestamp:   time.Now().Unix(),
			Connections: nil,
		}, "index.gohtml")
		if err != nil {
			return fmt.Errorf("%w for %q: %w", ErrExecutingTemplate, dir, err)
		}
	}

	return nil
}

func (h *HTMLGenerator) Finalize() error {
	// Generate 404 page
	if err := h.generate404(); err != nil {
		return fmt.Errorf("generating 404 page: %w", err)
	}

	return nil
}

func (h *HTMLGenerator) fm() template.FuncMap {
	return template.FuncMap{
		"config":       func() structs.Config { return h.config },
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
			h.muContents.Lock()
			defer h.muContents.Unlock()

			if c, ok := h.contents[path]; ok {
				return &c
			}
			return nil
		},
		// "connections" returns a list of connections for a given file path (no extension).
		"connections": func(path string) map[string][]structs.Connection {
			h.muConnections.Lock()
			defer h.muConnections.Unlock()

			if m, ok := h.connections[path]; ok {
				return m
			}

			return nil
		},
		"prev": func(id string) string {
			h.muChainPages.Lock()
			defer h.muChainPages.Unlock()

			if m, ok := h.chainPages[id]; ok {
				if prev, ok := m[false]; ok {
					return prev
				}
			}
			return ""
		},
		"next": func(id string) string {
			h.muChainPages.Lock()
			defer h.muChainPages.Unlock()

			if m, ok := h.chainPages[id]; ok {
				if next, ok := m[true]; ok {
					return next
				}
			}
			return ""
		},
		"crc32": h.crc32sum,
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
		"missing":    h.missing,
		"missingAwardsLen": func(id string) int {
			h.muAwardsMissingContent.Lock()
			defer h.muAwardsMissingContent.Unlock()
			return len(h.awardsMissingContent[id])
		},
		"image":       h.getImageForPath,
		"formatTitle": h.formatTitle,
		"title":       caser.String,
		"awardYear":   awardYear,
		"prefix":      prefix,
		"columns": func() []structs.Column {
			return structs.ColumnsList
		},
		"column":        column,
		"chooseColumns": chooseColumns,
		"rootTypes":     func() map[string]string { return structs.RootTypes },
		"renderPanel":   h.renderPanel,
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
	}
}

func (h *HTMLGenerator) copyStaticFiles() {
	if h.StaticDirectory == "" {
		log.Printf("No static files directory specified, skipping")
		return
	}

	log.Printf("Copying static files from %q to %q", h.StaticDirectory, h.OutputDirectory)

	if err := os.MkdirAll(h.OutputDirectory, 0o755); err != nil {
		log.Fatalf("Error creating output directory %q: %v", h.OutputDirectory, err)
	}

	err := filepath.Walk(
		h.StaticDirectory,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return os.MkdirAll(filepath.Join(h.OutputDirectory, strings.TrimPrefix(path, h.StaticDirectory)), 0o755)
			}

			relPath := strings.TrimPrefix(path, h.StaticDirectory+string(filepath.Separator))
			outPath := filepath.Join(h.OutputDirectory, relPath)

			if strings.HasSuffix(path, ".gojs") {
				outPath = strings.TrimSuffix(outPath, ".gojs") + ".js"
				log.Printf("Processing GoJS file %q to %q", path, outPath)
				return h.processGoJSFile(path, outPath)
			}

			return copyFile(path, outPath)
		},
	)
	if err != nil {
		log.Fatalf("Error walking static directory %q: %v", h.StaticDirectory, err)
	}

	log.Printf("Done copying static files from %q to %q", h.StaticDirectory, h.OutputDirectory)
}

func (h *HTMLGenerator) copyFunctionsFiles() {
	log.Printf("Copying functions files")

	// check if functions directory exists, if it does – exit
	if _, err := os.Stat("functions"); err == nil {
		log.Printf("Functions directory already exists, skipping")
		return
	}

	// unline static files, functions directory has to the directory where app is running
	// so we can't use h.OutputDirectory
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
		defer outFile.Close()

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

func (h *HTMLGenerator) processGoJSFile(src, out string) error {
	// treat file as a Go template
	b, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	t, err := template.New("").Funcs(h.fm()).Parse(string(b))
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	outFile, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer outFile.Close()

	if err = t.Execute(outFile, nil); err != nil {
		return fmt.Errorf("%w for %q: %w", ErrExecutingTemplate, src, err)
	}

	return nil
}

func (h *HTMLGenerator) getImageForPath(path string) *structs.Media {
	dir := filepath.Dir(path)
	if dir == "." {
		dir = ""
	}

	base := structs.EscapeFileName(filepath.Base(path))

	dirContent, ok := h.mediaDirContents[dir]
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

func (h *HTMLGenerator) buildPanels(path string, isFile bool) (structs.Panels, structs.Breadcrumbs) {
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
			dir = h.config.HomeLabel
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
			Files: h.getFilesForPath(cumulativePath),
		})
	}

	return panels, breadcrumbs
}

func (h *HTMLGenerator) getFilesForPath(path string) []structs.File {
	files, ok := h.dirContents[path]
	if !ok {
		return nil
	}

	return files
}

// renderPanel renders a panel and caches the result
// it is an optimisation to not render the same panel multiple times.
// drawback is that some content still has to be changed dynamically,
// that is why markInPathLinks function is used
func (h *HTMLGenerator) renderPanel(panel structs.Panel, index int, isLast bool, path string) string {
	h.muRenderedPanels.Lock()
	defer h.muRenderedPanels.Unlock()

	if rendered, ok := h.renderedPanelsCache[panel.Dir]; ok {
		return markInPathLinks(rendered, panel, path, isLast)
	}

	rendered, err := h.renderPanelImpl(panel, index)
	if err != nil {
		log.Fatalf("rendering panel %q: %v", path, err)
	}

	h.renderedPanelsCache[panel.Dir] = rendered

	return markInPathLinks(rendered, panel, path, isLast)
}

func (h *HTMLGenerator) renderPanelImpl(panel structs.Panel, index int) (string, error) {
	var b bytes.Buffer
	err := h.templates.Lookup("panel.gohtml").Execute(&b, struct {
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

func (h *HTMLGenerator) generate404() error {
	outputPath := filepath.Join(h.OutputDirectory, "404.html")

	return h.executeTemplate(outputPath, structs.PageData{
		CurrentPath: "404",
		Dir:         "",
		Breadcrumbs: structs.Breadcrumbs{
			{Name: h.config.HomeLabel},
			{Name: h.config.NotFoundHeader, IsCurrent: true},
		},
		Panels:    nil, // no panels on 404 page
		Timestamp: time.Now().Unix(),
	}, "404.gohtml")
}

func (h *HTMLGenerator) executeTemplate(path string, pageData structs.PageData, templateName string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}

	if err := h.templates.ExecuteTemplate(f, templateName, pageData); err != nil {
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

// todo: move missing to App
func (h *HTMLGenerator) missing() []structs.Missing {
	missing := map[string]map[string][]structs.Connection{}

	// add missing content referenced by other files

	h.muConnections.Lock()
	h.muContents.Lock()

	for to, from := range h.connections {
		if _, ok := h.contents[to]; !ok && len(from) > 1 {
			missing[to] = from
		}
	}
	h.muContents.Unlock()
	h.muConnections.Unlock()

	result := []structs.Missing{}
	for to, from := range missing {
		result = append(
			result,
			structs.Missing{
				To:     to,
				From:   from,
				Awards: h.awardsMissingContent[to],
			},
		)
	}

	// add missing content that got awards

	h.muAwardPages.Lock()
	for to, awards := range h.awardsMissingContent {
		if _, ok := h.contents[to]; !ok && len(awards) > 1 {
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
	h.muAwardPages.Unlock()

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

func (h *HTMLGenerator) formatTitle(b structs.Breadcrumbs) string {
	b = b[1:] // skip the first element (it's always "Home")
	if len(b) == 0 {
		return h.config.Title
	}

	var dirs []string
	for _, dir := range b {
		dirs = append(dirs, dir.Name)
	}

	slices.Reverse(dirs)

	return strings.Join(dirs, " \\ ")
}

var (
	crc32cache = map[string]string{}
	crc32mu    = sync.Mutex{}
)

// crc32 calculates CRC32 checksum for a file.
// It's used to add a get parameter to a static file URL,
// so that when the file is updated, the browser will download the new version.
func (h *HTMLGenerator) crc32sum(path string) string {
	crc32mu.Lock()
	defer crc32mu.Unlock()

	if crc, ok := crc32cache[path]; ok {
		return crc
	}

	// calculate CRC32 checksum for a file
	file, err := os.Open(filepath.Join(h.OutputDirectory, path))
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

func in(needle string, slice ...string) bool {
	for _, s := range slice {
		if needle == s {
			return true
		}
	}
	return false
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

func column(file structs.File, column string) string {
	return file.Columns.Get(column)
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

func removeFileExtention(path string) string {
	withoutExt := path[:len(path)-len(filepath.Ext(path))]
	if withoutExt != "" {
		return withoutExt
	}
	return path
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
