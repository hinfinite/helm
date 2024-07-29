package action

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/hinfinite/helm/pkg/agent/model"
	v1 "k8s.io/api/core/v1"
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
	customLabelOnResource map[string]string,
	hskpCommonLabelOnChart map[string]string,
	currentFlag bool) error {
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
		labels[model.LastUpdateTime] = time.Now().String()

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
		traceEnabled := customLabelOnChart["hskp.io/spring_boot_trace_enabled"]
		if metricsEnabled != "true" && traceEnabled != "true" {
			return
		}

		unstructuredPodTemplateSpec, _, err := unstructured.NestedFieldNoCopy(t.Object, "spec", "template")
		if err != nil {
			glog.Error("Get spec.template failed, %v", err)
		}
		if unstructuredPodTemplateSpec == nil {
			glog.Warningf(">>>>>>>>>>>>>>>>>podTemplateSpec is null")
			return
		}
		podTemplateSpec := v1.PodTemplateSpec{}

		ToObj(unstructuredPodTemplateSpec, &podTemplateSpec)
		if metricsEnabled == "true" {
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
		}

		podSpec := &podTemplateSpec.Spec
		if podSpec.Containers == nil {
			glog.Warningf(">>>>>>>>>>>>>>>>>podSpec.Containers is null")
			return
		}
		//挂载点
		volumeMount := v1.VolumeMount{
			Name:      "hskp-agent",
			MountPath: "/hskp/agent",
		}
		//更新的时候需要对当前的资源也进行处理，但是部分资源需要前后不一致，否则更新的时候无法生成patch path
		if currentFlag {
			volumeMount.Name = "hskp-agent-old"
		}
		//设置agentInitContainer
		if podSpec.InitContainers == nil {
			podSpec.InitContainers = []v1.Container{}
		}
		agentInitContainer := v1.Container{
			Name:            "hskp-java-agent",
			Image:           "harbor.open.hand-china.com/hskp/hskp-javaagent:v1.2.0",
			Command:         []string{"sh", "-c", "cp /data/agents/opentelemetry-* /hskp/agent"},
			ImagePullPolicy: v1.PullPolicy("IfNotPresent"),
			VolumeMounts:    []v1.VolumeMount{volumeMount},
		}
		podSpec.InitContainers = append(podSpec.InitContainers, agentInitContainer)
		//给Containers设置挂载点
		//给Containers设置挂载点。
		var mountedVolumes []v1.VolumeMount
		for index, _ := range podSpec.Containers {
			c := &podSpec.Containers[index]
			if c.VolumeMounts == nil {
				c.VolumeMounts = []v1.VolumeMount{}
			}
			mountPathExist := false
			for _, existVolumeMount := range c.VolumeMounts {
				if existVolumeMount.MountPath == volumeMount.MountPath {
					mountPathExist = true
					mountedVolumes = append(mountedVolumes, existVolumeMount)
					break
				}
			}
			if mountPathExist {
				continue
			}
			c.VolumeMounts = append(c.VolumeMounts, volumeMount)
			mountedVolumes = append(mountedVolumes, volumeMount)

		}
		//基于mountedVolumes添加volumes,
		if podSpec.Volumes == nil {
			podSpec.Volumes = []v1.Volume{}
		}
		for _, mountedVolume := range mountedVolumes {
			exist := false
			for _, existVolume := range podSpec.Volumes {
				if existVolume.Name == mountedVolume.Name {
					exist = true
					break
				}
			}
			if exist {
				continue
			}
			volume := v1.Volume{
				Name: mountedVolume.Name,
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			}
			podSpec.Volumes = append(podSpec.Volumes, volume)
		}
		encodeMap, err := EncodeToMap(podTemplateSpec)
		if err != nil {
			glog.Error("Get spec.template failed, %v", err)
		}
		if err := unstructured.SetNestedMap(t.Object, encodeMap, "spec", "template"); err != nil {
			glog.Error("Set spec failed, %v", err)
		}

	}
	//处理hzero1.10的license权限认证
	var handlerSvcLicense = func() {
		svcLicenseeEnabled := hskpCommonLabelOnChart["hskp.io/svc_licensee_enabled"]
		svcLicenseeLicUrl := hskpCommonLabelOnChart["hskp.io/svc_licensee_lic_url"]
		svcLicenseeAgentUrl := hskpCommonLabelOnChart["hskp.io/svc_licensee_agent_url"]
		if svcLicenseeEnabled != "true" {
			return
		}

		unstructuredPodTemplateSpec, _, err := unstructured.NestedFieldNoCopy(t.Object, "spec", "template")
		if err != nil {
			glog.Error("Get spec.template failed, %v", err)
		}
		if unstructuredPodTemplateSpec == nil {
			glog.Warningf(">>>>>>>>>>>>>>>>>podTemplateSpec is null")
			return
		}
		podTemplateSpec := v1.PodTemplateSpec{}

		ToObj(unstructuredPodTemplateSpec, &podTemplateSpec)

		podSpec := &podTemplateSpec.Spec
		if podSpec.Containers == nil {
			glog.Warningf(">>>>>>>>>>>>>>>>>podSpec.Containers is null")
			return
		}
		//挂载点
		volumeMount := v1.VolumeMount{
			Name:      "hskp-agent",
			MountPath: "/hskp/agent",
		}
		//更新的时候需要对当前的资源也进行处理，但是部分资源需要前后不一致，否则更新的时候无法生成patch path
		if currentFlag {
			volumeMount.Name = "hskp-agent-old"
		}

		//设置agentInitContainer
		if podSpec.InitContainers == nil {
			podSpec.InitContainers = []v1.Container{}
		}
		agentInitContainer := v1.Container{
			Name:            "hskp-license-agent",
			Image:           "harbor.open.hand-china.com/hskp/hskp-javaagent:v1.2.0",
			Command:         []string{"sh", "-c", "bash /data/agents/download_license.sh && /bin/cp -rf /data/agents/* /hskp/agent"},
			ImagePullPolicy: v1.PullPolicy("IfNotPresent"),
			VolumeMounts:    []v1.VolumeMount{volumeMount},
		}
		//设置lic文件下载路径
		if svcLicenseeLicUrl != "" {
			licenseeLicUrlEnv := v1.EnvVar{
				Name:  "HSKP_DOWNLOAD_LICENSE_URL",
				Value: svcLicenseeLicUrl,
			}
			agentInitContainer.Env = append(agentInitContainer.Env, licenseeLicUrlEnv)
		}
		//设置agent下载路径
		if svcLicenseeAgentUrl != "" {
			licenseeAgentUrlEnv := v1.EnvVar{
				Name:  "HSKP_DOWNLOAD_LICENSE_AGENT_URL",
				Value: svcLicenseeAgentUrl,
			}
			agentInitContainer.Env = append(agentInitContainer.Env, licenseeAgentUrlEnv)
		}

		podSpec.InitContainers = append(podSpec.InitContainers, agentInitContainer)
		//给Containers设置挂载点。
		var mountedVolumes []v1.VolumeMount
		for index, _ := range podSpec.Containers {
			c := &podSpec.Containers[index]
			if c.VolumeMounts == nil {
				c.VolumeMounts = []v1.VolumeMount{}
			}
			mountPathExist := false
			for _, existVolumeMount := range c.VolumeMounts {
				if existVolumeMount.MountPath == volumeMount.MountPath {
					mountPathExist = true
					mountedVolumes = append(mountedVolumes, existVolumeMount)
					break
				}
			}
			if mountPathExist {
				continue
			}
			c.VolumeMounts = append(c.VolumeMounts, volumeMount)
			mountedVolumes = append(mountedVolumes, volumeMount)

		}
		//基于mountedVolumes添加volumes,
		if podSpec.Volumes == nil {
			podSpec.Volumes = []v1.Volume{}
		}
		for _, mountedVolume := range mountedVolumes {
			exist := false
			for _, existVolume := range podSpec.Volumes {
				if existVolume.Name == mountedVolume.Name {
					exist = true
					break
				}
			}
			if exist {
				continue
			}
			volume := v1.Volume{
				Name: mountedVolume.Name,
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			}
			podSpec.Volumes = append(podSpec.Volumes, volume)
		}

		encodeMap, err := EncodeToMap(podTemplateSpec)
		if err != nil {
			glog.Error("Get spec.template failed, %v", err)
		}
		if err := unstructured.SetNestedMap(t.Object, encodeMap, "spec", "template"); err != nil {
			glog.Error("Set spec failed, %v", err)
		}

	}
	switch kind {
	case "ReplicationController", "ReplicaSet", "Deployment":
		addImagePullSecrets()
		addSelectorAppLabels()
		addTemplateAppLabels(kind, t.GetName())
		if kind == "Deployment" {
			handlerSpringBootMonitorMetrics()
			handlerSvcLicense()
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
	case "ConfigMap":
	case "Service":
	case "Ingress":
	case "Secret":
	case "Pod":
	default:
	}
	//if t.GetNamespace() != "" && t.GetNamespace() != namespace {
	//	return fmt.Errorf(" Kind:%s Name:%s. The namespace of this resource is not consistent with helm release", kind, t.GetName())
	//}

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
func EncodeToMap(obj interface{}) (map[string]interface{}, error) {
	if m, ok := obj.(map[string]interface{}); ok {
		return m, nil
	}

	if unstr, ok := obj.(*unstructured.Unstructured); ok {
		return unstr.Object, nil
	}

	b, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	result := map[string]interface{}{}
	dec := json.NewDecoder(bytes.NewBuffer(b))
	dec.UseNumber()
	return result, dec.Decode(&result)
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
