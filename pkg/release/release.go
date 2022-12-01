/*
Copyright The Helm Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package release

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/hinfinite/helm/pkg/chart"
	"github.com/mitchellh/copystructure"
)

// Release describes a deployment of a chart, together with the chart
// and the variables used to deploy that chart.
type Release struct {
	// Name is the name of the release
	Name string `json:"name,omitempty"`
	// Info provides information about a release
	Info *Info `json:"info,omitempty"`
	// Chart is the chart that was released.
	Chart *chart.Chart `json:"chart,omitempty"`
	// Config is the set of extra Values added to the chart.
	// These values override the default values inside of the chart.
	Config map[string]interface{} `json:"config,omitempty"`
	// ConfigRaw是config的string形式
	ConfigRaw string `json:"configRaw,omitempty"`
	// Manifest is the string representation of the rendered template.
	Manifest string `json:"manifest,omitempty"`
	// Hooks are all of the hooks declared for this release.
	Hooks []*Hook `json:"hooks,omitempty"`
	// Version is an int which represents the version of the release.
	Version int `json:"version,omitempty"`
	// Namespace is the kubernetes namespace of the release.
	Namespace string `json:"namespace,omitempty"`

	// 资源对应的chart名称 k=kind:name v=chart名称
	ResourceChartMap map[string]string `json:"resource_chart_map,omitempty"`
	// chart维度自定义标签，k=charName v=自定义标签
	ChartCustomLabelMap map[string]map[string]string `json:"resource_chart_map,omitempty"`
	// chart维度自定义标签选择器，k=charName v=自定义标签
	ChartCustomSelectorLabelMap map[string]map[string]string `json:"resource_chart_map,omitempty"`
	// resource维度自定义标签，k=kind_name v=自定义标签
	ResourceCustomLabelMap map[string]map[string]string `json:"resource_chart_map,omitempty"`
}

// SetStatus is a helper for setting the status on a release.
func (r *Release) SetStatus(status Status, msg string) {
	r.Info.Status = status
	r.Info.Description = msg
}

//基于资源类型和名称获取chart维度的自定义标签
func (r *Release) GetCustomLabelOnChart(kind string, name string) map[string]string {
	result := make(map[string]string)
	if r.ChartCustomLabelMap == nil {
		r.ChartCustomLabelMap = r.parseCustomLabel("hskp_devops_custom_label")
	}
	if r.ResourceChartMap == nil {
		glog.Info("ResourceChartMap空，无法获取自定义标签 ")
		return result
	}
	resourceChartKey := fmt.Sprintf("%s:%s", kind, name)
	if charName, ok := r.ResourceChartMap[resourceChartKey]; ok {
		if charCustomLabel, ok := r.ChartCustomLabelMap[charName]; ok {
			result = charCustomLabel
		}
	}
	return result
}

//基于资源类型和名称获取chart维度的自定义标签选择器
func (r *Release) GetCustomSelectorLabelOnChar(kind string, name string) map[string]string {
	result := make(map[string]string)
	if r.ChartCustomSelectorLabelMap == nil {
		r.ChartCustomSelectorLabelMap = r.parseCustomLabel("hskp_devops_custom_selector_label")
	}
	if r.ResourceChartMap == nil {
		glog.Info("ResourceChartMap空，无法获取自定义标签 ")
		return result
	}
	resourceChartKey := fmt.Sprintf("%s:%s", kind, name)
	if charName, ok := r.ResourceChartMap[resourceChartKey]; ok {
		if customSelectorLabel, ok := r.ChartCustomSelectorLabelMap[charName]; ok {
			result = customSelectorLabel
		}
	}
	return result
}

//基于资源类型和名称获取资源维度的自定义标签
func (r *Release) GetCustomLabelOnResource(kind string, name string) map[string]string {
	result := make(map[string]string)
	if r.ChartCustomSelectorLabelMap == nil {
		r.ChartCustomSelectorLabelMap = r.parseCustomLabel("hskp_devops_resource_custom_label")
	}

	resourceKey := fmt.Sprintf("%s_%s", kind, name)
	if targetResourceCustomLabel, ok := r.ChartCustomSelectorLabelMap[resourceKey]; ok {
		result = targetResourceCustomLabel
	}
	return result
}

func (r *Release) parseCustomLabel(labelType string) map[string]map[string]string {
	customLabelMap := make(map[string]map[string]string)
	v, err := copystructure.Copy(r.Config)

	if err != nil {
		return customLabelMap
	}

	valsCopy := v.(map[string]interface{})
	if valsCopy == nil {
		valsCopy = make(map[string]interface{})
	}

	if global, ok := valsCopy["global"]; ok {
		globalValue := global.(map[string]interface{})
		if customLabel, ok := globalValue[labelType]; ok {
			customLabelValues := customLabel.(map[string]interface{})
			for charName, charLabelsValue := range customLabelValues {
				if charLabelsValue == nil {
					continue
				}
				charLabels := make(map[string]string)
				for key, val := range charLabelsValue.(map[string]interface{}) {
					charLabels[key] = fmt.Sprintf("%v", val)
				}
				customLabelMap[charName] = charLabels
			}
		}
	}

	return customLabelMap

}
