package structs

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
	Authors     oneOrMany
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
	Soundcloud      string
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
	Amazon          string
	AppleTV         string `yaml:"apple_tv"`
	GooglePlay      string `yaml:"google_play"`
	MicrosoftStore  string `yaml:"microsoft_store"`
	Row8            string
	Redbox          string
	Vudu            string

	// for books
	ISBN        string
	ISBN10      string
	ISBN13      string
	OCLC        string
	Publisher   string
	Publication string // date or year of publication

	// for comics
	Artists  oneOrMany
	Colorist string
	UPC      string

	// for movies
	Genres         []string
	Trailer        string
	Rating         string
	Length         time.Duration
	Writers        oneOrMany
	Editor         string
	Directors      oneOrMany
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
