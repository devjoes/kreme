package datasources

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/golang/glog"
)

type DataSourceHelper struct {
	dataSource *interface{}
	context    *map[string]interface{}
	request    *http.Request
}

func NewDataSourceHelper(dataSourceInfo DataSourceInfo, context *map[string]interface{}, request *http.Request) DataSourceHelper {
	var dataSource interface{}
	dataSource = dataSourceInfo.DataSource
	return DataSourceHelper{dataSource: &dataSource, context: context, request: request}
}

func (dsh *DataSourceHelper) Context() map[string]interface{} {
	return *dsh.context
}

func (dsh *DataSourceHelper) Request() http.Request {
	return *dsh.request
}

func (dsh *DataSourceHelper) DataSource() interface{} {
	return *dsh.dataSource
}
func (dsh *DataSourceHelper) ExecuteTemplate(t *template.Template) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	err := t.Execute(&buf, dsh)
	if err != nil {
		glog.Warningf("Cant render template '%s'\n%s\n%v\n", t, err.Error(), *dsh)
	}
	return &buf, err
}

func (dsh *DataSourceHelper) ExecuteTemplateStr(templateStr string) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	t, err := template.New("").Parse(templateStr)
	if (err) != nil {
		return &buf, err
	}
	return dsh.ExecuteTemplate(t)
}
