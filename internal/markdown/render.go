package markdown

import (
	"bytes"
	"html/template"

	"github.com/yuin/goldmark"
)

func Render(source string) template.HTML {
	if source == "" {
		return template.HTML("")
	}

	var buffer bytes.Buffer
	if err := goldmark.Convert([]byte(source), &buffer); err != nil {
		return template.HTML(template.HTMLEscapeString(source))
	}
	return template.HTML(buffer.String())
}
