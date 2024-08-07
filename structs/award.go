package structs

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
