package structs

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Media struct {
	Path                string
	Width               int    `yaml:"width,omitempty" json:",omitempty"`
	Height              int    `yaml:"height,omitempty" json:",omitempty"`
	ThumbPath           string `yaml:"thumb,omitempty" json:",omitempty"`
	ThumbXOffset        int    `yaml:"thumb_x,omitempty" json:",omitempty"`
	ThumbYOffset        int    `yaml:"thumb_y,omitempty" json:",omitempty"`
	ThumbWidth          int    `yaml:"thumb_width,omitempty" json:",omitempty"`
	ThumbHeight         int    `yaml:"thumb_height,omitempty" json:",omitempty"`
	ThumbTotalWidth     int    `yaml:"thumb_total_width,omitempty" json:",omitempty"`
	ThumbTotalHeight    int    `yaml:"thumb_total_height,omitempty" json:",omitempty"`
	Blurhash            string `yaml:"blurhash,omitempty" json:",omitempty"`
	BlurhashImageBase64 string `yaml:"blurhash_image_base64,omitempty" json:",omitempty"`
}

func ParseMediaFile(path string) ([]Media, error) {
	var media []Media
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, &media)
	if err != nil {
		return nil, err
	}
	return media, nil
}
