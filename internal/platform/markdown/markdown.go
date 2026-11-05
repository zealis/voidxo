package markdown

import (
	"bytes"
	"html"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
)

var tagPattern = regexp.MustCompile(`<[^>]*>`)

type Renderer struct {
	converter goldmark.Markdown
	policy    *bluemonday.Policy
}

func New() *Renderer {
	return &Renderer{
		converter: goldmark.New(),
		policy:    bluemonday.UGCPolicy(),
	}
}

func (r *Renderer) Render(source string) (rendered, plain string, err error) {
	var output bytes.Buffer
	if err := r.converter.Convert([]byte(source), &output); err != nil {
		return "", "", err
	}
	sanitized := r.policy.SanitizeBytes(output.Bytes())
	return string(sanitized), PlainText(string(sanitized)), nil
}

func PlainText(renderedHTML string) string {
	withoutTags := tagPattern.ReplaceAllString(renderedHTML, " ")
	return strings.Join(strings.Fields(html.UnescapeString(withoutTags)), " ")
}
