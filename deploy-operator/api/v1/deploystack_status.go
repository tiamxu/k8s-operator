package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Status int

const (
	Pending Status = iota
	Running
	Succeeded
	Failed
	Unknown
	Terminating
)

func (s Status) String() string {
	switch s {
	case Pending:
		return "Pending"
	case Running:
		return "Running"
	case Succeeded:
		return "Succeeded"
	case Failed:
		return "Failed"
	default:
		return "Unknown"

	}
}

type DeployStackCondition struct {
	Type               string                 `json:"type"`
	Status             corev1.ConditionStatus `json:"status"`
	LastTransitionTime metav1.Time            `json:"lastTransitionTime,omitempty"`
	Reason             string                 `json:"reason,omitempty"`
	Message            string                 `json:"message,omitempty"`
}

// DeployStackStatus defines the observed state of DeployStack
type DeployStackStatus struct {
	Status     string                 `json:"status,omitempty"`
	Conditions []DeployStackCondition `json:"conditions,omitempty"`
}
