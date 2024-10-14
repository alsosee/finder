package structs

// Config represents a site configuration stored in a YAML file, e.g. config.yaml.
// It is used to store site-specific configuration values.
type Config struct {
	Title           string `yaml:"title"`
	Description     string `yaml:"description"`
	Lang            string `yaml:"lang"`
	Repo            string `yaml:"repo"`
	URL             string `yaml:"url"`
	MediaHost       string `yaml:"media_host"`
	SearchHost      string `yaml:"search_host"`
	SearchAPIKey    string `yaml:"search_api_key"`
	SearchIndexName string `yaml:"search_index"`
	OpenGraph       struct {
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
	LabelCancel    string `yaml:"label_cancel"`
	LabelUpload    string `yaml:"label_upload"`
	NoResultsLabel string `yaml:"no_results_label"`
	ColumnName     string `yaml:"column_name"`
	ColumnKind     string `yaml:"column_kind"`
	OfLabel        string `yaml:"of_label"`
}
