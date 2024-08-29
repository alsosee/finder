package main

import (
	"testing"

	"github.com/alsosee/finder/structs"
)

func TestGetFilesForPathSimple(t *testing.T) {
	cfg = Config{TemplatesDirectory: "templates", ConfigFile: "config.test.yml"}
	g, err := NewGenerator()
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
