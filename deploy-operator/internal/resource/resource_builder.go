package resource

import (
	"fmt"

	apiv1 "github.com/tiamxu/k8s-operator/deploy-operator/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeployStackBuild struct {
	Instance *apiv1.DeployStack
	Scheme   *runtime.Scheme
}
type ContainerPorts = apiv1.DefaultPorts
type ServicePorts = apiv1.DefaultPorts

type ResourceBuilder interface {
	Build(name, tag string) (client.Object, error)
	Update(object client.Object, name, tag string) (client.Object, error)
	ExecStrategy() bool
	GetObjectKind() (client.Object, error)
}
type labels map[string]string

// DeployStackBuild 上的方法ResourceBuilds，返回接口ResourceBuilder 类型
func (builder *DeployStackBuild) ResourceBuilds() []ResourceBuilder {
	builders := []ResourceBuilder{
		builder.Deployment(),
		builder.Service(),
		builder.ConfigMap(),
		builder.Secret(),
		builder.Ingress(),
	}
	return builders
}
func int64Ptr(i int64) *int64 { return &i }

// func int32Ptr(i int32) *int32 { return &i }

func Labels(name string) labels {
	return labels{
		"app":     name,
		"version": "prod",
	}
}

// string combination字符串组合
func StringCombin(prefix, modifier, suffix string) string {
	return fmt.Sprintf("%s%s%s", prefix, modifier, suffix)
}
