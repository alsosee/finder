package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/alsosee/finder/structs"
)

func TestGenerateGoTemplatesDoesNotMutateGraphContents(t *testing.T) {
	const id = "Pages/Example"
	const original = "Hello {{ \"World\" }}"

	graph := &BuildGraph{
		Contents: structs.Contents{
			id: {
				Source: id + ".gomd",
				HTML:   original,
			},
		},
	}

	projector := NewHTMLProjector(structs.Config{}, "", "", "", "")
	projector.graph = graph
	projector.contents = cloneContents(graph.Contents)
	projector.templates = template.New("").Funcs(projector.fm())

	if err := projector.generateGoTemplates(); err != nil {
		t.Fatalf("generateGoTemplates() error = %v", err)
	}

	if graph.Contents[id].HTML != original {
		t.Fatalf("graph content HTML was mutated: got %q, want %q", graph.Contents[id].HTML, original)
	}

	rendered := projector.contents[id].HTML
	if !strings.Contains(rendered, "Hello World") {
		t.Fatalf("projector content HTML was not rendered: got %q", rendered)
	}
}

func TestReferenceTemplateCanonicalizesColonPath(t *testing.T) {
	projector := NewHTMLProjector(structs.Config{}, "", "", "", "")
	projector.contents = structs.Contents{
		"Movies/2024/Dune Part Two": {
			Source: "Movies/2024/Dune Part Two.yml",
			Name:   "Dune: Part Two",
		},
	}
	projector.templates = template.Must(template.New("").Funcs(projector.fm()).ParseFiles(
		"templates/reference.gohtml",
		"templates/image_style.gohtml",
	))

	var rendered strings.Builder
	err := projector.templates.ExecuteTemplate(
		&rendered,
		"reference",
		map[string]interface{}{
			"Path":     "Movies/2024/Dune: Part Two",
			"Fallback": "Dune: Part Two",
		},
	)
	if err != nil {
		t.Fatalf("executing reference template: %v", err)
	}

	got := rendered.String()
	if !strings.Contains(got, `href="/Movies/2024/Dune Part Two"`) {
		t.Fatalf("rendered reference %q does not use canonical path", got)
	}
	if strings.Contains(got, `href="/Movies/2024/Dune: Part Two"`) {
		t.Fatalf("rendered reference %q still uses colon path", got)
	}
}

func TestHTMLProjectorExtractsPeoplePanel(t *testing.T) {
	outputDir := t.TempDir()
	config := structs.Config{
		HomeLabel:  "Home",
		ColumnName: "Name",
	}
	graph := &BuildGraph{
		Config: config,
		Contents: structs.Contents{
			"People/Alice": {
				Name:   "Alice",
				Source: "People/Alice.yml",
			},
			"People/Bob": {
				Name:   "Bob",
				Source: "People/Bob.yml",
			},
		},
		DirContents: map[string][]structs.File{
			"": {
				{Name: "People", Title: "People", IsFolder: true},
			},
			"People": {
				{Name: "Alice", Title: "Alice"},
				{Name: "Bob", Title: "Bob"},
			},
		},
	}

	projector := NewHTMLProjector(config, "", "", "templates", outputDir)
	if err := projector.Run(graph); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	panel := mustReadFile(t, filepath.Join(outputDir, sharedPanelPath("People")))
	if !strings.Contains(panel, `href="/People/Alice"`) || !strings.Contains(panel, `href="/People/Bob"`) {
		t.Fatalf("shared panel does not contain all people: %s", panel)
	}
	if strings.Contains(panel, "active in-path") {
		t.Fatalf("shared panel contains route-specific state: %s", panel)
	}

	personPage := mustReadFile(t, filepath.Join(outputDir, "People", "Alice.html"))
	if !strings.Contains(personPage, `id="shared-panel-people"`) {
		t.Fatalf("person page does not contain the shared panel placeholder")
	}
	if !strings.Contains(personPage, `data-panel-src="/_panels/People.html?crc=`) {
		t.Fatalf("person page does not reference the cache-busted shared panel")
	}
	if !strings.Contains(personPage, `hx-history="false"`) {
		t.Fatalf("person page does not exclude the shared panel from HTMX history snapshots")
	}
	placeholderStart := strings.Index(personPage, `id="shared-panel-people"`)
	placeholderEnd := strings.Index(personPage[placeholderStart:], `</ul>`)
	placeholder := personPage[placeholderStart : placeholderStart+placeholderEnd]
	if strings.Contains(placeholder, `hx-preserve`) {
		t.Fatalf("person page asks HTMX to preserve the oversized panel DOM")
	}
	if strings.Contains(personPage, `href="/People/Bob"`) {
		t.Fatalf("person page still embeds the People panel")
	}

	peopleIndex := mustReadFile(t, filepath.Join(outputDir, "People", "index.html"))
	if !strings.Contains(peopleIndex, `data-scroll-marker="true"`) {
		t.Fatalf("People index does not mark its shared panel as the current panel")
	}
}

func TestSharedPanelPath(t *testing.T) {
	if got := sharedPanelPath("People"); got != filepath.Join("_panels", "People.html") {
		t.Fatalf("sharedPanelPath(People) = %q", got)
	}
	if got := sharedPanelPath("Movies"); got != "" {
		t.Fatalf("sharedPanelPath(Movies) = %q, expected no shared panel", got)
	}
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(b)
}
