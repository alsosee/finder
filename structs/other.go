package structs

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

type PageData struct {
	Content     *Content
	Connections Connections
	OutputPath  string
	CurrentPath string
	Dir         string
	Breadcrumbs []Dir
	Panels      Panels
	Timestamp   int64
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
//
// In case of TV series, it's a bit more complicated, because there are multiple episodes.
// The problem is that same person can be a writer for a multiple episodes.
// And when adding connection for the same person to the same content, it will append to the list.
//
//	{
//		"Charlie Brooker": {
//			"Black Mirror": [
//				"Writer", "", "The National Anthem",
//				"Writer", "", "Fifteen Million Merits",
//				"Writer", "", "The Entire History of You",
//			],
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
	Image *Media

	Columns   Columns // extra fields to use in list view
	Name      string
	Title     string // value from YAML "name" field, may contain colons
	IsFolder  bool   // used to render folder icon and to sort files
	IsMissing bool   // for pages that have no source file; used to show striped background
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

	type alias Reference
	var a alias
	if err := value.Decode(&a); err != nil {
		return err
	}
	*r = Reference(a)
	return nil
}

// References represents a list of references.
type References []Reference

// UnmarshalYAML is a custom unmarshaler for References,
// allowing to unmarshal a single string as a reference.
func (rr *References) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		*rr = []Reference{{Path: value.Value}}
		return nil
	}

	if value.Kind != yaml.SequenceNode {
		return fmt.Errorf("expected a string or a sequence")
	}

	if len(value.Content) == 0 {
		return nil
	}

	*rr = make([]Reference, len(value.Content))
	for i, v := range value.Content {
		if v.Kind == yaml.ScalarNode {
			(*rr)[i].Path = v.Value
			continue
		}

		err := v.Decode(&(*rr)[i])
		if err != nil {
			return fmt.Errorf("failed to decode reference: %w", err)
		}
	}

	return nil
}

type Link struct {
	Title string `yaml:"title"`
	URL   string `yaml:"url"`
}

// UnmarshalYAML is a custom unmarshaler for Link.
// It can be either a string or a map.
func (l *Link) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		*l = Link{URL: value.Value}
		return nil
	}

	type alias Link
	var a alias
	if err := value.Decode(&a); err != nil {
		return err
	}
	*l = Link(a)
	return nil
}

type Links []Link

// UnmarshalYAML is a custom unmarshaler for Links,
// allowing to unmarshal a single string as a link.
func (ll *Links) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		*ll = []Link{{URL: value.Value}}
		return nil
	}

	if value.Kind != yaml.SequenceNode {
		return fmt.Errorf("expected a string or a sequence")
	}

	if len(value.Content) == 0 {
		return nil
	}

	*ll = make([]Link, len(value.Content))
	for i, v := range value.Content {
		if v.Kind == yaml.ScalarNode {
			(*ll)[i].URL = v.Value
			continue
		}

		err := v.Decode(&(*ll)[i])
		if err != nil {
			return fmt.Errorf("failed to decode link: %w", err)
		}
	}

	return nil
}

// Missing represents a missing reference.
// Used in "missing" template function to render Missing.gomd file.
type Missing struct {
	To     string
	From   map[string][]string
	Awards []Award
}

// Extra usind in "splitExtra" template function to separate episode titles from the rest of the data.
type Extra struct {
	Addon   string
	Primary []string
}
