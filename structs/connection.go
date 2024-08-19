package structs

const (
	// ConnectionPrevious is the label for a connection that goes to the previous node.
	ConnectionPrevious = "previous"

	// ConnectionSeries is the label for a connection that goes to a series page
	// (series of movies, games, books, etc).
	ConnectionSeries = "series"
)

// Connection represents a connection to another content.
type Connection struct {
	To    string
	Label string
	Meta  string
}
