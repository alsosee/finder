package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/alsosee/finder/structs"
	"github.com/meilisearch/meilisearch-go"
)

func buildProjectors(config structs.Config, outputs map[string]bool, openGraphHost string) []Projector {
	var projectors []Projector

	if outputs["html"] {
		projectors = append(projectors, NewHTMLProjector(
			config,
			cfg.InfoDirectory,
			cfg.StaticDirectory,
			cfg.TemplatesDirectory,
			cfg.OutputDirectory,
		))
	}
	if outputs["search"] && cfg.SearchMasterKey != "" {
		projectors = append(projectors, SearchProjector{
			stateFile: cfg.StateFile,
			indexName: cfg.SearchIndexName,
			force:     cfg.Force,
			host:      cfg.SearchHost,
			masterKey: cfg.SearchMasterKey,
			timeout:   cfg.Timeout,
		})
	}
	if outputs["opengraph"] {
		projectors = append(projectors, OpenGraphProjector{
			outputDir: cfg.OutputDirectory,
			stateFile: cfg.OpenGraphState,
			force:     cfg.Force,
			host:      openGraphHost,
			uploader:  buildOpenGraphUploader(),
		})
	}
	if outputs["json"] {
		projectors = append(projectors, JSONProjector{outputDir: cfg.OutputDirectory})
	}
	if outputs["markdown"] {
		projectors = append(projectors, MarkdownProjector{outputDir: cfg.OutputDirectory})
	}

	return projectors
}

func buildOpenGraphUploader() OpenGraphUploader {
	if cfg.OpenGraphR2Account == "" || cfg.OpenGraphR2KeyID == "" || cfg.OpenGraphR2Secret == "" || cfg.OpenGraphR2Bucket == "" {
		return NoopOpenGraphUploader{}
	}

	return R2OpenGraphUploader{
		accountID:       cfg.OpenGraphR2Account,
		accessKeyID:     cfg.OpenGraphR2KeyID,
		accessKeySecret: cfg.OpenGraphR2Secret,
		bucket:          cfg.OpenGraphR2Bucket,
		client:          &http.Client{Timeout: cfg.Timeout},
	}
}

func selectedOutputs() map[string]bool {
	outputs := map[string]bool{}
	if cfg.Outputs == "" {
		outputs["html"] = true
		if cfg.SearchMasterKey != "" {
			outputs["search"] = true
		}
		return outputs
	}

	for _, output := range strings.Split(cfg.Outputs, ",") {
		output = strings.TrimSpace(strings.ToLower(output))
		if output != "" {
			outputs[output] = true
		}
	}
	return outputs
}

type JSONProjector struct {
	outputDir string
}

func (p JSONProjector) Name() string {
	return "json"
}

type SearchProjector struct {
	stateFile string
	indexName string
	force     string
	host      string
	masterKey string
	timeout   time.Duration
}

func (p SearchProjector) Name() string {
	return "search"
}

func (p SearchProjector) Run(graph *BuildGraph) error {
	if p.masterKey == "" {
		return nil
	}

	log.Printf("Current state contains %d entries", len(graph.Hashes))

	client := meilisearch.New(
		p.host,
		meilisearch.WithAPIKey(p.masterKey),
		meilisearch.WithCustomClient(&http.Client{
			Timeout: p.timeout,
		}),
	)

	indexer := NewIndexer(meiliSearchClient{client: client}, graph)

	return indexer.Index(
		p.stateFile,
		p.indexName,
		p.force,
	)
}

func (p JSONProjector) Run(graph *BuildGraph) error {
	type jsonGraph struct {
		Contents    structs.Contents          `json:"contents"`
		Connections structs.Connections       `json:"connections"`
		Directories map[string][]structs.File `json:"directories"`
	}

	outPath := filepath.Join(p.outputDir, "data", "graph.json")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("creating json output dir: %w", err)
	}

	b, err := json.MarshalIndent(jsonGraph{
		Contents:    graph.Contents,
		Connections: graph.Connections,
		Directories: graph.DirContents,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling graph json: %w", err)
	}

	return os.WriteFile(outPath, b, 0o644)
}

type MarkdownProjector struct {
	outputDir string
}

func (p MarkdownProjector) Name() string {
	return "markdown"
}

func (p MarkdownProjector) Run(graph *BuildGraph) error {
	for id, content := range graph.Contents {
		outPath := filepath.Join(p.outputDir, "markdown", id+".md")
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return fmt.Errorf("creating markdown output dir: %w", err)
		}

		var b strings.Builder
		title := content.Header()
		if title == "" {
			title = filepath.Base(id)
		}
		b.WriteString("# " + title + "\n\n")
		if content.Subtitle != "" {
			b.WriteString(content.Subtitle + "\n\n")
		}
		if content.Description != "" {
			b.WriteString(content.Description + "\n\n")
		}

		columns := content.Columns()
		if len(columns) > 0 {
			keys := make([]string, 0, len(columns))
			for key, value := range columns {
				if value != "" {
					keys = append(keys, key)
				}
			}
			sort.Strings(keys)
			for _, key := range keys {
				b.WriteString(fmt.Sprintf("- **%s:** %s\n", key, columns[key]))
			}
			if len(keys) > 0 {
				b.WriteString("\n")
			}
		}

		if err := os.WriteFile(outPath, []byte(b.String()), 0o644); err != nil {
			return fmt.Errorf("writing markdown %q: %w", outPath, err)
		}
	}

	return nil
}
