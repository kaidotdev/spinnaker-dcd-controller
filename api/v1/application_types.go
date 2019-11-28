package v1

import (
	"encoding/json"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SpinnakerApplicationResource defines the resource of Spinnaker
type SpinnakerApplicationResource struct {
	ApplicationName string `json:"applicationName,omitempty"`
}

// ApplicationStatus defines the observed state of Application
type ApplicationStatus struct {
	SpinnakerResource SpinnakerApplicationResource `json:"spinnakerResource,omitempty"`
	Phase             string                       `json:"phase,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="PHASE",type=string,JSONPath=`.status.phase`

// Application is the schema for Spinnaker Application
type Application struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`

	Spec   json.RawMessage   `json:"spec,omitempty"`
	Status ApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApplicationList contains a list of Application
type ApplicationList struct {
	metaV1.TypeMeta `json:",inline"`
	metaV1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}
