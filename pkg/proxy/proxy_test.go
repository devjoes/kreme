package proxy

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const Path = "/foo/bar"
const RequestHeader = "Req-Foo"
const RequestHeaderFromProxy = "Req-Bar"
const RequestHeaderStripped = "TE"
const ResponseHeader = "Res-Bar"
const Ok = "OK"
const Get = "GET"

const ExtraClientHeader = "Client"
const ExtraProxyHeader = "Proxy"

func TestHandleRequestMultipleClientHeaders(t *testing.T) {
	headers := []string{ExtraClientHeader, ExtraClientHeader}
	testHandleRequest(t, headers, nil, nil, func(req *http.Request) {
		if !reflect.DeepEqual(req.Header[ExtraClientHeader], headers) {
			t.Errorf("Expected %s got %s", headers, req.Header[ExtraClientHeader])
		}
	})
}
func TestHandleRequestMultipleProxyHeaders(t *testing.T) {
	headers := []string{ExtraProxyHeader, ExtraProxyHeader}
	testHandleRequest(t, nil, headers, nil, func(req *http.Request) {
		if !reflect.DeepEqual(req.Header[ExtraProxyHeader], headers) {
			t.Errorf("Expected %s got %s", headers, req.Header[ExtraProxyHeader])
		}
	})
}
func TestReplaceOverlappingClientAndProxyHeaders(t *testing.T) {
	proxyHeaders := []string{ExtraClientHeader}
	clientHeaders := []string{strings.ToUpper(ExtraClientHeader), ExtraClientHeader}
	testHandleRequest(t, clientHeaders, proxyHeaders, nil, func(req *http.Request) {
		if !reflect.DeepEqual(req.Header[ExtraClientHeader], proxyHeaders) {
			t.Errorf("Expected %s got %s", proxyHeaders, req.Header[ExtraProxyHeader])
		}
	})
}

func TestAppendOverlappingClientAndProxyHeaders(t *testing.T) {
	proxyHeaders := []string{ExtraProxyHeader, ExtraProxyHeader, ExtraClientHeader}
	clientHeaders := []string{ExtraClientHeader, ExtraClientHeader}
	testHandleRequest(t, clientHeaders, proxyHeaders, map[string]struct{}{ExtraClientHeader: {}},
		func(req *http.Request) {
			if len(req.Header[ExtraProxyHeader]) != 2 {
				t.Errorf("Expected count of 2, got %s", req.Header[ExtraProxyHeader])
			}
			expected := append(clientHeaders, ExtraClientHeader)
			if !reflect.DeepEqual(req.Header[ExtraClientHeader], expected) {
				t.Errorf("Expected %s got %s", expected, req.Header[ExtraProxyHeader])
			}
		})
}

func TestHandleRequestWithSingleHeader(t *testing.T) {
	testHandleRequest(t, nil, nil, nil, nil)
}

func TestHandlesError(t *testing.T) {
	const errText = "Bang!"
	causeError := func(upstream bool, expectedCode int, exposeError bool) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if upstream {
				http.Error(rw, errText, 500)
			} else {
				rw.Write([]byte(Ok))
			}
		}))
		defer server.Close()

		writer := httptest.NewRecorder()
		p := Proxy{
			getHeaders: func(request *http.Request) (map[string][]string, error) {
				if !upstream {
					return nil, errors.New(errText)
				}
				return map[string][]string{}, nil
			},
			exposeErrorsToClient: exposeError,
		}
		req := httptest.NewRequest(Get, server.URL+Path, nil)
		p.ProxyRequest(writer, req)
		result := writer.Result()
		assert.Equal(t, expectedCode, result.StatusCode)
		bodyBytes, _ := ioutil.ReadAll(result.Body)
		body := string(bodyBytes)
		if exposeError {
			assert.Contains(t, body, errText)
		} else {
			assert.Regexp(t, regexp.MustCompile("Request error [0-9]+ see log for further details."), body)
			assert.NotContains(t, body, errText)
		}
	}

	t.Run("Proxy error hidden from client", func(t *testing.T) {
		causeError(false, 500, false)
	})
	t.Run("Upstream error hidden from client", func(t *testing.T) {
		causeError(true, 504, false)
	})

	t.Run("Proxy error", func(t *testing.T) {
		causeError(false, 500, true)
	})
	t.Run("Upstream error", func(t *testing.T) {
		causeError(true, 504, true)
	})
}
func testHandleRequest(t *testing.T, extraClientHeaders []string, extraProxyHeaders []string, headersToKeep map[string]struct{}, extraServerValidation func(*http.Request)) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, Get, req.Method)
		assert.Equal(t, Path, req.URL.String())
		assert.Nil(t, req.Header[RequestHeaderStripped])
		assert.Truef(t, headerEqual(req.Header, RequestHeader, RequestHeader),
			"req.Header[\"%s\"]: %s\n", RequestHeader, req.Header[RequestHeader][0])
		assert.Truef(t, headerEqual(req.Header, RequestHeaderFromProxy, RequestHeaderFromProxy),
			"req.Header[\"%s\"][0]: %s\n", RequestHeaderFromProxy, req.Header[RequestHeaderFromProxy][0])

		if extraServerValidation != nil {
			extraServerValidation(req)
		}

		rw.Header().Set(ResponseHeader, ResponseHeader)
		rw.Write([]byte(Ok))
	}))
	defer server.Close()

	writer := httptest.NewRecorder()
	req := httptest.NewRequest(Get, server.URL+Path, nil)
	req.Header.Set(RequestHeaderStripped, "Strip this")
	req.Header.Set(RequestHeader, RequestHeader)
	for _, h := range extraClientHeaders {
		req.Header.Add(h, h)
	}

	localExtraProxyHeaders := extraProxyHeaders
	p := Proxy{
		clientHeadersToNotOverrite: headersToKeep,
		getHeaders: func(request *http.Request) (map[string][]string, error) {
			headers := make(map[string][]string)
			headers[RequestHeaderFromProxy] = []string{RequestHeaderFromProxy}
			for _, h := range localExtraProxyHeaders {
				headers[h] = append(headers[h], h)
			}
			return headers, nil
		},
	}
	p.ProxyRequest(writer, req)
	result := writer.Result()
	if !headerEqual(result.Header, ResponseHeader, ResponseHeader) {
		t.Errorf("result.Header[\"%s\"]: %s\n", ResponseHeader, result.Header[ResponseHeader])
	}

	body, err := ioutil.ReadAll(result.Body)
	if err != nil {
		t.Error(err)
	}
	if bytes.Compare(body, []byte(Ok)) != 0 {
		t.Errorf("Body was %s\n", body)
	}
}

func headerEqual(header http.Header, key string, value string) bool {
	return len(header[key]) == 0 || header[key][0] == value
}
