package resource

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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

// func (builder *StatefulSetBuild) Build(name, tag string) (client.Object, error) {

// 	podTemplateSpec := builder.podTemplateSpec(name, tag)

// 	sts := appsv1.StatefulSet{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      name,
// 			Namespace: builder.Instance.Spec.Namespace,
// 			Labels:    map[string]string{},
// 		},
// 		Spec: appsv1.StatefulSetSpec{
// 			ServiceName: name,
// 			Selector:    &metav1.LabelSelector{MatchLabels: LabelsSelector(name, builder.Instance.Spec.Namespace)},
// 			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
// 				Type: appsv1.RollingUpdateStatefulSetStrategyType,
// 				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{
// 					Partition: int32Ptr(0),
// 				},
// 			},
// 			Replicas: builder.Instance.Spec.Replicas,
// 			Template: podTemplateSpec,
// 		},
// 	}

// 	return &sts, nil
// }

// func (builder *StatefulSetBuild) podTemplateSpec(name, tag string) corev1.PodTemplateSpec {
// 	var (
// 		image           string
// 		registrySecret  string
// 		ports           []corev1.ContainerPort
// 		resources       corev1.ResourceRequirements
// 		imagePullPolicy corev1.PullPolicy
// 	)
// 	var (
// 		configSuffix string = "config"
// 		// prefixSuffix string = "config"
// 	)
// 	namespace := builder.Instance.Spec.Namespace

// 	requestMem, limitMem := stringsSplit(builder.Instance.Spec.ResourcesMemoryForDefault)
// 	requestCpu, limitCpu := stringsSplit(builder.Instance.Spec.ResourcesCpuForDefault)
// 	defaultResources := corev1.ResourceRequirements{
// 		Requests: corev1.ResourceList{
// 			"memory": resource.MustParse(requestMem),
// 			"cpu":    resource.MustParse(requestCpu),
// 		},
// 		Limits: corev1.ResourceList{
// 			"memory": resource.MustParse(limitMem),
// 			"cpu":    resource.MustParse(limitCpu),
// 		},
// 	}
// 	affinity := corev1.Affinity{
// 		PodAffinity: &corev1.PodAffinity{
// 			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
// 				{
// 					Weight: 1,
// 					PodAffinityTerm: corev1.PodAffinityTerm{
// 						LabelSelector: &metav1.LabelSelector{
// 							MatchExpressions: []metav1.LabelSelectorRequirement{
// 								{
// 									Key:      "app",
// 									Operator: "In",
// 									Values:   []string{name},
// 								},
// 							},
// 						},
// 						TopologyKey: "kubernetes.io/hostname",
// 					},
// 				},
// 			},
// 		},
// 	}
// 	volumes := []corev1.Volume{
// 		{
// 			Name: fmt.Sprintf("%s-%s", name, configSuffix),
// 			VolumeSource: corev1.VolumeSource{
// 				ConfigMap: &corev1.ConfigMapVolumeSource{
// 					LocalObjectReference: corev1.LocalObjectReference{
// 						Name: name,
// 					},
// 				},
// 			},
// 		},
// 	}
// 	volumeMounts := []corev1.VolumeMount{
// 		{Name: fmt.Sprintf("%s-%s", name, configSuffix), MountPath: "/www/config/"},
// 	}
// 	lifecycle := corev1.Lifecycle{
// 		PreStop: &corev1.LifecycleHandler{Exec: &corev1.ExecAction{
// 			Command: []string{"/bin/sh", "-c", "sleep 20"},
// 		}}}
// 	livenessProbe := corev1.Probe{
// 		ProbeHandler: corev1.ProbeHandler{
// 			HTTPGet: &corev1.HTTPGetAction{
// 				Path: "/ops/alive",
// 				Port: intstr.FromInt(6060),
// 			},
// 		},
// 		InitialDelaySeconds: 30,
// 		TimeoutSeconds:      5,
// 	}
// 	readinessProbe := corev1.Probe{
// 		ProbeHandler: corev1.ProbeHandler{
// 			HTTPGet: &corev1.HTTPGetAction{
// 				Path: "/ops/alive",
// 				Port: intstr.FromInt(6060),
// 			},
// 		},
// 		InitialDelaySeconds: 15,
// 		TimeoutSeconds:      5,
// 	}

