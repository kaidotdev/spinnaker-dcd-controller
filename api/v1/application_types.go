package v1

import (
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// SpinnakerApplicationResource defines the resource of Spinnaker
type SpinnakerApplicationResource struct {
	ApplicationName string `json:"applicationName,omitempty"`
}

// ApplicationConditionType defines codition type
type ApplicationConditionType string

const (
	// ApplicationCreationComplete means creation has finished
	ApplicationCreationComplete ApplicationConditionType = "CreationComplete"
	// ApplicationDeletionComplete means deletion has finished
	ApplicationDeletionComplete ApplicationConditionType = "DeletionComplete"
)

// ApplicationCondition defines condition struct
type ApplicationCondition struct {
	Type   ApplicationConditionType `json:"type"`
	Status string                   `json:"status"`
}

// ApplicationStatus defines the observed state of Application
type ApplicationStatus struct {
	SpinnakerResource SpinnakerApplicationResource `json:"spinnakerResource,omitempty"`
	Conditions        []ApplicationCondition       `json:"conditions,omitempty"`
	Hash              string                       `json:"hash,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="SPINNAKER-APPLICATION-NAME",type=string,JSONPath=`.status.spinnakerResource.applicationName`

// Application is the schema for Spinnaker Application
type Application struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:pruning:PreserveUnknownFields
	Spec   runtime.RawExtension `json:"spec,omitempty"`
	Status ApplicationStatus    `json:"status,omitempty"`
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
