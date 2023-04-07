package resource

import (
	"encoding/base64"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultSecretName string = "global-secret"
)

type SecretBuild struct {
	*DeployStackBuild
}

func (builder *DeployStackBuild) Secret() *SecretBuild {

	return &SecretBuild{builder}
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
			Labels:      Labels(name),
			Annotations: map[string]string{},
		},
		Data: builder.convertString(),
	}
	return &secret, nil
}

func (builder *SecretBuild) Update(object client.Object, name, tag string) error {
	secret := object.(*corev1.Secret)
	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}
	secret.Data = builder.convertString()
	return nil
}

// convert secret string to byte.
func (builder *SecretBuild) convertString() map[string][]byte {
	secretObj := builder.Instance.Spec.Secret
	var data = make(map[string][]byte)
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

var (
	secretData = map[string][]byte{
		"CONFIG_DB_USERNAME": []byte("dW5pcG"),
	}
)
