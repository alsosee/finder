// A simple file browser written in Go.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/gomarkdown/markdown"
	gitignore "github.com/sabhiram/go-gitignore"
	"gopkg.in/yaml.v3"
)

// File represents a file or directory in the file system.
type File struct {
	Name            string
	Dir             string
	IsFolder        bool
	IsInBreakcrumbs bool
}

// Dir represents a directory in the breadcrumbs.
type Dir struct {
	InPath bool
	Name   string
	Path   string
}

// ByNameFolderOnTop sorts files by name, with folders on top.
type ByNameFolderOnTop []File

func (a ByNameFolderOnTop) Len() int      { return len(a) }
func (a ByNameFolderOnTop) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByNameFolderOnTop) Less(i, j int) bool {
	if a[i].IsFolder == a[j].IsFolder {
		return a[i].Name < a[j].Name
	}
	if a[i].IsFolder && !a[j].IsFolder {
		return true
	}
	if !a[i].IsFolder && a[j].IsFolder {
		return false
	}
	return a[i].Name < a[j].Name
}

// Panel represents a single directory with files.
type Panel struct {
	Files []File
}

// Panels represents a list of panels.
type Panels []Panel

// Reference represents a reference to another file.
// Often it has only a path.
type Reference struct {
	Path string
	Name string
}

// UnmarshalYAML is a custom unmarshaler for Reference.
// It can be either a string or a map.
func (r *Reference) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		r.Path = value.Value
		return nil
	}

	var ref Reference
	if err := value.Decode(&ref); err != nil {
		return err
	}

	r = &ref
	return nil
}

// Content represents the content of a file.
type Content struct {
	HTML string `yaml:"-"` // for Markdown files

	Name        string
	Subtitle    string
	Year        int
	Author      string
	Authors     string
	Description string

	DOB string
	DOD string

	Website         string
	Wikipedia       string
	GoodReads       string
	Twitch          string
	YouTube         string
	IMDB            string
	Steam           string
	Hulu            string
	AdultSwim       string
	AppStore        string `yaml:"app_store"`
	Fandom          string
	RottenTomatoes  string `yaml:"rotten_tomatoes"`
	Twitter         string
	Instagram       string
	TelegramChannel string `yaml:"telegram_channel"`
	X               string

	ISBN   string
	ISBN10 string
	ISBN13 string
	OCLC   string

	// unknown fields are stored in the Extra map
	Extra map[string]interface{} `yaml:",inline"`

	References []Reference `yaml:"refs"`
}

var errNotFound = fmt.Errorf("not found")

// listFiles collects all files in the given path (and all parent directories)
// and returns them as a list of panels.
func listFiles(path string) (panels Panels, content *Content, err error) {
	realDir := filepath.Join(*dir, path)

	content = nil
	// ensure that the path is a directory
	if _, err := os.Stat(realDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			content = tryFiles(path)
			if content != nil {
				path = filepath.Dir(path)
				realDir = filepath.Join(*dir, path)
			}
		}
	}

	if path == "/" {
		path = ""
	}

	dirs := strings.Split(path, string(filepath.Separator))
	for i := range dirs {
		realDir = filepath.Join(*dir, filepath.Join(dirs[:i+1]...))

		dir := strings.TrimPrefix(realDir, *dir)
		if len(dir) == 0 {
			dir = "/"
		}

		panel := Panel{
			Files: []File{},
		}

		entries, err := os.ReadDir(realDir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, nil, errNotFound
			}
			return nil, nil, err
		}

		for _, entry := range entries {
			name := entry.Name()

			if ignore.MatchesPath(name) {
				continue
			}

			if *hideExtensions {
				name = strings.TrimSuffix(name, filepath.Ext(name))
				if len(name) == 0 {
					name = entry.Name()
				}
			}

			panel.Files = append(panel.Files, File{
				Name:            name,
				Dir:             dir,
				IsFolder:        entry.IsDir(),
				IsInBreakcrumbs: entry.IsDir() && strings.HasPrefix(path, filepath.Join(dir, entry.Name())),
			})
		}

		sort.Sort(ByNameFolderOnTop(panel.Files))
		panels = append(panels, panel)
	}

	return panels, content, nil
}

func tryFiles(path string) *Content {
	// try to find a file with the same name as the directory
	// with a ".md" extension
	// if the file exists, read the content
	// otherwise, return nil

	extentions := []string{".yml", ".yaml", ".md"}
	for _, ext := range extentions {
		content, err := readContent(path, ext)
		if err == nil {
			return content
		}
	}
	return nil
}

func readContent(path, ext string) (*Content, error) {
	b, err := os.ReadFile(filepath.Join(*dir, path+ext))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errNotFound
		}
		return nil, err
	}

	var content Content

	switch ext {
	case ".yml", ".yaml":
		if err := yaml.Unmarshal(b, &content); err != nil {
			return nil, err
		}
	case ".md":
		htmlBody := markdown.ToHTML(b, nil, nil)
		content = Content{
			HTML: string(htmlBody),
		}
	}

	return &content, nil
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

var (
	bind           = flag.String("bind", ":8080", "address to bind to")
	dir            = flag.String("dir", "", "root directory to serve")
	hideExtensions = flag.Bool("he", false, "hide file extensions")
	ignoreFile     = flag.String("ignore", ".ignore", "file with list of files to ignore")

	ignore        *gitignore.GitIgnore
	indexTemplate *template.Template
)

func init() {
	var err error
	// parse index.gohtml with funcmap
	indexTemplate, err = template.New("index.gohtml").Funcs(template.FuncMap{
		"join": func(dir, name string) string {
			return filepath.Join(dir, name)
		},
	}).ParseFiles("index.gohtml")

	if err != nil {
		log.Fatalf("error parsing template: %v", err)
	}
}

func main() {
	flag.Parse()

	if *dir == "" {
		log.Fatal("dir is required")
	}

	// replace ~ with the home directory
	if strings.HasPrefix(*dir, "~") {
		*dir = filepath.Join(os.Getenv("HOME"), strings.TrimPrefix(*dir, "~"))
	}

	log.Printf("Serving directory %s", *dir)

	// read ".ignore" file from the root directory
	if _, err := os.Stat(filepath.Join(*dir, *ignoreFile)); err == nil {
		ignore, err = gitignore.CompileIgnoreFile(filepath.Join(*dir, *ignoreFile))
		if err != nil {
			log.Fatalf("error reading ignore file: %v", err)
		}
	} else {
		ignore = gitignore.CompileIgnoreLines()
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.URL.Path)

		// serve files from the static directory
		// check if file exists first
		if stat, err := os.Stat("static" + r.URL.Path); err == nil && !stat.IsDir() {
			http.ServeFile(w, r, "static"+r.URL.Path)
			return
		}

		panels, content, err := listFiles(r.URL.Path)
		if err != nil {
			if err == errNotFound {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Vary", "HX-Request")

		err = indexTemplate.Execute(w, struct {
			HXRequest   bool
			CurrentPath string
			Dirs        []Dir
			Panels      Panels
			Content     *Content
			Timestamp   int64
		}{
			HXRequest:   r.Header.Get("HX-Request") == "true",
			CurrentPath: r.URL.Path,
			Dirs:        buildDirs(r.URL.Path),
			Panels:      panels,
			Content:     content,
			Timestamp:   time.Now().Unix(),
		})
		if err != nil {
			log.Printf("error executing template: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	log.Printf("Starting server on port %s", *bind)
	log.Fatal(http.ListenAndServe(*bind, nil))
}
