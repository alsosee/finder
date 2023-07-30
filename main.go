package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

type File struct {
	Name     string
	Dir      string
	IsFolder bool
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

func listFiles(path string) (panels Panels) {
	root := "dir"
	path = strings.TrimPrefix(path, "/")
	dir := filepath.Join(root, path)

	if stat, err := os.Stat(dir); err == nil && !stat.IsDir() {
		dir = filepath.Dir(dir)
	}

	dirs := strings.Split(dir, string(filepath.Separator))
	for i := range dirs {
		dir = filepath.Join(dirs[:i+1]...)
		entries, err := os.ReadDir(dir)
		if err != nil {
			log.Fatal(err)
		}

		panel := Panel{
			Files: []File{},
		}

		for _, entry := range entries {
			panel.Files = append(panel.Files, File{
				Name:     entry.Name(),
				Dir:      strings.TrimPrefix(dir, root),
				IsFolder: entry.IsDir(),
			})
		}
		sort.Sort(ByNameFolderOnTop(panel.Files))
		panels = append(panels, panel)
	}

	return panels
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.URL.Path)

		// serve files from the static directory
		if r.URL.Path == "/style.css" || r.URL.Path == "/files.png" || r.URL.Path == "/sprite.png" {
			http.ServeFile(w, r, "static"+r.URL.Path)
			return
		}

		// serve the index template
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		indexTemplate.Execute(w, struct {
			Panels Panels
		}{
			Panels: listFiles(r.URL.Path),
		})

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
    <link rel="stylesheet" href="style.css">
    <script src="https://unpkg.com/htmx.org@1.9.3"></script>
    <script src="https://unpkg.com/htmx.org/dist/ext/ws.js"></script>
</head>
<body>
<div id="toolbar">
    <fieldset class="radio" title="Show items as icons, in a list or in columns" role="menubar">
        <legend>View</legend>
        <label tabindex="0" role="menuitem"><input type="radio" name="view" value="icons"> <span>Icons</span></label>
        <label tabindex="0" role="menuitem"><input type="radio" name="view" value="list"> <span>List</span></label>
        <label tabindex="0" role="menuitem"><input type="radio" name="view" value="columns" checked> <span>Columns</span></label>
    </fieldset>
</div>
<div id="container" data-view="columns" hx-swap-oob="insideHTML">
    {{- range $index, $panel := .Panels }}
    <ul class="panel" data-level="{{ $index }}">
        {{- range $panel.Files }}
        <li
            class="{{ if .IsFolder}}folder{{ end }}"
            hx-get="{{ join .Dir .Name }}"
            ><span>{{ .Name }}</span></li>
        {{- end }}
    </ul>
    {{- end }}
</div>
<script type="text/javascript">
    const toolbar = document.querySelector('#toolbar');
    const container = document.querySelector('#container');

    let view = localStorage.getItem('view') || 'icons';
    container.setAttribute('data-view', view);
    toolbar.querySelector(` + "`input[value=${view}]`" + `).checked = true;

    let setView = function(value) {
        localStorage.setItem('view', value);
        container.setAttribute('data-view', value);
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
</body>
</html>
`))
