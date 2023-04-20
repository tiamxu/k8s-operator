package resource

import (
	"fmt"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

const (
	defaultNamespace     string = "dev"
	defaultImageRegistry string = "registry.cn-hangzhou.aliyuncs.com/unipal"
	//IfNotPresent、Always
	defaultImagePullPolicy  corev1.PullPolicy = "IfNotPresent"
	defaultImagePullSecrets string            = "regcred-vpc"
	defaultTag              string            = "latest"
)

type DeploymentBuild struct {
	*DeployStackBuild
}

func (builder *DeployStackBuild) Deployment() *DeploymentBuild {

	return &DeploymentBuild{builder}
}

func (builder *DeploymentBuild) GetObjectKind() (client.Object, error) {
	return &appsv1.Deployment{}, nil
}

// func (builder *DeploymentBuild) GetObjectKind(name string, deployStack *unstructured.Unstructured, d DeployStackBuild) (schema.GroupVersionKind, error) {
// 	_, serviceType, _ := GetAppConf(name, deployStack, d)
// 	if serviceType == "sts" {
// 		return schema.GroupVersionKind{
// 			Group:   "apps",
// 			Kind:    "StatefulSet",
// 			Version: "v1",
// 		}, nil
// 	}
// 	return schema.GroupVersionKind{
// 		Group:   "apps",
// 		Kind:    "Deployment",
// 		Version: "v1",
// 	}, nil
// }

func (builder *DeploymentBuild) Build(name, tag string, deployStack *unstructured.Unstructured, d DeployStackBuild) (client.Object, error) {
	var (
		namespace, image, registrySecret, imageNamespace string
		replicas                                         *int32
		requestMem, limitMem                             string
		requestCpu, limitCpu                             string
		probePort                                        int
		probeReadyHttp                                   bool
		volumes                                          []corev1.Volume
		volumeMounts                                     []corev1.VolumeMount
	)
	var (
		ports           []corev1.ContainerPort
		imagePullPolicy corev1.PullPolicy
		resources       corev1.ResourceRequirements
	)

	if tag == "" {
		tag = defaultTag
	}

	// deployStackSpec, ok := deployStack.Object["spec"]
	// if !ok {
	// 	return nil, fmt.Errorf("Not Found deployStack Spec")
	// }
	appsConfObj, _, err := GetAppConf(name, deployStack, d)
	if err != nil {
		return nil, err
	}
	appConf, ok := appsConfObj[name]
	if !ok {
		return nil, fmt.Errorf("deployment appConf error:%v", appConf)
	}
	for key, valueConf := range appConf {
		// fmt.Printf("####key:%v,type:%T, value:%v\n", key, valueConf, valueConf)
		switch key {
		case "replicasForDefault", "replicasForWeb", "replicasForApp", fmt.Sprintf("replicasFor%s", strings.Title(name)):
			if value, ok := valueConf.(int64); ok {
				replicas = intFromPtr(value)

			} else if value, ok := valueConf.(string); ok {
				tmp, _ := strconv.Atoi(value)
				replicas = intFromPtr(int64(tmp))

			} else {
				return nil, fmt.Errorf("%v Error", key)
			}
		case "resourcesMemoryForDefault":
			value, ok := valueConf.(string)
			if !ok {
				return nil, fmt.Errorf("%v Error", key)
			}
			requestMem, limitMem = stringsSplit(value)

		case "resourcesCpuForDefault":
			value, ok := valueConf.(string)
			if !ok {
				return nil, fmt.Errorf("%v Error", key)

			}
			requestCpu, limitCpu = stringsSplit(value)

		case "imageRegistryForDefault":
			value, ok := valueConf.(string)
			if !ok {
				return nil, fmt.Errorf("%v Error", key)
			}
			image = value

		case "imageSecretsForDefault":
			value, ok := valueConf.(string)
			if !ok {
				return nil, fmt.Errorf("%v Error", key)
			}
			registrySecret = value

		case "imageNamespaceForDefault":
			value, ok := valueConf.(string)
			if !ok {
				return nil, fmt.Errorf("%v Error", key)
			}
			imageNamespace = value
		case "namespaceForDefault":
			value, ok := valueConf.(string)
			if !ok {
				return nil, fmt.Errorf("%v Error", key)
			}
			namespace = value
		case "volumeCmForDefault", "volumeSecretForCerts":
			suffix, volumeSource := volumeSourceForSuffix(key)
			value, ok := valueConf.(string)
			if !ok {
				return nil, fmt.Errorf("%v Error", key)
			}
			volumesPath := value
			volumes = append(volumes, containerVolumes(name, suffix, volumeSource)...)
			volumeMounts = append(volumeMounts, containerVolumeMounts(name, suffix, volumesPath)...)

		case "portForGrpc", "portForHttp":
			value, ok := valueConf.(int64)
			if !ok {
				return nil, fmt.Errorf("%v Error", key)
			}
			ports = append(ports, containerPorts(name, key, int32(value))...)
		case "probeReadyTcpForDefault":
			if value, ok := valueConf.(int64); ok {
				probePort = int(value)

			} else if value, ok := valueConf.(string); ok {
				tmp, _ := strconv.Atoi(value)
				probePort = tmp

			} else {
				return nil, fmt.Errorf("%v Error", key)
			}
		case "probeHttpEnable":
			if value, ok := valueConf.(bool); ok {
				probeReadyHttp = value

			} else if value, ok := valueConf.(string); ok {
				tmp, _ := strconv.ParseBool(value)
				probeReadyHttp = tmp

			} else {
				return nil, fmt.Errorf("%v Error", key)
			}
		default:
		}
	}
	resources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			"memory": resource.MustParse(requestMem),
			"cpu":    resource.MustParse(requestCpu),
		},
		Limits: corev1.ResourceList{
			"memory": resource.MustParse(limitMem),
			"cpu":    resource.MustParse(limitCpu),
		},
	}

	//image
	image = fmt.Sprintf("%s/%s_%s:%s", image, imageNamespace, name, tag)
	if image == "" {
		image = fmt.Sprintf("%s/%s_%s:%s", defaultImageRegistry, defaultNamespace, name, tag)
	}
	if registrySecret == "" {
		registrySecret = defaultImagePullSecrets
	}
	if tag == defaultTag {
		imagePullPolicy = "Always"
	} else {
		imagePullPolicy = defaultImagePullPolicy
	}
	//env
	env := envVarObject(namespace, name)
	envFrom := envVarFrom()

	affinity := corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: 100,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "app",
									Operator: "In",
									Values:   []string{name},
								},
							},
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		},
	}
	// volumeCmForData, _, _ := unstructured.NestedString(deployStack.Object, "spec", "volumeCmForData")

	//check health
	lifecycle := corev1.Lifecycle{
		PreStop: &corev1.LifecycleHandler{Exec: &corev1.ExecAction{
			Command: []string{"/bin/sh", "-c", "sleep 20"},
		}}}

	livenessProbe := corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				// Port: intstr.IntOrString{Type: intstr.Int, IntVal: probeTcpPort},
				Port: intstr.FromInt(probePort),
			},
		},
		InitialDelaySeconds: 30,
		TimeoutSeconds:      5,
	}
	readinessProbe := corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt(probePort),
			},
		},
		InitialDelaySeconds: 15,
		TimeoutSeconds:      5,
	}
	if probeReadyHttp {
		livenessProbe = corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/ops/alive",
					Port: intstr.FromInt(probePort),
				},
			},
			InitialDelaySeconds: 30,
			TimeoutSeconds:      5,
		}
		readinessProbe = corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/ops/alive",
					Port: intstr.FromInt(probePort),
				},
			},
			InitialDelaySeconds: 15,
			TimeoutSeconds:      5,
		}
	}
	//PodTemplateSpec
	podTemplateSpec := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
			Labels:      LabelsSelector(name, namespace),
		},
		Spec: corev1.PodSpec{
			NodeSelector: map[string]string{},
			Affinity:     &affinity,
			Containers: []corev1.Container{{
				Name:            name,
				Image:           image,
				ImagePullPolicy: imagePullPolicy,
				// Command:         command,
				// Args:            args,
				Ports:          ports,
				Resources:      resources,
				Env:            env,
				EnvFrom:        envFrom,
				VolumeMounts:   volumeMounts,
				LivenessProbe:  &livenessProbe,
				ReadinessProbe: &readinessProbe,
				Lifecycle:      &lifecycle,
			}},
			TerminationGracePeriodSeconds: int64Ptr(30),
			Volumes:                       volumes,
			ImagePullSecrets: []corev1.LocalObjectReference{{
				Name: registrySecret,
			}},
		},
	}

	//deployment
	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: map[string]string{},
			Labels:      Labels(name, namespace),
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: LabelsSelector(name, namespace),
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: int32(0)},
					MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: int32(1)},
				},
			},
			Replicas: replicas,
			Template: podTemplateSpec,
		},
	}

	// sts := appsv1.StatefulSet{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      name,
	// 		Namespace: namespace,
	// 		Labels:    map[string]string{},
	// 	},
	// 	Spec: appsv1.StatefulSetSpec{
	// 		ServiceName: name,
	// 		Selector:    &metav1.LabelSelector{MatchLabels: LabelsSelector(name, namespace)},
	// 		UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
	// 			Type: appsv1.RollingUpdateStatefulSetStrategyType,
	// 			RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{
	// 				Partition: int32Ptr(0),
	// 			},
	// 		},
	// 		Replicas: replicas,
	// 		Template: podTemplateSpec,
	// 	},
	// }

	// _, serviceType, _ := GetAppConf(name, deployStack, d)
	// if serviceType == "sts" {
	// 	return &sts, nil
	// }

	return &deployment, nil
}

