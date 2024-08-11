// codegen is a tool for generating code from provided schema definition.
package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"

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
	Properties map[string]Property
}

// Property represents a Content struct field.
type Property struct {
	Type        string
	Description string
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

func fieldType(name string, value Property) string {
	switch value.Type {
	case "string", "person":
		return "string"
	case "duration":
		return "time.Duration"
	case "references":
		return "oneOrMany"
	case "reference", "category":
		return caser.String(value.Type)
	case "character", "episode":
		// Characters may have images assigned to them.
		// Episodes have a list of characters.
		// Both are represented as a slice of pointers to the respective structs,
		// so that we can assign images to them.
		return "*" + caser.String(value.Type)
	case "array":
		if value.Items == nil {
			log.Fatalf("items field is required for array type")
			return ""
		}
		return "[]" + fieldType(name, *value.Items)
	default:
		log.Fatalf("unknown type %q for field %q (%s)", value.Type, name, value.Description)
		return ""
	}
}
