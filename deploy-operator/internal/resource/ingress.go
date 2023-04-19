package resource

import (
	"fmt"
	"strconv"
	"strings"

	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultSSL = "gopron.online"
)

var (
	defaultPathType  = v1.PathTypeImplementationSpecific
	ingressClassName = "nginx"
)

type IngressBuild struct {
	*DeployStackBuild
}

func (builder *DeployStackBuild) Ingress() *IngressBuild {

	return &IngressBuild{builder}
}
func (builder *IngressBuild) ExecStrategy() bool {
	return false
}

func (builder *IngressBuild) GetObjectKind() (client.Object, error) {

	return &v1.Ingress{}, nil

}
func (builder *IngressBuild) Build(name, tag string) (client.Object, error) {

	var (
		rules       []v1.IngressRule  = builder.ingressRules(name)
		tls         []v1.IngressTLS   = builder.tlsStrategy(name)
		annotations map[string]string = builder.getAnnotations(name)
	)

	ingress := v1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        StringCombin(name, "-", "ingress"),
			Namespace:   builder.Instance.Spec.Namespace,
			Annotations: annotations,
		},
		Spec: v1.IngressSpec{
			IngressClassName: &ingressClassName,
			TLS:              tls,
			Rules:            rules,
		},
	}
	return &ingress, nil
}

func (builder *IngressBuild) stringsSplit(name string) (string, int32) {
	var (
		svcName string
		port32  int32
	)
	trimmed := strings.TrimSpace(name)
	str := strings.Fields(trimmed)
	if len(str) >= 2 {
		svcName = str[0]
		port, err := strconv.Atoi(str[1])
		if err != nil {
			fmt.Println("Invalid port")
		}
		port32 = int32(port)
	} else if len(str) == 1 {
		svcName = str[0]
		port32 = 80
	}

	return svcName, port32
}
func (builder *IngressBuild) tlsStrategy(name string) []v1.IngressTLS {
	var (
		hosts []string
		tls   []v1.IngressTLS
	)
	if builder.Instance.Spec.Ingress != nil {
		for _, ingress := range builder.Instance.Spec.Ingress {
			if ingress.Name == name && ingress.Https {
				hosts = append(hosts, ingress.Host)
			}
		}
		if hosts != nil {
			tls = append(tls, v1.IngressTLS{
				Hosts:      hosts,
				SecretName: defaultSSL,
			})
		} else {
			tls = []v1.IngressTLS{}
		}

	}

	return tls
}

func (builder *IngressBuild) ingressRules(name string) []v1.IngressRule {
	var (
		rules []v1.IngressRule
	)
	if builder.Instance.Spec.Ingress != nil {
		for _, ingress := range builder.Instance.Spec.Ingress {
			if name == ingress.Name {
				var paths []v1.HTTPIngressPath
				if ingress.Match != nil {
					pathType := v1.PathTypeImplementationSpecific
					for path, service := range ingress.Match {
						svcName, svcPort := builder.stringsSplit(service)
						paths = append(paths, builder.httpIngressPath(path, pathType, svcName, svcPort))
					}

				}
				if ingress.Prefix != nil {
					pathType := v1.PathTypePrefix
					for path, service := range ingress.Prefix {
						svcName, svcPort := builder.stringsSplit(service)
						paths = append(paths, builder.httpIngressPath(path, pathType, svcName, svcPort))
					}

				}
				if ingress.Exact != nil {
					pathType := v1.PathTypeExact
					for path, service := range ingress.Exact {
						svcName, svcPort := builder.stringsSplit(service)
						paths = append(paths, builder.httpIngressPath(path, pathType, svcName, svcPort))
					}

				}

				if ingress.Match == nil && ingress.Prefix == nil && ingress.Exact == nil {
					return rules
				}
				rules = append(rules, v1.IngressRule{
					Host: ingress.Host,
					IngressRuleValue: v1.IngressRuleValue{
						HTTP: &v1.HTTPIngressRuleValue{
							Paths: paths,
						},
					},
				})
			}

		}
	} else {
		rules = []v1.IngressRule{}
	}
	return rules
}

func (builder *IngressBuild) httpIngressPath(path string, pathType v1.PathType, svcName string, svcPort int32) v1.HTTPIngressPath {
	return v1.HTTPIngressPath{
		Path:     path,
		PathType: &pathType,
		Backend: v1.IngressBackend{
			Service: &v1.IngressServiceBackend{
				Name: svcName,
				Port: v1.ServiceBackendPort{
					Number: svcPort,
				},
			},
		},
	}
}
func (builder *IngressBuild) getAnnotations(name string) map[string]string {
	var annotations map[string]string
	if builder.Instance.Spec.Ingress != nil {
		for _, ingress := range builder.Instance.Spec.Ingress {
			if name == ingress.Name && ingress.Annotations != nil {
				annotations = ingress.Annotations
				break
			} else {
				annotations = map[string]string{}
			}
		}
	}
	return annotations
}
