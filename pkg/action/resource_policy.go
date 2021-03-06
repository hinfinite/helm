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
	"strings"

	"github.com/hinfinite/helm/pkg/releaseutil"
)

// resourcePolicyAnno is the annotation name for a resource policy
const resourcePolicyAnno = "helm.sh/resource-policy"

// keepPolicy is the resource policy type for keep
//
// This resource policy type allows resources to skip being deleted
//   during an uninstallRelease action.
const keepPolicy = "keep"

func filterManifestsToKeep(manifests []releaseutil.Manifest) (keep, remaining []releaseutil.Manifest) {
	for _, m := range manifests {
		if m.Head.Metadata == nil || m.Head.Metadata.Annotations == nil || len(m.Head.Metadata.Annotations) == 0 {
			remaining = append(remaining, m)
			continue
		}

		resourcePolicyType, ok := m.Head.Metadata.Annotations[resourcePolicyAnno]
		if !ok {
			remaining = append(remaining, m)
			continue
		}

		resourcePolicyType = strings.ToLower(strings.TrimSpace(resourcePolicyType))
		if resourcePolicyType == keepPolicy {
			keep = append(keep, m)
		}

	}
	return keep, remaining
}
