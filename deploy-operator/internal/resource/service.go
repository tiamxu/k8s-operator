package resource

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	service corev1.Service
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

	service = corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind: "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: builder.Instance.Spec.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: Labels(name),
			Ports:    builder.servicePorts(name, builder.Instance.Spec.Ports),
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
	appsName := builder.Instance.Spec.Apps
	apps, ok := appsName[name]
	if ok {
		service.Spec.Ports = builder.servicePorts(name, apps.Ports)

	} else {
		service.Spec.Ports = builder.servicePorts(name, builder.Instance.Spec.Ports)
		service.Spec.Type = builder.Instance.Spec.Service.Type

	}

	return service, nil
}
