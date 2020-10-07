package proxy

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

type Config struct {
	Mode                 string   `yaml:"mode"`
	Port                 uint16   `yaml:"port"`
	ExposeErrorsToClient bool     `yaml:"exposeErrorsToClient"`
	PreserveHeaders      []string `yaml:"preserveHeaders"` //TODO: Rename to something like "dont replace client headers"
}

type getHeadersDelegate func(request *http.Request) (map[string][]string, error)

func setHeaders(req *http.Request, headers map[string][]string, clientHeadersNotToOverrite map[string]struct{}) (map[string][]string, []string) {
	overriteAllClientHeaders := len(clientHeadersNotToOverrite) == 0
	toRemove := make([]string, 0, len(req.Header))
	for h := range headers {
		if headers[h] == nil {
			toRemove = append(toRemove, h)
			continue
		}
		if overriteAllClientHeaders {
			req.Header[h] = nil
		} else {
			_, skip := clientHeadersNotToOverrite[h]
			if !skip {
				req.Header[h] = nil
			}
		}
		for _, v := range headers[h] {
			req.Header.Add(h, v)
		}

	}
	return headers, toRemove
}

func stripHeaders(req *http.Request, toRemove []string) {
	var defaultHeadersToRemove = [...]string{"Keep-Alive", "Transfer-Encoding", "TE", "Connection", "Trailer", "Upgrade", "Proxy-Authorization", "Proxy-Authenticate"}

	// TODO: handle Connection header
	for _, h := range append(toRemove, defaultHeadersToRemove[:]...) {
		req.Header[h] = nil
	}
}

func (p *Proxy) handleRequestError(err error, request *http.Request, response http.ResponseWriter, code int) error {
	id := rand.Uint64()
	glog.Warningf("Request error %d: Could not process URL %s: %s\n", id, request.RequestURI, err.Error())
	if p.exposeErrorsToClient {
		http.Error(response, err.Error(), code)
		return err
	}
	http.Error(response, fmt.Sprintf("Request error %d see log for further details.\n", id), code)
	return err
}

func errorFromHTTPResonse(resp *http.Response) error {
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return errors.New(string(data))
}

// ProxyRequest handles a request
func (p *Proxy) ProxyRequest(writer http.ResponseWriter, req *http.Request) {
	headers, err := p.getHeaders(req)
	if err != nil {
		p.handleRequestError(err, req, writer, 500)
		return
	}
	headers, toRemove := setHeaders(req, headers, p.clientHeadersToNotOverrite)
	stripHeaders(req, toRemove)
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil || resp.StatusCode/100 == 5 {
		if err == nil {
			err = errorFromHTTPResonse(resp)
		}
		p.handleRequestError(errors.Wrapf(err, "Upstream HTTP error: %s\n", resp.Status), req, writer, 504)
		return
	}
	glog.Infoln(req.URL)
	defer resp.Body.Close()
	copyHeader(writer.Header(), resp.Header)
	writer.WriteHeader(resp.StatusCode)
	io.Copy(writer, resp.Body)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// Proxy HTTP proxy
type Proxy struct {
	port                       uint16
	getHeaders                 getHeadersDelegate
	exposeErrorsToClient       bool
	clientHeadersToNotOverrite map[string]struct{}
}

// StartProxy starts the HTTP proxy
func (p *Proxy) StartProxy() error {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", p.port),
		Handler: http.HandlerFunc(p.ProxyRequest),
	}
	log.Printf("Starting on port %d", p.port)
	return server.ListenAndServe()
}

func NewProxy(config *Config, getHeaders getHeadersDelegate) (*Proxy, error) {
	switch strings.ToLower(config.Mode) {
	case "httpproxy":
		return NewHTTPProxy(config.Port, config.ExposeErrorsToClient, config.PreserveHeaders, getHeaders), nil
		//TODO: Reverse? Envoy filter based?
	}
	return nil, fmt.Errorf("unknown proxy mode '%s'", config.Mode)
}

//NewHTTPProxy Creates a new http proxy
func NewHTTPProxy(port uint16, exposeErrorsToClient bool, clientHeadersToNotOverrite []string, getHeaders getHeadersDelegate) *Proxy {
	p := &Proxy{port: port, exposeErrorsToClient: exposeErrorsToClient, clientHeadersToNotOverrite: toHashSet(clientHeadersToNotOverrite)}
	p.getHeaders = getHeaders
	return p
}

func toHashSet(arr []string) map[string]struct{} {
	hs := make(map[string]struct{}, len(arr))
	for _, x := range arr {
		hs[x] = struct{}{}
	}
	return hs
}
