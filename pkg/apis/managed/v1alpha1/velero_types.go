package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VeleroSpec defines the desired state of Velero
// +k8s:openapi-gen=true
type VeleroSpec struct{}

// VeleroStatus defines the observed state of Velero
// +k8s:openapi-gen=true
type VeleroStatus struct {
	// S3Bucket contains details of the S3 storage bucket for backups
	// +optional
	S3Bucket S3Bucket `json:"s3Bucket,omitempty"`
}

// S3Bucket defines the observed state of Velero
// +k8s:openapi-gen=true
type S3Bucket struct {
	// Name is the name of the S3 bucket created to store Velero backup details
	// +kubebuilder:validation:MaxLength=63
	Name string `json:"name,omitempty"`

	// Provisioned is true once the bucket has been initially provisioned.
	Provisioned bool `json:"provisioned"`

	// LastSyncTimestamp is the time that the bucket policy was last synced.
	LastSyncTimestamp *metav1.Time `json:"lastSyncTimestamp,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Velero is the Schema for the veleros API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Bucket",type="string",JSONPath=".status.s3Bucket.name",description="Name of the S3 bucket"
// +kubebuilder:printcolumn:name="Provisioned",type="boolean",JSONPath=".status.s3Bucket.provisioned",description="Has the S3 bucket been successfully provisioned"
// +kubebuilder:printcolumn:name="Last Sync",type="date",JSONPath=".status.s3Bucket.lastSyncTimestamp"
type Velero struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VeleroSpec   `json:"spec,omitempty"`
	Status VeleroStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VeleroList contains a list of Velero
type VeleroList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Velero `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Velero{}, &VeleroList{})
}
