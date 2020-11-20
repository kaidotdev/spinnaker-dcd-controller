package v1

import (
	"encoding/json"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SpinnakerCanaryConfigResource defines the resource of Spinnaker
type SpinnakerCanaryConfigResource struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// CanaryConfigConditionType defines codition type
type CanaryConfigConditionType string

const (
	// CanaryConfigCreationComplete means creation has finished
	CanaryConfigCreationComplete CanaryConfigConditionType = "CreationComplete"
	// CanaryConfigDeletionComplete means deletion has finished
	CanaryConfigDeletionComplete CanaryConfigConditionType = "DeletionComplete"
)

// CanaryConfigCondition defines condition struct
type CanaryConfigCondition struct {
	Type   CanaryConfigConditionType `json:"type"`
	Status string                    `json:"status"`
}

// CanaryConfigStatus defines the observed state of CanaryConfig
type CanaryConfigStatus struct {
	SpinnakerResource SpinnakerCanaryConfigResource `json:"spinnakerResource,omitempty"`
	Conditions        []CanaryConfigCondition       `json:"conditions,omitempty"`
	Hash              string                        `json:"hash,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="SPINNAKER-CANARY-CONFIG-NAME",type=string,JSONPath=`.status.spinnakerResource.name`
// +kubebuilder:printcolumn:name="SPINNAKER-CANARY-CONFIG-ID",type=string,JSONPath=`.status.spinnakerResource.id`

// CanaryConfig is the schema for Spinnaker CanaryConfig
type CanaryConfig struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`

	Spec   json.RawMessage    `json:"spec,omitempty"`
	Status CanaryConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CanaryConfigList contains a list of CanaryConfig
type CanaryConfigList struct {
	metaV1.TypeMeta `json:",inline"`
	metaV1.ListMeta `json:"metadata,omitempty"`
	Items           []CanaryConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CanaryConfig{}, &CanaryConfigList{})
}
