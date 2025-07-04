// This file is automatically generated. Do not edit.
// To update this file, run `make codegen` (assumes that `info` repository is cloned in the same parent directory).
// Template: codegen/content.tmpl
// Schema source: https://github.com/alsosee/info/blob/main/_finder/schema.yml
package structs

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var RootTypes = map[string]string{
	"People":    "person",
	"Books":     "book",
	"Games":     "game",
	"Movies":    "movie",
	"Companies": "company",
	"Podcasts":  "podcast",
}

type Column struct {
	Name       string // used to lookup property in search hits response
	Title      string // used for column name in UI
	Type       string // used to conditionally convert "duration" value from search hits response into human readable format
	AlwaysShow bool   // used when choosing columns for search results
}

var ColumnsList = []Column{
	{
		Name:       "dob",
		Title:      "Born",
		Type:       "string",
		AlwaysShow: false,
	},
	{
		Name:       "dod",
		Title:      "Died",
		Type:       "string",
		AlwaysShow: true,
	},
	{
		Name:       "publishers",
		Title:      "Publishers",
		Type:       "array",
		AlwaysShow: false,
	},
	{
		Name:       "length",
		Title:      "Length",
		Type:       "duration",
		AlwaysShow: false,
	},
	{
		Name:       "directors",
		Title:      "Directors",
		Type:       "array",
		AlwaysShow: false,
	},
	{
		Name:       "writers",
		Title:      "Writers",
		Type:       "array",
		AlwaysShow: false,
	},
	{
		Name:       "distributors",
		Title:      "Distributors",
		Type:       "array",
		AlwaysShow: false,
	},
	{
		Name:       "manufacturers",
		Title:      "Manufacturers",
		Type:       "array",
		AlwaysShow: false,
	},
	{
		Name:       "rating",
		Title:      "Rating",
		Type:       "string",
		AlwaysShow: false,
	},
	{
		Name:       "released",
		Title:      "Released",
		Type:       "string",
		AlwaysShow: false,
	},
	{
		Name:       "network",
		Title:      "Network",
		Type:       "company",
		AlwaysShow: false,
	},
	{
		Name:       "creators",
		Title:      "Creators",
		Type:       "array",
		AlwaysShow: false,
	},
	{
		Name:       "developed_by",
		Title:      "DevelopedBy",
		Type:       "array",
		AlwaysShow: false,
	},
	{
		Name:       "showrunners",
		Title:      "Showrunners",
		Type:       "array",
		AlwaysShow: false,
	},
	{
		Name:       "authors",
		Title:      "Authors",
		Type:       "array",
		AlwaysShow: false,
	},
	{
		Name:       "developers",
		Title:      "Developers",
		Type:       "array",
		AlwaysShow: false,
	},
	{
		Name:       "screenplay",
		Title:      "Screenplay",
		Type:       "array",
		AlwaysShow: false,
	},
	{
		Name:       "story_by",
		Title:      "StoryBy",
		Type:       "array",
		AlwaysShow: false,
	},
	{
		Name:       "dialogues_by",
		Title:      "DialoguesBy",
		Type:       "array",
		AlwaysShow: false,
	},
	{
		Name:       "hosts",
		Title:      "Hosts",
		Type:       "array",
		AlwaysShow: false,
	},
}

