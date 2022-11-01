package action

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/hinfinite/helm/pkg/agent/model"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
)

func AddLabel(imagePullSecret []v1.LocalObjectReference,
	clusterCode string,
	info *resource.Info,
	commit,
	version,
	releaseName,
	chartName,
	agentVersion,
	namespace string,
	isUpgrade bool,
	clientSet *kubernetes.Clientset,
	customLabel map[string]string) error {
	t := info.Object.(*unstructured.Unstructured)
	kind := info.Mapping.GroupVersionKind.Kind

	// 添加公共标签逻辑
	var addBaseLabels = func(labels map[string]string) {
		if labels == nil {
			return
		}
		labels[model.ResourceCluster] = clusterCode
		labels[model.AgentVersionLabel] = agentVersion
		labels[model.ReleaseLabel] = releaseName
		labels[model.CommitLabel] = commit

	}
	var addAppLabels = func(labels map[string]string) {
		if labels == nil {
			return
		}
		labels[model.AppLabel] = chartName
		labels[model.AppVersionLabel] = version
	}
	//按需添加自定义标签
	var addCustomLabelIfNotPresent = func(labels map[string]string) map[string]string {
		if customLabel == nil {
			return labels
		}
		for key, val := range customLabel {
			if _, ok := labels[key]; !ok {
				labels[key] = val
			}
		}
		return labels
	}

	// 获取并添加工作负载标签
	var fetchAndAddWorkloadTemplateLabels = func(workloadKind, workloadName string) map[string]string {
		tplLabels := getTemplateLabels(t.Object)
		if tplLabels == nil {
			tplLabels = make(map[string]string)
		}

		// Base Labels
		addBaseLabels(tplLabels)

		// App Labels
		addAppLabels(tplLabels)

		// Workload Labels
		tplLabels[model.ParentWorkloadLabel] = workloadKind
		tplLabels[model.ParentWorkloadNameLabel] = workloadName

		return tplLabels
	}

	var addTemplateAppLabels = func(workloadKind, workloadName string) {
		tplLabels := fetchAndAddWorkloadTemplateLabels(workloadKind, workloadName)
		// 按需添加自定义标签到标签选择器标签选择器
		tplLabels = addCustomLabelIfNotPresent(tplLabels)
		if err := setTemplateLabels(t.Object, tplLabels); err != nil {
			glog.Warningf("Set Template Labels failed, %v", err)
		}
	}

	var addCronJobTemplateAppLabels = func(workloadKind, workloadName string) {
		tplLabels := fetchAndAddWorkloadTemplateLabels(workloadKind, workloadName)

		if err := setCronJobPodTemplateLabels(t.Object, tplLabels); err != nil {
			glog.Warningf("Set Template Labels failed, %v", err)
		}
	}
	var addSelectorAppLabels = func() {
		selectorLabels, _, err := unstructured.NestedStringMap(t.Object, "spec", "selector", "matchLabels")
		if err != nil {
			glog.Warningf("Get Selector Labels failed, %v", err)
		}
		if selectorLabels == nil {
			selectorLabels = make(map[string]string)
		}
		selectorLabels[model.ReleaseLabel] = releaseName
		// 按需添加自定义标签到标签选择器标签选择器
		addCustomLabelIfNotPresent(selectorLabels)
		if err := unstructured.SetNestedStringMap(t.Object, selectorLabels, "spec", "selector", "matchLabels"); err != nil {
			glog.Warningf("Set Selector label failed, %v", err)
		}
	}

	// add private image pull secrets
	var addImagePullSecrets = func() {
		secrets, _, err := nestedLocalObjectReferences(t.Object, "spec", "template", "spec", "imagePullSecrets")
		if err != nil {
			glog.Warningf("Get ImagePullSecrets failed, %v", err)
		}
		if secrets == nil {
			secrets = make([]v1.LocalObjectReference, 0)

		}
		secrets = append(secrets, imagePullSecret...)
		// SetNestedField method just support a few types
		s := make([]interface{}, 0)
		for _, secret := range secrets {
			m := make(map[string]interface{})
			m["name"] = secret.Name
			s = append(s, m)
		}
		if err := unstructured.SetNestedField(t.Object, s, "spec", "template", "spec", "imagePullSecrets"); err != nil {
			glog.Warningf("Set ImagePullSecrets failed, %v", err)
		}
	}

	switch kind {
	case "ReplicationController", "ReplicaSet", "Deployment":
		addImagePullSecrets()
		addTemplateAppLabels(kind, t.GetName())
		addSelectorAppLabels()
		if isUpgrade {
			if kind == "ReplicaSet" {
				rs, err := clientSet.AppsV1().ReplicaSets(namespace).Get(t.GetName(), metav1.GetOptions{})
				if errors.IsNotFound(err) {
					break
				}
				if err != nil {
					glog.Warningf("Failed to get ReplicaSet,error is %s.", err.Error())
					return err
				}
				err = setReplicas(t.Object, int64(*rs.Spec.Replicas))
				if err != nil {
					glog.Warningf("Failed to set replicas,error is %s", err.Error())
					return err
				}
			}
			if kind == "Deployment" {
				dp, err := clientSet.AppsV1().Deployments(namespace).Get(t.GetName(), metav1.GetOptions{})
				if errors.IsNotFound(err) {
					break
				}
				if err != nil {
					glog.Warningf("Failed to get ReplicaSet,error is %s.", err.Error())
					return err
				}
				err = setReplicas(t.Object, int64(*dp.Spec.Replicas))
				if err != nil {
					glog.Warningf("Failed to set replicas,error is %s", err.Error())
					return err
				}
			}
		}
	case "Job":
		addImagePullSecrets()
		addTemplateAppLabels(kind, t.GetName())
	case "CronJob":
		addImagePullSecrets()
		addCronJobTemplateAppLabels(kind, t.GetName())
	case "DaemonSet", "StatefulSet":
		addImagePullSecrets()
		addTemplateAppLabels(kind, t.GetName())
		if isUpgrade {
			if kind == "StatefulSet" {
				sts, err := clientSet.AppsV1().StatefulSets(namespace).Get(t.GetName(), metav1.GetOptions{})
				if errors.IsNotFound(err) {
					break
				}
				if err != nil {
					glog.Warningf("Failed to get ReplicaSet,error is %s.", err.Error())
					return err
				}
				err = setReplicas(t.Object, int64(*sts.Spec.Replicas))
				if err != nil {
					glog.Warningf("Failed to set replicas,error is %s", err.Error())
					return err
				}
			}
		}
	case "ConfigMap":
	case "Service":
	case "Ingress":
	case "Secret":
	case "Pod":
	default:
	}
	if t.GetNamespace() != "" && t.GetNamespace() != namespace {
		return fmt.Errorf(" Kind:%s Name:%s. The namespace of this resource is not consistent with helm release", kind, t.GetName())
	}

	// Add base and app labels
	l := t.GetLabels()

	if l == nil {
		l = make(map[string]string)
	}

	addBaseLabels(l)
	addAppLabels(l)
	addCustomLabelIfNotPresent(l)

	t.SetLabels(l)
	return nil
}

