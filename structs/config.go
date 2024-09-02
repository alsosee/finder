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
	LogoShiftY   string `yaml:"logo_shift_y"`
	HomeLabel    string `yaml:"home_label"`
	SearchLabel  string `yaml:"search_label"`
	ViewsLabel   string `yaml:"views_label"`
	ViewsTooltip string `yaml:"views_tooltip"`
	ViewIcons    string `yaml:"view_icons"`
	ViewList     string `yaml:"view_list"`
	ViewColumns  string `yaml:"view_columns"`
	Menu         []struct {
		Title      string `yaml:"title"`
		URL        string `yaml:"url"`
		LogoShiftY string `yaml:"logo_shift_y"`
	} `yaml:"menu"`
	ButtonCancel string `yaml:"button_cancel"`
	ButtonUpload string `yaml:"button_upload"`
}