// Content represents the content of a file.
type Content struct {
	ID                string        `yaml:"-"`                   // used by Search
	Source            string        `yaml:"-"`                   // path to the file
	SourceNoExtention string        `yaml:"-"`                   // path to the file without extention
	HTML              string        `yaml:"-" json:",omitempty"` // for Markdown files
	Name              string        `yaml:"name,omitempty" json:"name,omitempty"`
	Title             string        `yaml:"title,omitempty" json:"title,omitempty"`
	Image             *Media        `yaml:"image,omitempty" json:"image,omitempty"`
	Subtitle          string        `yaml:"subtitle,omitempty" json:"subtitle,omitempty"`
	Description       string        `yaml:"description,omitempty" json:"description,omitempty"`
	CoverArtist       string        `yaml:"cover_artist,omitempty" json:"cover_artist,omitempty"`
	Designer          string        `yaml:"designer,omitempty" json:"designer,omitempty"`
	BasedOn           References    `yaml:"based_on,omitempty" json:"based_on,omitempty"`
	Series            string        `yaml:"series,omitempty" json:"series,omitempty"`
	Previous          *Reference    `yaml:"previous,omitempty" json:"previous,omitempty"`
	DOB               string        `yaml:"dob,omitempty" json:"dob,omitempty"`
	DOD               string        `yaml:"dod,omitempty" json:"dod,omitempty"`
	Contact           string        `yaml:"contact,omitempty" json:"contact,omitempty"`
	Parent            string        `yaml:"parent,omitempty" json:"parent,omitempty"`
	Founded           string        `yaml:"founded,omitempty" json:"founded,omitempty"`
	Founders          oneOrMany     `yaml:"founders,omitempty" json:"founders,omitempty"`
	Website           *Link         `yaml:"website,omitempty" json:"website,omitempty"`
	Websites          Links         `yaml:"websites,omitempty" json:"websites,omitempty"`
	Wikipedia         string        `yaml:"wikipedia,omitempty" json:"wikipedia,omitempty"`
	GoodReads         string        `yaml:"goodreads,omitempty" json:"goodreads,omitempty"`
	Bookshop          string        `yaml:"bookshop,omitempty" json:"bookshop,omitempty"`
	AnimeNewsNetwork  string        `yaml:"anime_news_network,omitempty" json:"anime_news_network,omitempty"`
	Twitch            string        `yaml:"twitch,omitempty" json:"twitch,omitempty"`
	YouTube           string        `yaml:"youtube,omitempty" json:"youtube,omitempty"`
	Vimeo             string        `yaml:"vimeo,omitempty" json:"vimeo,omitempty"`
	IMDB              string        `yaml:"imdb,omitempty" json:"imdb,omitempty"`
	TMDB              string        `yaml:"tmdb,omitempty" json:"tmdb,omitempty"`
	TPDB              string        `yaml:"tpdb,omitempty" json:"tpdb,omitempty"`
	Steam             string        `yaml:"steam,omitempty" json:"steam,omitempty"`
	Netflix           string        `yaml:"netflix,omitempty" json:"netflix,omitempty"`
	Spotify           string        `yaml:"spotify,omitempty" json:"spotify,omitempty"`
	Soundcloud        string        `yaml:"soundcloud,omitempty" json:"soundcloud,omitempty"`
	Hulu              string        `yaml:"hulu,omitempty" json:"hulu,omitempty"`
	Max               string        `yaml:"max,omitempty" json:"max,omitempty"`
	AdultSwim         string        `yaml:"adult_swim,omitempty" json:"adult_swim,omitempty"`
	AppStore          string        `yaml:"app_store,omitempty" json:"app_store,omitempty"`
	Fandom            string        `yaml:"fandom,omitempty" json:"fandom,omitempty"`
	RottenTomatoes    string        `yaml:"rotten_tomatoes,omitempty" json:"rotten_tomatoes,omitempty"`
	Metacritic        string        `yaml:"metacritic,omitempty" json:"metacritic,omitempty"`
	Opencritic        string        `yaml:"opencritic,omitempty" json:"opencritic,omitempty"`
	Twitter           string        `yaml:"twitter,omitempty" json:"twitter,omitempty"`
	Mastodon          string        `yaml:"mastodon,omitempty" json:"mastodon,omitempty"`
	Reddit            string        `yaml:"reddit,omitempty" json:"reddit,omitempty"`
	Facebook          string        `yaml:"facebook,omitempty" json:"facebook,omitempty"`
	Instagram         string        `yaml:"instagram,omitempty" json:"instagram,omitempty"`
	Threads           string        `yaml:"threads,omitempty" json:"threads,omitempty"`
	LinkedIn          string        `yaml:"linkedin,omitempty" json:"linkedin,omitempty"`
	TikTok            string        `yaml:"tiktok,omitempty" json:"tiktok,omitempty"`
	TelegramChannel   string        `yaml:"telegram_channel,omitempty" json:"telegram_channel,omitempty"`
	PlayStation       string        `yaml:"playstation,omitempty" json:"playstation,omitempty"`
	XBox              string        `yaml:"xbox,omitempty" json:"xbox,omitempty"`
	GOG               string        `yaml:"gog,omitempty" json:"gog,omitempty"`
	X                 string        `yaml:"x,omitempty" json:"x,omitempty"`
	Discord           string        `yaml:"discord,omitempty" json:"discord,omitempty"`
	Epic              string        `yaml:"epic,omitempty" json:"epic,omitempty"`
	IGN               string        `yaml:"ign,omitempty" json:"ign,omitempty"`
	Amazon            string        `yaml:"amazon,omitempty" json:"amazon,omitempty"`
	PrimeVideo        string        `yaml:"prime_video,omitempty" json:"prime_video,omitempty"`
	AppleTV           string        `yaml:"apple_tv,omitempty" json:"apple_tv,omitempty"`
	ApplePodcasts     string        `yaml:"apple_podcasts,omitempty" json:"apple_podcasts,omitempty"`
	AppleBooks        string        `yaml:"apple_books,omitempty" json:"apple_books,omitempty"`
	Peacock           string        `yaml:"peacock,omitempty" json:"peacock,omitempty"`
	GooglePlay        string        `yaml:"google_play,omitempty" json:"google_play,omitempty"`
	DisneyPlus        string        `yaml:"disney_plus,omitempty" json:"disney_plus,omitempty"`
	MicrosoftStore    string        `yaml:"microsoft_store,omitempty" json:"microsoft_store,omitempty"`
	Nintendo          string        `yaml:"nintendo,omitempty" json:"nintendo,omitempty"`
	HumbleBundle      string        `yaml:"humble_bundle,omitempty" json:"humble_bundle,omitempty"`
	Row8              string        `yaml:"row8,omitempty" json:"row8,omitempty"`
	Redbox            string        `yaml:"redbox,omitempty" json:"redbox,omitempty"`
	Vudu              string        `yaml:"vudu,omitempty" json:"vudu,omitempty"`
	DarkHorse         string        `yaml:"darkhorse,omitempty" json:"darkhorse,omitempty"`
	Kickstarter       string        `yaml:"kickstarter,omitempty" json:"kickstarter,omitempty"`
	ISBN              string        `yaml:"isbn,omitempty" json:"isbn,omitempty"`
	ISBN10            string        `yaml:"isbn10,omitempty" json:"isbn10,omitempty"`
	ISBN13            string        `yaml:"isbn13,omitempty" json:"isbn13,omitempty"`
	OCLC              string        `yaml:"oclc,omitempty" json:"oclc,omitempty"`
	Publishers        oneOrMany     `yaml:"publishers,omitempty" json:"publishers,omitempty"`
	Publication       string        `yaml:"publication,omitempty" json:"publication,omitempty"`
	Artists           oneOrMany     `yaml:"artists,omitempty" json:"artists,omitempty"`
	Colorist          string        `yaml:"colorist,omitempty" json:"colorist,omitempty"`
	Illustrators      oneOrMany     `yaml:"illustrators,omitempty" json:"illustrators,omitempty"`
	Imprint           string        `yaml:"imprint,omitempty" json:"imprint,omitempty"`
	UPC               string        `yaml:"upc,omitempty" json:"upc,omitempty"`
	Genres            []string      `yaml:"genres,omitempty" json:"genres,omitempty"`
	Length            time.Duration `yaml:"length,omitempty" json:"length,omitempty"`
	Directors         oneOrMany     `yaml:"directors,omitempty" json:"directors,omitempty"`
	Writers           oneOrMany     `yaml:"writers,omitempty" json:"writers,omitempty"`
	Distributors      oneOrMany     `yaml:"distributors,omitempty" json:"distributors,omitempty"`
	Manufacturers     oneOrMany     `yaml:"manufacturers,omitempty" json:"manufacturers,omitempty"`
	Rating            string        `yaml:"rating,omitempty" json:"rating,omitempty"`
	Released          string        `yaml:"released,omitempty" json:"released,omitempty"`
	Network           string        `yaml:"network,omitempty" json:"network,omitempty"`
	Engine            string        `yaml:"engine,omitempty" json:"engine,omitempty"`
	Creators          oneOrMany     `yaml:"creators,omitempty" json:"creators,omitempty"`
	DevelopedBy       oneOrMany     `yaml:"developed_by,omitempty" json:"developed_by,omitempty"`
	Showrunners       oneOrMany     `yaml:"showrunners,omitempty" json:"showrunners,omitempty"`
	Authors           oneOrMany     `yaml:"authors,omitempty" json:"authors,omitempty"`
	Developers        oneOrMany     `yaml:"developers,omitempty" json:"developers,omitempty"`
	Trailer           string        `yaml:"trailer,omitempty" json:"trailer,omitempty"`
	Editors           oneOrMany     `yaml:"editors,omitempty" json:"editors,omitempty"`
	Cinematography    oneOrMany     `yaml:"cinematography,omitempty" json:"cinematography,omitempty"`
	Producers         oneOrMany     `yaml:"producers,omitempty" json:"producers,omitempty"`
	Screenplay        oneOrMany     `yaml:"screenplay,omitempty" json:"screenplay,omitempty"`
	StoryBy           oneOrMany     `yaml:"story_by,omitempty" json:"story_by,omitempty"`
	DialoguesBy       oneOrMany     `yaml:"dialogues_by,omitempty" json:"dialogues_by,omitempty"`
	Music             oneOrMany     `yaml:"music,omitempty" json:"music,omitempty"`
	Production        oneOrMany     `yaml:"production,omitempty" json:"production,omitempty"`
	Composers         oneOrMany     `yaml:"composers,omitempty" json:"composers,omitempty"`
	Programmers       oneOrMany     `yaml:"programmers,omitempty" json:"programmers,omitempty"`
	Designers         oneOrMany     `yaml:"designers,omitempty" json:"designers,omitempty"`
	Hosts             oneOrMany     `yaml:"hosts,omitempty" json:"hosts,omitempty"`
	Guests            oneOrMany     `yaml:"guests,omitempty" json:"guests,omitempty"`
	RemakeOf          *Reference    `yaml:"remake_of,omitempty" json:"remake_of,omitempty"`
	Characters        []*Character  `yaml:"characters,omitempty" json:"characters,omitempty"`
	Categories        []Category    `yaml:"categories,omitempty" json:"categories,omitempty"`
	References        References    `yaml:"references,omitempty" json:"references,omitempty"`
	Episodes          []*Episode    `yaml:"episodes,omitempty" json:"episodes,omitempty"`

	// unknown fields are stored in the Extra map
	Extra map[string]interface{} `yaml:",inline" json:",omitempty"`

	// fields populated by the generator
	Awards               []Award `yaml:"-" json:",omitempty"`
	EditorsAwards        []Award `yaml:"-" json:",omitempty"`
	WritersAwards        []Award `yaml:"-" json:",omitempty"`
	DirectorsAwards      []Award `yaml:"-" json:",omitempty"`
	CinematographyAwards []Award `yaml:"-" json:",omitempty"`
	MusicAwards          []Award `yaml:"-" json:",omitempty"`
	ScreenplayAwards     []Award `yaml:"-" json:",omitempty"`
}

