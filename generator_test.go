package main

import (
	"testing"
)

func TestGetFilesForPathSimple(t *testing.T) {
	g, err := NewGenerator(Config{
		TemplatesDirectory: "templates",
	})
	if err != nil {
		t.Fatalf("error creating generator: %v", err)
	}

	tt := []struct {
		operations func(g *Generator)
		path       string
		expected   []File
	}{
		{
			operations: func(g *Generator) {
				g.addFile("test1.yml")
				g.addFile("test2.md")
			},
			path: "",
			expected: []File{
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
			expected: []File{
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
			expected: []File{
				{Name: "dir3", IsFolder: true},
				{Name: "test2"},
			},
		},
	}
	for _, tc := range tt {
		g.contents = Contents{}
		g.dirContents = map[string][]File{}

		tc.operations(g)

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
