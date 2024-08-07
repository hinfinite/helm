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
	"bytes"
	"fmt"
	"github.com/hinfinite/helm/pkg/agent/action"
	v1 "k8s.io/api/core/v1"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/hinfinite/helm/pkg/release"
	helmtime "github.com/hinfinite/helm/pkg/time"
)

// Rollback is the action for rolling back to a given release.
//
// It provides the implementation of 'helm rollback'.
type Rollback struct {
	cfg *Configuration

	Version       int
	Timeout       time.Duration
	Wait          bool
	DisableHooks  bool
	DryRun        bool
	Recreate      bool // will (if true) recreate pods after a rollback.
	Force         bool // will (if true) force resource upgrade through uninstall/recreate if needed
	CleanupOnFail bool

	ReleaseName     string
	Commit          string
	ImagePullSecret []v1.LocalObjectReference
	ClusterCode     string
	AgentVersion    string
}

// NewRollback creates a new Rollback object with the given configuration.
func NewRollback(cfg *Configuration) *Rollback {
	return &Rollback{
		cfg: cfg,
	}
}

// Run executes 'helm rollback' against the given release.
func (r *Rollback) Run(name string) error {
	if err := r.cfg.KubeClient.IsReachable(); err != nil {
		return err
	}

	r.cfg.Log("preparing rollback of %s", name)
	currentRelease, targetRelease, err := r.prepareRollback(name)
	if err != nil {
		return err
	}

	if !r.DryRun {
		r.cfg.Log("creating rolled back release for %s", name)
		if err := r.cfg.Releases.Create(targetRelease); err != nil {
			return err
		}
	}

	r.cfg.Log("performing rollback of %s", name)
	if _, err := r.performRollback(currentRelease, targetRelease); err != nil {
		return err
	}

	if !r.DryRun {
		r.cfg.Log("updating status for rolled back release for %s", name)
		if err := r.cfg.Releases.Update(targetRelease); err != nil {
			return err
		}
	}
	return nil
}

// prepareRollback finds the previous release and prepares a new release object with
// the previous release's configuration
func (r *Rollback) prepareRollback(name string) (*release.Release, *release.Release, error) {
	if err := validateReleaseName(name); err != nil {
		return nil, nil, errors.Errorf("prepareRollback: Release name is invalid: %s", name)
	}

	if r.Version < 0 {
		return nil, nil, errInvalidRevision
	}

	currentRelease, err := r.cfg.Releases.Last(name)
	if err != nil {
		return nil, nil, err
	}

	previousVersion := r.Version
	if r.Version == 0 {
		previousVersion = currentRelease.Version - 1
	}

	r.cfg.Log("rolling back %s (current: v%d, target: v%d)", name, currentRelease.Version, previousVersion)

	previousRelease, err := r.cfg.Releases.Get(name, previousVersion)
	if err != nil {
		return nil, nil, err
	}

	// Store a new release object with previous release's configuration
	targetRelease := &release.Release{
		Name:      name,
		Namespace: currentRelease.Namespace,
		Chart:     previousRelease.Chart,
		Config:    previousRelease.Config,
		Info: &release.Info{
			FirstDeployed: currentRelease.Info.FirstDeployed,
			LastDeployed:  helmtime.Now(),
			Status:        release.StatusPendingRollback,
			Notes:         previousRelease.Info.Notes,
			// Because we lose the reference to previous version elsewhere, we set the
			// message here, and only override it later if we experience failure.
			Description: fmt.Sprintf("Rollback to %d", previousVersion),
		},
		Version:  currentRelease.Version + 1,
		Manifest: previousRelease.Manifest,
		Hooks:    previousRelease.Hooks,

		ResourceChartMap:            previousRelease.ResourceChartMap,
		ChartCommonLabelMap:         previousRelease.ChartCommonLabelMap,
		ChartCustomLabelMap:         previousRelease.ChartCustomLabelMap,
		ChartCustomSelectorLabelMap: previousRelease.ChartCustomSelectorLabelMap,
		ResourceCustomLabelMap:      previousRelease.ResourceCustomLabelMap,
	}

	return currentRelease, targetRelease, nil
}

