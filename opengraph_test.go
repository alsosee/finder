package main

import (
	"testing"

	"github.com/alsosee/finder/structs"
)

func TestOpenGraphProjectorShouldGenerate(t *testing.T) {
	projector := OpenGraphProjector{force: ""}
	next := OpenGraphStateEntry{
		SourceHash:   "source",
		TemplateHash: "template",
		Key:          "opengraph/Movie.png",
	}

	if !projector.shouldGenerate("Movie", OpenGraphStateEntry{}, next) {
		t.Fatalf("expected missing state to generate")
	}

	old := next
	old.SourceHash = "source:image"
	if projector.shouldGenerate("Movie", old, next) {
		t.Fatalf("expected matching state to skip")
	}

	projector.force = "Movie"
	if !projector.shouldGenerate("Movie", old, next) {
		t.Fatalf("expected forced path to generate")
	}
}

func TestRenderOpenGraphImage(t *testing.T) {
	content := structs.Content{
		Source: "Movies/Example.yml",
		Name:   "Example",
	}
	content.GenerateID()

	b, err := renderOpenGraphImage(content, 1200, 630)
	if err != nil {
		t.Fatalf("renderOpenGraphImage() error = %v", err)
	}
	if len(b) == 0 {
		t.Fatalf("renderOpenGraphImage() returned empty image")
	}
}