// func (builder *DeploymentBuild) ExecStrategy() bool {
// 	return true
// }

func containerPorts(name, key string, port int32) []corev1.ContainerPort {
	var ports []corev1.ContainerPort
	trimStr := strings.TrimSpace(key)
	str := strings.Split(trimStr, "For")
	suffix := strings.ToLower(str[1])
	ports = append(ports, corev1.ContainerPort{
		Name:          fmt.Sprintf("%s-%s", suffix, name),
		ContainerPort: port,
	})
	return ports
}

func envVarObject(namespace, name string) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{Name: "CONFIG_ENV", Value: namespace},
		{Name: "MY_SERVICE_NAME", Value: name},
	}
	return env
}

func envVarFrom() []corev1.EnvFromSource {
	envFrom := []corev1.EnvFromSource{
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "global-config",
				}},
		},
		{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "global-secret"}},
		},
	}
	return envFrom
}

func stringsSplit(name string) (request string, limit string) {
	trimmed := strings.TrimSpace(name)
	str := strings.Split(trimmed, "-")
	if len(str) == 2 {
		request = str[0]
		limit = str[1]
	}
	return request, limit
}

func uniqueStrings(confValue []string) []string {
	uniqueMap := make(map[string]bool)
	for _, s := range confValue {
		uniqueMap[s] = true
	}
	uniqueConfValue := make([]string, 0, len(uniqueMap))
	for k := range uniqueMap {
		uniqueConfValue = append(uniqueConfValue, k)
	}
	return uniqueConfValue
}