func setCronJobPodTemplateLabels(obj map[string]interface{}, templateLabels map[string]string) error {
	return unstructured.SetNestedStringMap(obj, templateLabels, "spec", "jobTemplate", "spec", "template", "metadata", "labels")
}

func setTemplateLabels(obj map[string]interface{}, templateLabels map[string]string) error {
	return unstructured.SetNestedStringMap(obj, templateLabels, "spec", "template", "metadata", "labels")
}

func setReplicas(obj map[string]interface{}, value int64) error {
	return unstructured.SetNestedField(obj, value, "spec", "replicas")
}

func getTemplateLabels(obj map[string]interface{}) map[string]string {
	tplLabels, _, err := unstructured.NestedStringMap(obj, "spec", "template", "metadata", "labels")
	if err != nil {
		glog.Warningf("Get Template Labels failed, %v", err)
	}
	if tplLabels == nil {
		tplLabels = make(map[string]string)
	}
	return tplLabels
}

func nestedLocalObjectReferences(obj map[string]interface{}, fields ...string) ([]v1.LocalObjectReference, bool, error) {
	val, found, err := unstructured.NestedFieldNoCopy(obj, fields...)
	if !found || err != nil {
		return nil, found, err
	}

	m, ok := val.([]v1.LocalObjectReference)
	if ok {
		return m, true, nil
		//return nil, false, fmt.Errorf("%v accessor error: %v is of the type %T, expected []v1.LocalObjectReference", strings.Join(fields, "."), val, val)
	}

	if m, ok := val.([]interface{}); ok {
		secrets := make([]v1.LocalObjectReference, 0)
		for _, v := range m {
			if vv, ok := v.(map[string]interface{}); ok {
				v2 := vv["name"]
				secret := v1.LocalObjectReference{}
				if secret.Name, ok = v2.(string); ok {
					secrets = append(secrets, secret)
				}
			}
		}
		return secrets, true, nil
	}
	return m, true, nil
}
