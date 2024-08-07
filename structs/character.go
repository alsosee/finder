package structs

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
