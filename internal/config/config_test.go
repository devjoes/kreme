package config

import (
	"regexp"
	"strings"
	"testing"

	"github.com/devjoes/kreme/pkg/datasources"
	"github.com/stretchr/testify/assert"
)

const yamlConfig = `
proxy:
  mode: httpProxy
  port: 1234
  exposeErrorsToClient: true
  preserveHeaders: [Department]
cache:
  redisUrl: http://foo.com
matches:
- hosts:
    - "svc.ns"
  urlRegex: ".*"
  always: true
  errorIfMissing:
  - Department
  headers:
    Department: "{{.person.department}}"
`
const yamlConfigDataSources = `
  dataSources:
    getHeaders:
        type: headers
        cache: false
        options:
          headersToKeys:
            Host: hostKey
            Accept: foo
`

//   - jwtFoo:
//       type: jwt
//       cache: false
//       options:
//         oidcStuff: blah
//       data:
//         name: name
//         scope0:"scope.[0]"
//   - person:
//       type: http
//       cache: true
//       options:
//         url: "https://foo.com/getData/{{.jwtFoo.name}}"
//         method: POST
//         headers:
//           foo: bar
//         body: |
//           blah
//       data:
//         firstName: "person.[0].firstName"
//         department: "assignment.[0].department"
// `

func TestParsesBasicConfig(t *testing.T) {
	config, err := Parse(yamlConfig)

	assert.Nil(t, err)
	assert.Equal(t, "http://foo.com", config.Cache.RedisURL.String())
	assert.Equal(t, 1, len(config.Matches))
	assert.True(t, config.Matches[0].Hosts["svc.ns"])
	r, _ := regexp.Compile(".*")
	assert.Equal(t, r, config.Matches[0].URLRegex)
	assert.True(t, config.Matches[0].Always)
	assert.Equal(t, []string{"department"}, config.Matches[0].ErrorIfMissing)
	assert.Equal(t, map[string][]string{"department": {"{{.person.department}}"}}, config.Matches[0].Headers)
	assert.Equal(t, "httpproxy", config.Proxy.Mode)
	assert.Equal(t, uint16(1234), config.Proxy.Port)
	assert.True(t, config.Proxy.ExposeErrorsToClient)
	assert.Equal(t, []string{"department"}, config.Proxy.PreserveHeaders)

}

func TestParsesDataSources(t *testing.T) {
	//TODO: Make plugin
	config, err := Parse(yamlConfig + yamlConfigDataSources)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(config.Matches[0].DataSources))
	headers := config.Matches[0].DataSources["getHeaders"]
	assert.NotNil(t, headers)
}

func TestHeadersCanOnlyBeStringOrArray(t *testing.T) {
	config, err := Parse(strings.ReplaceAll(yamlConfig, "\"{{.person.department}}\"", "foo"))
	assert.Nil(t, err)
	assert.Equal(t, map[string][]string{"department": {"foo"}}, config.Matches[0].Headers)

	config, err = Parse(strings.ReplaceAll(yamlConfig, "\"{{.person.department}}\"", "[foo,bar]"))
	assert.Nil(t, err)
	assert.Equal(t, map[string][]string{"department": {"foo", "bar"}}, config.Matches[0].Headers)

	_, err = Parse(strings.ReplaceAll(yamlConfig, "\"{{.person.department}}\"", "1"))
	assert.NotNil(t, err)
}

func TestLoadsDataSources(t *testing.T) {
	datasources.DataSourceTypes = make(map[string]func(yml []byte) (datasources.DataSource, error))
	assert.Equal(t, 0, len(datasources.DataSourceTypes))
	_, err := Parse(yamlConfig + yamlConfigDataSources)
	assert.Nil(t, err)
	assert.Greater(t, len(datasources.DataSourceTypes), 0)
}
