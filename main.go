// A simple file browser written in Go.
package main

import (
	"log"

	flags "github.com/jessevdk/go-flags"
)

// Config represents an app configuration.
type Config struct {
	InfoDirectory      string `env:"INPUT_INFO" short:"i" long:"info" description:"Directory that contains info files" default:"info"`
	StaticDirectory    string `env:"INPUT_STATIC" short:"s" long:"static" description:"Directory that contains static files" default:""`
	IgnoreFile         string `env:"INPUT_IGNOREFILE" short:"f" long:"ignorefile" description:"File that contains ignore patterns" default:".ignore"`
	TemplatesDirectory string `env:"INPUT_TEMPLATES" short:"t" long:"templates" description:"Directory that contains templates" default:"templates"`
	OutputDirectory    string `env:"INPUT_OUTPUT" short:"o" long:"output" description:"Directory to output static site" default:"output"`
	NumWorkers         int    `env:"INPUT_NUMWORKERS" short:"w" long:"workers" description:"Number of workers to use" default:"4"`
}

var cfg Config // global config

func main() {
	if _, err := flags.Parse(&cfg); err != nil {
		log.Fatalf("Error parsing flags: %v", err)
	}

	generator, err := NewGenerator(cfg)
	if err != nil {
		log.Fatalf("Error creating generator: %v", err)
	}

	if err := generator.Run(); err != nil {
		log.Fatalf("Error running generator: %v", err)
	}
}
