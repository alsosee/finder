package main

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/alsosee/finder/structs"
)

type Parser struct {
	schema *SchemaMetadata
}

func NewParser(schema *SchemaMetadata) *Parser {
	return &Parser{schema: schema}
}

func (p *Parser) ParseContentYAML(path string, b []byte) (structs.Content, []Diagnostic, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(b, &node); err != nil {
		return structs.Content{}, nil, fmt.Errorf("unmarshaling yaml node: %w", err)
	}

	var diagnostics []Diagnostic
	if p.schema != nil {
		diagnostics = p.schema.ValidateYAML(path, &node)
	}

	var content structs.Content
	if err := yaml.Unmarshal(b, &content); err != nil {
		return structs.Content{}, diagnostics, fmt.Errorf("unmarshaling yaml: %w", err)
	}

	return content, diagnostics, nil
}
