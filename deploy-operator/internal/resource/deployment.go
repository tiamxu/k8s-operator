package resource

import (
	"fmt"
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

func (builder *DeploymentBuild) Build(name, tag string, deployStack *unstructured.Unstructured) (client.Object, error) {
	var (
		namespace, image, registrySecret string
		replicas                         *int32
		requestMem, limitMem             string
		requestCpu, limitCpu             string
	)
	var (
		ports           []corev1.ContainerPort
		imagePullPolicy corev1.PullPolicy
		resources       corev1.ResourceRequirements
	)

	var (
		configSuffix string = "config"
		// prefixSuffix string = "config"
	)
	if tag == "" {
		tag = defaultTag
	}

	// deployStackSpec, ok := deployStack.Object["spec"]
	// if !ok {
	// 	return nil, fmt.Errorf("Not Found deployStack Spec")
	// }

	deployStackSpec, ok := deployStack.Object["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("deployStack.Object is error")
	}
	//通用配置项
	defaultForConfig, ok := deployStackSpec["default"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("defaultForConfig  error")
	}
	for _, key := range defaultForConfig {
		switch key {
		case "replicasForDefault":
			value, ok := deployStackSpec[key.(string)].(int64)
			if !ok {
				return nil, fmt.Errorf("%v Error", key)
			}
			replicas = intFromPtr(value)
		case "resourcesMemoryForDefault":
			value, ok := deployStackSpec[key.(string)].(string)
			if !ok {
				return nil, fmt.Errorf("%v Error", key)
			}
			requestMem, limitMem = stringsSplit(value)

		case "resourcesCpuForDefault":
			value, ok := deployStackSpec[key.(string)].(string)
			if !ok {
				return nil, fmt.Errorf("%v Error", key)

			}
			requestCpu, limitCpu = stringsSplit(value)

		case "imageRegistryForDefault":
			value, ok := deployStackSpec[key.(string)].(string)
			if !ok {
				return nil, fmt.Errorf("%v Error", key)
			}
			image = value

		case "imageSecretsForDefault":
			value, ok := deployStackSpec[key.(string)].(string)
			if !ok {
				return nil, fmt.Errorf("%v Error", key)
			}
			registrySecret = value

		case "imageNamespaceForDefault":
			value, ok := deployStackSpec[key.(string)].(string)
			if !ok {
				return nil, fmt.Errorf("%v Error", key)
			}
			namespace = value

		default:
		}

	}

	// var appConfObj map[string]interface{}

	// for key, _ := range deployStackSpec {
	// 	// fmt.Println(key, appConf)
	// 	switch key {
	// 	case "web":
	// 		//web h5
	// 		if confApp, ok := deployStackSpec["web"]; ok {
	// 			if appConfObj, ok = confApp.(map[string]interface{}); !ok {
	// 				return nil, fmt.Errorf("confWeb  error")
	// 			}
	// 		}

	// }
	// fmt.Println(appConfObj)

	//server
	// if confApp, ok := deployStackSpec["app"]; ok {
	// 	if appConfObj, ok = confApp.(map[string]interface{}); !ok {
	// 		return nil, fmt.Errorf("confApp  error")
	// 	}
	// }
	// appConfObj = deployStackSpec

	//自定义服务配置
	if appServer, ok := deployStackSpec[name]; ok {
		for _, key := range appServer.([]interface{}) {
			switch key {
			case fmt.Sprintf("replicasFor%s", strings.Title(name)):
				// if value, ok := deployStackSpec[key.(string)].(int64); ok {
				// 	replicas = intFromPtr(value)
				// }
				value, _, _ := unstructured.NestedInt64(deployStack.Object, "spec", fmt.Sprintf("replicasFor%s", strings.Title(name)))
				replicas = intFromPtr(value)
			case fmt.Sprintf("resourcesMemoryFor%s", strings.Title(name)):
				value, ok := deployStackSpec[key.(string)].(string)
				if !ok {
					return nil, fmt.Errorf("%v Error", key)
				}
				requestMem, limitMem = stringsSplit(value)
			case fmt.Sprintf("resourcesCpuFor%s", strings.Title(name)):
				value, ok := deployStackSpec[key.(string)].(string)
				if !ok {
					return nil, fmt.Errorf("%v Error", key)
				}
				requestCpu, limitCpu = stringsSplit(value)
			default:
			}
		}
	}
	// fmt.Println(fmt.Sprintf("replicasFor%s", strings.Title(name)))

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
	if xx, ok := deployStackSpec["portForGrpc"].(int64); ok {
		ports = []corev1.ContainerPort{{
			Name:          StringCombin("grpc", "-", name),
			ContainerPort: int32(xx),
		},
		}
	}
	fmt.Printf("replicas:%v,namespace:%v,ports:%v,resources:%v\n", *replicas, namespace, ports, resources)

	appsConf := builder.Instance.Spec.AppsConf
	for k, _ := range deployStackSpec {
		for kk, vv := range appsConf {
			if kk == k {
				fmt.Printf("kk:%v,vv:%v\n", kk, vv)
				confApp, ok := deployStackSpec[k].([]interface{})
				if !ok {
					return nil, fmt.Errorf("confApp  error")
				}
				fmt.Printf("confApp:%v \n", confApp)
				for _, vvv := range confApp {
					fmt.Printf("vvv%v\n", vvv)
				}
			}
		}
		// fmt.Printf("kk:%v,vv:%v\n", kk, vv)

	}

	// namespace = builder.Instance.Spec.Namespace
	// podTemplateSpec := builder.podTemplateSpec(name, tag, deployStack)
	// env := envVarObject(namespace, name)
	// envFrom := envVarFrom()
	// appsName := builder.Instance.Spec.Apps
	// if apps, ok := appsName[name]; ok {

	// }

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
	volumeCmForData, _, _ := unstructured.NestedString(deployStack.Object, "spec", "volumeCmForData")
	volumes := []corev1.Volume{
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
	volumeMounts := []corev1.VolumeMount{
		{Name: fmt.Sprintf("%s-%s", name, configSuffix), MountPath: volumeCmForData},
	}
	lifecycle := corev1.Lifecycle{
		PreStop: &corev1.LifecycleHandler{Exec: &corev1.ExecAction{
			Command: []string{"/bin/sh", "-c", "sleep 20"},
		}}}

	probeReadyTcpPort := builder.Instance.Spec.ProbeReadyTcpPort
	// probeReadyHttpPort := builder.Instance.Spec.ProbeReadyHttpPort
	livenessProbe := corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				// Port: intstr.IntOrString{Type: intstr.Int, IntVal: probeTcpPort},
				Port: intstr.FromInt(probeReadyTcpPort),
			},
		},
		InitialDelaySeconds: 30,
		TimeoutSeconds:      5,
	}
	readinessProbe := corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt(probeReadyTcpPort),
			},
		},
		InitialDelaySeconds: 15,
		TimeoutSeconds:      5,
	}
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
			Template: corev1.PodTemplateSpec{
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
						Ports:     ports,
						Resources: resources,
						// Env:            env,
						// EnvFrom:        envFrom,
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

	return &deployment, nil
}

