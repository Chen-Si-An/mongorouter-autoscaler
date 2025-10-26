package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TargetRef defines the target resource for autoscaling
type TargetRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// ScaleBounds defines the min/max replica limits
type ScaleBounds struct {
	MinReplicas int `json:"minReplicas"`
	MaxReplicas int `json:"maxReplicas"`
}

// Policy defines scaling policy parameters
type Policy struct {
	CpuTargetPercent int    `json:"cpuTargetPercent"`
	TolerancePercent int    `json:"tolerancePercent"`
	Window           string `json:"window"`
	Step             int    `json:"step"`
	CooldownSeconds  int    `json:"cooldownSeconds"`
}

// Prometheus defines the Prometheus endpoint
type Prometheus struct {
	URL string `json:"url"`
}

// MongoRouterAutoscalerSpec defines desired state
type MongoRouterAutoscalerSpec struct {
	TargetRef   TargetRef   `json:"targetRef"`
	ScaleBounds ScaleBounds `json:"scaleBounds"`
	Policy      Policy      `json:"policy"`
	Prometheus  Prometheus  `json:"prometheus"`
}

// MongoRouterAutoscalerStatus defines observed state
type MongoRouterAutoscalerStatus struct {
	LastScaleTime       metav1.Time `json:"lastScaleTime,omitempty"`
	LastObservedCPU     string      `json:"lastObservedCPU,omitempty"`
	LastDesiredReplicas int32       `json:"lastDesiredReplicas,omitempty"`
}

// MongoRouterAutoscaler is the Schema for the CRD
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type MongoRouterAutoscaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MongoRouterAutoscalerSpec   `json:"spec,omitempty"`
	Status MongoRouterAutoscalerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type MongoRouterAutoscalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MongoRouterAutoscaler `json:"items"`
}

// Register the MongoRouterAutoscaler types with the scheme.
func init() {
	SchemeBuilder.Register(&MongoRouterAutoscaler{}, &MongoRouterAutoscalerList{})
}
