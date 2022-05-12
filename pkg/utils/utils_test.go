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
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/hinfinite/helm/pkg/paginator"
)

const (
	username = "hsopDeployer"
	password = ""
)

func TestListChart(t *testing.T) {
	type args struct {
		namespace  string
		repoConfig *RepoConfig
		listOpts   *ListOptions
	}
	tests := []struct {
		name    string
		args    args
		want    *paginator.Page
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "round 1",
			args: args{
				namespace: "hskp-demo",
				repoConfig: &RepoConfig{
					Name:     "hzero",
					Url:      "http://harbor.open.hand-china.com/chartrepo/hzero",
					Username: username,
					Password: password,
				},
				listOpts: &ListOptions{
					NameKeyword: "${artifactIdName}",
					Page:        0,
					Size:        2,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ListChart(tt.args.namespace, tt.args.repoConfig, tt.args.listOpts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "list chart with error: %s", err.Error())
			}

			data, err := json.MarshalIndent(got, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "marshal list chart page data with error: %s", err.Error())
			}

			fmt.Println(string(data))
		})
	}
}

func TestShowDetail(t *testing.T) {
	type args struct {
		namespace  string
		repoConfig *RepoConfig
		showOpts   *ShowOptions
	}
	tests := []struct {
		name    string
		args    args
		want    *ChartDetail
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "round 1",
			args: args{
				namespace: "hzero",
				repoConfig: &RepoConfig{
					Name:     "hzero",
					Url:      "http://harbor.open.hand-china.com/chartrepo/hzero",
					Username: username,
					Password: password,
				},
				showOpts: &ShowOptions{
					Name:    "${artifactIdName}",
					Version: "1.3.1",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ShowDetail(tt.args.namespace, tt.args.repoConfig, tt.args.showOpts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "show chart with error: %s", err.Error())
			}

			data, err := json.MarshalIndent(got, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "marshal show chart page data with error: %s", err.Error())
			}

			fmt.Println(string(data))
		})
	}
}
