package action

import (
	"encoding/json"
	"fmt"
	"strings"

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
	customLabelOnChart map[string]string,
	customSelectorLabelOnChart map[string]string,
	customLabelOnResource map[string]string) error {
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
	var addCustomLabelIfNotPresent = func(source map[string]string, target map[string]string) map[string]string {
		if source == nil {
			return target
		}
		for key, val := range source {
			if _, ok := target[key]; !ok {
				target[key] = val
			}
		}
		return target
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
		// 按需添加自定义标签到标签选择器标签选择器,优先资源维度，其次chart维度
		tplLabels = addCustomLabelIfNotPresent(customLabelOnResource, tplLabels)
		tplLabels = addCustomLabelIfNotPresent(customLabelOnChart, tplLabels)
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
		// 按需添加自定义选择器标签到selector
		addCustomLabelIfNotPresent(customSelectorLabelOnChart, selectorLabels)
		if err := unstructured.SetNestedStringMap(t.Object, selectorLabels, "spec", "selector", "matchLabels"); err != nil {
			glog.Warningf("Set Selector label failed, %v", err)
		}
		// 按需添加自定义选择器标签到template
		tplLabels := getTemplateLabels(t.Object)
		if tplLabels == nil {
			tplLabels = make(map[string]string)
		}
		addCustomLabelIfNotPresent(customSelectorLabelOnChart, tplLabels)
		if err := setTemplateLabels(t.Object, tplLabels); err != nil {
			glog.Warningf("Set Template Labels failed, %v", err)
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
	//添加Spring-boot应用监控指标指标
	var handlerSpringBootMonitorMetrics = func() {
		metricsEnabled := customLabelOnChart["hskp.io/spring_boot_metrics_enabled"]
		if metricsEnabled != "true" {
			return
		}

		unstructuredPodTemplateSpec, _, err := unstructured.NestedFieldNoCopy(t.Object, "spec", "template")
		if err != nil {
			glog.Warningf("Get spec.template failed, %v", err)
		}
		if unstructuredPodTemplateSpec == nil {
			glog.Warningf(">>>>>>>>>>>>>>>>>podTemplateSpec is null")
			return
		}
		podTemplateSpec := v1.PodTemplateSpec{}

		ToObj(unstructuredPodTemplateSpec, &podTemplateSpec)
		//注解
		if podTemplateSpec.Annotations == nil {
			podTemplateSpec.Annotations = map[string]string{}
		}
		podTemplateSpec.Annotations["prometheus.io/port"] = "9464"
		podTemplateSpec.Annotations["prometheus.io/scrape"] = "true"

		//标签
		if podTemplateSpec.Labels == nil {
			podTemplateSpec.Labels = map[string]string{}
		}
		podTemplateSpec.Labels["prometheus.io/port"] = "9464"
		podTemplateSpec.Labels["hskp.io/component"] = "springboot"

		podSpec := podTemplateSpec.Spec
		if podSpec.Containers == nil {
			glog.Warningf(">>>>>>>>>>>>>>>>>podSpec.Containers is null")
			return
		}
		//挂载点
		volumeMount := v1.VolumeMount{
			Name:      "agent",
			MountPath: "/agent",
		}
		//设置agentInitContainer
		if podSpec.InitContainers == nil {
			podSpec.InitContainers = make([]v1.Container, 1, 1)
		}
		agentInitContainer := v1.Container{
			Name:         "java-agent",
			Image:        "harbor.open.hand-china.com/hskp/hskp-javaagent:v1.0.0",
			Command:      []string{"sh", "-c", "cp /data/opentelemetry-javaagent.jar /agent"},
			VolumeMounts: []v1.VolumeMount{volumeMount},
		}
		podSpec.InitContainers = append(podSpec.InitContainers, agentInitContainer)
		//给Containers设置挂载点
		for _, c := range podSpec.Containers {
			if c.VolumeMounts == nil {
				c.VolumeMounts = make([]v1.VolumeMount, 1, 1)
			}
			c.VolumeMounts = append(c.VolumeMounts, volumeMount)
		}

		if podSpec.Volumes == nil {
			podSpec.Volumes = make([]v1.Volume, 1, 1)
		}
		volume := v1.Volume{
			Name: "agent",
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		}
		podSpec.Volumes = append(podSpec.Volumes, volume)

		if err := setNestedFieldNoCopy(t.Object, podTemplateSpec, "spec", "template"); err != nil {
			glog.Warningf("Set spec failed, %v", err)
		}

	}

	//var addMetricsInitContainer = func() {
	//	metricsEnabled := customLabelOnChart["hskp.io/metrics_enabled"]
	//	if metricsEnabled != "true" {
	//		return
	//	}
	//
	//}

	switch kind {
	case "ReplicationController", "ReplicaSet", "Deployment":
		addImagePullSecrets()
		addSelectorAppLabels()
		addTemplateAppLabels(kind, t.GetName())
		if kind == "Deployment" {
			handlerSpringBootMonitorMetrics()
		}
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
	addCustomLabelIfNotPresent(customLabelOnChart, l)

	t.SetLabels(l)
	return nil
}

func ToObj(data interface{}, into interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, into)
}
func setNestedFieldNoCopy(obj map[string]interface{}, value interface{}, fields ...string) error {
	m := obj

	for i, field := range fields[:len(fields)-1] {
		if val, ok := m[field]; ok {
			if valMap, ok := val.(map[string]interface{}); ok {
				m = valMap
			} else {
				return fmt.Errorf("value cannot be set because %v is not a map[string]interface{}", jsonPath(fields[:i+1]))
			}
		} else {
			newVal := make(map[string]interface{})
			m[field] = newVal
			m = newVal
		}
	}
	m[fields[len(fields)-1]] = value
	return nil
}
func jsonPath(fields []string) string {
	return "." + strings.Join(fields, ".")
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
