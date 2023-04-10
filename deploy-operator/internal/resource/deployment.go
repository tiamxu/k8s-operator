package resource

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// defaultNamespace               = "default"
	// defaultImageRegistry           = "registry-vpc.cn-hangzhou.aliyuncs.com"
	// defaultImagePullPolicy         = "IfNotPresent"
	// defaultImagePullSecrets        = "regcred-vpc"
	defaultTag string = "latest"
)

var (
	deployment     appsv1.Deployment
	volumes        []corev1.Volume
	volumeMounts   []corev1.VolumeMount
	lifecycle      corev1.Lifecycle
	readinessProbe corev1.Probe
	livenessProbe  corev1.Probe
	command        []string
	args           []string
	// containerPorts                                    []corev1.ContainerPort
	resources, defaultResources      corev1.ResourceRequirements
	registrySecret, image, namespace string
	replicas                         *int32
)
var ports []corev1.ContainerPort
var (
	configSuffix string = "config"
	// prefixSuffix string = "config"
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

// var defaultAppList = map[string]string{"test":"latest",}

func (builder *DeploymentBuild) Build(name, tag string) (client.Object, error) {
	if tag == "" {
		tag = defaultTag
	}
	var (
		ports []corev1.ContainerPort
		// namespace string
		affinity corev1.Affinity
	)
	appsName := builder.Instance.Spec.Apps
	// apps, ok := appsName[name]
	if apps, ok := appsName[name]; ok {
		if apps.Namespace != "" {
			namespace = apps.Namespace
		} else {
			namespace = builder.Instance.Spec.Namespace
		}
		if apps.Replicas != nil {
			replicas = apps.Replicas
		} else {
			replicas = builder.Instance.Spec.Replicas
		}
		if apps.RegistrySecrets != "" {
			registrySecret = apps.RegistrySecrets
		} else {
			registrySecret = builder.Instance.Spec.RegistrySecrets

		}
		if apps.Ports != nil {
			ports = builder.containerPorts(name, apps.Ports)
		} else {
			ports = builder.containerPorts(name, builder.Instance.Spec.Ports)
		}

	} else {
		namespace = builder.Instance.Spec.Namespace
		replicas = builder.Instance.Spec.Replicas
		registrySecret = builder.Instance.Spec.RegistrySecrets
		ports = builder.containerPorts(name, builder.Instance.Spec.Ports)
		image = fmt.Sprintf("%s/%s_%s:%s", builder.Instance.Spec.ImageRegistry, builder.Instance.Namespace, name, tag)
		if builder.Instance.Spec.Resources != nil {
			resources = *builder.Instance.Spec.Resources
		} else {
			resources = defaultResources
		}

	}

	deployment = appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    Labels(name),
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: Labels(name),
			},
			Replicas: replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: Labels(name),
				},
				Spec: corev1.PodSpec{
					Affinity: &affinity,
					Containers: []corev1.Container{{
						Name:            name,
						Image:           image,
						ImagePullPolicy: "Always",
						// Command:         command,
						// Args:            args,
						Ports:     ports,
						Resources: resources,
						// Env:,
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
			},
		},
	}
	affinity = corev1.Affinity{
		PodAffinity: &corev1.PodAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: 1,
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
	volumes = []corev1.Volume{
		{
			Name: fmt.Sprintf("%s-%s", name, configSuffix),
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: name,
					},
				},
			},
		},
	}
	volumeMounts = []corev1.VolumeMount{
		{Name: fmt.Sprintf("%s-%s", name, configSuffix), MountPath: "/www/config/"},
	}
	lifecycle = corev1.Lifecycle{
		PreStop: &corev1.LifecycleHandler{Exec: &corev1.ExecAction{
			Command: []string{"/bin/sh", "-c", "sleep 20"},
		}}}
	livenessProbe = corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/ops/alive",
				Port: intstr.FromInt(6060),
			},
		},
		InitialDelaySeconds: 30,
		TimeoutSeconds:      5,
	}
	readinessProbe = corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/ops/alive",
				Port: intstr.FromInt(6060),
			},
		},
		InitialDelaySeconds: 15,
		TimeoutSeconds:      5,
	}
	defaultResources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			"memory": resource.MustParse("100Mi"),
			"cpu":    resource.MustParse("10m"),
		},
		Limits: corev1.ResourceList{
			"memory": resource.MustParse("1024Mi"),
			"cpu":    resource.MustParse("500m"),
		},
	}
	command = []string{"/bin/bash", "-c", "sleep 5"}
	args = []string{"/bin/bash", "-c", "sleep 5"}
	// env := []corev1.EnvVar{{Name:"CONFIG_ENV",Value:"prod"}}
	// containerPorts = []corev1.ContainerPort{
	// 	{
	// 		Name: "grpc", ContainerPort: int32(5010),
	// 	},
	// 	{
	// 		Name: "http", ContainerPort: int32(8800),
	// 	},
	// }

	return &deployment, nil

}

