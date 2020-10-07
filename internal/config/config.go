package config

import (
	"flag"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/devjoes/kreme/internal/common"
	"github.com/devjoes/kreme/internal/data"
	"github.com/devjoes/kreme/internal/rules"
	"github.com/devjoes/kreme/pkg/datasources"
	"github.com/devjoes/kreme/pkg/datasources/headers"
	"github.com/devjoes/kreme/pkg/datasources/parsejwt"
	"github.com/devjoes/kreme/pkg/proxy"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type Options struct {
	Proxy proxy.Config `yaml:"proxy"`
	Cache struct {
		RedisURL string `yaml:"redisUrl"`
	} `yaml:"cache"`
	Matches   []rules.MatchOptions `yaml:"matches"`
	PluginDir string               `yaml:"pluginDir"`
}

type Config struct {
	Proxy proxy.Config
	Cache struct {
		RedisURL *url.URL
	}
	Matches   []rules.Match
	PluginDir string
}

func NewConfig(co Options) (*Config, error) {
	config := Config{
		PluginDir: "/etc/kreme/plugins/datasources",
		Proxy: proxy.Config{
			Mode:                 strings.ToLower(co.Proxy.Mode),
			Port:                 co.Proxy.Port,
			ExposeErrorsToClient: co.Proxy.ExposeErrorsToClient,
			PreserveHeaders:      co.Proxy.PreserveHeaders,
		}}

	common.LowerArray(&config.Proxy.PreserveHeaders)
	if co.PluginDir != "" {
		config.PluginDir = co.PluginDir
	}
	if co.Cache.RedisURL != "" {
		u, err := url.Parse(co.Cache.RedisURL)
		if err != nil {
			return nil, err
		}
		config.Cache = struct{ RedisURL *url.URL }{u}
	}
	loadDataSources(config.PluginDir, config)
	matches, err := parseMatches(co.Matches)
	config.Matches = matches
	return &config, err
}

func parseMatches(mos []rules.MatchOptions) ([]rules.Match, error) {
	matches := make([]rules.Match, len(mos))
	for i, mo := range mos {
		m, err := rules.NewMatch(i, mo.Hosts, mo.URLRegex, mo.Always, mo.ErrorIfMissing, mo.Headers)
		if err != nil {
			return nil, err
		}
		dataSources, err := datasources.Parse(mo.DataSources)
		if err != nil {
			return nil, err
		}
		var dsArr []datasources.DataSourceInfo
		for _, ds := range dataSources {
			dsArr = append(dsArr, ds)
		}
		if err := data.ValidateDependencies(dsArr); err != nil {
			return nil, err
		}
		m.DataSources = dataSources
		matches[i] = m
	}
	return matches, nil
}

func loadDataSources(path string, config Config) error {
	// This would ideally be in the datasources project
	// but it feels a bit too much like a circular reference to put it there
	initFuncs := []datasources.InitDataSourceFunc{
		headers.InitDataSource,
		parsejwt.InitDataSource,
	}
	if path != "" {
		//TODO: plugins
	}
	for _, ds := range initFuncs {
		var iConfig interface{}
		iConfig = config
		name, create, err := ds(iConfig)
		if err != nil {
			return errors.Wrapf(err, "error whilst initializing %s", name)
		}
		datasources.DataSourceTypes[name] = create
	}
	return nil
}

func Parse(yamlConfig string) (*Config, error) {
	co := Options{}
	err := yaml.Unmarshal([]byte(yamlConfig), &co)
	if err != nil {
		return nil, err
	}
	config, err := NewConfig(co)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func Load() (*Config, error) {
	var config = flag.String("config", "./config.yaml", "Config file path")
	flag.Parse()
	dat, err := ioutil.ReadFile(*config)
	if err != nil {
		return nil, err
	}
	return Parse(string(dat))
}

/*

	GetHeaders struct {
		Type    string `yaml:"type"`
		Cache   bool   `yaml:"cache"`
		Options struct {
		} `yaml:"options"`
		Data struct {
			HostKey string `yaml:"hostKey"`
			Foo     string `yaml:"foo"`
		} `yaml:"data"`
	} `yaml:"getHeaders,omitempty"`
	JwtFoo struct {
		Type    string `yaml:"type"`
		Cache   bool   `yaml:"cache"`
		Options struct {
			OidcStuff string `yaml:"oidcStuff"`
		} `yaml:"options"`
		Data struct {
			Name   string `yaml:"name"`
			Scope0 string `yaml:"scope0"`
		} `yaml:"data"`
	} `yaml:"jwtFoo,omitempty"`
	Person struct {
		Type    string `yaml:"type"`
		Cache   bool   `yaml:"cache"`
		Options struct {
			URL     string `yaml:"url"`
			Method  string `yaml:"method"`
			Headers struct {
				Foo string `yaml:"foo"`
			} `yaml:"headers"`
			Body string `yaml:"body"`
		} `yaml:"options"`
		Data struct {
			FirstName  string `yaml:"firstName"`
			Department string `yaml:"department"`
		} `yaml:"data"`
	} `yaml:"person,omitempty"`
*/
