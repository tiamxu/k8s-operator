package resource

import (
	"encoding/base64"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// 定义全局secret配置
var (
	defaultSecretName string = "global-secret"
	defaultEnvData           = map[string]string{
		"CONFIG_DB_USERNAME":    "cm9vdAo=",
		"CONFIG_DB_PASSWORD":    "MTIzNDU2Cg==",
		"CONFIG_REDIS_PASSWORD": "MTIzNDU2Cg==",
	}
)

type SecretBuild struct {
	*DeployStackBuild
}

func (builder *DeployStackBuild) Secret() *SecretBuild {

	return &SecretBuild{builder}
}
func (builder *SecretBuild) GetObjectKind() (client.Object, error) {
	return &corev1.Secret{}, nil
}

// whether to execute this resource.
func (builder *SecretBuild) ExecStrategy() bool {
	return false
}
func (builder *SecretBuild) Build(name, tag string) (client.Object, error) {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        defaultSecretName,
			Namespace:   builder.Instance.Spec.Namespace,
			Labels:      Labels(name, builder.Instance.Spec.Namespace),
			Annotations: map[string]string{},
		},
		Data: builder.convertString(),
		Type: corev1.SecretTypeOpaque,
	}
	return &secret, nil
}

func (builder *SecretBuild) Update(object client.Object, name, tag string) (client.Object, error) {
	secret := object.(*corev1.Secret)
	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}
	secret.Data = builder.convertString()
	return secret, nil
}

// convert secret base64 string to byte.
func (builder *SecretBuild) convertString() map[string][]byte {
	var (
		data      = make(map[string][]byte)
		secretObj = make(map[string]string)
	)
	if builder.Instance.Spec.Secret != nil {
		secretObj = builder.Instance.Spec.Secret
		for key, value := range secretObj {
			defaultEnvData[key] = value
		}
	}
	secretObj = defaultEnvData

	for key, value := range secretObj {
		//base64 Decode
		value, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			fmt.Printf("secret base64 Decode error:%s", err)
		}
		data[key] = value
	}
	return data
}

// secret type:
// 1、docker-registry: type: kubernetes.io/dockerconfigjson
// 2、type: Opaque

// const (
// 	registrySecret = "regcred-vpc"
// 	registryKey    = ".dockerconfigjson"
// 	dockerRegistryConfig = `
// 	{
// 		"auths": {
// 			"registry-vpc.cn-hangzhou.aliyuncs.com": {
// 				"auth": "eGlhb21lbmdjb3JwOld0elV3aGttOUtDb3hNc1EzR1JU"
// 			}
// 		}

// 	}
// 	`
// )
