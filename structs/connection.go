package structs

const (
	// ConnectionPrevious is the label for a connection that goes to the previous node.
	ConnectionPrevious = "previous"
)

// Connection represents a connection to another content.
type Connection struct {
	To    string
	Label string
	Meta  string
}
