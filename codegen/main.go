// codegen is a tool for generating code from provided schema definition.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"

	_ "embed"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

var (
	knownAbbreviations = []string{
		"ID",
		"URL",
		"DOB",
		"DOD",
		"IMDB",
		"TMDB",
		"TPDB",
		"GOG",
		"XBox",
		"IGN",
		"OCLC",
		"UPC",
		"ISBN",
		"ISBN10",
		"ISBN13",
		"GoodReads",
		"YouTube",
		"LinkedIn",
		"TikTok",
		"PlayStation",
		"AppleTV",
		"DarkHorse",
	}
	knownAbbreviationsMap = map[string]string{}
)

func init() {
	// populate knownAbbreviationsMap for faster lookups
	for _, abbr := range knownAbbreviations {
		knownAbbreviationsMap[strings.ToLower(abbr)] = abbr
	}
}

//go:embed content.tmpl
var contentTemplate string

// Schema represents a YAML schema definition for code generation.
type Schema struct {
	Content Content

	Extra map[string]Content `yaml:",inline"`
}

// HasExtraType checks if the schema has any extra schema types defined.
// Used in Connections() generation,
// for example, to connect Character in a Movie to an Actor.
func (s *Schema) HasExtraType(t string) bool {
	for name := range s.Extra {
		if t == name {
			return true
		}
	}
	return false
}

// Content represents a Content struct.
type Content struct {
	Type       string
	Properties PropertySlice
	RootTypes  RootTypes `yaml:"root_types"`
}

// RootTypes represents a list of root types for the schema.
type RootTypes []RootType

// HasType checks if the schema has a root type with the provided type.
func (rt *RootTypes) HasType(t string) bool {
	for _, rootType := range *rt {
		if rootType.Type == t {
			return true
		}
	}
	return false
}

// RootType represents a root type for the schema.
type RootType struct {
	Path string
	Type string
}

// PropertySlice is a slice of Property.
// Need to parse YAML map into a slice of structs to preserve the order of fields.
// That way order of fields in the generated code will be the same as in the schema.
// Especially useful for content.Columns() method.
type PropertySlice []Property

// UnmarshalYAML unmarshals a YAML mapping node into a slice of Property.
func (p *PropertySlice) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("expected a mapping node")
	}

	var properties []Property
	for i := 0; i < len(value.Content); i += 2 {
		nameNode := value.Content[i]
		propertyNode := value.Content[i+1]

		var property Property
		err := propertyNode.Decode(&property)
		if err != nil {
			return fmt.Errorf("failed to decode property: %w", err)
		}

		property.Name = nameNode.Value

		properties = append(properties, property)
	}

	*p = properties
	return nil
}

// Property represents a Content struct field.
type Property struct {
	Name        string
	Title       string // used to override Column title
	Type        string
	Description string
	Label       string // used for Connections to display reference on the other content page
	Meta        string // used for Connections to customize the logic (e.g. "previous" case)
	Column      bool   // indicates if the field should be included in the Columns method
	Items       *Property
}

func main() {
	log.Println("codegen started")

	in := flag.String("in", "", "input schema YAML file")
	out := flag.String("out", "", "output file")
	flag.Parse()

	if err := run(*in, *out); err != nil {
		log.Fatalf("error: %v", err)
	}
	log.Println("codegen finished")
}

// global schema to use in the template functions
var schema *Schema

func run(in, out string) error {
	if in == "" {
		return fmt.Errorf("input file is required")
	}

	if out == "" {
		return fmt.Errorf("output file is required")
	}

	var err error
	schema, err = parseSchema(in)
	if err != nil {
		return fmt.Errorf("failed to parse schema: %w", err)
	}

	err = generateCode(schema, out)
	if err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}

	return nil
}

func parseSchema(in string) (*Schema, error) {
	var schema Schema

	content, err := os.ReadFile(in)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	err = yaml.Unmarshal(content, &schema)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return &schema, nil
}

func generateCode(schema *Schema, out string) error {
	f, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	defer f.Close()

	tmpl, err := template.New("content").Funcs(fm).Parse(contentTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	err = tmpl.Execute(f, schema)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

var fm = template.FuncMap{
	"titleCase": titleCase,
	"fieldType": func(property Property, rootTypes RootTypes) string {
		return schema.FieldType(property, rootTypes)
	},
	"columnTitle": func(p Property) string {
		if p.Title != "" {
			return p.Title
		}
		return titleCase(p.Name)
	},
	"rootTypePath": func(t string, rootTypes RootTypes) string {
		for _, rt := range rootTypes {
			if rt.Type == t {
				return rt.Path
			}
		}
		return ""
	},
	"extraType": func(t string, schema *Schema) bool {
		return schema.HasExtraType(t)
	},
	"columnValue": func(p Property, rootTypes RootTypes) string {
		switch p.Type {
		case "string":
			return "c." + titleCase(p.Name)
		case "duration":
			return "length(c." + titleCase(p.Name) + ")"
		case "references":
			return "strings.Join(c." + titleCase(p.Name) + ", \", \")"
		case "array":
			return "strings.Join(c." + titleCase(p.Name) + ", \", \")"
		default:
			if rootTypes.HasType(p.Type) {
				return "c." + titleCase(p.Name)
			}

			log.Fatalf("columnValue: unknown type %q for field %q (%s)", p.Type, p.Name, p.Description)
			return ""
		}
	},
}

var caser = cases.Title(language.English)

func titleCase(s string) string {
	var result string
	words := strings.Split(s, "_")
	for _, word := range words {
		result += caser.String(word)
	}

	if knownAbbreviationsMap[strings.ToLower(result)] != "" {
		return knownAbbreviationsMap[strings.ToLower(result)]
	}

	return result
}

func (s *Schema) FieldType(property Property, rootTypes RootTypes) string {
	switch property.Type {
	case "string":
		return "string"
	case "duration":
		return "time.Duration"
	case "references":
		return "oneOrMany"
	case "category":
		return caser.String(property.Type)
	case "reference":
		// Reference should be a pointer, so that we can check if it's nil in the templates
		return "*Reference"
	case "award", "media": // todo move to default case
		return "*" + caser.String(property.Type)
	case "array":
		if property.Items == nil {
			log.Fatalf("items field is required for array type")
			return ""
		}

		if rootTypes.HasType(property.Items.Type) {
			return "oneOrMany"
		}

		if property.Items.Type == "reference" {
			return "References"
		}

		return "[]" + s.FieldType(*property.Items, rootTypes)
	default:
		// iterate over root types to find the type
		if rootTypes.HasType(property.Type) {
			return "string"
		}

		// check if type is defined in the extra content
		if s.HasExtraType(property.Type) {
			return "*" + titleCase(property.Type)
		}

		log.Fatalf("unknown type %q for field %q (%s)", property.Type, property.Name, property.Description)
		return ""
	}
}
