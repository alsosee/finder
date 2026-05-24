package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alsosee/finder/structs"
)

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