// GenerateID generates an ID for the content.
// Used for identifying the content in connections and search.
func (c *Content) GenerateID() string {
	if c.ID != "" {
		return c.ID
	}

	c.SourceNoExtention = removeFileExtention(c.Source)
	c.ID = formatID(c.SourceNoExtention)

	return c.ID
}

// Type return a type of the content in singular form
// (e.g. "person" for "People", "book" for "Books", etc.)
// it used to add an additional context to reference link
// when current page and the reference have the same name
func (c Content) Type() string {
	// get first part of the Source path
	// (e.g. "People" or "Book")
	root := pathType(c.Source)
	switch root {
	case "People":
		return "person"
	case "Books":
		return "book"
	case "Games":
		return "game"
	case "Movies":
		return "movie"
	case "Companies":
		return "company"
	case "Podcasts":
		return "podcast"
	default:
		return strings.ToLower(root)
	}
}

// Header returns a string to be displayed in the header of the content.
// Title is used by default, Name is a fallback.
func (c Content) Header() string {
	if c.Title != "" {
		return c.Title
	}
	return c.Name
}

func (c *Content) SetName(name string) {
	c.Name = name
}

func (c Content) GetName() string {
	return c.Name
}

// Columns defines the columns to be displayed in the List view.
func (c Content) Columns() map[string]string {
	return map[string]string{
		"Born":          c.DOB,
		"Died":          c.DOD,
		"Publishers":    strings.Join(c.Publishers, ", "),
		"Length":        length(c.Length),
		"Directors":     strings.Join(c.Directors, ", "),
		"Writers":       strings.Join(c.Writers, ", "),
		"Distributors":  strings.Join(c.Distributors, ", "),
		"Manufacturers": strings.Join(c.Manufacturers, ", "),
		"Rating":        c.Rating,
		"Released":      c.Released,
		"Network":       c.Network,
		"Creators":      strings.Join(c.Creators, ", "),
		"DevelopedBy":   strings.Join(c.DevelopedBy, ", "),
		"Showrunners":   strings.Join(c.Showrunners, ", "),
		"Authors":       strings.Join(c.Authors, ", "),
		"Developers":    strings.Join(c.Developers, ", "),
		"Screenplay":    strings.Join(c.Screenplay, ", "),
		"StoryBy":       strings.Join(c.StoryBy, ", "),
		"DialoguesBy":   strings.Join(c.DialoguesBy, ", "),
		"Hosts":         strings.Join(c.Hosts, ", "),
	}
}

