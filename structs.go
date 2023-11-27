package main

import "gopkg.in/yaml.v3"

// Connections represents a list of connectiones that initiated by a reference.
// Key is a file path, where reference is pointing to.
// Value is a list of files that are pointing to the key.
// For example, if a file "A" has a reference to file "B",
// and file "C" has a reference to file "B" as well,
// then the Connections map will look like this:
//
//	{
//	  "B": ["A", "C"]
//	}
type Connections map[string]map[string]bool

// Contents represents a list of contents, where key is a file path.
// It is used to properly render references.
type Contents map[string]Content

// File represents a file or directory in the file system.
type File struct {
	Name            string
	Dir             string
	IsFolder        bool
	IsInBreakcrumbs bool
}

// Dir represents a directory in the breadcrumbs.
type Dir struct {
	InPath bool
	Name   string
	Path   string
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

// Panel represents a single directory with files.
type Panel struct {
	Files []File
}

// FileLists is a map, where key is a directory path, and value is a Panel that corresponds to that directory.
type FileLists map[string]Panel

// Panels represents a list of panels.
type Panels []Panel

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

	var ref Reference
	if err := value.Decode(&ref); err != nil {
		return err
	}

	r = &ref
	return nil
}

// Content represents the content of a file.
type Content struct {
	HTML string `yaml:"-"` // for Markdown files

	Name        string
	Subtitle    string
	Year        int
	Author      string
	Authors     string
	Description string

	DOB string
	DOD string

	Website         string
	Wikipedia       string
	GoodReads       string
	Twitch          string
	YouTube         string
	IMDB            string
	Steam           string
	Hulu            string
	AdultSwim       string
	AppStore        string `yaml:"app_store"`
	Fandom          string
	RottenTomatoes  string `yaml:"rotten_tomatoes"`
	Twitter         string
	Instagram       string
	TelegramChannel string `yaml:"telegram_channel"`
	X               string

	ISBN   string
	ISBN10 string
	ISBN13 string
	OCLC   string

	// unknown fields are stored in the Extra map
	Extra map[string]interface{} `yaml:",inline"`

	References []Reference `yaml:"refs"`
}
