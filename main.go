// A simple file browser written in Go.
package main

import (
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"time"

	flags "github.com/jessevdk/go-flags"
)

// Config represents an app configuration.
type Config struct {
	InfoDirectory      string `env:"INPUT_INFO" short:"i" long:"info" description:"Directory that contains info files" default:"info"`
	MediaDirectory     string `env:"INPUT_MEDIA" short:"m" long:"media" description:"Directory that contains media files" default:""`
	StaticDirectory    string `env:"INPUT_STATIC" short:"s" long:"static" description:"Directory that contains static files" default:""`
	ConfigFile         string `env:"INPUT_CONFIG" short:"c" long:"config" description:"File that contains config" default:"config.yml"`
	IgnoreFile         string `env:"INPUT_IGNOREFILE" short:"f" long:"ignore" description:"File that contains ignore patterns" default:".ignore"`
	TemplatesDirectory string `env:"INPUT_TEMPLATES" short:"t" long:"templates" description:"Directory that contains templates" default:"templates"`
	OutputDirectory    string `env:"INPUT_OUTPUT" short:"o" long:"output" description:"Directory to output static site" default:"output"`
	MediaHost          string `env:"INPUT_MEDIA_HOST" short:"M" long:"media-host" description:"Host for media" default:""`
	OpenGraphHost      string `env:"INPUT_OPENGRAPH_HOST" long:"opengraph-host" description:"Host for generated OpenGraph images" default:""`
	OpenGraphR2Account string `env:"INPUT_OPENGRAPH_R2_ACCOUNT_ID" long:"opengraph-r2-account-id" description:"Cloudflare account ID for OpenGraph R2 uploads" default:""`
	OpenGraphR2KeyID   string `env:"INPUT_OPENGRAPH_R2_ACCESS_KEY_ID" long:"opengraph-r2-access-key-id" description:"Cloudflare R2 access key ID for OpenGraph uploads" default:""`
	OpenGraphR2Secret  string `env:"INPUT_OPENGRAPH_R2_ACCESS_KEY_SECRET" long:"opengraph-r2-access-key-secret" description:"Cloudflare R2 access key secret for OpenGraph uploads" default:""`
	OpenGraphR2Bucket  string `env:"INPUT_OPENGRAPH_R2_BUCKET" long:"opengraph-r2-bucket" description:"Cloudflare R2 bucket for OpenGraph uploads" default:""`
	SearchHost         string `env:"INPUT_SEARCH_HOST" short:"h" long:"search-host" description:"Host for search" default:""`
	SearchAPIKey       string `env:"INPUT_SEARCH_API_KEY" short:"k" long:"search-api-key" description:"API key for search" default:""`
	Outputs            string `env:"INPUT_OUTPUTS" long:"outputs" description:"comma-separated projectors to run: html,search,opengraph,json,markdown" default:""`
	NumWorkers         int    `env:"INPUT_NUMWORKERS" short:"w" long:"workers" description:"Number of workers to use" default:"4"`

	SearchMasterKey string        `env:"INPUT_SEARCH_MASTER_KEY" long:"master-key" description:"search master key"`
	SearchIndexName string        `env:"INPUT_SEARCH_INDEX" long:"index" description:"search index name" default:"info"`
	StateFile       string        `env:"INPUT_SEARCH_STATE" long:"state-file" description:"path to state file" default:".state"`
	OpenGraphState  string        `env:"INPUT_OPENGRAPH_STATE" long:"opengraph-state" description:"path to OpenGraph image state file" default:".opengraph-state"`
	Force           string        `env:"INPUT_FORCE" long:"force" description:"force reindexing specified path (\"all\" will reindex everything)" default:""`
	Timeout         time.Duration `env:"INPUT_TIMEOUT" long:"timeout" description:"search timeout" default:"5s"`

	Profile bool `env:"INPUT_PROFILE" long:"profile" description:"enable profiling"`
}

var cfg Config // global env config

func main() {
	if _, err := flags.Parse(&cfg); err != nil {
		log.Fatalf("Error parsing flags: %v", err)
	}

	fn := run
	if cfg.Profile {
		fn = profileWrapper(run, "cpu.pprof", "mem.pprof")
	}

	if err := fn(); err != nil {
		log.Fatalf("Error %v", err)
	}
}

func run() error {
	ignore, err := processIgnoreFile(cfg.IgnoreFile)
	if err != nil {
		return fmt.Errorf("processing ignore file: %w", err)
	}

	config, err := parseConfig(cfg.ConfigFile)
	if err != nil {
		return fmt.Errorf("parsing site config: %w", err)
	}
	overrideConfig(&config)

	schema, err := LoadSchemaMetadata(cfg.InfoDirectory)
	if err != nil {
		return fmt.Errorf("loading schema metadata: %w", err)
	}
	parser := NewParser(schema)

	outputs := selectedOutputs()
	scan, err := NewScanner(cfg.InfoDirectory, cfg.MediaDirectory, ignore).Scan()
	if err != nil {
		return fmt.Errorf("scanning inputs: %w", err)
	}

	defer measureTime()()

	graph, err := NewGraphBuilder(config, scan, parser, cfg.InfoDirectory, outputs["opengraph"]).Build()
	if err != nil {
		return fmt.Errorf("building graph: %w", err)
	}

	projectors := buildProjectors(config, outputs, graph.Config.OpenGraphHost)
	if err := RunProjectors(graph, projectors...); err != nil {
		return err
	}

	return nil
}

func profileWrapper(fn func() error, cpuProfile, memProfile string) func() error {
	return func() error {
		f, err := os.Create(cpuProfile)
		if err != nil {
			return fmt.Errorf("creating cpu profile: %v", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("starting cpu profile: %v", err)
		}

		err = fn()
		if err != nil {
			return err
		}

		// Stop CPU profiling and take a memory snapshot
		pprof.StopCPUProfile()
		f, err = os.Create(memProfile)
		if err != nil {
			return fmt.Errorf("creating memory profile: %v", err)
		}
		if err := pprof.WriteHeapProfile(f); err != nil {
			return fmt.Errorf("writing memory profile: %v", err)
		}
		if err = f.Close(); err != nil {
			return fmt.Errorf("closing memory profile: %v", err)
		}

		return nil
	}
}