// 	//image
// 	if tag == defaultTag {
// 		imagePullPolicy = "Always"
// 	} else {
// 		imagePullPolicy = defaultImagePullPolicy
// 	}
// 	//registry secret
// 	if builder.Instance.Spec.RegistrySecrets != "" {
// 		registrySecret = builder.Instance.Spec.RegistrySecrets
// 	} else {
// 		registrySecret = defaultImagePullSecrets
// 	}
// 	//container port
// 	if builder.Instance.Spec.Ports != nil {
// 		ports = builder.containerPorts(name, builder.Instance.Spec.Ports)
// 	} else {
// 		if builder.Instance.Spec.PortForGrpc != 0 {
// 			ports = []corev1.ContainerPort{{
// 				Name:          StringCombin("grpc", "-", name),
// 				ContainerPort: builder.Instance.Spec.PortForGrpc,
// 			}}
// 		}
// 	}
// 	if builder.Instance.Spec.ImageRegistry != "" {
// 		image = fmt.Sprintf("%s/%s_%s:%s", builder.Instance.Spec.ImageRegistry, builder.Instance.Namespace, name, tag)
// 	} else {
// 		image = fmt.Sprintf("%s/%s:%s", defaultImageRegistry, name, tag)
// 	}
// 	if builder.Instance.Spec.Resources != nil {
// 		resources = *builder.Instance.Spec.Resources
// 	} else {
// 		resources = defaultResources
// 	}

// 	appsName := builder.Instance.Spec.Apps
// 	if apps, ok := appsName[name]; ok {
// 		if apps.RegistrySecrets != "" {
// 			registrySecret = apps.RegistrySecrets
// 		}
// 		if apps.Ports != nil {
// 			ports = append(ports, builder.containerPorts(name, apps.Ports)...)
// 		}
// 		if apps.ImageRegistry != "" {
// 			image = fmt.Sprintf("%s/%s:%s", apps.ImageRegistry, name, tag)
// 		}
// 		// resources = defaultResources
// 		//
// 	}

// 	podTemplateSpec := corev1.PodTemplateSpec{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Annotations: map[string]string{},
// 			Labels:      LabelsSelector(name, namespace),
// 		},
// 		Spec: corev1.PodSpec{
// 			Affinity: &affinity,
// 			Containers: []corev1.Container{{
// 				Name:            name,
// 				Image:           image,
// 				ImagePullPolicy: imagePullPolicy,
// 				// Command:         command,
// 				// Args:            args,
// 				Ports:     ports,
// 				Resources: resources,
// 				// Env:,
// 				VolumeMounts:   volumeMounts,
// 				LivenessProbe:  &livenessProbe,
// 				ReadinessProbe: &readinessProbe,
// 				Lifecycle:      &lifecycle,
// 			}},
// 			TerminationGracePeriodSeconds: int64Ptr(30),
// 			Volumes:                       volumes,
// 			ImagePullSecrets: []corev1.LocalObjectReference{{
// 				Name: registrySecret,
// 			}},
// 		},
// 	}

// 	return podTemplateSpec
// }

func (builder *StatefulSetBuild) containerPorts(name string, containerPorts []ContainerPorts) []corev1.ContainerPort {
	var ports []corev1.ContainerPort
	for _, containerPort := range containerPorts {
		ports = append(ports, corev1.ContainerPort{
			Name:          StringCombin(containerPort.Name, "-", name),
			ContainerPort: containerPort.Port,
		})
	}

	return ports
}
