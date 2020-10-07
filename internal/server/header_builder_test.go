package server

import (
	"testing"
)

func TestBlanksHeadersWhenThereIsNoData(t *testing.T) {
	const header = "Foo"
	const value = "bar"

	templates := map[string]string{header: value}
	builder, err := NewHeaderBuilder(0, templates)
	if err != nil {
		t.Error(err)
	}

	headers, err := builder.generateHeaders(0, make(map[string]interface{}))
	if err != nil {
		t.Error(err)
	}
	if len(headers) != 1 {
		t.Errorf("headers should have length 1 not %v\n", len(headers))
	}
	if (headers[header][0]) != value {
		t.Errorf("headers should be Foo:foo not %v\n", headers)
	}
}
