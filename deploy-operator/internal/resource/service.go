package resource

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ServiceBuild struct {
	*DeployStackBuild
}

func (builder *DeployStackBuild) Service() *ServiceBuild {

	return &ServiceBuild{builder}
}
func (builder *ServiceBuild) ExecStrategy() bool {
	return true
}

func (builder *ServiceBuild) GetObjectKind() (client.Object, error) {
	return &corev1.Service{}, nil
}

func (builder *ServiceBuild) Build(name, tag string) (client.Object, error) {
	//container port
	var ports []corev1.ServicePort
	defaultPorts := []corev1.ServicePort{{
		Name: StringCombin("grpc", "-", name),
		Port: portForGrpcDefault,
	}}
	namespace := builder.Instance.Spec.Namespace

	if builder.Instance.Spec.Ports != nil {
		ports = builder.servicePorts(name, builder.Instance.Spec.Ports)
	} else {
		if builder.Instance.Spec.PortForGrpc != 0 {
			ports = []corev1.ServicePort{{
				Name: StringCombin("grpc", "-", name),
				Port: builder.Instance.Spec.PortForGrpc,
			}}
		} else {
			ports = defaultPorts
		}
	}
	service := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind: "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    Labels(name, namespace),
		},
		Spec: corev1.ServiceSpec{
			Selector: LabelsSelector(name, namespace),
			Ports:    ports,
			Type:     builder.Instance.Spec.Service.Type,
		},
	}
	return &service, nil
}

func (builder *ServiceBuild) servicePorts(name string, servicePorts []ServicePorts) []corev1.ServicePort {
	var ports []corev1.ServicePort
	// servicePorts := builder.Instance.Spec.Ports
	for _, svcPort := range servicePorts {
		ports = append(ports, corev1.ServicePort{
			Name: StringCombin(svcPort.Name, "-", name),
			Port: svcPort.Port,
		})
	}
	return ports
}

func (builder *ServiceBuild) Update(object client.Object, name, tag string) (client.Object, error) {
	service := object.(*corev1.Service)
	service.Labels = Labels(name, builder.Instance.Spec.Namespace)
	service.Spec.Type = builder.Instance.Spec.Service.Type
	if builder.Instance.Spec.Ports != nil {
		service.Spec.Ports = builder.servicePorts(name, builder.Instance.Spec.Ports)
	} else {
		if builder.Instance.Spec.PortForGrpc != 0 {
			service.Spec.Ports = []corev1.ServicePort{{
				Name: StringCombin("grpc", "-", name),
				Port: builder.Instance.Spec.PortForGrpc,
			}}
		}
	}

	appsName := builder.Instance.Spec.Apps
	if apps, ok := appsName[name]; ok {
		if apps.Ports != nil {
			service.Spec.Ports = append(service.Spec.Ports, builder.servicePorts(name, apps.Ports)...)
		}
	}

	return service, nil
}
