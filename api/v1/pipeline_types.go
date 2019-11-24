package v1

import (
	"encoding/json"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PipelineStatus defines the observed state of Pipeline
type PipelineStatus struct {
	ApplicationName string `json:"applicationName,omitempty"`
	ID              string `json:"id,omitempty"`
}

// +kubebuilder:object:root=true

// Pipeline is the schema for Spinnaker Pipeline
type Pipeline struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`

	Spec   json.RawMessage `json:"spec,omitempty"`
	Status PipelineStatus  `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PipelineList contains a list of Pipeline
type PipelineList struct {
	metaV1.TypeMeta `json:",inline"`
	metaV1.ListMeta `json:"metadata,omitempty"`
	Items           []Pipeline `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Pipeline{}, &PipelineList{})
}
