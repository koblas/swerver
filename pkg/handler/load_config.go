package handler

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
)

// Configuration file format as defined by the serve utility
type serveConfiguration = struct {
	Public string `json:"public"`
	// CleanUrls []string `json:"cleanUrls"`
	CleanUrls json.RawMessage `json:"cleanUrls"`
	Rewrites  []struct {
		Source      string `json:"source" validate:"min=1"`
		Destination string `json:"destination" validate:"min=1"`
	} `json:"rewrites"`
	Redirects []struct {
		Source      string `json:"source" validate:"min=1"`
		Destination string `json:"destination" validate:"min=1"`
		Type        int    `json:"type"`
	} `json:"redirects"`
	Proxy []struct {
		Source      string `json:"source" validate:"min=1"`
		Destination string `json:"destination" validate:"min=1"`
	} `json:"proxy"`
	Headers []struct {
		Source  string `json:"source" validate:"min=1,max=100"`
		Headers []struct {
			Key   string `json:"key" validate:"min=1,max=128,"`
			Value string `json:"value" validate:"min=1,max=2048,"`
		}
	} `json:"headers"`
	DirectoryListing json.RawMessage `json:"directoryListing"`
	Unlisted         *[]string       `json:"unlisted"`
	TrailingSlash    *bool           `json:"trailingSlash"`
	RenderSingle     bool            `json:"renderSingle"`
	Symlinks         bool            `json:"symlinks"`

	Ssl struct {
		KeyFile  string `json:"keyFile"`
		CertFile string `json:"certFile"`
	} `json:"ssl"`
}

func LoadServeConfiguration(filepath string) (Configuration, error) {
	config := Configuration{}
	data := serveConfiguration{}

	file, err := ioutil.ReadFile(filepath)
	if err == nil {
		if err = json.Unmarshal([]byte(file), &data); err != nil {
			return config, err
		}
	}

	if cwd, err := os.Getwd(); err != nil {
		panic(err)
	} else {
		if data.Public == "" {
			config.Public = cwd
		} else {
			config.Public = path.Join(cwd, data.Public)
		}
	}

	// if data.CleanUrls != nil {
	// 	var boolValue bool
	// 	var strValue []string

	// 	if err := json.Unmarshal(data.CleanUrls, &boolValue); err == nil {
	// 		config.NoCleanUrls = !boolValue
	// 	} else if err := json.Unmarshal(data.CleanUrls, &strValue); err == nil {
	// 		config.CleanUrls = strValue
	// 	}
	// }

	// config.Rewrites = data.Rewrites
	// config.Redirects = data.Redirects
	// config.Headers = data.Headers
	config.Proxy = data.Proxy

	if data.DirectoryListing != nil {
		var boolValue bool
		var strValue []string

		if err := json.Unmarshal(data.DirectoryListing, &boolValue); err == nil {
			config.NoDirectoryListing = !boolValue
		} else if err := json.Unmarshal(data.DirectoryListing, &strValue); err == nil {
			config.DirectoryListing = strValue
		}
	}

	if data.Unlisted != nil {
		config.Unlisted = *data.Unlisted
	}
	// Provide senible defaults for these
	if len(config.Unlisted) == 0 {
		config.Unlisted = append(config.Unlisted, ".DS_Store", ".git")
	}

	// config.TrailingSlash = true
	// if data.TrailingSlash != nil {
	// 	config.TrailingSlash = *data.TrailingSlash
	// }
	config.RenderSingle = data.RenderSingle
	// if config.RenderSingle {
	// 	config.Rewrites = append(config.Rewrites, ConfigRewrite{
	// 		Source:      "**",
	// 		Destination: "/index.html",
	// 	})
	// }
	// config.Symlinks = data.Symlinks
	config.Ssl = data.Ssl

	return config, nil
}
