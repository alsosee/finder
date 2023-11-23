// A simple file browser written in Go.
package main

import (
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

	gitignore "github.com/sabhiram/go-gitignore"
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

var errNotFound = fmt.Errorf("not found")

// listFiles collects all files in the given path (and all parent directories)
// and returns them as a list of panels.
func listFiles(path string) (panels Panels, err error) {
	if path == "/" {
		path = ""
	}

	realDir := filepath.Join(*dir, path)

	// ensure that the path is a directory
	if stat, err := os.Stat(realDir); err == nil && !stat.IsDir() {
		realDir = filepath.Dir(realDir)
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
			log.Printf("error reading directory %s: %v", realDir, err)
			if os.IsNotExist(err) {
				return nil, errNotFound
			}
			return nil, err
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

	return panels, nil
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

	ignore *gitignore.GitIgnore
)

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

		panels, err := listFiles(r.URL.Path)
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
			Timestamp   int64
		}{
			HXRequest:   r.Header.Get("HX-Request") == "true",
			CurrentPath: r.URL.Path,
			Dirs:        buildDirs(r.URL.Path),
			Panels:      panels,
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

var indexTemplate = template.Must(template.New("index").Funcs(map[string]interface{}{
	"join": func(dir, name string) string {
		return filepath.Join(dir, name)
	},
}).Parse(`
<!doctype html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Finder</title>
    <link rel="stylesheet" href="/style.css?ts={{ .Timestamp }}">
    <script src="https://unpkg.com/htmx.org@1.9.4" integrity="sha384-zUfuhFKKZCbHTY6aRR46gxiqszMk5tcHjsVFxnUo8VMus4kHGVdIYVbOYYNlKmHV" crossorigin="anonymous"></script>
    <script src="/navigation.js?ts={{ .Timestamp }}"></script>
</head>
<body data-view="columns">
<div id="toolbar" hx-preserve="true">
	<fieldset
		class="radio menubar-navigation"
		title="Show items as icons, in a list or in columns"
		role="menubar"
		aria-label="View"
	>
		<legend>View</legend>
		<label tabindex="0" role="menuitem"><input type="radio" name="view" value="icons"> <span>Icons</span></label>
		<label tabindex="0" role="menuitem"><input type="radio" name="view" value="list"> <span>List</span></label>
		<label tabindex="0" role="menuitem"><input type="radio" name="view" value="columns" checked> <span>Columns</span></label>
	</fieldset>
</div>
<div id="container" hx-boost="true">
    <nav>
        <ul id="breadcrumbs" class="menubar-navigation" role="menubar" aria-label="breadcrumbs">
            {{- range .Dirs }}
            {{- $isCurrent := eq $.CurrentPath .Path }}
            {{- if $isCurrent }}
            <li role="none"><a role="menuitem" href="{{ .Path }}"{{ if .InPath }} class="secondary"{{ end }} aria-current="page">{{ .Name }}</a></li>
            {{- else }}
            <li role="none"><a role="menuitem" href="{{ .Path }}"{{ if .InPath }} class="secondary"{{ end }}>{{ .Name }}</a></li>
            {{- end }}
            {{- end }}
        </ul>
    </nav>
    <nav id="panels">
	{{- range $index, $panel := .Panels }}
        <ul class="panel menubar-navigation" role="menu" data-level="{{ $index }}">
            {{- range $panel.Files }}
            {{- $path := join .Dir .Name }}
            <li role="none">
                <a
                    role="menuitem"
                    class="{{ if .IsFolder }}folder{{ end }}{{ if eq $.CurrentPath $path }} active{{ end }}{{ if .IsInBreakcrumbs }} in-breadcrumbs{{ end }}"
                    href="{{ $path }}"
                >
                    <span>{{ .Name }}</span>
                </a>
            </li>
            {{- end }}
        </ul>
        {{- end }}
    </nav>
</div>
{{- if not .HXRequest }}
<script type="text/javascript">
    const toolbar = document.querySelector('#toolbar');
    const container = document.querySelector('#container');

    let view = localStorage.getItem('view') || 'icons';
    console.log('view', view);
    document.body.setAttribute('data-view', view);
    toolbar.querySelector(` + "`input[value=${view}]`" + `).checked = true;

    let setView = function(value) {
        localStorage.setItem('view', value);
        document.body.setAttribute('data-view', value);
    };

    // if enter or space is pressed on a toolbar item, check the radio button
    toolbar.addEventListener('keydown', (event) => {
        if (event.key === 'Enter' || event.key === ' ') {
            event.target.querySelector('input').checked = true;
            setView(event.target.querySelector('input').value);
        }
    });

    toolbar.addEventListener('change', (event) => {
        setView(event.target.value);
    });
</script>
{{- end }}
</body>
</html>
`))