// func (builder *DeploymentBuild) Update(object client.Object, name, tag string, deployStack *unstructured.Unstructured) (client.Object, error) {
// 	deploy := object.(*appsv1.Deployment)
// 	if tag == "" {
// 		tag = defaultTag
// 	}
// 	appsName := builder.Instance.Spec.Apps
// 	if apps, ok := appsName[name]; ok {
// 		//Replicas
// 		if apps.Replicas != nil {
// 			deploy.Spec.Replicas = apps.Replicas
// 		} else {
// 			deploy.Spec.Replicas = builder.Instance.Spec.Replicas
// 		}
// 		if apps.Namespace != "" {
// 			deploy.Namespace = apps.Namespace
// 		} else {
// 			deploy.Namespace = builder.Instance.Spec.Namespace
// 		}

// 	} else {
// 		deploy.Spec.Replicas = builder.Instance.Spec.Replicas
// 		deploy.Namespace = builder.Instance.Spec.Namespace
// 	}
// 	//pod template
// 	//标签字段不可变，不能更新
// 	deploy.Labels = Labels(name, builder.Instance.Spec.Namespace)
// 	// deploy.Spec.Selector = &metav1.LabelSelector{MatchLabels: LabelsSelector(name, builder.Instance.Spec.Namespace)}
// 	deploy.Spec.Template = builder.podTemplateSpec(name, tag, deployStack)
// 	return deploy, nil
// }

