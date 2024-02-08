package structs

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

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
	ID     string `yaml:"-"`                   // used by Search
	Source string `yaml:"-"`                   // path to the file
	HTML   string `yaml:"-" json:",omitempty"` // for Markdown files

	// for everything
	Name        string    `json:",omitempty"` // name of the file, used in the breadcrumbs
	Title       string    `json:",omitempty"` // override for the name, used as page title, fallback to Name
	Subtitle    string    `json:",omitempty"`
	Year        int       `json:",omitempty"`
	Author      string    `json:",omitempty"`
	Authors     oneOrMany `json:",omitempty"`
	Developers  string    `json:",omitempty"`
	Description string    `json:",omitempty"`
	CoverArtist string    `yaml:"cover_artist" json:",omitempty"`
	Designer    string    `json:",omitempty"`

	BasedOn  oneOrMany `yaml:"based_on,omitempty" json:",omitempty"`
	Series   string    `json:",omitempty"`
	Previous string    `json:",omitempty"` // reference to previous in the series

	// for people
	DOB     string `json:",omitempty"` // date of birth
	DOD     string `json:",omitempty"` // date of death
	Contact string `yaml:"contact" json:",omitempty"`

	Founded  string `json:",omitempty"` // for companies
	Released string `json:",omitempty"` // for games, ...

	// general external links
	Website         string   `json:",omitempty"`
	Websites        []string `json:",omitempty"`
	Wikipedia       string   `json:",omitempty"`
	GoodReads       string   `json:",omitempty"`
	Bookshop        string   `json:",omitempty"`
	Twitch          string   `json:",omitempty"`
	YouTube         string   `json:",omitempty"`
	IMDB            string   `json:",omitempty"`
	TMDB            string   `json:",omitempty"`
	Steam           string   `json:",omitempty"`
	Netflix         string   `json:",omitempty"`
	Spotify         string   `json:",omitempty"`
	Soundcloud      string   `json:",omitempty"`
	Hulu            string   `json:",omitempty"`
	AdultSwim       string   `json:",omitempty"`
	AppStore        string   `yaml:"app_store" json:",omitempty"`
	Fandom          string   `json:",omitempty"`
	RottenTomatoes  string   `yaml:"rotten_tomatoes" json:",omitempty"`
	Twitter         string   `json:",omitempty"`
	Reddit          string   `json:",omitempty"`
	Facebook        string   `json:",omitempty"`
	Instagram       string   `json:",omitempty"`
	TikTok          string   `json:",omitempty"`
	TelegramChannel string   `yaml:"telegram_channel" json:",omitempty"`
	PlayStation     string   `yaml:"playstation" json:",omitempty"`
	XBox            string   `yaml:"xbox" json:",omitempty"`
	GOG             string   `yaml:"gog" json:",omitempty"`
	X               string   `json:",omitempty"`
	Discord         string   `json:",omitempty"`
	Epic            string   `json:",omitempty"`
	IGN             string   `yaml:"ign" json:",omitempty"`
	Amazon          string   `json:",omitempty"`
	PrimeVideo      string   `yaml:"prime_video" json:",omitempty"`
	AppleTV         string   `yaml:"apple_tv" json:",omitempty"`
	Peacock         string   `json:",omitempty"`
	GooglePlay      string   `yaml:"google_play" json:",omitempty"`
	MicrosoftStore  string   `yaml:"microsoft_store" json:",omitempty"`
	Row8            string   `json:",omitempty"`
	Redbox          string   `json:",omitempty"`
	Vudu            string   `json:",omitempty"`

	// for books
	ISBN        string    `json:",omitempty"`
	ISBN10      string    `json:",omitempty"`
	ISBN13      string    `json:",omitempty"`
	OCLC        string    `json:",omitempty"`
	Publishers  oneOrMany `json:",omitempty"`
	Publication string    `json:",omitempty"` // date or year of publication

	// for comics
	Artists      oneOrMany `json:",omitempty"`
	Colorist     string    `json:",omitempty"`
	Illustrators oneOrMany `json:",omitempty"`
	Imprint      string    `json:",omitempty"`
	UPC          string    `json:",omitempty"`

	// for movies
	Genres         []string      `json:",omitempty"`
	Trailer        string        `json:",omitempty"`
	Rating         string        `json:",omitempty"`
	Length         time.Duration `json:",omitempty"`
	Creators       oneOrMany     `json:",omitempty"`
	Writers        oneOrMany     `json:",omitempty"`
	Editor         string        `json:",omitempty"`
	Directors      oneOrMany     `json:",omitempty"`
	Cinematography string        `json:",omitempty"`
	Producers      oneOrMany     `json:",omitempty"`
	Music          string        `json:",omitempty"`
	Production     oneOrMany     `json:",omitempty"`
	Distributor    string        `json:",omitempty"`
	Network        string        `json:",omitempty"`
	Characters     []*Character  `json:",omitempty"`

	// for awards
	Categories []Category `json:",omitempty"`

	// unknown fields are stored in the Extra map
	Extra map[string]interface{} `yaml:",inline" json:",omitempty"`

	References []Reference `yaml:"refs" json:",omitempty"`

	// fields populated by the generator
	Image           *Media  `yaml:"-" json:",omitempty"`
	Awards          []Award `yaml:"-" json:",omitempty"`
	EditorAwards    []Award `yaml:"-" json:",omitempty"`
	WritersAwards   []Award `yaml:"-" json:",omitempty"`
	DirectorsAwards []Award `yaml:"-" json:",omitempty"`
}

// Character represents a character in a movie, tv show, etc.
type Character struct {
	Name       string
	Actor      string `json:",omitempty"`
	Voice      string `json:",omitempty"`
	Image      *Media `json:",omitempty"`
	ActorImage *Media `json:",omitempty"`

	// populated by the generator
	Awards []Award `yml:"-" json:",omitempty"`
}

type Award struct {
	Category  string `json:",omitempty"`
	Reference string `json:",omitempty"` // who gave the award
}

type Category struct {
	Name   string `json:",omitempty"`
	Winner Winner `json:",omitempty"`
}

type Winner struct {
	Reference string    `yaml:"ref" json:",omitempty"` // full path to referenced content
	Movie     string    `json:",omitempty"`
	Game      string    `json:",omitempty"`
	Series    string    `json:",omitempty"`
	Person    string    `json:",omitempty"`
	Actor     string    `json:",omitempty"`
	Editor    string    `json:",omitempty"`
	Track     string    `json:",omitempty"`
	Directors oneOrMany `json:",omitempty"`
	Writers   oneOrMany `json:",omitempty"`

	Fallback string `yaml:"-" json:"-,omitempty"` // used to store the fallback value for template
}
