package resource

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

func (builder *ServiceBuild) Build(name, tag string, deployStack *unstructured.Unstructured, d DeployStackBuild) (client.Object, error) {

	var (
		namespace string
		ports     []corev1.ServicePort
	)
	appsConfObj, _, err := GetAppConf(name, deployStack, d)
	if err != nil {
		return nil, err
	}
	appConf, ok := appsConfObj[name]
	if !ok {
		return nil, fmt.Errorf("Service appConf error:%v", appConf)
	}
	for key, valueConf := range appConf {
		switch key {
		case "namespaceForDefault":
			value, ok := valueConf.(string)
			if !ok {
				return nil, fmt.Errorf("%v Error", key)
			}
			namespace = value
		case "portForGrpc", "portForHttp":
			value, ok := valueConf.(int64)
			if !ok {
				return nil, fmt.Errorf("%v Error", key)
			}
			ports = append(ports, servicePorts(name, key, int32(value))...)

		default:
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
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
	return &service, nil
}

func servicePorts(name, key string, port int32) []corev1.ServicePort {
	var ports []corev1.ServicePort
	trimStr := strings.TrimSpace(key)
	str := strings.Split(trimStr, "For")
	suffix := strings.ToLower(str[1])
	ports = append(ports, corev1.ServicePort{
		Name: fmt.Sprintf("%s-%s", suffix, name),
		Port: port,
	})
	return ports
}