// func (builder *DeploymentBuild) ExecStrategy() bool {
// 	return true
// }

// func (builder *DeploymentBuild) containerPorts(name string, containerPorts []ContainerPorts) []corev1.ContainerPort {
// 	var ports []corev1.ContainerPort
// 	for _, containerPort := range containerPorts {
// 		ports = append(ports, corev1.ContainerPort{
// 			Name:          StringCombin(containerPort.Name, "-", name),
// 			ContainerPort: containerPort.Port,
// 		})
// 	}

// 	return ports
// }

// func (builder *DeploymentBuild) containerPort(name string) []corev1.ContainerPort {
// 	var ports []corev1.ContainerPort
// 	if builder.Instance.Spec.PortForGrpc != 0 {
// 		ports = []corev1.ContainerPort{{
// 			Name:          StringCombin("grpc", "-", name),
// 			ContainerPort: builder.Instance.Spec.PortForGrpc,
// 		},
// 		}
// 	}
// 	if builder.Instance.Spec.PortForHttp != 0 {
// 		ports = append(ports, corev1.ContainerPort{
// 			Name:          StringCombin("http", "-", name),
// 			ContainerPort: builder.Instance.Spec.PortForHttp,
// 		})
// 	}

// 	return ports
// }

// func (builder *DeploymentBuild) podTemplateSpec(name, tag string, deployStack *unstructured.Unstructured) corev1.PodTemplateSpec {
// 	var (
// 		namespace       string
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

// 	namespace = builder.Instance.Spec.Namespace
// 	env := envVarObject(namespace, name)
// 	envFrom := envVarFrom()
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
// 		PodAntiAffinity: &corev1.PodAntiAffinity{
// 			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
// 				{
// 					Weight: 100,
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
// 	// var objMap map[string]interface{}
// 	// var deployment appsv1.Deployment
// 	// crdUnstructured, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(&deployment)
// 	// fmt.Println(crdUnstructured)
// 	// _ = runtime.DefaultUnstructuredConverter.FromUnstructured(deployStack.Object, &objMap)
// 	// fmt.Printf("Type:%T,objMap:%v", objMap["spec"], objMap) .(map[string]interface{})

// 	// if spec, ok := deployStack.Object["spec"]; ok {
// 	// 	if appList,ok :=
// 	// 	if volumeCmForConf, ok := spec.(map[string]interface{})["volumeCmForConf"]; ok {
// 	// 		fmt.Println(volumeCmForConf)
// 	// 	}
// 	// }

// 	volumeCmForData, _, _ := unstructured.NestedString(deployStack.Object, "spec", "volumeCmForData")
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
// 		{Name: fmt.Sprintf("%s-%s", name, configSuffix), MountPath: volumeCmForData},
// 	}
// 	lifecycle := corev1.Lifecycle{
// 		PreStop: &corev1.LifecycleHandler{Exec: &corev1.ExecAction{
// 			Command: []string{"/bin/sh", "-c", "sleep 20"},
// 		}}}
// 	var (
// 		livenessProbe  corev1.Probe
// 		readinessProbe corev1.Probe
// 	)
// 	probeReadyTcpPort := builder.Instance.Spec.ProbeReadyTcpPort
// 	probeReadyHttpPort := builder.Instance.Spec.ProbeReadyHttpPort

