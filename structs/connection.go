package structs

const (
	// ConnectionPrevious is the label for a connection that goes to the previous node.
	ConnectionPrevious = "previous"

	// ConnectionSeries is the label for a connection that goes to a series page
	// (series of movies, games, books, etc).
	ConnectionSeries = "series"

	// ConnectionNone is the label for a connection that has no specific label.
	ConnectionNone = "none"
)

// Connection represents a connection to another content.
type Connection struct {
	To     string
	Label  string
	Meta   string `yaml:"meta,omitempty"`
	Info   string `yaml:"info,omitempty"`
	Parent string `yaml:"parent,omitempty"`
}

type ConnectionLineItem struct {
	Label string   `yaml:"label,omitempty"`
	Info  []string `yaml:"info_grouped,omitempty"`
}

// ConnectionGroup is populated by `groupConnections` template function
type ConnectionLine struct {
	From    string               `yaml:"from,omitempty"`
	Groups  []ConnectionLineItem `yaml:"groups,omitempty"`
	Parents []string             `yaml:"parents,omitempty"`
}
