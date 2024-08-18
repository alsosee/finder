package structs

const (
	// ConnectionPrevious is the label for a connection that goes to the previous node.
	ConnectionPrevious = "previous"
)

type Connection struct {
	To    string
	Label string
	Meta  string
}
