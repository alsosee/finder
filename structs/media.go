package structs

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Media struct {
	Path             string
	ThumbPath        string `yaml:"thumb,omitempty" json:"thumb,omitempty"`
	Width            int    `yaml:"width,omitempty" json:"width,omitempty"`
	Height           int    `yaml:"height,omitempty" json:"height,omitempty"`
	ThumbXOffset     int    `yaml:"thumb_x,omitempty" json:"thumb_x,omitempty"`
	ThumbYOffset     int    `yaml:"thumb_y,omitempty" json:"thumb_y,omitempty"`
	ThumbWidth       int    `yaml:"thumb_width,omitempty" json:"thumb_width,omitempty"`
	ThumbHeight      int    `yaml:"thumb_height,omitempty" json:"thumb_height,omitempty"`
	ThumbTotalWidth  int    `yaml:"thumb_total_width,omitempty" json:"thumb_total_width,omitempty"`
	ThumbTotalHeight int    `yaml:"thumb_total_height,omitempty" json:"thumb_total_height,omitempty"`
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
