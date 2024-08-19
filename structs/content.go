package structs

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Content represents the content of a file.
type Content struct {
	ID               string        `yaml:"-"`                   // used by Search
	Source           string        `yaml:"-"`                   // path to the file
	HTML             string        `yaml:"-" json:",omitempty"` // for Markdown files
	Name             string        `yaml:"name,omitempty" json:"name,omitempty"`
	Title            string        `yaml:"title,omitempty" json:"title,omitempty"`
	Subtitle         string        `yaml:"subtitle,omitempty" json:"subtitle,omitempty"`
	Description      string        `yaml:"description,omitempty" json:"description,omitempty"`
	CoverArtist      string        `yaml:"cover_artist,omitempty" json:"cover_artist,omitempty"`
	Designer         string        `yaml:"designer,omitempty" json:"designer,omitempty"`
	BasedOn          oneOrMany     `yaml:"based_on,omitempty" json:"based_on,omitempty"`
	Series           string        `yaml:"series,omitempty" json:"series,omitempty"`
	Previous         *Reference    `yaml:"previous,omitempty" json:"previous,omitempty"`
	DOB              string        `yaml:"dob,omitempty" json:"dob,omitempty"`
	DOD              string        `yaml:"dod,omitempty" json:"dod,omitempty"`
	Contact          string        `yaml:"contact,omitempty" json:"contact,omitempty"`
	Parent           string        `yaml:"parent,omitempty" json:"parent,omitempty"`
	Founded          string        `yaml:"founded,omitempty" json:"founded,omitempty"`
	Founders         oneOrMany     `yaml:"founders,omitempty" json:"founders,omitempty"`
	Website          string        `yaml:"website,omitempty" json:"website,omitempty"`
	Websites         []string      `yaml:"websites,omitempty" json:"websites,omitempty"`
	Wikipedia        string        `yaml:"wikipedia,omitempty" json:"wikipedia,omitempty"`
	GoodReads        string        `yaml:"goodreads,omitempty" json:"goodreads,omitempty"`
	Bookshop         string        `yaml:"bookshop,omitempty" json:"bookshop,omitempty"`
	AnimeNewsNetwork string        `yaml:"anime_news_network,omitempty" json:"anime_news_network,omitempty"`
	Twitch           string        `yaml:"twitch,omitempty" json:"twitch,omitempty"`
	YouTube          string        `yaml:"youtube,omitempty" json:"youtube,omitempty"`
	Vimeo            string        `yaml:"vimeo,omitempty" json:"vimeo,omitempty"`
	IMDB             string        `yaml:"imdb,omitempty" json:"imdb,omitempty"`
	TMDB             string        `yaml:"tmdb,omitempty" json:"tmdb,omitempty"`
	TPDB             string        `yaml:"tpdb,omitempty" json:"tpdb,omitempty"`
	Steam            string        `yaml:"steam,omitempty" json:"steam,omitempty"`
	Netflix          string        `yaml:"netflix,omitempty" json:"netflix,omitempty"`
	Spotify          string        `yaml:"spotify,omitempty" json:"spotify,omitempty"`
	Soundcloud       string        `yaml:"soundcloud,omitempty" json:"soundcloud,omitempty"`
	Hulu             string        `yaml:"hulu,omitempty" json:"hulu,omitempty"`
	Max              string        `yaml:"max,omitempty" json:"max,omitempty"`
	AdultSwim        string        `yaml:"adult_swim,omitempty" json:"adult_swim,omitempty"`
	AppStore         string        `yaml:"app_store,omitempty" json:"app_store,omitempty"`
	Fandom           string        `yaml:"fandom,omitempty" json:"fandom,omitempty"`
	RottenTomatoes   string        `yaml:"rotten_tomatoes,omitempty" json:"rotten_tomatoes,omitempty"`
	Metacritic       string        `yaml:"metacritic,omitempty" json:"metacritic,omitempty"`
	Opencritic       string        `yaml:"opencritic,omitempty" json:"opencritic,omitempty"`
	Twitter          string        `yaml:"twitter,omitempty" json:"twitter,omitempty"`
	Mastodon         string        `yaml:"mastodon,omitempty" json:"mastodon,omitempty"`
	Reddit           string        `yaml:"reddit,omitempty" json:"reddit,omitempty"`
	Facebook         string        `yaml:"facebook,omitempty" json:"facebook,omitempty"`
	Instagram        string        `yaml:"instagram,omitempty" json:"instagram,omitempty"`
	Threads          string        `yaml:"threads,omitempty" json:"threads,omitempty"`
	LinkedIn         string        `yaml:"linkedin,omitempty" json:"linkedin,omitempty"`
	TikTok           string        `yaml:"tiktok,omitempty" json:"tiktok,omitempty"`
	TelegramChannel  string        `yaml:"telegram_channel,omitempty" json:"telegram_channel,omitempty"`
	PlayStation      string        `yaml:"playstation,omitempty" json:"playstation,omitempty"`
	XBox             string        `yaml:"xbox,omitempty" json:"xbox,omitempty"`
	GOG              string        `yaml:"gog,omitempty" json:"gog,omitempty"`
	X                string        `yaml:"x,omitempty" json:"x,omitempty"`
	Discord          string        `yaml:"discord,omitempty" json:"discord,omitempty"`
	Epic             string        `yaml:"epic,omitempty" json:"epic,omitempty"`
	IGN              string        `yaml:"ign,omitempty" json:"ign,omitempty"`
	Amazon           string        `yaml:"amazon,omitempty" json:"amazon,omitempty"`
	PrimeVideo       string        `yaml:"prime_video,omitempty" json:"prime_video,omitempty"`
	AppleTV          string        `yaml:"apple_tv,omitempty" json:"apple_tv,omitempty"`
	ApplePodcasts    string        `yaml:"apple_podcasts,omitempty" json:"apple_podcasts,omitempty"`
	AppleBooks       string        `yaml:"apple_books,omitempty" json:"apple_books,omitempty"`
	Peacock          string        `yaml:"peacock,omitempty" json:"peacock,omitempty"`
	GooglePlay       string        `yaml:"google_play,omitempty" json:"google_play,omitempty"`
	DisneyPlus       string        `yaml:"disney_plus,omitempty" json:"disney_plus,omitempty"`
	MicrosoftStore   string        `yaml:"microsoft_store,omitempty" json:"microsoft_store,omitempty"`
	Nintendo         string        `yaml:"nintendo,omitempty" json:"nintendo,omitempty"`
	HumbleBundle     string        `yaml:"humble_bundle,omitempty" json:"humble_bundle,omitempty"`
	Row8             string        `yaml:"row8,omitempty" json:"row8,omitempty"`
	Redbox           string        `yaml:"redbox,omitempty" json:"redbox,omitempty"`
	Vudu             string        `yaml:"vudu,omitempty" json:"vudu,omitempty"`
	DarkHorse        string        `yaml:"darkhorse,omitempty" json:"darkhorse,omitempty"`
	ISBN             string        `yaml:"isbn,omitempty" json:"isbn,omitempty"`
	ISBN10           string        `yaml:"isbn10,omitempty" json:"isbn10,omitempty"`
	ISBN13           string        `yaml:"isbn13,omitempty" json:"isbn13,omitempty"`
	OCLC             string        `yaml:"oclc,omitempty" json:"oclc,omitempty"`
	Publishers       oneOrMany     `yaml:"publishers,omitempty" json:"publishers,omitempty"`
	Publication      string        `yaml:"publication,omitempty" json:"publication,omitempty"`
	Artists          oneOrMany     `yaml:"artists,omitempty" json:"artists,omitempty"`
	Colorist         string        `yaml:"colorist,omitempty" json:"colorist,omitempty"`
	Illustrators     oneOrMany     `yaml:"illustrators,omitempty" json:"illustrators,omitempty"`
	Imprint          string        `yaml:"imprint,omitempty" json:"imprint,omitempty"`
	UPC              string        `yaml:"upc,omitempty" json:"upc,omitempty"`
	Genres           []string      `yaml:"genres,omitempty" json:"genres,omitempty"`
	Length           time.Duration `yaml:"length,omitempty" json:"length,omitempty"`
	Directors        oneOrMany     `yaml:"directors,omitempty" json:"directors,omitempty"`
	Writers          oneOrMany     `yaml:"writers,omitempty" json:"writers,omitempty"`
	Distributors     oneOrMany     `yaml:"distributors,omitempty" json:"distributors,omitempty"`
	Rating           string        `yaml:"rating,omitempty" json:"rating,omitempty"`
	Released         string        `yaml:"released,omitempty" json:"released,omitempty"`
	Network          string        `yaml:"network,omitempty" json:"network,omitempty"`
	Engine           string        `yaml:"engine,omitempty" json:"engine,omitempty"`
	Creators         oneOrMany     `yaml:"creators,omitempty" json:"creators,omitempty"`
	Authors          oneOrMany     `yaml:"authors,omitempty" json:"authors,omitempty"`
	Developers       oneOrMany     `yaml:"developers,omitempty" json:"developers,omitempty"`
	Trailer          string        `yaml:"trailer,omitempty" json:"trailer,omitempty"`
	Editors          oneOrMany     `yaml:"editors,omitempty" json:"editors,omitempty"`
	Cinematography   oneOrMany     `yaml:"cinematography,omitempty" json:"cinematography,omitempty"`
	Producers        oneOrMany     `yaml:"producers,omitempty" json:"producers,omitempty"`
	Screenplay       oneOrMany     `yaml:"screenplay,omitempty" json:"screenplay,omitempty"`
	StoryBy          oneOrMany     `yaml:"story_by,omitempty" json:"story_by,omitempty"`
	DialoguesBy      oneOrMany     `yaml:"dialogues_by,omitempty" json:"dialogues_by,omitempty"`
	Music            oneOrMany     `yaml:"music,omitempty" json:"music,omitempty"`
	Production       oneOrMany     `yaml:"production,omitempty" json:"production,omitempty"`
	Composers        oneOrMany     `yaml:"composers,omitempty" json:"composers,omitempty"`
	Programmers      oneOrMany     `yaml:"programmers,omitempty" json:"programmers,omitempty"`
	Designers        oneOrMany     `yaml:"designers,omitempty" json:"designers,omitempty"`
	Hosts            oneOrMany     `yaml:"hosts,omitempty" json:"hosts,omitempty"`
	Guests           oneOrMany     `yaml:"guests,omitempty" json:"guests,omitempty"`
	RemakeOf         *Reference    `yaml:"remake_of,omitempty" json:"remake_of,omitempty"`
	Characters       []*Character  `yaml:"characters,omitempty" json:"characters,omitempty"`
	Categories       []Category    `yaml:"categories,omitempty" json:"categories,omitempty"`
	References       []*Reference  `yaml:"references,omitempty" json:"references,omitempty"`
	Episodes         []*Episode    `yaml:"episodes,omitempty" json:"episodes,omitempty"`

	// unknown fields are stored in the Extra map
	Extra map[string]interface{} `yaml:",inline" json:",omitempty"`

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

// GenerateID generates an ID for the content.
// Used for identifying the content in connections and search.
func (c Content) GenerateID() string {
	if c.ID != "" {
		return c.ID
	}

	c.ID = removeFileExtention(c.Source)
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
	default:
		return strings.ToLower(root)
	}
}