// Connections returns a list of connections to other content.
func (c Content) Connections() []Connection {
	var connections []Connection

	if c.CoverArtist != "" {
		connections = append(connections, Connection{
			To:    "People/" + c.CoverArtist,
			Label: "Cover artist",
		})
	}
	if c.Designer != "" {
		connections = append(connections, Connection{
			To:    "People/" + c.Designer,
			Label: "Designer",
		})
	}
	for _, reference := range c.BasedOn {
		connections = append(connections, Connection{
			To:    reference.Path,
			Label: "Source",
		})
	}
	if c.Series != "" {
		connections = append(connections, Connection{
			To:   c.Series,
			Meta: "series",
		})
	}
	if c.Previous != nil {
		connections = append(connections, Connection{
			To:    c.Previous.Path,
			Label: "",
			Meta:  "previous",
		})
	}
	for _, person := range c.Founders {
		connections = append(connections, Connection{
			To:    "People/" + person,
			Label: "Founder",
		})
	}
	for _, company := range c.Publishers {
		connections = append(connections, Connection{
			To:    "Companies/" + company,
			Label: "Publishers",
		})
	}
	for _, person := range c.Artists {
		connections = append(connections, Connection{
			To:    "People/" + person,
			Label: "Artist",
		})
	}
	if c.Colorist != "" {
		connections = append(connections, Connection{
			To:    "People/" + c.Colorist,
			Label: "Colorist",
		})
	}
	for _, person := range c.Illustrators {
		connections = append(connections, Connection{
			To:    "People/" + person,
			Label: "Illustrator",
		})
	}
	for _, person := range c.Directors {
		connections = append(connections, Connection{
			To:    "People/" + person,
			Label: "Director",
		})
	}
	for _, person := range c.Writers {
		connections = append(connections, Connection{
			To:    "People/" + person,
			Label: "Writer",
		})
	}
	for _, company := range c.Distributors {
		connections = append(connections, Connection{
			To:    "Companies/" + company,
			Label: "Distributor",
		})
	}
	for _, company := range c.Manufacturers {
		connections = append(connections, Connection{
			To:    "Companies/" + company,
			Label: "Manufacturer",
		})
	}
	if c.Network != "" {
		connections = append(connections, Connection{
			To:    "Companies/" + c.Network,
			Label: "Network",
		})
	}
	for _, person := range c.Creators {
		connections = append(connections, Connection{
			To:    "People/" + person,
			Label: "Creator",
		})
	}
	for _, person := range c.DevelopedBy {
		connections = append(connections, Connection{
			To:    "People/" + person,
			Label: "Developer",
		})
	}
	for _, person := range c.Showrunners {
		connections = append(connections, Connection{
			To:    "People/" + person,
			Label: "Showrunner",
		})
	}
	for _, person := range c.Authors {
		connections = append(connections, Connection{
			To:    "People/" + person,
			Label: "Author",
		})
	}
	for _, company := range c.Developers {
		connections = append(connections, Connection{
			To:    "Companies/" + company,
			Label: "Developer",
		})
	}
	for _, person := range c.Editors {
		connections = append(connections, Connection{
			To:    "People/" + person,
			Label: "Editor",
		})
	}
	for _, person := range c.Cinematography {
		connections = append(connections, Connection{
			To:    "People/" + person,
			Label: "Cinematography",
		})
	}
	for _, person := range c.Producers {
		connections = append(connections, Connection{
			To:    "People/" + person,
			Label: "Producer",
		})
	}
	for _, person := range c.Screenplay {
		connections = append(connections, Connection{
			To:    "People/" + person,
			Label: "Screenplay",
		})
	}
	for _, person := range c.StoryBy {
		connections = append(connections, Connection{
			To:    "People/" + person,
			Label: "Story",
		})
	}
	for _, person := range c.DialoguesBy {
		connections = append(connections, Connection{
			To:    "People/" + person,
			Label: "Dialogues",
		})
	}
	for _, person := range c.Music {
		connections = append(connections, Connection{
			To:    "People/" + person,
			Label: "Music",
		})
	}
	for _, company := range c.Production {
		connections = append(connections, Connection{
			To:    "Companies/" + company,
			Label: "Production",
		})
	}
	for _, person := range c.Composers {
		connections = append(connections, Connection{
			To:    "People/" + person,
			Label: "Composer",
		})
	}
	for _, person := range c.Programmers {
		connections = append(connections, Connection{
			To:    "People/" + person,
			Label: "Programmer",
		})
	}
	for _, person := range c.Designers {
		connections = append(connections, Connection{
			To:    "People/" + person,
			Label: "Designer",
		})
	}
	for _, person := range c.Hosts {
		connections = append(connections, Connection{
			To:    "People/" + person,
			Label: "Host",
		})
	}
	for _, person := range c.Guests {
		connections = append(connections, Connection{
			To:    "People/" + person,
			Label: "Guest",
		})
	}
	if c.RemakeOf != nil {
		connections = append(connections, Connection{
			To:    c.RemakeOf.Path,
			Label: "Remake",
		})
	}
	for _, character := range c.Characters {
		if character.Actor != "" {
			connections = append(connections, Connection{
				To:    "People/" + character.Actor,
				Label: "Played",
				Info:  character.Name,
			})
		}
		if character.Voice != "" {
			connections = append(connections, Connection{
				To:    "People/" + character.Voice,
				Label: "Voiced",
				Info:  character.Name,
			})
		}
	}
	for _, reference := range c.References {
		connections = append(connections, Connection{
			To:    reference.Path,
			Label: "References",
			Meta:  "none",
		})
	}
	for _, episode := range c.Episodes {
		for _, episodePerson := range episode.Directors {
			connections = append(connections, Connection{
				To:     "People/" + episodePerson,
				Label:  "Director",
				Parent: episode.Name,
			})
		}
		for _, episodePerson := range episode.Writers {
			connections = append(connections, Connection{
				To:     "People/" + episodePerson,
				Label:  "Writer",
				Parent: episode.Name,
			})
		}
		for _, episodePerson := range episode.Editors {
			connections = append(connections, Connection{
				To:     "People/" + episodePerson,
				Label:  "Editor",
				Parent: episode.Name,
			})
		}
		for _, episodePerson := range episode.Cinematography {
			connections = append(connections, Connection{
				To:     "People/" + episodePerson,
				Label:  "Cinematography",
				Parent: episode.Name,
			})
		}
		for _, episodePerson := range episode.Teleplay {
			connections = append(connections, Connection{
				To:     "People/" + episodePerson,
				Label:  "Teleplay",
				Parent: episode.Name,
			})
		}
		for _, episodePerson := range episode.Story {
			connections = append(connections, Connection{
				To:     "People/" + episodePerson,
				Label:  "Story",
				Parent: episode.Name,
			})
		}
		if episode.Studio != "" {
			connections = append(connections, Connection{
				To:     "Companies/" + episode.Studio,
				Label:  "Studio",
				Parent: episode.Name,
			})
		}
		for _, episodeCharacter := range episode.Characters {
			if episodeCharacter.Actor != "" {
				connections = append(connections, Connection{
					To:     "People/" + episodeCharacter.Actor,
					Label:  "Played",
					Info:   episodeCharacter.Name,
					Parent: episode.Name,
				})
			}
			if episodeCharacter.Voice != "" {
				connections = append(connections, Connection{
					To:     "People/" + episodeCharacter.Voice,
					Label:  "Voiced",
					Info:   episodeCharacter.Name,
					Parent: episode.Name,
				})
			}
		}
	}
	return connections
}