// func (builder *DeploymentBuild) getDefaultConf(confValue []string, confDefault, deployStackSpec map[string]interface{}) (map[string]interface{}, error) {
// 	// var confDefault = make(map[string]interface{})
// 	// deployStackSpec, ok := deployStack.Object["spec"].(map[string]interface{})
// 	// if !ok {
// 	// 	return nil, fmt.Errorf("deployStack.Object is error")
// 	// }
// 	for _, key := range confValue {
// 		switch key {
// 		case "replicasForDefault":
// 			value, ok := deployStackSpec[key].(int64)
// 			if !ok {
// 				return nil, fmt.Errorf("%v Error", key)
// 			}
// 			confDefault[key] = value
// 		case "resourcesMemoryForDefault":
// 			value, ok := deployStackSpec[key].(string)
// 			if !ok {
// 				return nil, fmt.Errorf("%v Error", key)
// 			}
// 			confDefault[key] = value

// 		case "resourcesCpuForDefault":
// 			value, ok := deployStackSpec[key].(string)
// 			if !ok {
// 				return nil, fmt.Errorf("%v Error", key)
// 			}
// 			confDefault[key] = value

// 		case "imageRegistryForDefault":
// 			value, ok := deployStackSpec[key].(string)
// 			if !ok {
// 				return nil, fmt.Errorf("%v Error", key)
// 			}
// 			confDefault[key] = value

