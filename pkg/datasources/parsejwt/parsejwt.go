package parsejwt

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/devjoes/kreme/pkg/datasources"
	"github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
)

var InitDataSource datasources.InitDataSourceFunc = func(config interface{}) (string, func(yml []byte) (datasources.DataSource, error), error) {
	return "parsejwt", func(yml []byte) (datasources.DataSource, error) {
		var ds datasources.DataSource
		pj := NewParseJwt(nil)
		if err := yaml.Unmarshal(yml, &pj); err != nil {
			return nil, err
		}
		ds = &pj
		return ds, nil
	}, nil
}

type ParseJwt struct {
	Type     string
	Cache    bool
	Options  ParseJwtOptions `yaml:"options"`
	request  *http.Request
	context  *map[string]interface{}
	helper   *datasources.DataSourceHelper
	tokenStr string
	token    jwt.Token
}

type ParseJwtOptions struct {
	IgnoreAuthorizationHeader bool   `yaml:"ignoreAuthorizationHeader"`
	TokenTemplate             string `yaml:"tokenTemplate"`
	ErrorIfTokenMissing       bool   `yaml:"errorIfTokenMissing"`
	SigningSecret             []byte `yaml:"signingSecret"`
}

func NewParseJwt(options *ParseJwtOptions) ParseJwt {
	pj := ParseJwt{
		Type: "parsejwt",
		Options: ParseJwtOptions{
			ErrorIfTokenMissing: true,
		},
	}
	if options != nil {
		pj.Options = *options
	}
	return pj
}

func (pj *ParseJwt) Setup(request *http.Request, context *(map[string]interface{}), helper *datasources.DataSourceHelper) (string, *time.Duration, error) {
	pj.request = request
	pj.context = context
	pj.helper = helper
	if err := pj.getToken(); err != nil {
		return "", nil, err
	}
	claims := pj.token.Claims.(jwt.MapClaims)
	var exp int64
	switch e := claims["exp"].(type) {
	case float64:
		exp = int64(e)
	case json.Number:
		v, _ := e.Int64()
		exp = v
	}
	expiresIn := time.Unix(exp, 0).Sub(time.Now().UTC()) //TODO: minus a bit
	//TODO: Use fingerprint - sha1 sum is crackable
	return fmt.Sprintf("%x", sha1.Sum([]byte(pj.tokenStr))), &expiresIn, nil
}

func (pj *ParseJwt) GetData() (interface{}, error) {
	data := make(map[string][]string)

	return data, nil
}

func (pj ParseJwt) GetTemplateData(data *interface{}) (map[string]interface{}, error) {
	output := make(map[string]interface{}, 0)
	return output, nil
}

func (pj *ParseJwt) getToken() error {
	var token string
	if !pj.Options.IgnoreAuthorizationHeader {
		authorization := pj.request.Header.Get("Authorization")
		const preamble = "Bearer,"
		if authorization != "" && strings.Index(authorization, "Bearer,") == 0 {
			token = strings.TrimSpace(strings.TrimPrefix(authorization, preamble))
		}
	}
	if token == "" {
		if pj.Options.TokenTemplate != "" {
			result, err := pj.helper.ExecuteTemplateStr(pj.Options.TokenTemplate)
			if err != nil {
				glog.Warningf("TokenTemplate errored with %s", err.Error())
			} else {
				token = result.String()
			}

		}
		if token == "" && pj.Options.ErrorIfTokenMissing {
			return errors.New("JWT token is missing")
		}
	}

	pj.tokenStr = token
	if token != "" {
		jwtToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}
			return pj.Options.SigningSecret, nil
		})
		if err != nil {
			return err
		}
		pj.token = *jwtToken
	}
	return nil
}

func (pj ParseJwt) Context() map[string]interface{} {
	return *pj.context
}