type Character struct {
	Name       string   `yaml:"name,omitempty" json:"name,omitempty"`
	Actor      string   `yaml:"actor,omitempty" json:"actor,omitempty"`
	Voice      string   `yaml:"voice,omitempty" json:"voice,omitempty"`
	Image      *Media   `yaml:"image,omitempty" json:"image,omitempty"`
	ActorImage *Media   `yaml:"actor_image,omitempty" json:"actor_image,omitempty"`
	Awards     []*Award `yaml:"awards,omitempty" json:"awards,omitempty"`

	// unknown fields are stored in the Extra map
	Extra map[string]interface{} `yaml:",inline" json:",omitempty"`
}

type Episode struct {
	Name           string        `yaml:"name,omitempty" json:"name,omitempty"`
	Description    string        `yaml:"description,omitempty" json:"description,omitempty"`
	Length         time.Duration `yaml:"length,omitempty" json:"length,omitempty"`
	Released       string        `yaml:"released,omitempty" json:"released,omitempty"`
	Directors      oneOrMany     `yaml:"directors,omitempty" json:"directors,omitempty"`
	Writers        oneOrMany     `yaml:"writers,omitempty" json:"writers,omitempty"`
	Editors        oneOrMany     `yaml:"editors,omitempty" json:"editors,omitempty"`
	Cinematography oneOrMany     `yaml:"cinematography,omitempty" json:"cinematography,omitempty"`
	Teleplay       oneOrMany     `yaml:"teleplay,omitempty" json:"teleplay,omitempty"`
	Story          oneOrMany     `yaml:"story,omitempty" json:"story,omitempty"`
	Studio         string        `yaml:"studio,omitempty" json:"studio,omitempty"`
	Characters     []*Character  `yaml:"characters,omitempty" json:"characters,omitempty"`
	IMDB           string        `yaml:"imdb,omitempty" json:"imdb,omitempty"`
	TMDB           string        `yaml:"tmdb,omitempty" json:"tmdb,omitempty"`
	Netflix        string        `yaml:"netflix,omitempty" json:"netflix,omitempty"`
	Wikipedia      string        `yaml:"wikipedia,omitempty" json:"wikipedia,omitempty"`
	Fandom         string        `yaml:"fandom,omitempty" json:"fandom,omitempty"`

	// unknown fields are stored in the Extra map
	Extra map[string]interface{} `yaml:",inline" json:",omitempty"`
}

