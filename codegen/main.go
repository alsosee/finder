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
}

// Content represents a Content struct.
type Content struct {
	Type       string
	Properties PropertySlice
	RootTypes  []RootType `yaml:"root_types"`
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

func run(in, out string) error {
	if in == "" {
		return fmt.Errorf("input file is required")
	}

	if out == "" {
		return fmt.Errorf("output file is required")
	}

	schema, err := parseSchema(in)
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
	"fieldType": fieldType,
	"columnTitle": func(p Property) string {
		if p.Title != "" {
			return p.Title
		}
		return titleCase(p.Name)
	},
	"columnValue": func(p Property, rootTypes []RootType) string {
		switch p.Type {
		case "string":
			return "c." + titleCase(p.Name)
		case "duration":
			return "length(c." + titleCase(p.Name) + ")"
		case "references":
			return "strings.Join(c." + titleCase(p.Name) + ", \", \")"
		default:
			for _, rootType := range rootTypes {
				if rootType.Type == p.Type {
					return "c." + titleCase(p.Name)
				}
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

func fieldType(property Property, rootTypes []RootType) string {
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
	case "character", "episode":
		// Characters may have images assigned to them.
		// Episodes have a list of characters.
		// Both are represented as a slice of pointers to the respective structs,
		// so that we can assign images to them.
		return "*" + caser.String(property.Type)
	case "array":
		if property.Items == nil {
			log.Fatalf("items field is required for array type")
			return ""
		}
		return "[]" + fieldType(*property.Items, rootTypes)
	default:
		// iterate over root types to find the type
		for _, rootType := range rootTypes {
			if rootType.Type == property.Type {
				return "string"
			}
		}

		log.Fatalf("unknown type %q for field %q (%s)", property.Type, property.Name, property.Description)
		return ""
	}
}
