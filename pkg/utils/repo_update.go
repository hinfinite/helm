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

package utils

import (
	"fmt"
	"github.com/hinfinite/helm/pkg/cli"
	"io"
	"sync"

	"github.com/hinfinite/helm/pkg/getter"
	"github.com/hinfinite/helm/pkg/repo"
	"github.com/pkg/errors"
)

var (
	settings = cli.New()
)

var errNoRepositories = errors.New("no repositories found. You must add one before updating")

type repoUpdateOptions struct {
	update   func([]*repo.ChartRepository, io.Writer)
	repoFile string
}

func (o *repoUpdateOptions) run(out io.Writer) error {
	f, err := repo.LoadFile(o.repoFile)
	if isNotExist(err) || len(f.Repositories) == 0 {
		return errNoRepositories
	}
	var repos []*repo.ChartRepository
	for _, cfg := range f.Repositories {
		r, err := repo.NewChartRepository(cfg, getter.All(settings))
		if err != nil {
			return err
		}
		repos = append(repos, r)
	}

	o.update(repos, out)
	return nil
}

func updateCharts(repos []*repo.ChartRepository, out io.Writer) {
	fmt.Fprintln(out, "Hang tight while we grab the latest from your chart repositories...")
	var wg sync.WaitGroup
	for _, re := range repos {
		wg.Add(1)
		go func(re *repo.ChartRepository) {
			defer wg.Done()
			if _, _, err := re.DownloadIndexFile(); err != nil {
				fmt.Fprintf(out, "...Unable to get an update from the %q chart repository (%s):\n\t%s\n", re.Config.Name, re.Config.URL, err)
			} else {
				fmt.Fprintf(out, "...Successfully got an update from the %q chart repository\n", re.Config.Name)
			}
		}(re)
	}
	wg.Wait()
	fmt.Fprintln(out, "Update Complete. ⎈ Happy Helming!⎈ ")
}
