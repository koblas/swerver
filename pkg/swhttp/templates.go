package swhttp

import (
	_ "embed"
	"text/template"
)

//go:embed error.html
var errorHtml string

//go:embed directory.html
var directoryHtml string

var errorTemplate = template.Must(template.New("error").Parse(errorHtml))
var directoryTemplate = template.Must(template.New("directory").Parse(directoryHtml))
