package resource

import (
	"context"
	"fmt"
	"strings"

	apiv1 "github.com/tiamxu/k8s-operator/deploy-operator/api/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	portForGrpcDefault int32 = 5010
	portForHttpDefault int32 = 8800
)

type DeployStackBuild struct {
	Instance *apiv1.DeployStack
	Scheme   *runtime.Scheme
}
type ContainerPorts = apiv1.DefaultPorts
type ServicePorts = apiv1.DefaultPorts

type ResourceBuilder interface {
	Build(name, tag string, deployStack *unstructured.Unstructured) (client.Object, error)
	// Update(object client.Object, name, tag string, deployStack *unstructured.Unstructured) (client.Object, error)
	// ExecStrategy() bool
	GetObjectKind() (client.Object, error)
}
type labels map[string]string

// DeployStackBuild 上的方法ResourceBuilds，返回接口ResourceBuilder 类型
func (builder *DeployStackBuild) ResourceBuilds() []ResourceBuilder {
	builders := []ResourceBuilder{
		builder.Deployment(),
		// builder.Service(),
		// builder.ConfigMap(),
		// builder.Secret(),
		// builder.Ingress(),
	}
	return builders
}

func GetUnstructObject(ctx context.Context) (*unstructured.Unstructured, error) {
	deployStack := &unstructured.Unstructured{}
	deployStack.SetGroupVersionKind(schema.GroupVersionKind{Group: "gopron.online", Kind: "DeployStack", Version: "v1"})
	var client client.Client
	var namespaceName types.NamespacedName
	if err := client.Get(ctx, namespaceName, deployStack); err != nil {
		return deployStack, err
	}
	// deployStackSpec := deployStack.Object["spec"]
	return deployStack, nil
}

func int64Ptr(i int64) *int64 { return &i }
func intFromPtr(i int64) *int32 {
	m := int32(i)
	return &m
}

func int32Ptr(i int32) *int32 { return &i }

func Labels(name, env string) labels {
	return labels{
		"app":                    name,
		"env":                    env,
		"app.kubernetes.io/name": "deploystack",
	}
}

func LabelsSelector(name, env string) labels {
	return labels{
		"app":     name,
		"version": env,
		// "env":     env,
	}
}

// string combination字符串组合
func StringCombin(prefix, modifier, suffix string) string {
	return fmt.Sprintf("%s%s%s", prefix, modifier, suffix)
}

// 字符串切割
func StringsSplit(name string) (request string, limit string) {
	trimmed := strings.TrimSpace(name)
	str := strings.Split(trimmed, "-")
	if len(str) == 2 {
		request = str[0]
		limit = str[1]
	}
	return request, limit
}
