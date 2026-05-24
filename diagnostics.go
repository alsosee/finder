package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Diagnostic struct {
	File    string
	Line    int
	Column  int
	Path    string
	Message string
}

func (d Diagnostic) Log() {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		fmt.Printf("::warning file=%s,line=%d,col=%d::%s\n", d.File, d.Line, d.Column, d.Message)
		return
	}
	log.Printf("%s:%d:%d: %s", d.File, d.Line, d.Column, d.Message)
}

type SchemaMetadata struct {
	types map[string]map[string]string
}

func LoadSchemaMetadata(infoDir string) (*SchemaMetadata, error) {
	path := filepath.Join(infoDir, "_finder", "schema.yml")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &SchemaMetadata{types: map[string]map[string]string{}}, nil
		}
		return nil, fmt.Errorf("reading schema metadata: %w", err)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(b, &root); err != nil {
		return nil, fmt.Errorf("unmarshaling schema metadata: %w", err)
	}

	meta := &SchemaMetadata{types: map[string]map[string]string{}}
	if len(root.Content) == 0 {
		return meta, nil
	}

	doc := root.Content[0]
	for i := 0; i < len(doc.Content); i += 2 {
		typeName := doc.Content[i].Value
		typeNode := doc.Content[i+1]
		props := schemaProperties(typeNode)
		if props != nil {
			meta.types[typeName] = props
		}
	}

	return meta, nil
}

func schemaProperties(typeNode *yaml.Node) map[string]string {
	if typeNode == nil || typeNode.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i < len(typeNode.Content); i += 2 {
		if typeNode.Content[i].Value != "properties" {
			continue
		}
		propsNode := typeNode.Content[i+1]
		if propsNode.Kind != yaml.MappingNode {
			return nil
		}

		props := map[string]string{}
		for j := 0; j < len(propsNode.Content); j += 2 {
			name := propsNode.Content[j].Value
			props[name] = schemaPropertyItemType(propsNode.Content[j+1])
		}
		return props
	}

	return nil
}

func schemaPropertyItemType(propertyNode *yaml.Node) string {
	if propertyNode == nil || propertyNode.Kind != yaml.MappingNode {
		return ""
	}

	var propertyType string
	for i := 0; i < len(propertyNode.Content); i += 2 {
		switch propertyNode.Content[i].Value {
		case "type":
			propertyType = propertyNode.Content[i+1].Value
		case "items":
			if itemType := schemaPropertyType(propertyNode.Content[i+1]); itemType != "" {
				return itemType
			}
		}
	}

	return propertyType
}

func schemaPropertyType(propertyNode *yaml.Node) string {
	if propertyNode == nil || propertyNode.Kind != yaml.MappingNode {
		return ""
	}
	for i := 0; i < len(propertyNode.Content); i += 2 {
		if propertyNode.Content[i].Value == "type" {
			return propertyNode.Content[i+1].Value
		}
	}
	return ""
}

func (s *SchemaMetadata) ValidateYAML(file string, root *yaml.Node) []Diagnostic {
	if s == nil || len(s.types) == 0 || root == nil || len(root.Content) == 0 {
		return nil
	}

	return s.validateMapping(file, "content", root.Content[0], "")
}

func (s *SchemaMetadata) validateMapping(file, typeName string, node *yaml.Node, path string) []Diagnostic {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}

	known := s.types[typeName]
	if len(known) == 0 {
		return nil
	}

	var diagnostics []Diagnostic
	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i]
		value := node.Content[i+1]
		childPath := key.Value
		if path != "" {
			childPath = path + "." + key.Value
		}

		childType, ok := known[key.Value]
		if !ok {
			diagnostics = append(diagnostics, Diagnostic{
				File:    file,
				Line:    key.Line,
				Column:  key.Column,
				Path:    childPath,
				Message: fmt.Sprintf("Unknown field %q at %s", key.Value, childPath),
			})
			continue
		}

		diagnostics = append(diagnostics, s.validateNested(file, childType, value, childPath)...)
	}

	return diagnostics
}

func (s *SchemaMetadata) validateNested(file, typeName string, node *yaml.Node, path string) []Diagnostic {
	if _, ok := s.types[typeName]; !ok {
		return nil
	}

	switch node.Kind {
	case yaml.MappingNode:
		return s.validateMapping(file, typeName, node, path)
	case yaml.SequenceNode:
		var diagnostics []Diagnostic
		for i, item := range node.Content {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			diagnostics = append(diagnostics, s.validateNested(file, typeName, item, itemPath)...)
		}
		return diagnostics
	default:
		return nil
	}
}
