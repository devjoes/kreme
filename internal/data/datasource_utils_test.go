package data

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/devjoes/kreme/internal/rules"
	"github.com/devjoes/kreme/pkg/datasources"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasttemplate"
)

const ds1Name = "testDataSource1"
const ds2Name = "testDataSource2"
const ds3Name = "testDataSource3"
const foo = "foo"
const bar = "bar"

func TestGetsDataFromSource(t *testing.T) {
	var fFoo, fBar fasttemplate.TagFunc
	fFoo = func(w io.Writer, tag string) (int, error) { return w.Write([]byte(foo)) }
	fBar = func(w io.Writer, tag string) (int, error) { return w.Write([]byte(bar)) }
	t.Run("supports string", func(t *testing.T) {
		test(t, []string{foo, bar}, map[string]interface{}{ds1Name + "_First": foo, ds1Name + "_Second": bar})
	})
	t.Run("supports byte array", func(t *testing.T) {
		test(t, [][]byte{[]byte(foo), []byte(bar)}, map[string]interface{}{ds1Name + "_First": []byte(foo), ds1Name + "_Second": []byte(bar)})
	})
	t.Run("supports func", func(t *testing.T) {
		test(t, []fasttemplate.TagFunc{fFoo, fBar}, map[string]interface{}{ds1Name + "_First": "func foo", ds1Name + "_Second": "func bar"})
	})
	t.Run("supports mix", func(t *testing.T) {
		test(t, []interface{}{foo, fBar}, map[string]interface{}{ds1Name + "_First": foo, ds1Name + "_Second": "func bar"})
	})
}
func test(t *testing.T, input interface{}, output map[string]interface{}) {
	ds1 := datasources.DataSourceInfo{
		Name:       ds1Name,
		DataSource: &testDataSource{data: input},
		Type:       "TestDataSource",
	}

	m := rules.Match{
		DataSources: map[string]datasources.DataSourceInfo{
			ds1Name: ds1,
		},
	}
	req, _ := http.NewRequest("GET", "/foo", nil)
	data, err := FromRequest(&m, req)
	assert.Nil(t, err)
	for i, v := range data {
		if tf, ok := v.(fasttemplate.TagFunc); ok {
			buf := new(bytes.Buffer)
			tf(buf, "tag")
			data[i] = "func " + buf.String()
		}
	}
	assert.Equal(t, output, data)
	assert.Equal(t, 1, ds1.DataSource.(*testDataSource).SetupCalls)
	assert.Equal(t, 1, ds1.DataSource.(*testDataSource).GetDataCalls)
}

var getDataCounter safeCounter = safeCounter{}
var templateDataCounter safeCounter = safeCounter{}

func getDependentDs() []datasources.DataSourceInfo {
	counterFunc := func(text string) fasttemplate.TagFunc {
		count := templateDataCounter.Inc()
		return func(w io.Writer, tag string) (int, error) {
			fmt.Printf("%s %d\n", text, count)
			return w.Write([]byte(fmt.Sprintf("%s %d", text, count)))
		}
	}
	ds1 := datasources.DataSourceInfo{
		Name: ds1Name,
		DataSource: &testDataSource{
			getData: func(requestData *(map[string]interface{})) interface{} {
				time.Sleep(time.Millisecond * 500)
				return []fasttemplate.TagFunc{counterFunc("foo1"), counterFunc("bar1")}
			},
		},
		Type: "TestDataSource",
	}
	ds2 := datasources.DataSourceInfo{
		Name: ds2Name,
		DataSource: &testDataSource{
			getData: func(requestData *(map[string]interface{})) interface{} {
				return []fasttemplate.TagFunc{
					counterFunc("foo2"),
					counterFunc(fmt.Sprintf("%s returned %d entries. getDataCounter is %d", ds1Name, len((*requestData)[ds1Name].([]fasttemplate.TagFunc)), getDataCounter.Val())),
					counterFunc(fmt.Sprintf("requestData is %v.", *requestData)),
				}
			},
		},
		Type:      "TestDataSource",
		DependsOn: []string{ds1Name},
	}
	ds3 := datasources.DataSourceInfo{
		Name:       ds3Name,
		DataSource: &testDataSource{data: []fasttemplate.TagFunc{counterFunc("foo3"), counterFunc("bar3")}},
		Type:       "TestDataSource",
	}
	return []datasources.DataSourceInfo{ds1, ds2, ds3}
}

func TestDependencies(t *testing.T) {
	getDataCounter = safeCounter{}
	templateDataCounter = safeCounter{}
	depDs := getDependentDs()
	m := rules.Match{
		DataSources: map[string]datasources.DataSourceInfo{
			ds1Name: depDs[0],
			ds2Name: depDs[1],
			ds3Name: depDs[2],
		},
	}
	req, _ := http.NewRequest("GET", "/foo", nil)
	data, err := FromRequest(&m, req)
	assert.Nil(t, err)
	for i, v := range data {
		if tf, ok := v.(fasttemplate.TagFunc); ok {
			buf := new(bytes.Buffer)
			tf(buf, "tag")
			data[i] = "func " + buf.String()
		}
	}
	for _, ds := range m.DataSources {
		assert.Equal(t, 1, ds.DataSource.(*testDataSource).SetupCalls)
		assert.Equal(t, 1, ds.DataSource.(*testDataSource).GetDataCalls)
	}

	assert.Regexp(t, regexp.MustCompile("func foo3 [1-4]"), data[ds3Name+"_First"])
	assert.Regexp(t, regexp.MustCompile("func bar3 [1-4]"), data[ds3Name+"_Second"])
	assert.Regexp(t, regexp.MustCompile("func foo1 [1-4]"), data[ds1Name+"_First"])
	assert.Regexp(t, regexp.MustCompile("func bar1 [1-4]"), data[ds1Name+"_Second"])
	assert.Regexp(t, regexp.MustCompile("func foo2 [5-9]"), data[ds2Name+"_First"])
	assert.Regexp(t, regexp.MustCompile("func "+ds1Name+" returned 2 entries\\. getDataCounter is 3 [5-9]"), data[ds2Name+"_Second"])

}

