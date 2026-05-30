package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/alsosee/finder/structs"
)

const openGraphTemplateVersion = "og-v1"

type OpenGraphState map[string]OpenGraphStateEntry

type OpenGraphStateEntry struct {
	SourceHash   string `json:"source_hash"`
	TemplateHash string `json:"template_hash"`
	Key          string `json:"key"`
	URL          string `json:"url"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	ContentType  string `json:"content_type"`
}

type OpenGraphUploader interface {
	Upload(key string, body []byte, contentType string) error
}

type NoopOpenGraphUploader struct{}

func (NoopOpenGraphUploader) Upload(_ string, _ []byte, _ string) error {
	return nil
}

type OpenGraphProjector struct {
	outputDir string
	stateFile string
	force     string
	host      string
	uploader  OpenGraphUploader
}

func (p OpenGraphProjector) Name() string {
	return "opengraph"
}

func (p OpenGraphProjector) Run(graph *BuildGraph) error {
	if p.host == "" {
		return nil
	}
	if p.uploader == nil {
		p.uploader = NoopOpenGraphUploader{}
	}

	state, err := readOpenGraphState(p.stateFile)
	if err != nil {
		return err
	}

	for id, content := range graph.Contents {
		source := content.Source
		if source == "" {
			source = id + ".yml"
		}

		key := openGraphKey(id)
		url := joinURL(p.host, key)
		sourceHash := graph.Hashes[source]
		templateHash := openGraphTemplateHash()

		entry := OpenGraphStateEntry{
			SourceHash:   sourceHash,
			TemplateHash: templateHash,
			Key:          key,
			URL:          url,
			Width:        graph.Config.OpenGraph.Width,
			Height:       graph.Config.OpenGraph.Height,
			ContentType:  "image/png",
		}
		if entry.Width == 0 {
			entry.Width = 1200
		}
		if entry.Height == 0 {
			entry.Height = 630
		}

		if !p.shouldGenerate(id, state[id], entry) {
			continue
		}

		imageBytes, err := renderOpenGraphImage(content, entry.Width, entry.Height)
		if err != nil {
			return fmt.Errorf("rendering OpenGraph image for %q: %w", id, err)
		}
		entry.SourceHash = entry.SourceHash + ":" + fmt.Sprintf("%x", crc32.ChecksumIEEE(imageBytes))

		if err := p.uploader.Upload(key, imageBytes, entry.ContentType); err != nil {
			return fmt.Errorf("uploading OpenGraph image %q: %w", key, err)
		}

		outPath := filepath.Join(p.outputDir, key)
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return fmt.Errorf("creating local OpenGraph output dir: %w", err)
		}
		if err := os.WriteFile(outPath, imageBytes, 0o644); err != nil {
			return fmt.Errorf("writing local OpenGraph image: %w", err)
		}

		state[id] = entry
	}

	return writeOpenGraphState(p.stateFile, state)
}

func (p OpenGraphProjector) shouldGenerate(id string, old, next OpenGraphStateEntry) bool {
	if p.force == "all" {
		return true
	}
	if p.force != "" {
		for _, item := range strings.Split(p.force, ",") {
			item = strings.TrimSpace(removeFileExtention(item))
			if item == id {
				return true
			}
		}
	}

	return old.Key == "" || old.SourceHash == "" || !strings.HasPrefix(old.SourceHash, next.SourceHash+":") || old.TemplateHash != next.TemplateHash
}

func readOpenGraphState(path string) (OpenGraphState, error) {
	state := OpenGraphState{}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return nil, fmt.Errorf("reading OpenGraph state: %w", err)
	}
	if len(b) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(b, &state); err != nil {
		return nil, fmt.Errorf("unmarshaling OpenGraph state: %w", err)
	}
	return state, nil
}

func writeOpenGraphState(path string, state OpenGraphState) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return fmt.Errorf("creating OpenGraph state dir: %w", err)
	}

	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling OpenGraph state: %w", err)
	}
	return os.WriteFile(path, b, 0o644)
}

func openGraphTemplateHash() string {
	sum := sha256.Sum256([]byte(openGraphTemplateVersion))
	return hex.EncodeToString(sum[:])
}

func openGraphKey(id string) string {
	key := strings.NewReplacer(
		"\\", "/",
		":", "",
		"?", "",
		"#", "",
		"&", "and",
	).Replace(id)
	return filepath.ToSlash(filepath.Join("opengraph", key+".png"))
}

func openGraphURL(host, id string) string {
	if host == "" || id == "" {
		return ""
	}
	return joinURL(host, openGraphKey(id))
}

func joinURL(host, path string) string {
	return strings.TrimRight(host, "/") + "/" + strings.TrimLeft(path, "/")
}

func renderOpenGraphImage(content structs.Content, width, height int) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	bg := color.RGBA{R: 250, G: 249, B: 246, A: 255}
	fg := color.RGBA{R: 40, G: 37, B: 35, A: 255}
	accent := color.RGBA{R: 199, G: 58, B: 74, A: 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(0, 0, width, height/18), &image.Uniform{C: accent}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(width/16, height/5, width-width/16, height/5+6), &image.Uniform{C: fg}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(width/16, height-height/5, width-width/16, height-height/5+6), &image.Uniform{C: fg}, image.Point{}, draw.Src)

	var b strings.Builder
	title := content.Header()
	if title == "" {
		title = filepath.Base(content.SourceNoExtention)
	}
	b.WriteString(strings.ToUpper(title))
	if content.Subtitle != "" {
		b.WriteString(" / ")
		b.WriteString(strings.ToUpper(content.Subtitle))
	}

	drawBlockText(img, b.String(), width/16, height/3, 8, fg)

	return encodePNG(img)
}

func encodePNG(img image.Image) ([]byte, error) {
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func drawBlockText(img *image.RGBA, text string, x, y, scale int, c color.Color) {
	maxX := img.Bounds().Dx() - x
	cursorX := x
	cursorY := y
	for _, r := range text {
		if cursorX+6*scale > maxX+x || r == '\n' {
			cursorX = x
			cursorY += 8 * scale
			continue
		}
		drawBlockRune(img, r, cursorX, cursorY, scale, c)
		cursorX += 6 * scale
	}
}

func drawBlockRune(img *image.RGBA, r rune, x, y, scale int, c color.Color) {
	if r == ' ' {
		return
	}
	pattern := blockFont[r]
	if pattern == nil {
		pattern = blockFont['?']
	}
	for row, line := range pattern {
		for col, bit := range line {
			if bit != '1' {
				continue
			}
			rect := image.Rect(x+col*scale, y+row*scale, x+(col+1)*scale, y+(row+1)*scale)
			draw.Draw(img, rect, &image.Uniform{C: c}, image.Point{}, draw.Src)
		}
	}
}

var blockFont = map[rune][]string{
	'?': {"111", "001", "011", "000", "010"},
}