// 		case "imageSecretsForDefault":
// 			value, ok := deployStackSpec[key].(string)
// 			if !ok {
// 				return nil, fmt.Errorf("%v Error", key)
// 			}
// 			confDefault[key] = value

// 		case "imageNamespaceForDefault":
// 			value, ok := deployStackSpec[key].(string)
// 			if !ok {
// 				return nil, fmt.Errorf("%v Error", key)
// 			}
// 			confDefault[key] = value
// 		default:
// 		}
// 	}
// 	return confDefault, nil
// }

// 字符串后缀是否为Default,自定义服务配置处理
func getConfKeyValue(conf string) (string, interface{}) {
	var (
		key   string
		value interface{}
	)
	trimStr := strings.TrimSpace(conf)
	//使用空格分割字符串
	// str := strings.Fields(trimStr)
	str := strings.Split(trimStr, ":")
	if len(str) == 1 {
		key = trimStr
		value = nil
	} else if len(str) == 2 {
		key = str[0]
		value = str[1]
	} else {
		key = str[0]
		value = str[1:]
	}
	return key, value
}

func volumeSourceForSuffix(key string) (string, string) {
	var volumeSource string
	trimStr := strings.TrimSpace(key)
	str := strings.Split(trimStr, "For")
	prefix := str[0]
	suffix := strings.ToLower(str[1])
	if strings.Contains(trimStr, prefix) {
		switch prefix {
		case "volumeCm":
			volumeSource = "ConfigMap"
		case "volumeSecret":
			volumeSource = "Secret"
		default:
			volumeSource = "ConfigMap"
		}
		if suffix == "default" {
			suffix = "conf"
		}
		return suffix, volumeSource
	} else {
		suffix = "conf"
		volumeSource = "ConfigMap"
	}
	return suffix, volumeSource

}
func containerVolumes(name, configSuffix, volumeSource string) []corev1.Volume {
	var volumes []corev1.Volume
	switch volumeSource {
	case "ConfigMap":
		volumes = append(volumes, corev1.Volume{
			Name: fmt.Sprintf("%s-%s", name, configSuffix),
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: name,
					},
				},
			},
		})
	case "Secret":
		volumes = append(volumes, corev1.Volume{

			Name: fmt.Sprintf("%s-%s", name, configSuffix),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: fmt.Sprintf("%s-%s", name, configSuffix),
				},
			},
		})
	default:
	}

	return volumes
}
func containerVolumeMounts(name, configSuffix, path string) []corev1.VolumeMount {
	var volumeMounts []corev1.VolumeMount
	volumeMounts = append(volumeMounts, []corev1.VolumeMount{
		{Name: fmt.Sprintf("%s-%s", name, configSuffix), MountPath: path},
	}...)

	return volumeMounts
}

// 判断配置keyName是否是默认,还是自定义
func getKeySuffixName(key string) string {
	var (
		name string
	)
	trimStr := strings.TrimSpace(key)
	if strings.HasSuffix(trimStr, "Default") {

	}
	str := strings.Split(trimStr, "For")
	if len(str) == 2 {
		name = strings.ToLower(str[1])
	}
	return name
}