// AddMedia populates the Image field with a Media object.
func (c *Content) AddMedia(getImage func(string) *Media) {
	c.Image = getImage(c.SourceNoExtention)
	for _, character := range c.Characters {
		character.Image = getImage(c.SourceNoExtention + "/Characters/" + character.Name)
		character.ActorImage = getImage("People/" + character.Actor)
	}
	for _, episode := range c.Episodes {
		for _, episodeCharacter := range episode.Characters {
			episodeCharacter.Image = getImage(c.SourceNoExtention + "/Characters/" + episodeCharacter.Name)
			episodeCharacter.ActorImage = getImage("People/" + episodeCharacter.Actor)
		}
	}
}

func IsPerson(path string) bool {
	return pathType(path) == "People"
}

func PersonPrefix() string {
	return "People"
}

func ContentFieldName(field string) string {
	return field
}

func length(a time.Duration) string {
	if a == 0 {
		return ""
	}

	if a < time.Hour {
		// format duration as "2m"
		return fmt.Sprintf("%dm", int(a.Minutes()))
	}

	// format duration as "1h 2m"
	return fmt.Sprintf("%dh %dm", int(a.Hours()), int(a.Minutes())%60)
}

func pathType(path string) string {
	return strings.Split(path, string(filepath.Separator))[0]
}

func removeFileExtention(path string) string {
	withoutExt := path[:len(path)-len(filepath.Ext(path))]
	if withoutExt != "" {
		return withoutExt
	}
	return path
}

var reNonID = regexp.MustCompile("[^a-zA-Z0-9-_]")

// formatID formats an ID for MeiliSearch.
// A document identifier can be of type integer or string,
// only composed of alphanumeric characters (a-z A-Z 0-9), hyphens (-) and underscores (_).
func formatID(id string) string {
	return reNonID.ReplaceAllString(id, "_")
}
