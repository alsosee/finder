// A simple file browser written in Go.
package main

import (
	"log"
	"os"
	"reflect"
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
	SearchHost         string `env:"INPUT_SEARCH_HOST" short:"h" long:"search-host" description:"Host for search" default:""`
	SearchAPIKey       string `env:"INPUT_SEARCH_API_KEY" short:"k" long:"search-api-key" description:"API key for search" default:""`
	NumWorkers         int    `env:"INPUT_NUMWORKERS" short:"w" long:"workers" description:"Number of workers to use" default:"4"`
}

var cfg Config // global config

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
		log.Fatalf("Error running app: %v", err)
	}
}

func run() error {
	generator, err := NewGenerator()
	if err != nil {
		log.Fatalf("Error creating generator: %v", err)
	}

	if err := generator.Run(); err != nil {
		log.Fatalf("Error running generator: %v", err)
	}

	return nil
}

func profileWrapper(fn func() error, cpuProfile, memProfile string) func() error {
	return func() error {
		f, err := os.Create("cpu.pprof")
		if err != nil {
			return err
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			return err
		}

		err = fn()
		if err != nil {
			return err
		}

		// Stop CPU profiling and take a memory snapshot
		pprof.StopCPUProfile()
		f, err = os.Create("mem.pprof")
		if err != nil {
			return err
		}
		if err := pprof.WriteHeapProfile(f); err != nil {
			return err
		}
		f.Close()

		return err
	}
}
