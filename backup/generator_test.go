package main

import (
	"testing"

	gitignore "github.com/sabhiram/go-gitignore"

	"github.com/alsosee/finder/structs"
)

func TestGetFilesForPathSimple(t *testing.T) {
	cfg = Config{TemplatesDirectory: "templates", ConfigFile: "config.test.yml"}
	g, err := NewGenerator(&gitignore.GitIgnore{})
	if err != nil {
		t.Fatalf("error creating generator: %v", err)
	}

	tt := []struct {
		operations func(g *Generator)
		path       string
		expected   []structs.File
	}{
		{
			operations: func(g *Generator) {
				g.addFile("test1.yml")
				g.addFile("test2.md")
			},
			path: "",
			expected: []structs.File{
				{Name: "test1"},
				{Name: "test2"},
			},
		},
		{
			operations: func(g *Generator) {
				g.addFile("test1")
				g.addFile("test2")
				g.addDir("dir1")
				g.addDir("dir2")
				g.addFile("dir1/test3")
				g.addFile("dir2/test4")
			},
			path: "dir1",
			expected: []structs.File{
				{Name: "test3"},
			},
		},
		{
			operations: func(g *Generator) {
				g.addFile("dir1/test1")
				g.addDir("dir1/dir2")
				g.addFile("dir1/dir2/test2")
				g.addDir("dir1/dir2/dir3")
				g.addFile("dir1/dir2/dir3/test3")
			},
			path: "dir1/dir2",
			expected: []structs.File{
				{Name: "dir3", IsFolder: true},
				{Name: "test2"},
			},
		},
	}
	for _, tc := range tt {
		g.contents = structs.Contents{}
		g.dirContents = map[string][]structs.File{}

		tc.operations(g)

		g.processPanels()

		got := g.getFilesForPath(tc.path)
		if len(got) != len(tc.expected) {
			t.Fatalf("got %#v, expected %#v", got, tc.expected)
		}

		for i := range got {
			if got[i].Name != tc.expected[i].Name {
				t.Fatalf("got %#v, expected %#v", got, tc.expected)
			}
		}
	}
}

func TestRemoveFileExtention(t *testing.T) {
	tt := []struct {
		input    string
		expected string
	}{
		{
			input:    "test1.yml",
			expected: "test1",
		},
		{
			input:    "test.something.md",
			expected: "test.something",
		},
		{
			input:    "Mrs. Davis.md",
			expected: "Mrs. Davis",
		},
	}

	for _, tc := range tt {
		got := removeFileExtention(tc.input)
		if got != tc.expected {
			t.Fatalf("got %s, expected %s", got, tc.expected)
		}
	}
}

func TestGroupConnections(t *testing.T) {
	connections := map[string][]structs.Connection{}
	connections["Movies/2025/A"] = []structs.Connection{
		{To: "People/Alice", Label: "Played", Info: "Villian"},
	}
	connections["Movies/2024/B"] = []structs.Connection{
		{To: "People/Alice", Label: "Played", Info: "Hero"},
		{To: "People/Alice", Label: "Director"},
	}
	connections["Movies/2024/C"] = []structs.Connection{
		{To: "People/Alice", Label: "Played", Info: "Civilian", Parent: "Episode 1"},
		{To: "People/Alice", Label: "Played", Info: "Judge", Parent: "Episode 1"},
		{To: "People/Alice", Label: "Writer", Parent: "Episode 2"},
	}

	expected := []structs.ConnectionLine{
		{
			From: "Movies/2024/B",
			Groups: []structs.ConnectionLineItem{
				{Label: "Director", Info: []string{}},
				{Label: "played", Info: []string{"Hero"}},
			},
		},
		{
			From: "Movies/2024/C",
			Groups: []structs.ConnectionLineItem{
				{Label: "Writer"},
				{Label: "played", Info: []string{"Civilian", "Judge"}},
			},
			Parents: []string{"Episode 1", "Episode 2"},
		},
		{
			From: "Movies/2025/A",
			Groups: []structs.ConnectionLineItem{
				{Label: "Played", Info: []string{"Villian"}},
			},
		},
	}

	actual := groupConnections(connections)
	if len(actual) != len(expected) {
		t.Fatalf("expected %d connection lines, got %d", len(expected), len(actual))
	}

	for i := range actual {
		if actual[i].From != expected[i].From {
			t.Errorf("expected From %s, got %s", expected[i].From, actual[i].From)
			continue
		}
		if len(actual[i].Groups) != len(expected[i].Groups) {
			t.Errorf("expected %d groups, got %d", len(expected[i].Groups), len(actual[i].Groups))
		}
		for j := range actual[i].Groups {
			if actual[i].Groups[j].Label != expected[i].Groups[j].Label {
				t.Errorf("expected group label %s, got %s", expected[i].Groups[j].Label, actual[i].Groups[j].Label)
			}
			if len(actual[i].Groups[j].Info) != len(expected[i].Groups[j].Info) {
				t.Errorf("expected %d info items, got %d", len(expected[i].Groups[j].Info), len(actual[i].Groups[j].Info))
			}
			for k := range actual[i].Groups[j].Info {
				if actual[i].Groups[j].Info[k] != expected[i].Groups[j].Info[k] {
					t.Errorf("expected info item %s, got %s", expected[i].Groups[j].Info[k], actual[i].Groups[j].Info[k])
				}
			}
		}
		if len(actual[i].Parents) != len(expected[i].Parents) {
			t.Errorf("%s expected %d parents, got %d", actual[i].From, len(expected[i].Parents), len(actual[i].Parents))
		}
		for j := range actual[i].Parents {
			if actual[i].Parents[j] != expected[i].Parents[j] {
				t.Errorf("%s expected parent %s, got %s", actual[i].From, expected[i].Parents[j], actual[i].Parents[j])
			}
		}
	}
}
