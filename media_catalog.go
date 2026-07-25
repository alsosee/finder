package main

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/alsosee/finder/structs"
)

type MediaCatalog map[string][]structs.Media

func (m MediaCatalog) AddThumbsFile(path string, media []structs.Media) {
	dir := filepath.Dir(path)
	if dir == "." {
		dir = ""
	}
	m[dir] = media
}

func (m MediaCatalog) ImageForPath(path string) *structs.Media {
	dir := filepath.Dir(path)
	if dir == "." {
		dir = ""
	}

	base := structs.EscapeFileName(filepath.Base(path))
	for _, media := range m[dir] {
		mediaImage := media
		if removeFileExtention(media.Path) == base {
			return &mediaImage
		}
	}

	return nil
}

func (m MediaCatalog) PathsSharingThumb(path string) []string {
	dir := filepath.Dir(path)
	if dir == "." {
		dir = ""
	}

	base := filepath.Base(path)
	if base == ".thumbs.yml" {
		return m.contentPathsForDir(dir)
	}

	if strings.HasPrefix(base, "thumbnails_") {
		return m.contentPathsForThumbFile(dir, base)
	}

	image := m.ImageForPath(removeFileExtention(path))
	if image == nil || image.ThumbPath == "" {
		return nil
	}

	var result []string
	for _, media := range m[dir] {
		if media.ThumbPath == image.ThumbPath && media.Path != image.Path {
			result = append(result, filepath.Join(dir, removeFileExtention(media.Path)+".yml"))
		}
	}
	sort.Strings(result)
	return result
}

func (m MediaCatalog) contentPathsForDir(dir string) []string {
	var result []string
	for _, media := range m[dir] {
		result = append(result, filepath.Join(dir, removeFileExtention(media.Path)+".yml"))
	}
	sort.Strings(result)
	return result
}

func (m MediaCatalog) contentPathsForThumbFile(dir, thumbFile string) []string {
	var result []string
	for _, media := range m[dir] {
		thumbPath := strings.Split(media.ThumbPath, "?")[0]
		if thumbPath == thumbFile {
			result = append(result, filepath.Join(dir, removeFileExtention(media.Path)+".yml"))
		}
	}
	sort.Strings(result)
	return result
}
