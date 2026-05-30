package main

import (
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
