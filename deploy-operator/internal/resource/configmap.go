package resource

import (
	// 	appsv1 "k8s.io/api/apps/v1"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	// 	"sigs.k8s.io/controller-runtime/pkg/client"
	// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// "k8s.io/apimachinery/pkg/runtime"
)

const (
	defaultConfigMapName string = "global-config"

	ConfigMapName = "config.yaml"
	defaultConf   = `
	log_level: info
    srv:
      network: tcp
      listen_address: :5010
      with_proxy: true
	`
)

type ConfigMapBuild struct {
	*DeployStackBuild
}

func (builder *DeployStackBuild) ConfigMap() *ConfigMapBuild {

	return &ConfigMapBuild{builder}
}

func (builder *ConfigMapBuild) ExecStrategy() bool {
	return false
}

func (builder *ConfigMapBuild) GetObjectKind() (client.Object, error) {
	return &corev1.ConfigMap{}, nil
}

func (builder *ConfigMapBuild) Build(name, tag string, deployStack *unstructured.Unstructured, d DeployStackBuild) (client.Object, error) {
	var (
		namespace string
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

		default:
		}
	}
	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        defaultConfigMapName,
			Namespace:   namespace,
			Labels:      Labels(name, namespace),
			Annotations: map[string]string{},
		},
		Data: builder.Instance.Spec.Configs,
	}
	return &configMap, nil
}

// func (builder *ConfigMapBuild) configData() (map[string]string, error) {
// 	var data map[string]string
// 	data = builder.Instance.Spec.Configs
// 	return data, nil
// }
