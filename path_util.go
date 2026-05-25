package main

import (
	"path/filepath"
	"strings"
)

func removeFileExtention(path string) string {
	withoutExt := path[:len(path)-len(filepath.Ext(path))]
	if withoutExt != "" {
		return withoutExt
	}
	return path
}

func pathType(path string) string {
	return strings.Split(path, string(filepath.Separator))[0]
}