func (r *Rollback) performRollback(currentRelease, targetRelease *release.Release) (*release.Release, error) {
	if r.DryRun {
		r.cfg.Log("dry run for %s", targetRelease.Name)
		return targetRelease, nil
	}

	current, err := r.cfg.KubeClient.Build(bytes.NewBufferString(currentRelease.Manifest), false)
	if err != nil {
		return targetRelease, errors.Wrap(err, "unable to build kubernetes objects from current release manifest")
	}
	target, err := r.cfg.KubeClient.Build(bytes.NewBufferString(targetRelease.Manifest), false)
	if err != nil {
		return targetRelease, errors.Wrap(err, "unable to build kubernetes objects from new release manifest")
	}

	// 如果是agent升级，则跳过添加标签这一步，因为agent原本是直接在集群中安装的没有对应标签，如果在这里加标签k8s会报错
	if targetRelease.Chart.Name() != "hskp-devops-cluster-agent" {
		// 在这里对要新chart包中的对象添加标签
		// current于target相同的部分不会更新
		for _, t := range target {
			customLabelOnChart := targetRelease.GetCustomLabelOnChart(t.Mapping.GroupVersionKind.Kind, t.Name)
			customSelectorLabelOnChart := targetRelease.GetCustomSelectorLabelOnChar(t.Mapping.GroupVersionKind.Kind, t.Name)
			customLabelOnResource := targetRelease.GetCustomLabelOnResource(t.Mapping.GroupVersionKind.Kind, t.Name)
			commonLabelOnChart := targetRelease.GetCommonLabelOnChart(t.Mapping.GroupVersionKind.Kind, t.Name)
			err = action.AddLabel(r.ImagePullSecret,
				r.ClusterCode,
				t,
				r.Commit,
				targetRelease.Chart.Metadata.Version,
				r.ReleaseName,
				targetRelease.Chart.Metadata.Name,
				r.AgentVersion,
				currentRelease.Namespace,
				true,
				r.cfg.ClientSet,
				customLabelOnChart,
				customSelectorLabelOnChart,
				customLabelOnResource,
				commonLabelOnChart,
				false)
			if err != nil {
				return nil, err
			}
		}
		for _, c := range current {
			customLabelOnChart := currentRelease.GetCustomLabelOnChart(c.Mapping.GroupVersionKind.Kind, c.Name)
			customSelectorLabelOnChart := currentRelease.GetCustomSelectorLabelOnChar(c.Mapping.GroupVersionKind.Kind, c.Name)
			customLabelOnResource := currentRelease.GetCustomLabelOnResource(c.Mapping.GroupVersionKind.Kind, c.Name)
			commonLabelOnChart := currentRelease.GetCommonLabelOnChart(c.Mapping.GroupVersionKind.Kind, c.Name)
			err = action.AddLabel(r.ImagePullSecret,
				r.ClusterCode,
				c,
				"",
				currentRelease.Chart.Metadata.Version,
				r.ReleaseName,
				currentRelease.Chart.Metadata.Name,
				r.AgentVersion,
				currentRelease.Namespace,
				true,
				r.cfg.ClientSet,
				customLabelOnChart,
				customSelectorLabelOnChart,
				customLabelOnResource,
				commonLabelOnChart,
				true)
			if err != nil {
				return nil, err
			}
		}
	}

	// pre-rollback hooks
	if !r.DisableHooks {
		if err := r.cfg.execHook(targetRelease, release.HookPreRollback, r.Timeout, nil, "", "", "", "", "", "", ""); err != nil {
			return targetRelease, err
		}
	} else {
		r.cfg.Log("rollback hooks disabled for %s", targetRelease.Name)
	}

	results, err := r.cfg.KubeClient.Update(current, target, r.Force)

	if err != nil {
		msg := fmt.Sprintf("Rollback %q failed: %s", targetRelease.Name, err)
		r.cfg.Log("warning: %s", msg)
		currentRelease.Info.Status = release.StatusSuperseded
		targetRelease.Info.Status = release.StatusFailed
		targetRelease.Info.Description = msg
		r.cfg.recordRelease(currentRelease)
		r.cfg.recordRelease(targetRelease)
		if r.CleanupOnFail {
			r.cfg.Log("Cleanup on fail set, cleaning up %d resources", len(results.Created))
			_, errs := r.cfg.KubeClient.Delete(results.Created)
			if errs != nil {
				var errorList []string
				for _, e := range errs {
					errorList = append(errorList, e.Error())
				}
				return targetRelease, errors.Wrapf(fmt.Errorf("unable to cleanup resources: %s", strings.Join(errorList, ", ")), "an error occurred while cleaning up resources. original rollback error: %s", err)
			}
			r.cfg.Log("Resource cleanup complete")
		}
		return targetRelease, err
	}

	if r.Recreate {
		// NOTE: Because this is not critical for a release to succeed, we just
		// log if an error occurs and continue onward. If we ever introduce log
		// levels, we should make these error level logs so users are notified
		// that they'll need to go do the cleanup on their own
		if err := recreate(r.cfg, results.Updated); err != nil {
			r.cfg.Log(err.Error())
		}
	}

	if r.Wait {
		if err := r.cfg.KubeClient.Wait(target, r.Timeout); err != nil {
			targetRelease.SetStatus(release.StatusFailed, fmt.Sprintf("Release %q failed: %s", targetRelease.Name, err.Error()))
			r.cfg.recordRelease(currentRelease)
			r.cfg.recordRelease(targetRelease)
			return targetRelease, errors.Wrapf(err, "release %s failed", targetRelease.Name)
		}
	}

	// post-rollback hooks
	if !r.DisableHooks {
		if err := r.cfg.execHook(targetRelease, release.HookPostRollback, r.Timeout, nil, "", "", "", "", "", "", ""); err != nil {
			return targetRelease, err
		}
	}

	deployed, err := r.cfg.Releases.DeployedAll(currentRelease.Name)
	if err != nil {
		return nil, err
	}
	// Supersede all previous deployments, see issue #2941.
	for _, rel := range deployed {
		r.cfg.Log("superseding previous deployment %d", rel.Version)
		rel.Info.Status = release.StatusSuperseded
		r.cfg.recordRelease(rel)
	}

	targetRelease.Info.Status = release.StatusDeployed

	return targetRelease, nil
}
