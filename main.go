// A simple file browser written in Go.
package main

import (
	"fmt"
	"log"

	flags "github.com/jessevdk/go-flags"
)

// Config represents an app configuration.
type Config struct {
	InfoDirectory      string `env:"INPUT_INFO" short:"i" long:"info" description:"Directory that contains info files" default:"info"`
	IgnoreFile         string `env:"INPUT_IGNOREFILE" short:"f" long:"ignorefile" description:"File that contains ignore patterns" default:".ignore"`
	TemplatesDirectory string `env:"INPUT_TEMPLATES" short:"t" long:"templates" description:"Directory that contains templates" default:"templates"`
	OutputDirectory    string `env:"INPUT_OUTPUT" short:"o" long:"output" description:"Directory to output generated files" default:"output"`
	NumWorkers         int    `env:"INPUT_NUMWORKERS" short:"w" long:"workers" description:"Number of workers to use" default:"4"`
}

var (
	errNotFound = fmt.Errorf("not found")
	cfg         Config // global config
)

func main() {
	_, err := flags.Parse(&cfg)
	if err != nil {
		log.Fatalf("error parsing flags: %v", err)
	}

	if cfg.InfoDirectory == "" {
		log.Fatalf("info directory must be specified")
	}

	generator, err := NewGenerator(cfg)
	if err != nil {
		log.Fatalf("error creating generator: %v", err)
	}

	if err := generator.Run(); err != nil {
		log.Fatalf("error running generator: %v", err)
	}
}
