package main

import (
	"testing"

	"github.com/alsosee/finder/structs"
	"gopkg.in/yaml.v3"
)

func TestGetFilesForPathSimple(t *testing.T) {
	tt := []struct {
		operations func(b *GraphBuilder)
		path       string
		expected   []structs.File
	}{
		{
			operations: func(b *GraphBuilder) {
				b.addFile("test1.yml")
				b.addFile("test2.md")
			},
			path: "",
			expected: []structs.File{
				{Name: "test1"},
				{Name: "test2"},
			},
		},
		{
			operations: func(b *GraphBuilder) {
				b.addFile("test1")
				b.addFile("test2")
				b.addDir("dir1")
				b.addDir("dir2")
				b.addFile("dir1/test3")
				b.addFile("dir2/test4")
			},
			path: "dir1",
			expected: []structs.File{
				{Name: "test3"},
			},
		},
		{
			operations: func(b *GraphBuilder) {
				b.addFile("dir1/test1")
				b.addDir("dir1/dir2")
				b.addFile("dir1/dir2/test2")
				b.addDir("dir1/dir2/dir3")
				b.addFile("dir1/dir2/dir3/test3")
			},
			path: "dir1/dir2",
			expected: []structs.File{
				{Name: "dir3", IsFolder: true},
				{Name: "test2"},
			},
		},
	}
	for _, tc := range tt {
		b := NewGraphBuilder(structs.Config{}, &ScanResult{}, nil, "", false)

		tc.operations(b)

		b.processPanels()

		got := b.dirContents[tc.path]
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

func TestContentNotReferences(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantPaths []string
	}{
		{
			name: "single",
			input: `
name: Outer Wilds
not: Games/2019/Outer Worlds
`,
			wantPaths: []string{"Games/2019/Outer Worlds"},
		},
		{
			name: "list",
			input: `
name: Outer Wilds
not:
  - Games/2019/Outer Worlds
  - Movies/1995/Heat
`,
			wantPaths: []string{"Games/2019/Outer Worlds", "Movies/1995/Heat"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var content structs.Content
			if err := yaml.Unmarshal([]byte(tt.input), &content); err != nil {
				t.Fatalf("unmarshaling content: %v", err)
			}

			if len(content.Not) != len(tt.wantPaths) {
				t.Fatalf("expected %d not references, got %d", len(tt.wantPaths), len(content.Not))
			}

			for i, wantPath := range tt.wantPaths {
				if content.Not[i].Path != wantPath {
					t.Fatalf("reference %d path = %q, expected %q", i, content.Not[i].Path, wantPath)
				}
			}

			connections := content.Connections()
			if len(connections) != len(tt.wantPaths) {
				t.Fatalf("expected %d connections, got %d", len(tt.wantPaths), len(connections))
			}

			for i, wantPath := range tt.wantPaths {
				if connections[i].To != wantPath {
					t.Errorf("connection %d To = %q, expected %q", i, connections[i].To, wantPath)
				}
				if connections[i].Label != "Not" {
					t.Errorf("connection %d Label = %q, expected %q", i, connections[i].Label, "Not")
				}
			}
		})
	}
}

func TestAddAwardsCanonicalizesColonReferences(t *testing.T) {
	b := NewGraphBuilder(structs.Config{}, &ScanResult{}, nil, "", false)

	b.contents["Movies/2024/Dune Part Two"] = structs.Content{
		Source: "Movies/2024/Dune Part Two.yml",
		Name:   "Dune: Part Two",
	}
	b.contents["Movies/Awards/Test/2024"] = structs.Content{
		Source: "Movies/Awards/Test/2024.yml",
		Categories: []structs.Category{
			{
				Name: "Best Picture",
				Winner: structs.Winner{
					Movie: "Dune: Part Two",
				},
			},
		},
	}
	b.awardPages = []string{"Movies/Awards/Test/2024"}

	b.addAwards()

	awardPage := b.contents["Movies/Awards/Test/2024"]
	gotReference := awardPage.Categories[0].Winner.Reference
	if gotReference != "Movies/2024/Dune Part Two" {
		t.Fatalf("got reference %q, expected %q", gotReference, "Movies/2024/Dune Part Two")
	}

	awarded := b.contents["Movies/2024/Dune Part Two"]
	if len(awarded.Awards) != 1 {
		t.Fatalf("got %d awards, expected 1", len(awarded.Awards))
	}

	if len(b.awardsMissingContent) != 0 {
		t.Fatalf("got missing award content %#v, expected none", b.awardsMissingContent)
	}
}
