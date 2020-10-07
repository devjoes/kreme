package data

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/devjoes/kreme/internal/rules"
	"github.com/devjoes/kreme/pkg/datasources"
)

type dsReturnInfo struct {
	name         string
	data         interface{}
	templateData map[string]interface{}
	err          error
}

func FromRequest(m *rules.Match, req *http.Request) (map[string]interface{}, error) {
	templateData := make(map[string]interface{}, len(m.DataSources))
	context := make(map[string]interface{}, len(m.DataSources))
	for _, ds := range m.DataSources {
		helper := datasources.NewDataSourceHelper(ds, &context, req)
		ds.DataSource.Setup(req, &context, &helper)
	}

	complete := make(map[string]bool)
	started := make(map[string]bool)
	for len(complete) < len(m.DataSources) {
		dsGrouped := getDsGroupedByDependencyCount(m.DataSources, complete)

		dsCh := make(chan (dsReturnInfo))
		waitCount := 0
		for _, ds := range dsGrouped[0] {
			if started[ds.Name] {
				continue
			}
			waitCount++
			started[ds.Name] = true
			go triggerGetData(ds, dsCh)
		}
		for i := 0; i < waitCount; i++ {
			dsRetInfo := <-dsCh
			complete[dsRetInfo.name] = true
			context[dsRetInfo.name] = dsRetInfo.data
			for k, v := range dsRetInfo.templateData {
				templateData[k] = v
			}
		}
	}
	return templateData, nil
}

func triggerGetData(ds datasources.DataSourceInfo, dsCh chan (dsReturnInfo)) {
	data, err := ds.DataSource.GetData()
	if err != nil {
		dsCh <- dsReturnInfo{err: err}
		return
	}
	templateData := make(map[string]interface{})
	if err := getTemplateData(&templateData, ds.DataSource, ds.Name, &data); err != nil {
		dsCh <- dsReturnInfo{err: err}
		return
	}
	dsCh <- dsReturnInfo{name: ds.Name, data: data, templateData: templateData}
}

func getDsGroupedByDependencyCount(dss map[string]datasources.DataSourceInfo, complete map[string]bool) map[int][]datasources.DataSourceInfo {
	grouped := make(map[int][]datasources.DataSourceInfo)
	for _, ds := range dss {
		key := len(whereNotIntersected(ds.DependsOn, complete))
		group, _ := grouped[key]
		grouped[key] = append(group, ds)
	}
	return grouped
}

func whereNotIntersected(arr1 []string, arr2Map map[string]bool) []string {
	var result []string
	for _, v := range arr1 {
		if found, _ := arr2Map[v]; !found {
			result = append(result, v)
		}
	}
	return result
}

func getTemplateData(output *map[string]interface{}, ds datasources.DataSource, dsName string, dsData *interface{}) error {
	td, err := ds.GetTemplateData(dsData)
	if err != nil {
		return err
	}
	for k, v := range td {
		(*output)[fmt.Sprintf("%s_%s", dsName, k)] = v
	}
	return nil
}

func ValidateDependencies(datasources []datasources.DataSourceInfo) error {
	nameToDeps := make(map[string][]string, len(datasources))
	var errorTxt []string
	for _, ds := range datasources {
		nameToDeps[ds.Name] = ds.DependsOn
	}
	for _, ds := range datasources {
		for _, dep := range ds.DependsOn {
			deps, found := nameToDeps[dep]
			if !found {
				errorTxt = append(errorTxt, fmt.Sprintf("%s depends on %s which was not found.", ds.Name, dep))
				continue
			}
			nameToDeps[dep] = append(deps, nameToDeps[ds.Name]...)
		}
	}
	for name, deps := range nameToDeps {
		for _, dep := range deps {
			if dep == name {
				errorTxt = append(errorTxt, fmt.Sprintf("Circular dependency found between %s and %s.", name, dep))
			}
		}
	}

	if len(errorTxt) > 0 {
		return fmt.Errorf("The following errors occurred:\n%s", strings.Join(errorTxt, "\n"))
	}
	return nil
}
