package structs

import (
	"time"

	"gopkg.in/yaml.v3"
)

type PageData struct {
	OutputPath  string
	CurrentPath string
	Dir         string
	Breadcrumbs []Dir
	Panels      Panels
	Content     *Content
	Timestamp   int64
	Connections Connections
}

// Contents represents a list of contents, where key is a file path.
// It is used to properly render references.
type Contents map[string]Content

// Connections represents a list of connectiones that initiated by a reference.
// Key is a file path, where reference is pointing to.
// Value is a map, where key is a file path, where reference is located,
// and value is a type of reference.
// For example, three files "Alice", "Book" and "Cinema" are referencing to file "Dave",
// but in different contexts. File "Alice" just has a reference to file "Dave",
// file "Book" has a reference "Dave" as an "Author",
// and file "Cinema" has a reference "Dave" as a "Voice" for "Bob" (presumably, a character).
// Then the Connections map will look like this:
//
//	{
//		"Dave": {
//			"Alice": []
//			"Book": ["Author"]
//			"Cinema": ["Voice", "Bob"],
//		}
//	}
type Connections map[string]map[string][]string

type Columns map[string]string

func (c *Columns) Add(key, value string) {
	if *c == nil {
		*c = make(map[string]string)
	}

	if value == "" {
		return
	}

	(*c)[key] = value
}

func (c *Columns) Get(key string) string {
	if *c == nil {
		return ""
	}

	if value, ok := (*c)[key]; ok {
		return value
	}

	return ""
}

// File represents a file or directory in the file system.
type File struct {
	Name      string
	Title     string // value from YAML "name" field, may contain colons
	IsFolder  bool   // used to render folder icon and to sort files
	IsMissing bool   // for pages that have no source file; used to show striped background
	Image     *Media

	Columns Columns // extra fields to use in list view
}

// ByNameFolderOnTop sorts files by name, with folders on top.
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

// ByYearDesk sorts files by year, with newest on top.
// Directories that does not match the year format are sorted alphabetically.
type ByYearDesk []File

func (f ByYearDesk) Len() int      { return len(f) }
func (f ByYearDesk) Swap(i, j int) { f[i], f[j] = f[j], f[i] }
func (f ByYearDesk) Less(i, j int) bool {
	a, err := time.Parse("2006", f[i].Name)
	aIsYear := err == nil

	b, err := time.Parse("2006", f[j].Name)
	bIsYear := err == nil

	if aIsYear && bIsYear {
		return a.After(b)
	}

	if aIsYear && !bIsYear {
		return false
	}

	if !aIsYear && bIsYear {
		return true
	}

	return ByNameFolderOnTop(f).Less(i, j)
}

// Panel represents a single directory with files.
type Panel struct {
	Dir   string
	Files []File
}

// Dir represents a directory in the breadcrumbs.
type Dir struct {
	Name string
	Path string
}

// FileLists is a map, where key is a directory path, and value is a Panel that corresponds to that directory.
type FileLists map[string]Panel

// Panels represents a list of panels.
type Panels []Panel

// Breadcrumbs represents a list of directories in the breadcrumbs.
type Breadcrumbs []Dir

// Reference represents a reference to another file.
// Often it has only a path.
type Reference struct {
	Path string
	Name string
}

// UnmarshalYAML is a custom unmarshaler for Reference.
// It can be either a string or a map.
func (r *Reference) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		r.Path = value.Value
		return nil
	}

	return value.Decode(&r)
}

// Missing represents a missing reference.
// Used in "missing" template function to render Missing.gomd file.
type Missing struct {
	To     string
	From   map[string][]string
	Awards []Award
}
