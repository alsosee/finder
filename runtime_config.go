package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/alsosee/finder/structs"
)

func parseConfig(infoDir, configFile string) (structs.Config, error) {
	b, err := os.ReadFile(filepath.Join(infoDir, configFile))
	if err != nil {
		return structs.Config{}, fmt.Errorf("reading config file: %w", err)
	}

	var config structs.Config
	if err = yaml.Unmarshal(b, &config); err != nil {
		return structs.Config{}, fmt.Errorf("unmarshaling config: %w", err)
	}

	return config, nil
}

func overrideConfig(config *structs.Config, runtime Config) {
	if runtime.MediaHost != "" {
		config.MediaHost = runtime.MediaHost
	}
	if runtime.OpenGraphHost != "" {
		config.OpenGraphHost = runtime.OpenGraphHost
	}
	if runtime.SearchHost != "" {
		config.SearchHost = runtime.SearchHost
	}
	if runtime.SearchIndexName != "" {
		config.SearchIndexName = runtime.SearchIndexName
	}
	if runtime.SearchAPIKey != "" {
		config.SearchAPIKey = runtime.SearchAPIKey
	}
}
