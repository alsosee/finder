package main

import (
	"errors"
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

func runServer(bind *string) {
	// parse index.gohtml with funcmap
	indexTemplate, err := template.New("index.gohtml").Funcs(template.FuncMap{
		"join": func(dir, name string) string {
			return filepath.Join(dir, name)
		},
	}).ParseFiles("index.gohtml")
	if err != nil {
		log.Fatalf("error parsing index.gohtml: %v", err)
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
		ignore = &gitignore.GitIgnore{}
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

		if content != nil {
			contents[r.URL.Path] = *content

			// process references
			for _, ref := range content.References {
				refPath := string(filepath.Separator) + ref.Path

				if _, ok := connections[refPath]; !ok {
					connections[refPath] = map[string]bool{}
				}

				connections[refPath][r.URL.Path] = true
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Vary", "HX-Request")

		log.Printf("Connections: %v", connections)

		err = indexTemplate.Execute(w, struct {
			HXRequest   bool
			CurrentPath string
			Dirs        []Dir
			Panels      Panels
			Content     *Content
			Timestamp   int64
			Connections Connections
		}{
			HXRequest:   r.Header.Get("HX-Request") == "true",
			CurrentPath: r.URL.Path,
			Dirs:        buildDirs(r.URL.Path),
			Panels:      panels,
			Content:     content,
			Timestamp:   time.Now().Unix(),
			Connections: connections,
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
