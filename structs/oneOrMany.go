package structs

import (
	"fmt"

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
		return fmt.Errorf("expected a string or a sequence")
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

func (b oneOrMany) MarshalYAML() (interface{}, error) {
	if len(b) == 1 {
		// If there is only one element, return it as a string
		return b[0], nil
	}

	// Otherwise, Marshal it as a list
	return []string(b), nil
}
