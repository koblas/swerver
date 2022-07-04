package handler

type ConfigRewrite = struct {
	Source      string `json:"source" validate:"min=1"`
	Destination string `json:"destination" validate:"min=1"`
}

type Configuration = struct {
	Public      string `json:"public"`
	NoCleanUrls bool
	CleanUrls   []string        `json:"cleanUrls"`
	Rewrites    []ConfigRewrite `json:"rewrites"`
	Redirects   []struct {
		Source      string `json:"source" validate:"min=1"`
		Destination string `json:"destination" validate:"min=1"`
		Type        int    `json:"type"`
	} `json:"redirects"`
	Headers []struct {
		Source  string `json:"source" validate:"min=1,max=100"`
		Headers []struct {
			Key   string `json:"key" validate:"min=1,max=128,"`
			Value string `json:"value" validate:"min=1,max=2048,"`
		}
	} `json:"headers"`
	NoDirectoryListing bool
	DirectoryListing   []string `json:"directoryListing"`
	Unlisted           []string `json:"unlisted"`
	TrailingSlash      bool     `json:"trailingSlash"`
	RenderSingle       bool     `json:"renderSingle"`
	Symlinks           bool     `json:"symlinks"`
	Ssl                struct {
		KeyFile  string `json:"keyFile"`
		CertFile string `json:"certFile"`
	} `json:"ssl"`

	// Not in the config spec
	Debug         bool
	Listen        string
	Clipboard     bool
	NoCompression bool
}
