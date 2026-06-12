// codegen is a tool for generating code from provided schema definition.
package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
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
		"RSS",
		"GoodReads",
		"GitHub",
		"YouTube",
		"LinkedIn",
		"VK",
		"TikTok",
		"PlayStation",
		"AppleTV",
		"DarkHorse",
		"DTF",
	}
	knownAbbreviationsMap = map[string]string{}
)

func init() {
	// populate knownAbbreviationsMap for faster lookups
	for _, abbr := range knownAbbreviations {
		knownAbbreviationsMap[strings.ToLower(abbr)] = abbr
	}
}

//go:embed templates/*
var templatesFS embed.FS

// Schema represents a YAML schema definition for code generation.
type Schema struct {
	Extra     map[string]Content `yaml:",inline"`
	Content   Content
	RootTypes RootTypes `yaml:"root_types"`
	HashIDs   bool      `yaml:"hash_ids"`
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
	Meta string
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
	Items            *Property
	Name             string
	Title            string // used to override Column title
	Type             string
	Description      string
	Alias            string // field name to use in the template
	Label            string // used for Connections to display reference on the other content page
	Meta             string // used for Connections to customize the logic (e.g. "previous" case)
	Info             string
	Path             string // for fields with type "media": template path to media
	Column           bool   // indicates if the field should be included in the Columns method
	ColumnAlwaysShow bool   `yaml:"column_always_show"` // indicates if the column should be shown in search results, regardless of how many other rows have this field populated
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

	defer func() { _ = f.Close() }()

	tmpls, err := template.New("").Funcs(fm).ParseFS(templatesFS, "templates/*")
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	err = tmpls.Lookup("content.gogo").Execute(f, schema)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

var fm = template.FuncMap{
	"add": func(a, b int) int {
		return a + b
	},
	"titleCase":        titleCase,
	"fieldName":        fieldName,
	"contentFieldName": contentFieldName,
	"fieldNameFor": func(property Property, avoidGeneratedContentFieldCollisions bool) string {
		if avoidGeneratedContentFieldCollisions {
			return contentFieldName(property)
		}
		return fieldName(property)
	},
	"fieldType": func(property Property) string {
		return schema.FieldType(property)
	},
	"columnTitle": func(p Property) string {
		if p.Title != "" {
			return p.Title
		}
		return titleCase(p.Name)
	},
	"rootTypePath": func(t string) string {
		for _, rt := range schema.RootTypes {
			if rt.Type == t {
				return rt.Path
			}
		}
		return ""
	},
	"extraType": func(t string) bool {
		return schema.HasExtraType(t)
	},
	"lookupExtraType": func(t string) Content {
		return schema.Extra[t]
	},
	"hasConnections": hasConnections,
	"hasMedia":       hasMedia,
	"columnValue": func(p Property) string {
		field := "c." + contentFieldName(p)
		switch p.Type {
		case "string":
			return field
		case "duration":
			return "length(" + field + ")"
		case "references":
			return "strings.Join(" + field + ", \", \")"
		case "array":
			return "strings.Join(" + field + ", \", \")"
		default:
			if schema.RootTypes.HasType(p.Type) {
				return field
			}

			log.Fatalf("columnValue: unknown type %q for field %q (%s)", p.Type, p.Name, p.Description)
			return ""
		}
	},
	// "dict" used to pass multiple key-value pairs to a template
	// (e.g. {{ template "something" dict "Key1" "value1" "Key2" "value2" }})
	"dict": func(values ...interface{}) (map[string]interface{}, error) {
		if len(values)%2 != 0 {
			return nil, fmt.Errorf("dict must have an even number of arguments, got %d", len(values))
		}
		dict := make(map[string]interface{}, len(values)/2)
		for i := 0; i < len(values); i += 2 {
			key, ok := values[i].(string)
			if !ok {
				return nil, fmt.Errorf("dict keys must be strings, got %T", values[i])
			}
			dict[key] = values[i+1]
		}
		return dict, nil
	},
	"camelCaseConcat": func(item ...string) string {
		// contact strings together and upper case the first letter of each word
		// except for first word

		// also skip the first work if it's "c"
		result := strings.Builder{}

		var firstWordSeen bool
		for _, word := range item {
			if word == "c" {
				continue
			}

			if !firstWordSeen {
				result.WriteString(word)
				firstWordSeen = true
				continue
			}
			result.WriteString(caser.String(word))
		}

		return result.String()
	},
	"structRef": structRef,
}

var (
	caser           = cases.Title(language.English, cases.NoLower)
	structRefRegexp = regexp.MustCompile(`(\$[a-zA-Z0-9_$]+)`)
)

func structRef(ref, prefix string, escapeFileName bool) string {
	// replace "$" with prefix and convert to camel case
	// e.g. "$name" -> prefix.Name
	// "$ID" is a special case, it's used to reference the ID field of the content
	// e.g. "$ID/Characters/$name" -> c.ID + "/Characters/" + character.Name
	// "$$" is another special case, points to the source without extension.
	// The need for this is to support media lookups for when hash_ids is enabled.

	if ref == "" {
		return ""
	}

	matches := structRefRegexp.FindAllStringSubmatchIndex(ref, -1)
	if matches == nil {
		return ref
	}

	result := strings.Builder{}

	for i, match := range matches {
		if i == 0 && match[0] > 0 {
			result.WriteString("\"" + ref[:match[0]] + "\" + ")
		}

		switch ref[match[0]:match[1]] {
		case "$ID":
			result.WriteString("c.ID")
		case "$$":
			result.WriteString("c.SourceNoExtention")
		default:
			field := prefix + "." + caser.String(ref[match[0]+1:match[1]])
			if escapeFileName {
				result.WriteString("EscapeFileName(" + field + ")")
			} else {
				result.WriteString(field)
			}
		}

		if i < len(matches)-1 {
			result.WriteString(" + \"" + ref[match[1]:matches[i+1][0]] + "\" + ")
		}
	}

	return result.String()
}

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

var generatedContentFieldNames = map[string]bool{
	"Awards":               true,
	"EditorsAwards":        true,
	"WritersAwards":        true,
	"DirectorsAwards":      true,
	"CinematographyAwards": true,
	"MusicAwards":          true,
	"ScreenplayAwards":     true,
}

func fieldName(p Property) string {
	if p.Alias != "" && p.Alias != "name" && p.Alias != "title" {
		return titleCase(p.Alias)
	}

	return titleCase(p.Name)
}

func contentFieldName(p Property) string {
	name := fieldName(p)
	if generatedContentFieldNames[name] {
		if strings.HasSuffix(name, "s") {
			return strings.TrimSuffix(name, "s") + "Items"
		}
		return name + "Field"
	}

	return name
}

func hasConnections(content Content) bool {
	for _, property := range content.Properties {
		if property.Type == "reference" || schema.RootTypes.HasType(property.Type) || property.Meta == "series" {
			return true
		}

		if property.Type != "array" || property.Items == nil {
			continue
		}

		if property.Items.Type == "reference" || schema.RootTypes.HasType(property.Items.Type) {
			return true
		}

		if schema.HasExtraType(property.Items.Type) && hasConnections(schema.Extra[property.Items.Type]) {
			return true
		}
	}

	return false
}

func hasMedia(content Content) bool {
	for _, property := range content.Properties {
		if property.Type == "media" {
			return true
		}

		if property.Type == "array" && property.Items != nil && schema.HasExtraType(property.Items.Type) && hasMedia(schema.Extra[property.Items.Type]) {
			return true
		}
	}

	return false
}

func (s *Schema) FieldType(property Property) string {
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
	case "link":
		return "*Link"
	case "award", "media": // todo move to default case
		return "*" + caser.String(property.Type)
	case "array":
		if property.Items == nil {
			log.Fatalf("items field is required for array type")
			return ""
		}

		if s.RootTypes.HasType(property.Items.Type) {
			return "oneOrMany"
		}

		if property.Items.Type == "reference" {
			return "References"
		}

		if property.Items.Type == "link" {
			return "Links"
		}

		return "[]" + s.FieldType(*property.Items)
	default:
		// iterate over root types to find the type
		if s.RootTypes.HasType(property.Type) {
			return "string"
		}

		// check if type is defined in the extra content
		if s.HasExtraType(property.Type) {
			return "*" + titleCase(property.Type)
		}

		customTypes := []string{}
		for _, rt := range s.RootTypes {
			customTypes = append(customTypes, rt.Type)
		}
		for name := range s.Extra {
			customTypes = append(customTypes, name)
		}

		log.Fatalf(
			"unknown type %q for field %q, known custom types: %v",
			property.Type,
			property.Name,
			customTypes,
		)
		return ""
	}
}
