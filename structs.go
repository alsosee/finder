package main

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// Character represents a character in a movie, tv show, etc.
type Character struct {
	Name       string
	Actor      string
	Voice      string
	Image      *Media
	ActorImage *Media
}

// oneOrMany represents a list of strings that can be passed as a single string in YAML.
type oneOrMany []string

// UnmarshalYAML makes BasedOn support both a string and a list of strings.
func (b *oneOrMany) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		*b = []string{value.Value}
		return nil
	}

	if value.Kind != yaml.SequenceNode {
		return fmt.Errorf("based_on must be a string or a list of strings")
	}

	if len(value.Content) == 0 {
		return nil
	}

	*b = make([]string, len(value.Content))
	for i, v := range value.Content {
		(*b)[i] = v.Value
	}

	return nil
}

// Content represents the content of a file.
type Content struct {
	Source string `yaml:"-"` // path to the file
	HTML   string `yaml:"-"` // for Markdown files

	// for everything
	Name        string
	Subtitle    string
	Year        int
	Author      string
	Authors     string
	Developers  string
	Description string
	CoverArtist string `yaml:"cover_artist"`
	Designer    string

	BasedOn oneOrMany `yaml:"based_on"`
	Series  string

	// for people
	DOB     string // date of birth
	DOD     string // date of death
	Contact string `yaml:"contact"`

	// general external links
	Website         string
	Websites        []string
	Wikipedia       string
	GoodReads       string
	Bookshop        string
	Twitch          string
	YouTube         string
	IMDB            string
	Steam           string
	Netflix         string
	Spotify         string
	Hulu            string
	AdultSwim       string
	AppStore        string `yaml:"app_store"`
	Fandom          string
	RottenTomatoes  string `yaml:"rotten_tomatoes"`
	Twitter         string
	Reddit          string
	Facebook        string
	Instagram       string
	TikTok          string
	TelegramChannel string `yaml:"telegram_channel"`
	PlayStation     string `yaml:"playstation"`
	XBox            string `yaml:"xbox"`
	GOG             string `yaml:"gog"`
	X               string
	Discord         string
	Epic            string
	IGN             string `yaml:"ign"`

	// for books
	ISBN        string
	ISBN10      string
	ISBN13      string
	OCLC        string
	Publisher   string
	Publication string // date or year of publication

	// for movies
	Genres         []string
	Trailer        string
	Rating         string
	Length         time.Duration
	Writers        []string
	Editor         string
	Director       string
	Cinematography string
	Producers      oneOrMany
	Music          string
	Production     oneOrMany
	Distributor    string
	Characters     []*Character

	// unknown fields are stored in the Extra map
	Extra map[string]interface{} `yaml:",inline"`

	References []Reference `yaml:"refs"`

	Image *Media
}

// Contents represents a list of contents, where key is a file path.
// It is used to properly render references.
type Contents map[string]Content

// Connections represents a list of connectiones that initiated by a reference.
// Key is a file path, where reference is pointing to.
// Value is a map, where key is a file path, where reference is located,
// and value is a type of reference.
// For example, three files "A", "B" and "C" are referencing to file "D",
// but in different contexts. File "A" just has a reference to file "D",
// file "B" has a reference "D" as an "Author",
// and file "C" has a reference "D" as a "Voice" for "Bob" (presumably, a character).
// Then the Connections map will look like this:
//
//	{
//		"D": {
//			"A": []
//			"B": ["Auhor"]
//			"C": ["Voice", "Bob"],
//		}
//	}
type Connections map[string]map[string][]string

// File represents a file or directory in the file system.
type File struct {
	Name     string
	Title    string // value from YAML "name" field, may contain colons
	IsFolder bool   // used to render folder icon and to sort files
	Image    *Media
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
