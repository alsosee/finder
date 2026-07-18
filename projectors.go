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

func buildProjectors(runtime Config, config structs.Config, outputs map[string]bool, openGraphHost string) []Projector {
	var projectors []Projector

	if outputs["html"] {
		projectors = append(projectors, NewHTMLProjector(
			config,
			runtime.InfoDirectory,
			runtime.StaticDirectory,
			runtime.TemplatesDirectory,
			runtime.OutputDirectory,
		))
	}
	if outputs["search"] && runtime.SearchMasterKey != "" {
		projectors = append(projectors, SearchProjector{
			stateFile: runtime.StateFile,
			indexName: runtime.SearchIndexName,
			force:     runtime.Force,
			host:      runtime.SearchHost,
			masterKey: runtime.SearchMasterKey,
			timeout:   runtime.Timeout,
		})
	}
	if outputs["opengraph"] {
		projectors = append(projectors, OpenGraphProjector{
			outputDir: runtime.OutputDirectory,
			stateFile: runtime.OpenGraphState,
			force:     runtime.Force,
			host:      openGraphHost,
			uploader:  buildOpenGraphUploader(runtime),
		})
	}
	if outputs["json"] {
		projectors = append(projectors, JSONProjector{outputDir: runtime.OutputDirectory})
	}
	if outputs["markdown"] {
		projectors = append(projectors, MarkdownProjector{outputDir: runtime.OutputDirectory})
	}
	if outputs["worker-redirects"] {
		projectors = append(projectors, WorkerRedirectsProjector{
			infoDir: runtime.InfoDirectory,
			output:  runtime.WorkerRedirectsOut,
		})
	}

	return projectors
}

func buildOpenGraphUploader(runtime Config) OpenGraphUploader {
	if runtime.OpenGraphR2Account == "" || runtime.OpenGraphR2KeyID == "" || runtime.OpenGraphR2Secret == "" || runtime.OpenGraphR2Bucket == "" {
		return NoopOpenGraphUploader{}
	}

	return R2OpenGraphUploader{
		accountID:       runtime.OpenGraphR2Account,
		accessKeyID:     runtime.OpenGraphR2KeyID,
		accessKeySecret: runtime.OpenGraphR2Secret,
		bucket:          runtime.OpenGraphR2Bucket,
		client:          &http.Client{Timeout: runtime.Timeout},
	}
}

func selectedOutputs(runtime Config) map[string]bool {
	outputs := map[string]bool{}
	if runtime.Outputs == "" {
		outputs["html"] = true
		if runtime.SearchMasterKey != "" {
			outputs["search"] = true
		}
		return outputs
	}

	for _, output := range strings.Split(runtime.Outputs, ",") {
		output = strings.TrimSpace(strings.ToLower(output))
		if output != "" {
			outputs[output] = true
		}
	}
	return outputs
}

func onlyWorkerRedirects(outputs map[string]bool) bool {
	return len(outputs) == 1 && outputs["worker-redirects"]
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
				fmt.Fprintf(&b, "- **%s:** %s\n", key, columns[key])
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
