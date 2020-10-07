package datasources

import (
	"fmt"
	"net/http"
	"time"

	"gopkg.in/yaml.v3"
)

type DataSourceOptions struct {
	Type      string      `yaml:"type"`
	Cache     bool        `yaml:"cache"`
	DependsOn []string    `yaml:"dependsOn"`
	Options   interface{} `yaml:"options"`
}

type DataSourceInfo struct {
	Name       string
	Type       string
	Cache      bool
	DependsOn  []string
	DataSource DataSource
}

type InitDataSourceFunc func(config interface{}) (string, func(yml []byte) (DataSource, error), error)
type DataSource interface {
	Setup(request *http.Request, context *map[string]interface{}, dataSourceHelper *DataSourceHelper) (string, *time.Duration, error)
	GetData() (interface{}, error)
	GetTemplateData(data *interface{}) (map[string]interface{}, error)
}

func Parse(ds map[string]DataSourceOptions) (map[string]DataSourceInfo, error) {
	sources := make(map[string]DataSourceInfo, len(ds))
	for k, d := range ds {
		dataSrc, err := initDataSource(k, d)
		if err != nil {
			return nil, err
		}
		sources[k] = dataSrc
	}
	return sources, nil
}

var DataSourceTypes map[string](func(yml []byte) (DataSource, error)) = make(map[string]func(yml []byte) (DataSource, error))

func initDataSource(name string, dso DataSourceOptions) (DataSourceInfo, error) {
	create, found := DataSourceTypes[dso.Type]
	if !found {
		return DataSourceInfo{}, fmt.Errorf("could not find data source '%s'", dso.Type)
	}

	yml, err := yaml.Marshal(dso)
	ds, err := create(yml)
	if err != nil {
		return DataSourceInfo{}, err
	}

	return DataSourceInfo{Name: name, Cache: dso.Cache, DependsOn: dso.DependsOn, Type: dso.Type, DataSource: ds}, nil
}
func getKeys(input map[string]DataSource) []string {
	var keys []string
	for k := range input {
		keys = append(keys, k)
	}
	return keys
}
