package server

import (
	"net/http"

	"github.com/devjoes/kreme/internal/data"
	"github.com/devjoes/kreme/internal/rules"
	"github.com/golang/glog"
)

func GenerateHeadersForRequest(builders []*HeaderBuilder, matches []rules.Match) func(req *http.Request) (map[string][]string, error) {
	return func(req *http.Request) (map[string][]string, error) {
		m := rules.GetMatch(matches, req)
		if m == nil {
			glog.Infof("Match not found for %s\n", req.RequestURI)
			//TODO: Option to block
			return make(map[string][]string), nil
		}
		data, err := data.FromRequest(m, req)
		if err != nil {
			return nil, err
		}
		return builders[m.Index].generateHeaders(m.Index, data)
	}
}
