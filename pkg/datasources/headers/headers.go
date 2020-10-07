package headers

import (
	"net/http"
	"time"

	"github.com/devjoes/kreme/pkg/datasources"
	"gopkg.in/yaml.v2"
)

var InitDataSource datasources.InitDataSourceFunc = func(config interface{}) (string, func(yml []byte) (datasources.DataSource, error), error) {
	return "headers", func(yml []byte) (datasources.DataSource, error) {
		var ds datasources.DataSource
		x := Headers{}
		if err := yaml.Unmarshal(yml, &x); err != nil {
			return nil, err
		}
		ds = &x
		return ds, nil
	}, nil
}

type Headers struct {
	Type    string
	Cache   bool
	Options struct {
		HeadersToKeys map[string]string `yaml:"headersToKeys"`
	} `yaml:"options"`
	request *http.Request
}

// func (h Headers) Setup(raw datasources.DataSourceOptions) error {
// 	h.Cache = raw.Cache
// 	h.Type = raw.Type
// 	yml, err := yaml.Marshal(&raw.Options)
// 	if err != nil {
// 		return err
// 	}
// 	if err := yaml.Unmarshal(yml, &h.Options); err != nil {
// 		return err
// 	}
// 	// h.Options = &struct{ HeadersToKeys map[string]string }{HeadersToKeys: map[string]string{
// 	// 	"hostKey": "Host",
// 	// 	"foo":     "Accept",
// 	// }}

// 	// //TODO: Work out how to make dynamic
// 	// opts, ok := raw.Options.(struct{ HeadersToKeys map[string]string })
// 	// if !ok {
// 	// 	return fmt.Errorf("Could not get Headers.Options from:\n%v\n", raw.Options)
// 	// }
// 	// h.Options = &opts
// 	// data, ok := raw.Data.(map[string]string)
// 	// if !ok {
// 	// 	return fmt.Errorf("Could not get Headers.Data from:\n%v\n", raw.Data)
// 	// }
// 	// h.Data = data
// 	return nil
// }

func (h *Headers) Setup(request *http.Request, context *(map[string]interface{}), helper *datasources.DataSourceHelper) (string, *time.Duration, error) {
	h.request = request
	return "", nil, nil
}

func (h *Headers) GetData() (interface{}, error) {
	keysToValues := make(map[string][]string)
	for headerName := range h.Options.HeadersToKeys {
		headerVal := h.request.Header[headerName]
		if headerVal != nil && len(headerVal) > 0 {
			keysToValues[h.Options.HeadersToKeys[headerName]] = headerVal
		}
	}
	return keysToValues, nil
}

func (h Headers) GetTemplateData(data *interface{}) (map[string]interface{}, error) {
	headerData := (*data).(map[string][]string)
	output := make(map[string]interface{}, len(headerData))
	for k, v := range headerData {
		output[k] = v[0]
	}
	return output, nil
}