// 	livenessProbe = corev1.Probe{
// 		ProbeHandler: corev1.ProbeHandler{
// 			TCPSocket: &corev1.TCPSocketAction{
// 				// Port: intstr.IntOrString{Type: intstr.Int, IntVal: probeTcpPort},
// 				Port: intstr.FromInt(probeReadyTcpPort),
// 			},
// 		},
// 		InitialDelaySeconds: 30,
// 		TimeoutSeconds:      5,
// 	}
// 	readinessProbe = corev1.Probe{
// 		ProbeHandler: corev1.ProbeHandler{
// 			TCPSocket: &corev1.TCPSocketAction{
// 				Port: intstr.FromInt(probeReadyTcpPort),
// 			},
// 		},
// 		InitialDelaySeconds: 15,
// 		TimeoutSeconds:      5,
// 	}
// 	if builder.Instance.Spec.ProbeReadyForHttp {
// 		livenessProbe = corev1.Probe{
// 			ProbeHandler: corev1.ProbeHandler{
// 				HTTPGet: &corev1.HTTPGetAction{
// 					Path: "/ops/alive",
// 					Port: intstr.FromInt(probeReadyHttpPort),
// 				},
// 			},
// 			InitialDelaySeconds: 30,
// 			TimeoutSeconds:      5,
// 		}
// 		readinessProbe = corev1.Probe{
// 			ProbeHandler: corev1.ProbeHandler{
// 				HTTPGet: &corev1.HTTPGetAction{
// 					Path: "/ops/alive",
// 					Port: intstr.FromInt(probeReadyHttpPort),
// 				},
// 			},
// 			InitialDelaySeconds: 15,
// 			TimeoutSeconds:      5,
// 		}
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
// 	// image = "registry.cn-hangzhou.aliyuncs.com/unipal/"
// 	image = fmt.Sprintf("registry.cn-hangzhou.aliyuncs.com/unipal/%s_%s:%s", namespace, name, tag)

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
// 		// resources = defaultResources
// 		//
// 	}

// 	podTemplateSpec := corev1.PodTemplateSpec{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Annotations: map[string]string{},
// 			Labels:      LabelsSelector(name, namespace),
// 		},
// 		Spec: corev1.PodSpec{
// 			NodeSelector: map[string]string{},
// 			Affinity:     &affinity,
// 			Containers: []corev1.Container{{
// 				Name:            name,
// 				Image:           image,
// 				ImagePullPolicy: imagePullPolicy,
// 				// Command:         command,
// 				// Args:            args,
// 				Ports:          ports,
// 				Resources:      resources,
// 				Env:            env,
// 				EnvFrom:        envFrom,
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

// func envVarObject(namespace, name string) []corev1.EnvVar {
// 	env := []corev1.EnvVar{
// 		{Name: "CONFIG_ENV", Value: namespace},
// 		{Name: "MY_SERVICE_NAME", Value: name},
// 	}
// 	return env
// }
// func envVarFrom() []corev1.EnvFromSource {
// 	envFrom := []corev1.EnvFromSource{
// 		{
// 			ConfigMapRef: &corev1.ConfigMapEnvSource{
// 				LocalObjectReference: corev1.LocalObjectReference{
// 					Name: "global-config",
// 				}},
// 		},
// 		{
// 			SecretRef: &corev1.SecretEnvSource{
// 				LocalObjectReference: corev1.LocalObjectReference{
// 					Name: "global-secret"}},
// 		},
// 	}
// 	return envFrom
// }

// 字符串切割
func stringsSplit(name string) (request string, limit string) {
	trimmed := strings.TrimSpace(name)
	str := strings.Split(trimmed, "-")
	if len(str) == 2 {
		request = str[0]
		limit = str[1]
	}
	return request, limit
}
