package resource

import (
	"fmt"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatefulSetBuild struct {
	*DeployStackBuild
}

func (builder *DeployStackBuild) StatefulSet() *StatefulSetBuild {

	return &StatefulSetBuild{builder}
}
func (builder *StatefulSetBuild) GetObjectKind() (client.Object, error) {
	return &appsv1.StatefulSet{}, nil
}

func (builder *StatefulSetBuild) Build(name, tag string, deployStack *unstructured.Unstructured, d DeployStackBuild) (client.Object, error) {
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
	appsConfObj, _, err := GetAppConf(name, deployStack, d)
	if err != nil {
		return nil, err
	}
	appConf, ok := appsConfObj[name]
	if !ok {
		return nil, fmt.Errorf("appConf error:%v", appConf)
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
	//sts
	sts := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: map[string]string{},
			Labels:      Labels(name, namespace),
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: name,
			Selector:    &metav1.LabelSelector{MatchLabels: LabelsSelector(name, namespace)},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{
					Partition: int32Ptr(0),
				},
			},
			Replicas: replicas,
			Template: podTemplateSpec,
		},
	}

	return &sts, nil
}
