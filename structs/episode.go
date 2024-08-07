package structs

import "time"

type Episode struct {
	Name           string
	Description    string        `yaml:",omitempty" json:",omitempty"`
	Length         time.Duration `yaml:",omitempty" json:",omitempty"`
	Released       string        `yaml:",omitempty" json:",omitempty"`
	Directors      oneOrMany     `yaml:",omitempty" json:",omitempty"`
	Writers        oneOrMany     `yaml:",omitempty" json:",omitempty"`
	Editors        oneOrMany     `yaml:",omitempty" json:",omitempty"`
	Cinematography oneOrMany     `yaml:",omitempty" json:",omitempty"`
	Teleplay       oneOrMany     `yaml:",omitempty" json:",omitempty"`
	Story          oneOrMany     `yaml:",omitempty" json:",omitempty"`
	Studio         string        `yaml:",omitempty" json:",omitempty"`
	Characters     []*Character  `yaml:",omitempty" json:",omitempty"`

	IMDB      string `yaml:",omitempty" json:",omitempty"`
	TMDB      string `yaml:",omitempty" json:",omitempty"`
	Netflix   string `yaml:",omitempty" json:",omitempty"`
	Wikipedia string `yaml:",omitempty" json:",omitempty"`
	Fandom    string `yaml:",omitempty" json:",omitempty"`

	// unknown fields are stored in the Extra map
	Extra map[string]interface{} `yaml:",inline" json:",omitempty"`
}
