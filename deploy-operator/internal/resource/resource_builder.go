package resource

import (
	"fmt"
	"strings"

	apiv1 "github.com/tiamxu/k8s-operator/deploy-operator/api/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeployStackBuild struct {
	Instance *apiv1.DeployStack
	Scheme   *runtime.Scheme
}

type ResourceBuilder interface {
	Build(name, tag string, deployStack *unstructured.Unstructured, d DeployStackBuild) (client.Object, error)
	// Update(object client.Object, name, tag string, deployStack *unstructured.Unstructured) (client.Object, error)
	// ExecStrategy() bool
	GetObjectKind() (client.Object, error)
}
type labels map[string]string

// DeployStackBuild 上的方法ResourceBuilds，返回接口ResourceBuilder 类型
func (builder *DeployStackBuild) ResourceBuilds() []ResourceBuilder {
	builders := []ResourceBuilder{
		builder.Deployment(),
		builder.StatefulSet(),
		builder.Service(),
		// builder.ConfigMap(),
		// builder.Secret(),
		// builder.Ingress(),
	}
	return builders
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

func GetAppConf(name string, deployStack *unstructured.Unstructured, builder DeployStackBuild) (map[string]map[string]interface{}, string, error) {
	var appConf = make(map[string]map[string]interface{})
	var confMap = make(map[string]interface{})
	var serverType string
	deployStackSpec, ok := deployStack.Object["spec"].(map[string]interface{})
	if !ok {
		return nil, "", fmt.Errorf("deployStack.Object is error")
	}
	defaultForConfig := builder.Instance.Spec.Default
	if len(defaultForConfig) == 0 || defaultForConfig == nil {
		return nil, "", fmt.Errorf("defaultForConfig error,not values or nil")
	}
	//默认配置项
	for _, key := range defaultForConfig {
		if _, ok := deployStackSpec[key]; ok {
			confMap[key] = deployStackSpec[key]
		}
	}

	appsConf := builder.Instance.Spec.AppsConf
	if len(appsConf) == 0 {
		return nil, "", fmt.Errorf("appsConf error,not values")
	}
	for appType, appValue := range appsConf {
		if appTypeConf, ok := appValue[name]; ok {
			//自定义服务配置：type：web、app、sts
			if keys, ok := deployStackSpec[appType]; ok {
				for _, key := range keys.([]interface{}) {
					// confValue = append(confValue, key.(string))
					//分类配置项
					if _, ok := deployStackSpec[key.(string)]; ok {
						confMap[key.(string)] = deployStackSpec[key.(string)]
					}
					k, value := getConfKeyValue(key.(string))
					if value != nil {
						confMap[k] = value
					}
					if strings.HasSuffix(strings.TrimSpace(key.(string)), strings.Title(appType)) {
						delete(confMap, strings.Replace(key.(string), strings.Title(appType), "Default", 1))
					}
				}
			}
			//自定义服务配置项
			for _, key := range appTypeConf {
				if _, ok := deployStackSpec[key]; ok {
					confMap[key] = deployStackSpec[key]
				}
				k, value := getConfKeyValue(key)
				if value != nil {
					confMap[k] = value
				}
				if strings.HasSuffix(strings.TrimSpace(key), strings.Title(name)) {
					delete(confMap, strings.Replace(key, strings.Title(name), strings.Title(appType), 1))
				}
				if strings.HasSuffix(strings.TrimSpace(key), strings.Title(appType)) {
					delete(confMap, strings.Replace(key, strings.Title(appType), "Default", 1))
				}
			}
			appConf[name] = confMap
			serverType = appType

		}
	}

	return appConf, serverType, nil
}
