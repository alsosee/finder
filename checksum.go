package main

import (
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"os"
	"path/filepath"
)

// crc32sum calculates a CRC32 checksum for a generated output file.
// It is used in templates to add a cache-busting query parameter to static file URLs.
func (g *HTMLProjector) crc32sum(path string) string {
	g.crc32mu.Lock()
	defer g.crc32mu.Unlock()

	if crc, ok := g.crc32cache[path]; ok {
		return crc
	}

	filePath := filepath.Join(g.outputDir, path)
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Error opening file %q: %v", filePath, err)
		return ""
	}
	defer file.Close()

	hash := crc32.NewIEEE()
	if _, err := io.Copy(hash, file); err != nil {
		log.Printf("Error calculating CRC32 checksum for file %q: %v", filePath, err)
		return ""
	}

	g.crc32cache[path] = fmt.Sprintf("%x", hash.Sum32())

	return g.crc32cache[path]
}