type testDataSource struct {
	Options              struct{} `yaml:"options"`
	SetupCalls           int
	GetDataCalls         int
	GetTemplateDataCalls int
	requestData          *(map[string]interface{})
	getData              func(requestData *(map[string]interface{})) interface{}
	data                 interface{}
}

func (h *testDataSource) Setup(request *http.Request, requestData *(map[string]interface{}), helper *datasources.DataSourceHelper) (string, *time.Duration, error) {
	h.SetupCalls++
	h.requestData = requestData
	return "", nil, nil
}

func (h *testDataSource) GetData() (interface{}, error) {
	h.GetDataCalls++
	getDataCounter.Inc()
	data := h.data
	if h.getData != nil {
		data = h.getData(h.requestData)
	}
	return data, nil
}

func (h testDataSource) GetTemplateData(data *interface{}) (map[string]interface{}, error) {
	var arr []interface{}

	switch d := (*data).(type) {
	case []string:
		{
			for _, v := range d {
				arr = append(arr, v)
			}
			break
		}
	case [][]byte:
		{
			for _, v := range d {
				arr = append(arr, v)
			}
			break
		}
	case []fasttemplate.TagFunc:
		{
			for _, v := range d {
				arr = append(arr, v)
			}
			break
		}
	default:
		{
			arr = (*data).([]interface{})
		}
	}

	return map[string]interface{}{"First": arr[0], "Second": arr[1]}, nil
}

func TestGroupsByDependency(t *testing.T) {
	const extraName = "extra"
	getGrouping := func(complete map[string]bool) map[int][]datasources.DataSourceInfo {
		depDs := getDependentDs()
		dsMap := make(map[string]datasources.DataSourceInfo)
		for _, ds := range depDs {
			dsMap[ds.Name] = ds
			dsMap[ds.Name+"2"] = ds
		}
		extra := dsMap[ds3Name]
		extra.Name = extraName
		extra.DependsOn = []string{ds1Name, ds3Name}
		dsMap[extraName] = extra

		return getDsGroupedByDependencyCount(dsMap, complete)
	}

	t.Run("Two levels", func(t *testing.T) {
		grouped := getGrouping(make(map[string]bool))
		assert.Equal(t, 4, len(grouped[0]))
		assert.Equal(t, 2, len(grouped[1]))
		assert.Equal(t, ds2Name, grouped[1][0].Name)
		assert.Equal(t, 1, len(grouped[2]))
		assert.Equal(t, extraName, grouped[2][0].Name)
	})

	t.Run("One level", func(t *testing.T) {
		grouped := getGrouping(map[string]bool{ds1Name: true})
		assert.Equal(t, 6, len(grouped[0]))
		assert.Equal(t, 1, len(grouped[1]))
		assert.Equal(t, extraName, grouped[1][0].Name)
	})

	t.Run("0 levels", func(t *testing.T) {
		grouped := getGrouping(map[string]bool{ds1Name: true, ds3Name: true})
		assert.Equal(t, 7, len(grouped[0]))
	})
}

func TestValidateDependencyReturnsError(t *testing.T) {
	t.Run("Non existent dependency", func(t *testing.T) {
		missingDs := getDependentDs()
		missingDs[0].DependsOn = []string{"IDontExist"}
		assert.Error(t, ValidateDependencies(missingDs))
	})
	t.Run("Direct loop", func(t *testing.T) {
		circularDs := getDependentDs()
		circularDs[0].DependsOn = []string{circularDs[1].Name}
		circularDs[1].DependsOn = []string{circularDs[0].Name}
		assert.Error(t, ValidateDependencies(circularDs))
	})
	t.Run("Indirect loop", func(t *testing.T) {
		circularDs := getDependentDs()
		circularDs[0].DependsOn = []string{circularDs[1].Name}
		circularDs[1].DependsOn = []string{circularDs[2].Name}
		circularDs[2].DependsOn = []string{circularDs[0].Name}
		assert.Error(t, ValidateDependencies(circularDs))
	})
	t.Run("Big indirect loop", func(t *testing.T) {
		circularDs := getDependentDs()
		circularDs = append(circularDs, getDependentDs()...)
		circularDs[5].Name = "foo"
		circularDs[4].Name = "bar"
		circularDs[3].Name = "baz"
		circularDs[5].DependsOn = []string{circularDs[4].Name}
		circularDs[4].DependsOn = []string{circularDs[3].Name}
		circularDs[3].DependsOn = []string{circularDs[2].Name}
		circularDs[2].DependsOn = []string{circularDs[1].Name}
		circularDs[1].DependsOn = []string{circularDs[0].Name}
		circularDs[0].DependsOn = []string{circularDs[5].Name}
		assert.Error(t, ValidateDependencies(circularDs))
	})
}

type safeCounter struct {
	v   int
	mux sync.Mutex
}

func (c *safeCounter) Inc() int {
	c.mux.Lock()
	c.v++
	defer c.mux.Unlock()
	return c.v
}

func (c *safeCounter) Val() int {
	c.mux.Lock()
	defer c.mux.Unlock()
	return c.v
}
