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

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hinfinite/helm/cmd/helm/search"
	"github.com/hinfinite/helm/pkg/action"
	"github.com/hinfinite/helm/pkg/cli"
	"github.com/hinfinite/helm/pkg/paginator"
	"github.com/hinfinite/helm/pkg/paginator/adapter"
)

// getCfg get helm config
func getCfg(namespace string) (*action.Configuration, *cli.EnvSettings) {
	settings := cli.New()
	settings.SetNamespace(namespace)
	actionConfig := &action.Configuration{}

	helmDriver := os.Getenv("HELM_DRIVER")
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), helmDriver, debug); err != nil {
		log.Fatal(err)
	}
	return actionConfig, settings
}

// RepoConfig repo config to add
type RepoConfig struct {
	name     string
	url      string
	username string
	password string
}

// AddRepo add repo
func AddRepo(namespace string, repoConfig *RepoConfig) error {
	_, settings := getCfg(namespace)

	o := &repoAddOptions{
		repoFile:              settings.RepositoryConfig,
		repoCache:             settings.RepositoryCache,
		name:                  repoConfig.name,
		url:                   repoConfig.url,
		username:              repoConfig.username,
		password:              repoConfig.password,
		noUpdate:              false,
		insecureSkipTLSverify: false,
	}
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

	// Others
	Created time.Time `json:"created,omitempty"`
	Removed bool      `json:"removed,omitempty"`
	Digest  string    `json:"digest,omitempty"`
	Score   int       `json:"score,omitempty"`
}

type ListOptions struct {
	NameKeyword string `json:"nameKeyword,omitempty"`
	Page        int    `json:"page,omitempty"`
	Size        int    `json:"size,omitempty"`
}

func ListChart(namespace string, repoConfig *RepoConfig, listOpts *ListOptions) (*paginator.Page, error) {
	// Add repo
	AddRepo(namespace, repoConfig)

	// Set search options
	_, settings := getCfg(namespace)
	o := &searchRepoOptions{
		repoFile:     settings.RepositoryConfig,
		repoCacheDir: settings.RepositoryCache,
		regexp:       true,
	}
	o.setupSearchedVersion()

	// Build index
	index, err := o.buildIndex()
	if err != nil {
		return nil, err
	}

	// List in current repo, note the keyword construct
	var res []*search.Result
	if listOpts.NameKeyword == "" {
		res = index.All()
	} else {
		q := fmt.Sprintf("%s/*%s*", repoConfig.name, listOpts.NameKeyword)
		res, err = index.Search(q, searchMaxScore, o.regexp)
		if err != nil {
			return nil, err
		}
	}

	search.SortScore(res)
	data, err := o.applyConstraint(res)
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
			Created:     item.Chart.Created,
			Removed:     item.Chart.Removed,
			Digest:      item.Chart.Digest,
			Score:       item.Score,
		})
	}

	// Paginate
	p := paginator.New(adapter.NewSliceAdapter(chartSummarySlice), listOpts.Size)
	p.SetPage(listOpts.Page)

	chartSummaryInCurrentPage := make([]*ChartSummary, 0)
	// Note: here must be the pointer to slice
	err = p.Results(&chartSummaryInCurrentPage)
	if err != nil {
		return nil, err
	}

	pageResult, err := p.ToPageResults(chartSummaryInCurrentPage)
	if err != nil {
		return nil, err
	}

	return pageResult, nil
}
