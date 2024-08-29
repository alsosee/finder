package structs

// Config represents a site configuration stored in a YAML file, e.g. config.yaml.
// It is used to store site-specific configuration values.
type Config struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Lang        string `yaml:"lang"`
	Repo        string `yaml:"repo"`
	URL         string `yaml:"url"`
	OpenGraph   struct {
		Image        string `yaml:"image"`
		Width        int    `yaml:"width"`
		Height       int    `yaml:"height"`
		TwitterImage string `yaml:"twitter_image"`
	} `yaml:"opengraph"`
}
