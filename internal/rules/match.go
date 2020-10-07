package rules

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/devjoes/kreme/internal/common"
	"github.com/devjoes/kreme/pkg/datasources"
	"github.com/golang/glog"
)

type Match struct {
	Index          int
	Hosts          map[string]bool
	URLRegex       *regexp.Regexp
	Always         bool
	ErrorIfMissing []string
	Headers        map[string][]string
	DataSources    map[string]datasources.DataSourceInfo
}

type MatchOptions struct {
	Hosts          []string                                 `yaml:"hosts"`
	URLRegex       string                                   `yaml:"urlRegex"`
	Always         bool                                     `yaml:"always"`
	ErrorIfMissing []string                                 `yaml:"errorIfMissing"`
	Headers        map[string]interface{}                   `yaml:"headers"`
	DataSources    map[string]datasources.DataSourceOptions `yaml:"dataSources"`
}

// NewMatch creates a new match which is used to pick data sources/templates
func NewMatch(index int, hosts []string, urlRegex string, alwaysMatch bool, errorIfMissing []string, headers map[string]interface{}) (Match, error) {
	var rx *regexp.Regexp
	if urlRegex != "" {
		rx = regexp.MustCompile(urlRegex)
	}
	hostsMap := make(map[string]bool, len(hosts))
	for _, k := range hosts {
		hostsMap[strings.ToLower(k)] = true
	}
	common.LowerArray(&errorIfMissing)

	cleanHeaders := make(map[string][]string, len(headers))
	for k, x := range headers {
		lk := strings.ToLower(k)
		s, ok := x.(string)
		if ok {
			cleanHeaders[lk] = []string{s}
			continue
		}
		a, ok := x.([]interface{})
		if ok {
			var values []string
			for _, v := range a {
				s, ok := v.(string)
				if !ok {
					break
				}
				values = append(values, s)
			}
			if values != nil {
				cleanHeaders[lk] = values
				continue
			}
		}
		return Match{}, fmt.Errorf("matches[%d].headers[%s] should be a string or an array of strings", index, k)
	}

	if !alwaysMatch && len(hostsMap) == 0 && rx == nil {
		glog.Warningf("matches[%d] will never match", index)
	}

	return Match{
		Index: index,
		Hosts: hostsMap, URLRegex: rx,
		Always: alwaysMatch, ErrorIfMissing: errorIfMissing,
		Headers: cleanHeaders,
	}, nil
}

//GetMatch returns first valid match
func GetMatch(matches []Match, req *http.Request) *Match {
	for _, m := range matches {
		if m.Always || m.Hosts[strings.ToLower(req.URL.Host)] || (m.URLRegex != nil && m.URLRegex.Match([]byte(req.RequestURI))) {
			return &m
		}
	}
	return nil
}
