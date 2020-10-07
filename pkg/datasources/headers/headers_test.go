package headers

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeadersMapToKeys(t *testing.T) {
	const headerName = "Foo"
	const headerValue = "bar"
	const key = "baz"
	headersToKeys := make(map[string]string)
	headersToKeys[headerName] = key
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.Header.Add(headerName, headerValue)
	headers := &Headers{
		Options: struct {
			HeadersToKeys map[string]string "yaml:\"headersToKeys\""
		}{HeadersToKeys: headersToKeys}}
	cacheKey, cacheFor, err := headers.Setup(req, nil, nil)
	assert.Equal(t, "", cacheKey)
	assert.Nil(t, cacheFor)
	assert.Nil(t, err)
	result, err := headers.GetData()
	assert.Nil(t, err)
	assert.Equal(t, []string{headerValue}, result.(map[string][]string)[key])
}

func TestMultipleHeadersMapToArrays(t *testing.T) {
	const headerName = "Foo"
	const headerValue1 = "bar1"
	const headerValue2 = "bar2"
	const key = "baz"
	headersToKeys := make(map[string]string)
	headersToKeys[headerName] = key
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.Header.Add(headerName, headerValue1)
	req.Header.Add(headerName, headerValue2)
	headers := &Headers{
		Options: struct {
			HeadersToKeys map[string]string "yaml:\"headersToKeys\""
		}{HeadersToKeys: headersToKeys}}
	cacheKey, cacheFor, err := headers.Setup(req, nil, nil)
	assert.Equal(t, "", cacheKey)
	assert.Nil(t, cacheFor)
	assert.Nil(t, err)
	result, err := headers.GetData()
	assert.Nil(t, err)
	assert.Equal(t, []string{headerValue1, headerValue2}, result.(map[string][]string)[key])
}
