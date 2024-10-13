// A simple file browser written in Go.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"runtime/pprof"
	"time"

	flags "github.com/jessevdk/go-flags"
	"github.com/meilisearch/meilisearch-go"
	gitignore "github.com/sabhiram/go-gitignore"
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
	SearchHost         string `env:"INPUT_SEARCH_HOST" short:"h" long:"search-host" description:"Host for search" default:""`
	SearchAPIKey       string `env:"INPUT_SEARCH_API_KEY" short:"k" long:"search-api-key" description:"API key for search" default:""`
	NumWorkers         int    `env:"INPUT_NUMWORKERS" short:"w" long:"workers" description:"Number of workers to use" default:"4"`

	SearchMasterKey string        `env:"INPUT_SEARCH_MASTER_KEY" long:"master-key" description:"search master key"`
	SearchIndexName string        `env:"INPUT_SEARCH_INDEX" long:"index" description:"search index name" default:"info"`
	StateFile       string        `env:"INPUT_SEARCH_STATE" long:"state-file" description:"path to state file" default:".state"`
	Force           string        `env:"INPUT_FORCE" long:"force" description:"force reindexing specified path (\"all\" will reindex everything)" default:""`
	Timeout         time.Duration `env:"INPUT_TIMEOUT" long:"timeout" description:"search timeout" default:"5s"`

	Profile bool `env:"INPUT_PROFILE" long:"profile" description:"enable profiling"`
}

var cfg Config // global env config

// GetString returns the value of the environment variable named by the key.
// If the variable is not present, GetString returns empty string.
// Used in `config` template function to access config values.
func (c Config) GetString(key string) string {
	// use reflect to get the value of the key
	v := reflect.ValueOf(c)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Kind() != reflect.String {
			continue
		}

		if v.Type().Field(i).Name == key {
			return v.Field(i).String()
		}
	}
	return ""
}

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

	generator, err := NewGenerator(ignore)
	if err != nil {
		return fmt.Errorf("creating generator: %v", err)
	}

	if err := generator.Run(); err != nil {
		return fmt.Errorf("running generator: %v", err)
	}

	if cfg.SearchMasterKey != "" {
		if err := indexSite(ignore, generator.hashes); err != nil {
			return fmt.Errorf("indexing site: %v", err)
		}
	}

	return nil
}

func indexSite(ignore *gitignore.GitIgnore, state map[string]string) error {
	log.Printf("Current state contains %d entries", len(state))

	client := meilisearch.New(
		cfg.SearchHost,
		meilisearch.WithAPIKey(cfg.SearchMasterKey),
		meilisearch.WithCustomClient(&http.Client{
			Timeout: cfg.Timeout,
		}),
	)

	indexer, err := NewIndexer(
		client,
		ignore,
		cfg.InfoDirectory,
		cfg.MediaDirectory,
		state,
	)
	if err != nil {
		return fmt.Errorf("creating indexer: %v", err)
	}

	return indexer.Index(
		cfg.StateFile,
		cfg.SearchIndexName,
		cfg.Force,
	)
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
