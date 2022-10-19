/*
Copyright The Hand Authors.

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

package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/golang/glog"

	"github.com/hinfinite/helm/pkg/action"
	"github.com/hinfinite/helm/pkg/chart"
	"github.com/hinfinite/helm/pkg/paginator"
	"github.com/hinfinite/helm/pkg/paginator/adapter"
	"github.com/hinfinite/helm/pkg/repo"
	"sigs.k8s.io/yaml"
)

// RepoConfig repo config to add
type RepoConfig struct {
	Name     string
	Url      string
	Username string
	Password string
	NoUpdate bool
}

// AddRepo add repo
func AddRepo(namespace string, repoConfig *RepoConfig) error {
	_, settings := getCfg(namespace)

	o := &RepoAddOptions{
		repoFile:              settings.RepositoryConfig,
		repoCache:             settings.RepositoryCache,
		name:                  repoConfig.Name,
		url:                   repoConfig.Url,
		username:              repoConfig.Username,
		password:              repoConfig.Password,
		noUpdate:              repoConfig.NoUpdate,
		insecureSkipTLSverify: true,
		settings:              settings,
	}
	return o.run(os.Stdout)
}

func RepoUpdate() error {
	o := &repoUpdateOptions{update: updateCharts}
	o.repoFile = settings.RepositoryConfig
	return o.run(os.Stdout)
}

type ChartSummary struct {
	Name string `json:"name,omitempty"`
	// A SemVer 2 conformant version string of the chart
	Version string `json:"version,omitempty"`
	// The version of the application enclosed inside of this chart.
	AppVersion string `json:"appVersion,omitempty"`
	// A one-sentence description of the chart
	Description string `json:"description,omitempty"`
	// The URL to an icon file.
	Icon string `json:"icon,omitempty"`
}

type ListOptions struct {
	NameKeyword string `json:"nameKeyword,omitempty"`
	Page        int    `json:"page,omitempty"`
	Size        int    `json:"size,omitempty"`
}

func CheckExist(namespace string, repoConfig *RepoConfig) bool {
	_, settings := getCfg(namespace)

	repoFile := settings.RepositoryConfig

	err := os.MkdirAll(filepath.Dir(repoFile), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return false
	}
	b, err := ioutil.ReadFile(repoFile)
	if err != nil && !os.IsNotExist(err) {
		return false
	}

	var f repo.File
	if err := yaml.Unmarshal(b, &f); err != nil {
		return false
	}
	return f.Has(repoConfig.Name)

}
func ListChart(namespace string, repoConfig *RepoConfig, listOpts *ListOptions) (*paginator.Page, error) {
	// Add repo
	if !CheckExist(namespace, repoConfig) {
		glog.Info("<<<<<<<<执行ListChart发现仓库不存在，执行AddRepo")
		AddRepo(namespace, repoConfig)
	}

	// Set search options
	_, settings := getCfg(namespace)
	o := &SearchRepoOptions{
		repoFile:     settings.RepositoryConfig,
		repoCacheDir: settings.RepositoryCache,
		regexp:       true,
	}

	// List in current repo
	args := make([]string, 0)
	if listOpts.NameKeyword != "" {
		args = append(args, fmt.Sprintf("^%s/.*%s.*$", repoConfig.Name, regexp.QuoteMeta(listOpts.NameKeyword)))
	} else {
		args = append(args, fmt.Sprintf("^%s/.*$", repoConfig.Name))
	}

	// run search
	data, err := o.run(args)
	if err != nil {
		return nil, err
	}

	chartSummarySlice := make([]*ChartSummary, 0)
	for _, item := range data {
		chartSummarySlice = append(chartSummarySlice, &ChartSummary{
			Name:        item.Chart.Name,
			Version:     item.Chart.Version,
			AppVersion:  item.Chart.AppVersion,
			Description: item.Chart.Description,
			Icon:        item.Chart.Icon,
		})
	}

	// Paginate
	p := paginator.New(adapter.NewSliceAdapter(chartSummarySlice), listOpts.Size)
	chartSummaryInCurrentPage := make([]*ChartSummary, 0)
	// Note the data arguments must be the pointer to slice
	pageResult, err := p.PageResults(listOpts.Page, &chartSummaryInCurrentPage)
	if err != nil {
		return nil, err
	}

	return pageResult, nil
}

type ShowOptions struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type ChartDetail struct {
	ChartSummary
	Values   string   `json:"values,omitempty"`
	Readme   string   `json:"readme,omitempty"`
	Versions []string `json:"versions,omitempty"`
}

// ShowDetail show chart details with multiple versions
func ShowDetail(namespace string, repoConfig *RepoConfig, showOpts *ShowOptions) (*ChartDetail, error) {
	// Add repo
	if !repoConfig.NoUpdate {
		AddRepo(namespace, repoConfig)
	}

	// Set show options
	cfg, settings := getCfg(namespace)
	client := action.NewShow(cfg, action.ShowAll, action.ChartPathOptions{Version: showOpts.Version})
	client.Namespace = settings.Namespace()

	qualifiedName := fmt.Sprintf("%s/%s", repoConfig.Name, showOpts.Name)

	// Values
	client.OutputFormat = action.ShowValues
	values, err := runShow([]string{qualifiedName}, client, nil, settings)
	if err != nil {
		return nil, err
	}

	// Readme
	client.OutputFormat = action.ShowReadme
	readme, err := runShow([]string{qualifiedName}, client, nil, settings)
	if err != nil {
		return nil, err
	}

	// Chart
	client.OutputFormat = action.ShowChart
	chartStr, err := runShow([]string{qualifiedName}, client, nil, settings)
	if err != nil {
		return nil, err
	}

	chart := &chart.Metadata{}
	chartStr = strings.Replace(chartStr, "\n--- ChartInfo\n", "", 1)
	err = yaml.Unmarshal([]byte(chartStr), chart)
	if err != nil {
		return nil, err
	}

	// Versions
	versions := make([]string, 0)
	o := &SearchRepoOptions{
		repoFile:     settings.RepositoryConfig,
		repoCacheDir: settings.RepositoryCache,
		regexp:       false,
		versions:     true,
	}

	// Run Search
	data, err := o.run([]string{qualifiedName})
	if err != nil {
		return nil, err
	}

	for _, item := range data {
		// Fix: the repo name part of item.Name in result of search will be automatically convert to lower case,
		// then, the == with qualifiedName will be mismatched. use strings.EqualFold to ignore case
		if strings.EqualFold(item.Name, qualifiedName) {
			versions = append(versions, item.Chart.Version)
		}
	}

	return &ChartDetail{
		ChartSummary: ChartSummary{
			Name:        chart.Name,
			Version:     chart.Version,
			AppVersion:  chart.AppVersion,
			Description: chart.Description,
			Icon:        chart.Icon,
		},
		Values:   values,
		Readme:   readme,
		Versions: versions,
	}, nil
}