func (builder *DeploymentBuild) ExecStrategy() bool {
	return true
}

// type Ports struct {
// 	Name string `json:"name,omitempty"`
// 	Port int32  `json:"port,omitempty"`
// }

func (builder *DeploymentBuild) containerPorts(name string, containerPorts []ContainerPorts) []corev1.ContainerPort {
	var ports []corev1.ContainerPort
	// containerPorts := builder.Instance.Spec.Ports
	for _, containerPort := range containerPorts {
		ports = append(ports, corev1.ContainerPort{
			Name:          StringCombin(containerPort.Name, "-", name),
			ContainerPort: containerPort.Port,
		})
	}

	return ports
}
func (builder *DeploymentBuild) Update(object client.Object, name, tag string) (client.Object, error) {
	deploy := object.(*appsv1.Deployment)

	//Replicas
	deploy.Spec.Replicas = builder.Instance.Spec.Replicas
	//pod template
	deploy.Spec.Template = builder.podTemplateSpec(name, tag)
	return deploy, nil
}

func (builder *DeploymentBuild) podTemplateSpec(name, tag string) corev1.PodTemplateSpec {
	if tag == "" {
		tag = defaultTag
	}
	// var ports []corev1.ContainerPort
	appsName := builder.Instance.Spec.Apps
	apps, ok := appsName[name]
	if ok {
		replicas = apps.Replicas
		registrySecret = apps.RegistrySecrets
		image = fmt.Sprintf("%s/%s:%s", apps.ImageRegistry, name, tag)
		ports = builder.containerPorts(name, apps.Ports)

	} else {
		replicas = builder.Instance.Spec.Replicas
		registrySecret = builder.Instance.Spec.RegistrySecrets
		image = fmt.Sprintf("%s:%s", builder.Instance.Spec.ImageRegistry, tag)
		ports = builder.containerPorts(name, builder.Instance.Spec.Ports)

	}
	// image := fmt.Sprintf("%s/%s_%s:%s", builder.Instance.Spec.ImageRegistry, builder.Instance.Namespace, name, tag)
	podTemplateSpec := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: Labels(name),
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:            name,
				Image:           image,
				ImagePullPolicy: "Always",
				// Command:         command,
				// Args:            args,
				Ports:     ports,
				Resources: resources,
				// Env:,
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
	volumes = []corev1.Volume{
		{
			Name: fmt.Sprintf("%s-%s", name, configSuffix),
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: name,
					},
				},
			},
		},
	}
	volumeMounts = []corev1.VolumeMount{
		{Name: fmt.Sprintf("%s-%s", name, configSuffix), MountPath: "/www/config/"},
	}
	lifecycle = corev1.Lifecycle{
		PreStop: &corev1.LifecycleHandler{Exec: &corev1.ExecAction{
			Command: []string{"/bin/sh", "-c", "sleep 20"},
		}}}
	livenessProbe = corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/ops/alive",
				Port: intstr.FromInt(6060),
			},
		},
		InitialDelaySeconds: 30,
		TimeoutSeconds:      5,
	}
	readinessProbe = corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/ops/alive",
				Port: intstr.FromInt(6060),
			},
		},
		InitialDelaySeconds: 15,
		TimeoutSeconds:      5,
	}
	resources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			"memory": resource.MustParse("100Mi"),
			"cpu":    resource.MustParse("10m"),
		},
		Limits: corev1.ResourceList{
			"memory": resource.MustParse("1024Mi"),
			"cpu":    resource.MustParse("500m"),
		},
	}
	return podTemplateSpec
}

// func (builder *DeploymentBuild) containerVolumeMounts(name string, obj []client.Object) []corev1.VolumeMount {
// 	var volumeMounts []corev1.VolumeMount
// 	// containerPorts := builder.Instance.Spec.Ports
// 	for _, containerPort := range volumeMounts {
// 		ports = append(ports, corev1.ContainerPort{
// 			Name:          stringCombin(containerPort.Name, "-", name),
// 			ContainerPort: containerPort.Port,
// 		})
// 	}

//		return volumeMounts
//	}
func envVarObject() []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name: "CONFIG_ENV", Value: "prod"},
		{
			Name: "CONFIG_DB_USERNAME",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test"},
					Key: "username"}}},
	}
	return env
}
