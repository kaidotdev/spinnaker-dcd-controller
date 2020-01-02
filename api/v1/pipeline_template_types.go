package v1

import (
	"encoding/json"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SpinnakerPipelineTemplateResource defines the resource of Spinnaker
type SpinnakerPipelineTemplateResource struct {
	ID string `json:"id,omitempty"`
}

// PipelineTemplateStatus defines the observed state of PipelineTemplate
type PipelineTemplateStatus struct {
	SpinnakerResource SpinnakerPipelineTemplateResource `json:"spinnakerResource,omitempty"`
	Hash              string                            `json:"hash,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="SPINNAKER-PIPELINE-TEMPLATE-ID",type=string,JSONPath=`.status.spinnakerResource.id`

// PipelineTemplate is the schema for Spinnaker PipelineTemplate
type PipelineTemplate struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`

	Spec   json.RawMessage        `json:"spec,omitempty"`
	Status PipelineTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PipelineTemplateList contains a list of PipelineTemplate
type PipelineTemplateList struct {
	metaV1.TypeMeta `json:",inline"`
	metaV1.ListMeta `json:"metadata,omitempty"`
	Items           []PipelineTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PipelineTemplate{}, &PipelineTemplateList{})
}
