package main

import (
	"reflect"
	"testing"

	"github.com/alsosee/finder/structs"
)

func TestBuildProjectorsKeepsExplicitOrder(t *testing.T) {
	oldCfg := cfg
	cfg = Config{
		InfoDirectory:      "info",
		StaticDirectory:    "static",
		TemplatesDirectory: "templates",
		OutputDirectory:    "output",
		SearchMasterKey:    "master-key",
		SearchIndexName:    "info",
		StateFile:          ".state",
		OpenGraphState:     ".opengraph-state",
	}
	defer func() {
		cfg = oldCfg
	}()

	projectors := buildProjectors(structs.Config{}, map[string]bool{
		"html":      true,
		"search":    true,
		"opengraph": true,
		"json":      true,
		"markdown":  true,
	}, "https://images.example.test")

	names := make([]string, 0, len(projectors))
	for _, projector := range projectors {
		names = append(names, projector.Name())
	}

	want := []string{"html", "search", "opengraph", "json", "markdown"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("buildProjectors() names = %v, want %v", names, want)
	}
}

func TestBuildProjectorsSkipsSearchWithoutMasterKey(t *testing.T) {
	oldCfg := cfg
	cfg = Config{
		InfoDirectory:      "info",
		StaticDirectory:    "static",
		TemplatesDirectory: "templates",
		OutputDirectory:    "output",
	}
	defer func() {
		cfg = oldCfg
	}()

	projectors := buildProjectors(structs.Config{}, map[string]bool{
		"html":   true,
		"search": true,
	}, "")

	names := make([]string, 0, len(projectors))
	for _, projector := range projectors {
		names = append(names, projector.Name())
	}

	want := []string{"html"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("buildProjectors() names = %v, want %v", names, want)
	}
}
