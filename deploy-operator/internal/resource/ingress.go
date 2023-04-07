package resource

import (
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

func (builder *IngressBuild) Build(name, tag string) (client.Object, error) {

	var (
		rules []v1.IngressRule = builder.ingressRules(name)
		tls   []v1.IngressTLS  = builder.tlsStrategy(name)
	)

	ingress := v1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        StringCombin(name, "-", "ingress"),
			Namespace:   builder.Instance.Spec.Namespace,
			Annotations: map[string]string{},
		},
		Spec: v1.IngressSpec{
			IngressClassName: &ingressClassName,
			TLS:              tls,
			Rules:            rules,
		},
	}
	return &ingress, nil
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
		tls = append(tls, v1.IngressTLS{
			Hosts:      hosts,
			SecretName: defaultSSL,
		})
	}

	return tls
}

func (builder *IngressBuild) ingressRules(name string) []v1.IngressRule {
	var rules []v1.IngressRule
	if builder.Instance.Spec.Ingress != nil {
		for _, ingress := range builder.Instance.Spec.Ingress {
			if ingress.Name == name {
				rules = append(rules, v1.IngressRule{
					Host: ingress.Host,
					IngressRuleValue: v1.IngressRuleValue{
						HTTP: &v1.HTTPIngressRuleValue{
							Paths: []v1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &defaultPathType,
									Backend: v1.IngressBackend{
										Service: &v1.IngressServiceBackend{
											Name: ingress.Name,
											Port: v1.ServiceBackendPort{
												Number: ingress.Port,
											},
										},
									},
								},
							},
						},
					},
				})
			}

		}
	}

	// rules = []v1.IngressRule{
	// 	{
	// 		Host: "",
	// 		IngressRuleValue: v1.IngressRuleValue{
	// 			HTTP: &v1.HTTPIngressRuleValue{
	// 				Paths: []v1.HTTPIngressPath{
	// 					{
	// 						Path:     "/",
	// 						PathType: &defaultPathType,
	// 						Backend: v1.IngressBackend{
	// 							Service: &v1.IngressServiceBackend{
	// 								Name: name,
	// 								Port: v1.ServiceBackendPort{
	// 									Name: name, Number: int32(8800),
	// 								},
	// 							},
	// 						},
	// 					},
	// 				},
	// 			},
	// 		},
	// 	},
	// }

	return rules
}

func (builder *IngressBuild) Update(object client.Object, name, tag string) error {
	return nil
}
