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

package action

import (
	"fmt"
	"strings"

	"github.com/hinfinite/helm/pkg/release"

	"sigs.k8s.io/yaml"

	"github.com/hinfinite/helm/pkg/chart"
	"github.com/hinfinite/helm/pkg/chart/loader"
	"github.com/hinfinite/helm/pkg/chartutil"
)

// ShowOutputFormat is the format of the output of `helm show`
type ShowOutputFormat string

const (
	// ShowAll is the format which shows all the information of a chart
	ShowAll ShowOutputFormat = "all"
	// ShowChart is the format which only shows the chart's definition
	ShowChart ShowOutputFormat = "chart"
	// ShowValues is the format which only shows the chart's values
	ShowValues ShowOutputFormat = "values"
	// ShowReadme is the format which only shows the chart's README
	ShowReadme ShowOutputFormat = "readme"
	// ShowHook is the format which only show the chart's hooks
	ShowHook ShowOutputFormat = "hook"
)

var readmeFileNames = []string{"readme.md", "readme.txt", "readme"}

func (o ShowOutputFormat) String() string {
	return string(o)
}

// Show is the action for checking a given release's information.
//
// It provides the implementation of 'helm show' and its respective subcommands.
type Show struct {
	cfg       *Configuration
	Namespace string
	ChartPathOptions
	Devel        bool
	OutputFormat ShowOutputFormat
}

// NewShow creates a new Show object with the given configuration.
func NewShow(cfg *Configuration, output ShowOutputFormat, chartPathOptions ChartPathOptions) *Show {
	return &Show{
		cfg:              cfg,
		OutputFormat:     output,
		ChartPathOptions: chartPathOptions,
	}
}

// Run executes 'helm show' against the given release.
func (s *Show) Run(chartpath string, vals map[string]interface{}) (string, error) {
	var out strings.Builder
	chrt, err := loader.Load(chartpath)
	if err != nil {
		return "", err
	}
	cf, err := yaml.Marshal(chrt.Metadata)
	if err != nil {
		return "", err
	}

	if s.OutputFormat == ShowChart || s.OutputFormat == ShowAll {
		fmt.Fprintln(&out, "\n--- ChartInfo")

		fmt.Fprintf(&out, "%s\n", cf)
	}

	if (s.OutputFormat == ShowValues || s.OutputFormat == ShowAll) && chrt.Values != nil {
		if s.OutputFormat == ShowAll {
			fmt.Fprintln(&out, "\n--- Values")
		}
		for _, f := range chrt.Raw {
			if f.Name == chartutil.ValuesfileName {
				fmt.Fprintln(&out, string(f.Data))
			}
		}
	}

	if s.OutputFormat == ShowHook || s.OutputFormat == ShowAll {
		if s.OutputFormat == ShowAll {
			fmt.Fprintln(&out, "\n--- Hooks")
		}
		hooks, err := s.FindHooks(chrt, vals)
		if err != nil {
			return "", nil
		}
		if hooks == nil {
			return out.String(), nil
		}
		for _, hook := range hooks {
			fmt.Fprintf(&out, "# Source: %s\n%s\n", hook.Path, hook.Manifest)
		}
	}

	if s.OutputFormat == ShowReadme || s.OutputFormat == ShowAll {
		if s.OutputFormat == ShowAll {
			fmt.Fprintln(&out, "\n--- README")
		}
		readme := findReadme(chrt.Files)
		if readme == nil {
			fmt.Fprintln(&out, "\n(README is empty)")
			return out.String(), nil
		}
		fmt.Fprintf(&out, "%s\n", readme.Data)
	}
	return out.String(), nil
}

func findReadme(files []*chart.File) (file *chart.File) {
	for _, file := range files {
		for _, n := range readmeFileNames {
			if strings.EqualFold(file.Name, n) {
				return file
			}
		}
	}
	return nil
}

func (s *Show) FindHooks(chrt *chart.Chart, vals map[string]interface{}) ([]*release.Hook, error) {
	options := chartutil.ReleaseOptions{
		Name:      chrt.Name(),
		Namespace: s.Namespace,
		Revision:  1,
	}
	caps, err := s.cfg.getCapabilities()
	if err != nil {
		return nil, err
	}
	valuesToRender, err := chartutil.ToRenderValues(chrt, vals, options, caps)
	hooks, _, _, _, err := s.cfg.renderResources(chrt, valuesToRender, chrt.Name(), "", false, true, false, nil)
	if err != nil {
		return nil, err
	}
	return hooks, nil
}
