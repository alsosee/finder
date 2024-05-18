package structs

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// oneOrMany represents a list of strings that can be passed as a single string in YAML.
type oneOrMany []string

// UnmarshalYAML makes oneOrMany support both a string and a list of strings.
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
	Name        string    `yaml:"имя,omitempty" json:",omitempty"`      // name of the file, used in the breadcrumbs
	Title       string    `yaml:"название,omitempty" json:",omitempty"` // override for the name, used as page title, fallback to Name
	Subtitle    string    `yaml:"подзаголовок,omitempty" json:",omitempty"`
	Year        int       `yaml:",omitempty" json:",omitempty"`
	Authors     oneOrMany `yaml:",omitempty" json:",omitempty"`
	Developers  string    `yaml:",omitempty" json:",omitempty"`
	Description string    `yaml:"описание,omitempty" json:",omitempty"`
	CoverArtist string    `yaml:"cover_artist,omitempty" json:",omitempty"`
	Designer    string    `yaml:",omitempty" json:",omitempty"`

	BasedOn  oneOrMany `yaml:"based_on,omitempty" json:",omitempty"`
	Series   string    `yaml:",omitempty" json:",omitempty"`
	Previous string    `yaml:",omitempty" json:",omitempty"` // reference to previous in the series

	// for people
	DOB     string `yaml:",omitempty" json:",omitempty"` // date of birth
	DOD     string `yaml:",omitempty" json:",omitempty"` // date of death
	Contact string `yaml:"contact,omitempty" json:",omitempty"`
	Nick    string `yaml:"ник,omitempty" json:",omitempty"`

	Parent   string    `yaml:",omitempty" json:",omitempty"` // for companies
	Founded  string    `yaml:",omitempty" json:",omitempty"` // for companies
	Founders oneOrMany `yaml:",omitempty" json:",omitempty"` // for companies
	Released string    `yaml:",omitempty" json:",omitempty"` // for games, ...

	// general external links
	Website          string   `yaml:",omitempty" json:",omitempty"`
	Websites         []string `yaml:",omitempty" json:",omitempty"`
	RSS              string   `yaml:",omitempty" json:",omitempty"`
	Wikipedia        string   `yaml:",omitempty" json:",omitempty"`
	GoodReads        string   `yaml:",omitempty" json:",omitempty"`
	Bookshop         string   `yaml:",omitempty" json:",omitempty"`
	AnimeNewsNetwork string   `yaml:"anime_news_network,omitempty" json:",omitempty"`
	Twitch           string   `yaml:",omitempty" json:",omitempty"`
	YouTube          string   `yaml:",omitempty" json:",omitempty"`
	Vimeo            string   `yaml:",omitempty" json:",omitempty"`
	IMDB             string   `yaml:",omitempty" json:",omitempty"`
	TMDB             string   `yaml:",omitempty" json:",omitempty"`
	TPDB             string   `yaml:",omitempty" json:",omitempty"`
	Steam            string   `yaml:",omitempty" json:",omitempty"`
	Netflix          string   `yaml:",omitempty" json:",omitempty"`
	Spotify          string   `yaml:",omitempty" json:",omitempty"`
	Soundcloud       string   `yaml:",omitempty" json:",omitempty"`
	Hulu             string   `yaml:",omitempty" json:",omitempty"`
	AdultSwim        string   `yaml:",omitempty" json:",omitempty"`
	AppStore         string   `yaml:"app_store,omitempty" json:",omitempty"`
	Fandom           string   `yaml:",omitempty" json:",omitempty"`
	RottenTomatoes   string   `yaml:"rotten_tomatoes,omitempty" json:",omitempty"`
	Metacritic       string   `yaml:",omitempty" json:",omitempty"`
	GitHub           string   `yaml:",omitempty" json:",omitempty"`
	Twitter          string   `yaml:",omitempty" json:",omitempty"`
	Reddit           string   `yaml:",omitempty" json:",omitempty"`
	Facebook         string   `yaml:",omitempty" json:",omitempty"`
	Instagram        string   `yaml:",omitempty" json:",omitempty"`
	Threads          string   `yaml:",omitempty" json:",omitempty"`
	LinkedIn         string   `yaml:"linkedin,omitempty" json:",omitempty"`
	TikTok           string   `yaml:",omitempty" json:",omitempty"`
	TelegramChannel  string   `yaml:"telegram_channel,omitempty" json:",omitempty"`
	TelegramChat     string   `yaml:"telegram_chat,omitempty" json:",omitempty"`
	Mave             string   `yaml:",omitempty" json:",omitempty"`
	Bento            string   `yaml:",omitempty" json:",omitempty"`
	PlayStation      string   `yaml:"playstation,omitempty" json:",omitempty"`
	XBox             string   `yaml:"xbox,omitempty" json:",omitempty"`
	GOG              string   `yaml:"gog,omitempty" json:",omitempty"`
	X                string   `yaml:",omitempty" json:",omitempty"`
	Discord          string   `yaml:",omitempty" json:",omitempty"`
	Epic             string   `yaml:",omitempty" json:",omitempty"`
	IGN              string   `yaml:"ign,omitempty" json:",omitempty"`
	Amazon           string   `yaml:",omitempty" json:",omitempty"`
	PrimeVideo       string   `yaml:"prime_video,omitempty" json:",omitempty"`
	AppleTV          string   `yaml:"apple_tv,omitempty" json:",omitempty"`
	ApplePodcasts    string   `yaml:"apple_podcasts,omitempty" json:",omitempty"`
	GooglePodcasts   string   `yaml:"google_podcasts,omitempty" json:",omitempty"`
	YandexMusic      string   `yaml:"yandex_music,omitempty" json:",omitempty"`
	Boosty           string   `yaml:",omitempty" json:",omitempty"`
	Patreon          string   `yaml:",omitempty" json:",omitempty"`
	Donatty          string   `yaml:",omitempty" json:",omitempty"`
	Destream         string   `yaml:",omitempty" json:",omitempty"`
	Mastodon         string   `yaml:",omitempty" json:",omitempty"`
	Bluesky          string   `yaml:",omitempty" json:",omitempty"`
	DTF              string   `yaml:",omitempty" json:",omitempty"`
	Peacock          string   `yaml:",omitempty" json:",omitempty"`
	GooglePlay       string   `yaml:"google_play,omitempty" json:",omitempty"`
	MicrosoftStore   string   `yaml:"microsoft_store,omitempty" json:",omitempty"`
	Nintendo         string   `yaml:",omitempty" json:",omitempty"`
	HumbleBundle     string   `yaml:"humble_bundle,omitempty" json:",omitempty"`
	Row8             string   `yaml:",omitempty" json:",omitempty"`
	Redbox           string   `yaml:",omitempty" json:",omitempty"`
	Vudu             string   `yaml:",omitempty" json:",omitempty"`
	DarkHorse        string   `yaml:",omitempty" json:",omitempty"`
	VK               string   `yaml:",omitempty" json:",omitempty"`
	Unsplash         string   `yaml:",omitempty" json:",omitempty"`
	Medium           string   `yaml:",omitempty" json:",omitempty"`
	Kinopoisk        string   `yaml:",omitempty" json:",omitempty"`
	PrevouslyKnownAs string   `yaml:"ранее_известен_как,omitempty" json:",omitempty"`

	// for books
	ISBN        string    `yaml:",omitempty" json:",omitempty"`
	ISBN10      string    `yaml:",omitempty" json:",omitempty"`
	ISBN13      string    `yaml:",omitempty" json:",omitempty"`
	OCLC        string    `yaml:",omitempty" json:",omitempty"`
	Publishers  oneOrMany `yaml:",omitempty" json:",omitempty"`
	Publication string    `yaml:",omitempty" json:",omitempty"` // date or year of publication

	// for comics
	Artists      oneOrMany `yaml:",omitempty" json:",omitempty"`
	Colorist     string    `yaml:",omitempty" json:",omitempty"`
	Illustrators oneOrMany `yaml:",omitempty" json:",omitempty"`
	Imprint      string    `yaml:",omitempty" json:",omitempty"`
	UPC          string    `yaml:",omitempty" json:",omitempty"`

	// for movies, games, series, ...
	Genres         []string      `yaml:",omitempty" json:",omitempty"`
	Engine         string        `yaml:",omitempty" json:",omitempty"`
	Trailer        string        `yaml:",omitempty" json:",omitempty"`
	Rating         string        `yaml:",omitempty" json:",omitempty"`
	Length         time.Duration `yaml:",omitempty" json:",omitempty"`
	Creators       oneOrMany     `yaml:",omitempty" json:",omitempty"`
	Writers        oneOrMany     `yaml:",omitempty" json:",omitempty"`
	Editors        oneOrMany     `yaml:",omitempty" json:",omitempty"`
	Directors      oneOrMany     `yaml:",omitempty" json:",omitempty"`
	Cinematography oneOrMany     `yaml:",omitempty" json:",omitempty"`
	Producers      oneOrMany     `yaml:",omitempty" json:",omitempty"`
	Screenplay     oneOrMany     `yaml:",omitempty" json:",omitempty"`
	StoryBy        oneOrMany     `yaml:"story_by,omitempty" json:",omitempty"`
	DialoguesBy    oneOrMany     `yaml:"dialogues_by,omitempty" json:",omitempty"`
	Music          oneOrMany     `yaml:",omitempty" json:",omitempty"`
	Production     oneOrMany     `yaml:",omitempty" json:",omitempty"`
	Distributors   oneOrMany     `yaml:",omitempty" json:",omitempty"`
	Network        string        `yaml:",omitempty" json:",omitempty"`
	Composers      oneOrMany     `yaml:",omitempty" json:",omitempty"`
	Programmers    oneOrMany     `yaml:",omitempty" json:",omitempty"`
	Designers      oneOrMany     `yaml:",omitempty" json:",omitempty"`

	// for podcasts
	Hosts  oneOrMany `yaml:"ведущие,omitempty" json:",omitempty"`
	Guests oneOrMany `yaml:"гости,omitempty" json:",omitempty"`

	From string `yaml:"от,omitempty" json:",omitempty"`

	RemakeOf string `yaml:"remake_of,omitempty" json:",omitempty"`

	Characters []*Character `yaml:",omitempty" json:",omitempty"`

	// for awards
	Categories []Category `yaml:",omitempty" json:",omitempty"`

	// unknown fields are stored in the Extra map
	Extra map[string]interface{} `yaml:",inline" json:",omitempty"`

	References []Reference `yaml:"refs,omitempty" json:",omitempty"`

	// fields populated by the generator
	Image                *Media  `yaml:"-" json:",omitempty"`
	Awards               []Award `yaml:"-" json:",omitempty"`
	EditorsAwards        []Award `yaml:"-" json:",omitempty"`
	WritersAwards        []Award `yaml:"-" json:",omitempty"`
	DirectorsAwards      []Award `yaml:"-" json:",omitempty"`
	CinematographyAwards []Award `yaml:"-" json:",omitempty"`
	MusicAwards          []Award `yaml:"-" json:",omitempty"`
	ScreenplayAwards     []Award `yaml:"-" json:",omitempty"`
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
	Reference      string    `yaml:"ref,omitempty" json:",omitempty"` // full path to referenced content
	Movie          string    `yaml:",omitempty" json:",omitempty"`
	Game           string    `yaml:",omitempty" json:",omitempty"`
	Series         string    `yaml:",omitempty" json:",omitempty"`
	Person         string    `yaml:",omitempty" json:",omitempty"`
	Actor          string    `yaml:",omitempty" json:",omitempty"`
	Editors        oneOrMany `yaml:",omitempty" json:",omitempty"`
	Track          string    `yaml:",omitempty" json:",omitempty"`
	Directors      oneOrMany `yaml:",omitempty" json:",omitempty"`
	Writers        oneOrMany `yaml:",omitempty" json:",omitempty"`
	Cinematography oneOrMany `yaml:",omitempty" json:",omitempty"`
	Music          oneOrMany `yaml:",omitempty" json:",omitempty"`
	Screenplay     oneOrMany `yaml:",omitempty" json:",omitempty"`
	Producers      oneOrMany `yaml:",omitempty" json:",omitempty"`
	Casting        oneOrMany `yaml:",omitempty" json:",omitempty"`
	ConstumeDesign oneOrMany `yaml:",omitempty" json:",omitempty"`
	MakeUpAndHair  oneOrMany `yaml:",omitempty" json:",omitempty"`

	Fallback string `yaml:"-" json:"-,omitempty"` // used to store the fallback value for template
}
