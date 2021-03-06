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

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"sigs.k8s.io/yaml"

	"github.com/hinfinite/helm/internal/test/ensure"
	"github.com/hinfinite/helm/pkg/helmpath"
	"github.com/hinfinite/helm/pkg/helmpath/xdg"
	"github.com/hinfinite/helm/pkg/repo"
	"github.com/hinfinite/helm/pkg/repo/repotest"
)

func TestRepoAddCmd(t *testing.T) {
	srv, err := repotest.NewTempServer("testdata/testserver/*.*")
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	repoFile := filepath.Join(ensure.TempDir(t), "repositories.yaml")

	tests := []cmdTestCase{{
		name:   "add a repository",
		cmd:    fmt.Sprintf("repo add test-name %s --repository-config %s", srv.URL(), repoFile),
		golden: "output/repo-add.txt",
	}}

	runTestCmd(t, tests)
}

func TestRepoAdd(t *testing.T) {
	ts, err := repotest.NewTempServer("testdata/testserver/*.*")
	if err != nil {
		t.Fatal(err)
	}
	defer ts.Stop()

	rootDir := ensure.TempDir(t)
	repoFile := filepath.Join(rootDir, "repositories.yaml")

	const testRepoName = "test-name"

	o := &repoAddOptions{
		name:     testRepoName,
		url:      ts.URL(),
		noUpdate: true,
		repoFile: repoFile,
	}
	os.Setenv(xdg.CacheHomeEnvVar, rootDir)

	if err := o.run(ioutil.Discard); err != nil {
		t.Error(err)
	}

	f, err := repo.LoadFile(repoFile)
	if err != nil {
		t.Fatal(err)
	}

	if !f.Has(testRepoName) {
		t.Errorf("%s was not successfully inserted into %s", testRepoName, repoFile)
	}

	idx := filepath.Join(helmpath.CachePath("repository"), helmpath.CacheIndexFile(testRepoName))
	if _, err := os.Stat(idx); os.IsNotExist(err) {
		t.Errorf("Error cache index file was not created for repository %s", testRepoName)
	}
	idx = filepath.Join(helmpath.CachePath("repository"), helmpath.CacheChartsFile(testRepoName))
	if _, err := os.Stat(idx); os.IsNotExist(err) {
		t.Errorf("Error cache charts file was not created for repository %s", testRepoName)
	}

	o.noUpdate = false

	if err := o.run(ioutil.Discard); err != nil {
		t.Errorf("Repository was not updated: %s", err)
	}

	if err := o.run(ioutil.Discard); err != nil {
		t.Errorf("Duplicate repository name was added")
	}
}

func TestRepoAddConcurrentGoRoutines(t *testing.T) {
	const testName = "test-name"
	repoFile := filepath.Join(ensure.TempDir(t), "repositories.yaml")
	repoAddConcurrent(t, testName, repoFile)
}

func TestRepoAddConcurrentDirNotExist(t *testing.T) {
	const testName = "test-name-2"
	repoFile := filepath.Join(ensure.TempDir(t), "foo", "repositories.yaml")
	repoAddConcurrent(t, testName, repoFile)
}

func repoAddConcurrent(t *testing.T, testName, repoFile string) {
	ts, err := repotest.NewTempServer("testdata/testserver/*.*")
	if err != nil {
		t.Fatal(err)
	}
	defer ts.Stop()

	var wg sync.WaitGroup
	wg.Add(3)
	for i := 0; i < 3; i++ {
		go func(name string) {
			defer wg.Done()
			o := &repoAddOptions{
				name:     name,
				url:      ts.URL(),
				noUpdate: true,
				repoFile: repoFile,
			}
			if err := o.run(ioutil.Discard); err != nil {
				t.Error(err)
			}
		}(fmt.Sprintf("%s-%d", testName, i))
	}
	wg.Wait()

	b, err := ioutil.ReadFile(repoFile)
	if err != nil {
		t.Error(err)
	}

	var f repo.File
	if err := yaml.Unmarshal(b, &f); err != nil {
		t.Error(err)
	}

	var name string
	for i := 0; i < 3; i++ {
		name = fmt.Sprintf("%s-%d", testName, i)
		if !f.Has(name) {
			t.Errorf("%s was not successfully inserted into %s: %s", name, repoFile, f.Repositories[0])
		}
	}
}
