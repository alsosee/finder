package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	gitignore "github.com/sabhiram/go-gitignore"
)

func processIgnoreFile(infoDir, ignoreFile string) (*gitignore.GitIgnore, error) {
	ignore := &gitignore.GitIgnore{}
	ignoreFilepath := filepath.Join(infoDir, ignoreFile)
	if _, err := os.Stat(ignoreFilepath); err == nil {
		ignore, err = gitignore.CompileIgnoreFile(ignoreFilepath)
		if err != nil {
			return nil, fmt.Errorf("compiling ignore file: %w", err)
		}
	} else {
		log.Printf("Ignore file %q not found, ignoring", ignoreFilepath)
	}

	return ignore, nil
}
