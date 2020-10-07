package server

import (
	"strings"

	"github.com/devjoes/kreme/internal/config"
	"github.com/pkg/errors"
	"github.com/valyala/fasttemplate"
)

// HeaderBuilder converts data in to HTTP headers
type HeaderBuilder struct {
	headerToTemplate map[string]*fasttemplate.Template
}

// NewHeaderBuilder creates a new TemplateBuilder and parses the Go templates
func NewHeaderBuilder(matchIndex int, headerToTemplateStrs map[string]string) (*HeaderBuilder, error) {
	headerToTemplate := make(map[string]*fasttemplate.Template, len(headerToTemplateStrs))
	for headerName, templateStr := range headerToTemplateStrs {
		t := fasttemplate.New(templateStr, "{{", "}}") //TODO: multi headers
		headerToTemplate[headerName] = t
	}
	hb := HeaderBuilder{headerToTemplate: headerToTemplate}
	return &hb, nil
}

// generateHeaders attempts to find a matching template and generate headers using supplied data
func (h *HeaderBuilder) generateHeaders(matchIndex int, data map[string]interface{}) (map[string][]string, error) {
	headers := make(map[string][]string)
	headersToTemplates := h.headerToTemplate
	if headersToTemplates == nil {
		return nil, errors.Errorf("No template for match '%d' was found\n", matchIndex)
	}

	for headerName, t := range headersToTemplates {
		headerValues := t.ExecuteString(data)

		if strings.TrimSpace(headerValues) == "" {
			headers[headerName] = nil
		} else {
			headers[headerName] = strings.Split(headerValues, "\n")
		}
	}
	return headers, nil
}

func FromConfig(config *config.Config) ([]*HeaderBuilder, error) {
	headerBuilders := make([]*HeaderBuilder, len(config.Matches))
	for i, m := range config.Matches {
		//TODO: multival
		singleValHeaders := make(map[string]string, len(m.Headers))
		for h, v := range m.Headers {
			singleValHeaders[h] = v[0]
		}
		hb, err := NewHeaderBuilder(i, singleValHeaders)
		if err != nil {
			return nil, err
		}
		headerBuilders[i] = hb
	}
	return headerBuilders, nil
}
