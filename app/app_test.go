package app

import (
	"io/fs"
	"log"
	"testing"
	"testing/fstest"

	"github.com/alsosee/finder/structs"
)

// createTestConfig creates a valid config YAML for testing
func createTestConfig() string {
	return `title: Test Site
description: A test site
lang: en
repo: test/repo
url: https://test.example.com
media_host: https://media.example.com
search_host: https://search.example.com
search_api_key: test_key
search_index: test_index
opengraph:
  image: test.png
  width: 1200
  height: 630
  twitter_image: test_twitter.png
logo_shift_y: "0px"
home_label: Home
search_label: Search
not_found_header: Not Found
not_found_description: Page not found
not_found_but_description: But here are some options
views_label: Views
views_tooltip: Change view
view_icons: Icons
view_list: List
view_columns: Columns
menu:
  - title: Home
    url: /
    logo_shift_y: "0px"
label_cancel: Cancel
label_upload: Upload
no_results_label: No results found
column_name: Name
column_kind: Kind
of_label: of
and_label: and`
}

func createTestInfoFileSystem() fs.FS {
	return fstest.MapFS{
		"config.yaml": &fstest.MapFile{
			Data: []byte(createTestConfig()),
		},
		// "invalid.yaml": &fstest.MapFile{
		// 	Data: []byte("invalid: yaml: content: ["),
		// },
		"content/movie.yaml": &fstest.MapFile{
			Data: []byte(`name: Test Movie
year: 2023
type: movie`),
		},
		"content/person.yaml": &fstest.MapFile{
			Data: []byte(`name: Test Person
type: person
dob: 1990-01-01`),
		},
		"content/article.md": &fstest.MapFile{
			Data: []byte("# Test Article\n\nThis is a test markdown file."),
		},
		"content/.thumbs.yml": &fstest.MapFile{
			Data: []byte(`- name: person.jpg
  width: 800
  height: 600
- name: image2.png
  width: 1024
  height: 768`),
		},
		".gitignore": &fstest.MapFile{
			Data: []byte("*.tmp\n.DS_Store\n"),
		},
		".finderignore": &fstest.MapFile{
			Data: []byte(`*.gitignore
config.yaml
.gitignore`),
		},
	}
}

type StubProcessor struct {
	Content map[string]structs.Content
}

func (sp *StubProcessor) ProcessContent(content structs.Content) error {
	log.Println("Adding content:", content.SourceNoExtention)
	if sp.Content == nil {
		sp.Content = make(map[string]structs.Content)
	}

	sp.Content[content.SourceNoExtention] = content
	return nil
}

func TestNewApp(t *testing.T) {
	app, err := NewApp(
		structs.Config{},
		createTestInfoFileSystem(),
	)
	if err != nil {
		t.Fatalf("NewApp() failed with valid config: %v", err)
	}
	if app == nil {
		t.Fatal("NewApp() returned nil app")
	}

	app.NumWorkers = 1 // Set to 1 for testing

	processor := &StubProcessor{}
	app.AddProcessor(processor)

	err = app.Run()
	if err != nil {
		t.Fatalf("App.Run() failed: %v", err)
	}

	// if !processor.initCalled {
	// 	t.Error("Processor Init() was not called")
	// }
}
