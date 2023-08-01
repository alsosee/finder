package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"
)

type File struct {
	Name            string
	Dir             string
	IsFolder        bool
	IsInBreakcrumbs bool
}

type Dir struct {
	InPath bool
	Name   string
	Path   string
}

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

type Panel struct {
	Files []File
}

type Panels []Panel

var errNotFound = fmt.Errorf("not found")

func listFiles(path string) (panels Panels, err error) {
	root := "dir"
	realDir := filepath.Join(root, path)

	if stat, err := os.Stat(realDir); err == nil && !stat.IsDir() {
		realDir = filepath.Dir(realDir)
	}

	dirs := strings.Split(realDir, string(filepath.Separator))
	for i := range dirs {
		realDir = filepath.Join(dirs[:i+1]...)
		entries, err := os.ReadDir(realDir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, errNotFound
			}
			return nil, err
		}

		panel := Panel{
			Files: []File{},
		}

		dir := strings.TrimPrefix(realDir, root)
		if len(dir) == 0 {
			dir = "/"
		}

		for _, entry := range entries {
			panel.Files = append(panel.Files, File{
				Name:            entry.Name(),
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

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.URL.Path)

		// serve files from the static directory
		if r.URL.Path == "/style.css" || r.URL.Path == "/files.png" || r.URL.Path == "/sprite.png" {
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

	log.Println("Starting server on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
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
</head>
<body data-view="columns">
<div id="toolbar" hx-preserve="true">
    <fieldset class="radio" title="Show items as icons, in a list or in columns" role="menubar">
        <legend>View</legend>
        <label tabindex="0" role="menuitem"><input type="radio" name="view" value="icons"> <span>Icons</span></label>
        <label tabindex="0" role="menuitem"><input type="radio" name="view" value="list"> <span>List</span></label>
        <label tabindex="0" role="menuitem"><input type="radio" name="view" value="columns" checked> <span>Columns</span></label>
    </fieldset>
</div>
<div id="container" hx-boost="true">
    <ul id="breadcrumbs">
        {{- range .Dirs }}
        <li><a href="{{ .Path }}"{{ if .InPath }} class="secondary"{{ end }}>{{ .Name }}</a></li>
        {{- end }}
    </ul>
    <div id="panels">
        {{- range $index, $panel := .Panels }}
        <ul class="panel" data-level="{{ $index }}">
            {{- range $panel.Files }}
            {{- $path := join .Dir .Name }}
            <li>
                <a class="{{ if .IsFolder }}folder{{ end }}{{ if eq $.CurrentPath $path }} active{{ end }}{{ if .IsInBreakcrumbs }} in-breadcrumbs{{ end }}" href="{{ $path }}"
                ><span>{{ .Name }}</span></a>
            </li>
            {{- end }}
        </ul>
        {{- end }}
    </div>
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
