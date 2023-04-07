/*
Copyright 2023.
*/

package v1

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DeployStackSpec defines the desired state of DeployStack
type DeployStackSpec struct {
	Apps            map[string]AppsName          `json:"apps,omitempty"`
	AppsList        map[string]string            `json:"appsList,omitempty"`
	Replicas        *int32                       `json:"replicas,omitempty"`
	ImageRegistry   string                       `json:"imageRegistry,omitempty"`
	ImagePullPolicy string                       `json:"imagePullPolicy,omitempty"`
	RegistrySecrets string                       `json:"registrySecrets,omitempty"`
	Namespace       string                       `json:"namespace,omitempty"`
	Service         DeployStackServiceSpec       `json:"service,omitempty"`
	Resources       *corev1.ResourceRequirements `json:"resources,omitempty"`
	Affinity        *corev1.Affinity             `json:"affinity,omitempty"`
	Toleration      *corev1.Toleration           `json:"toleration,omitempty"`
	Default         map[string]string            `json:"default,omitempty"`
	Ports           []DefaultPorts               `json:"ports,omitempty"`
	Configs         map[string]string            `json:"configs,omitempty"`
	Secret          map[string]string            `json:"secret,omitempty"`
	Ingress         []IngressSpec                `json:"ingress,omitempty"`

	// Override        DeployStackOverrideSpec      `json:"override,omitempty"`

}

type IngressSpec struct {
	Name       string `json:"name,omitempty"`
	Https      bool   `json:"https,omitempty"`
	Host       string `json:"host,omitempty"`
	Port       int32  `json:"port,omitempty"`
	SecretName string `json:"secretName,omitempty"`
}
type AppsName struct {
	Name            string         `json:"name,omitempty"`
	Replicas        *int32         `json:"replicas,omitempty"`
	Namespace       string         `json:"namespace,omitempty"`
	ImageRegistry   string         `json:"imageRegistry,omitempty"`
	ImagePullPolicy string         `json:"imagePullPolicy,omitempty"`
	RegistrySecrets string         `json:"registrySecrets,omitempty"`
	Ports           []DefaultPorts `json:"ports,omitempty"`
}
type DeployStackServiceSpec struct {
	Type  corev1.ServiceType  `json:"type,omitempty"`
	Ports *corev1.ServicePort `json:"ports,omitempty"`
}
type DefaultPorts struct {
	Name string `json:"name,omitempty"`
	Port int32  `json:"port,omitempty"`
}
type DeployStackOverrideSpec struct {
	Deployment *Deployment `json:"depoyment,omitempty"`
	// Service    *Service    `json:"service,omitempty"`
}

// ObjectMeta
type Service struct {
	Spec *corev1.ServiceSpec `json:"spec,omitempty"`
}
type Deployment struct {
	Spec *DeploymentSpec `json:"spec,omitempty"`
}

// DeploymentSpec
type DeploymentSpec struct {
	Replicas *int32                `json:"replicas,omitempty"`
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
	Template *PodTemplateSpec      `json:"template,omitempty"`
}
type PodTemplateSpec struct {
	Spec *corev1.PodSpec `json:"spec,omitempty"`
}

// DeployStackStatus defines the observed state of DeployStack
type DeployStackStatus struct {
	Status *appsv1.DeploymentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DeployStack is the Schema for the deploystacks API
type DeployStack struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeployStackSpec   `json:"spec,omitempty"`
	Status DeployStackStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DeployStackList contains a list of DeployStack
type DeployStackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeployStack `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DeployStack{}, &DeployStackList{})
}