// Columns defines the columns to be displayed in the List view.
func (c Content) Columns() map[string]string {
	return map[string]string{
		"Born":         c.DOB,
		"Died":         c.DOD,
		"Publishers":   strings.Join(c.Publishers, ", "),
		"Length":       length(c.Length),
		"Directors":    strings.Join(c.Directors, ", "),
		"Writers":      strings.Join(c.Writers, ", "),
		"Distributors": strings.Join(c.Distributors, ", "),
		"Rating":       c.Rating,
		"Released":     c.Released,
		"Network":      c.Network,
		"Creators":     strings.Join(c.Creators, ", "),
		"Authors":      strings.Join(c.Authors, ", "),
		"Developers":   strings.Join(c.Developers, ", "),
		"Screenplay":   strings.Join(c.Screenplay, ", "),
		"StoryBy":      strings.Join(c.StoryBy, ", "),
		"DialoguesBy":  strings.Join(c.DialoguesBy, ", "),
		"Hosts":        strings.Join(c.Hosts, ", "),
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
	if c.Previous != nil {
		connections = append(connections, Connection{
			To:    c.Previous.Path,
			Label: "",
			Meta:  "previous",
		})
	}
	if c.Colorist != "" {
		connections = append(connections, Connection{
			To:    "People/" + c.Colorist,
			Label: "Colorist",
		})
	}
	if c.RemakeOf != nil {
		connections = append(connections, Connection{
			To:    c.RemakeOf.Path,
			Label: "Remake",
			Meta:  "",
		})
	}
	return connections
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
