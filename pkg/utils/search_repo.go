package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/hinfinite/helm/cmd/helm/search"
	"github.com/hinfinite/helm/pkg/cli/output"
	"github.com/hinfinite/helm/pkg/helmpath"
	"github.com/hinfinite/helm/pkg/repo"
	"github.com/pkg/errors"
)

// searchMaxScore suggests that any score higher than this is not considered a match.
const searchMaxScore = 25

type SearchRepoOptions struct {
	versions     bool
	regexp       bool
	devel        bool
	version      string
	maxColWidth  uint
	repoFile     string
	repoCacheDir string
	outputFormat output.Format
}

func (o *SearchRepoOptions) run(args []string) ([]*search.Result, error) {
	o.setupSearchedVersion()

	// Build index
	index, err := o.buildIndex()
	if err != nil {
		return nil, err
	}

	// List in current repo, note the keyword construct
	var res []*search.Result
	if len(args) == 0 {
		res = index.All()
	} else {
		q := strings.Join(args, " ")
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

	return data, nil
}

func (o *SearchRepoOptions) setupSearchedVersion() {
	debug("Original chart version: %q", o.version)

	if o.version != "" {
		return
	}

	if o.devel { // search for releases and prereleases (alpha, beta, and release candidate releases).
		debug("setting version to >0.0.0-0")
		o.version = ">0.0.0-0"
	} else { // search only for stable releases, prerelease versions will be skip
		debug("setting version to >0.0.0")
		o.version = ">0.0.0"
	}
}

func (o *SearchRepoOptions) applyConstraint(res []*search.Result) ([]*search.Result, error) {
	if len(o.version) == 0 {
		return res, nil
	}

	constraint, err := semver.NewConstraint(o.version)
	if err != nil {
		return res, errors.Wrap(err, "an invalid version/constraint format")
	}

	data := res[:0]
	foundNames := map[string]bool{}
	for _, r := range res {
		if _, found := foundNames[r.Name]; found {
			continue
		}
		v, err := semver.NewVersion(r.Chart.Version)
		if err != nil || constraint.Check(v) {
			data = append(data, r)
			if !o.versions {
				foundNames[r.Name] = true // If user hasn't requested all versions, only show the latest that matches
			}
		}
	}

	return data, nil
}

func (o *SearchRepoOptions) buildIndex() (*search.Index, error) {
	// Load the repositories.yaml
	rf, err := repo.LoadFile(o.repoFile)
	if isNotExist(err) || len(rf.Repositories) == 0 {
		return nil, errors.New("no repositories configured")
	}

	i := search.NewIndex()
	for _, re := range rf.Repositories {
		n := re.Name
		f := filepath.Join(o.repoCacheDir, helmpath.CacheIndexFile(n))
		ind, err := repo.LoadIndexFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: Repo %q is corrupt or missing. Try 'helm repo update'.", n)
			continue
		}

		i.AddRepo(n, ind, o.versions || len(o.version) > 0)
	}
	return i, nil
}

func isNotExist(err error) bool {
	return os.IsNotExist(errors.Cause(err))
}
