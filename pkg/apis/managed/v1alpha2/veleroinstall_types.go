package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VeleroInstallSpec defines the desired state of Velero
// +k8s:openapi-gen=true
type VeleroInstallSpec struct{}

// VeleroInstallStatus defines the observed state of Velero
// +k8s:openapi-gen=true
type VeleroInstallStatus struct {
	// StorageBucket contains details of the storage bucket for backups
	// +optional
	StorageBucket StorageBucket `json:"storageBucket,omitempty"`
}

// StorageBucket contains details of the storage bucket for backups
// +k8s:openapi-gen=true
type StorageBucket struct {
	// Name is the name of the storage bucket created to store Velero backup details
	// +kubebuilder:validation:MaxLength=63
	Name string `json:"name,omitempty"`

	// Provisioned is true once the bucket has been initially provisioned.
	Provisioned bool `json:"provisioned"`

	// LastSyncTimestamp is the time that the bucket policy was last synced.
	LastSyncTimestamp *metav1.Time `json:"lastSyncTimestamp,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VeleroInstall is the Schema for the veleroinstalls API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=veleroinstalls,scope=Namespaced
// +kubebuilder:printcolumn:name="Bucket",type="string",JSONPath=".status.storageBucket.name",description="Name of the storage bucket"
// +kubebuilder:printcolumn:name="Provisioned",type="boolean",JSONPath=".status.storageBucket.provisioned",description="Has the storage bucket been successfully provisioned"
// +kubebuilder:printcolumn:name="Last Sync",type="date",JSONPath=".status.storageBucket.lastSyncTimestamp"
type VeleroInstall struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VeleroInstallSpec   `json:"spec,omitempty"`
	Status VeleroInstallStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VeleroInstallList contains a list of VeleroInstalls
type VeleroInstallList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VeleroInstall `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VeleroInstall{}, &VeleroInstallList{})
}
