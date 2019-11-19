package v1

import metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// ApplicationSpec defines the desired state of Application
type ApplicationSpec struct {
	Email       string      `json:"email,omitempty"`
	DataSources DataSources `json:"dataSources,omitempty"`
}

// DataSources defines dataSources
type DataSources struct {
	Disabled []string `json:"disabled,omitempty"`
	Enabled  []string `json:"enabled,omitempty"`
}

// ApplicationStatus defines the observed state of Application
type ApplicationStatus struct{}

// +kubebuilder:object:root=true

// Application is the schema for Spinnaker Application
type Application struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec,omitempty"`
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
