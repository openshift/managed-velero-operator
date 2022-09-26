package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VeleroInstallSpec defines the desired state of Velero
type VeleroInstallSpec struct{}

// VeleroInstallStatus defines the observed state of Velero
type VeleroInstallStatus struct {
	// StorageBucket contains details of the storage bucket for backups
	// +optional
	StorageBucket StorageBucket `json:"storageBucket,omitempty"`
}

//+kubebuilder:object:root=true

// VeleroInstall is the Schema for the veleroinstalls API
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

//+kubebuilder:object:root=true

// VeleroInstallList contains a list of VeleroInstall
type VeleroInstallList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VeleroInstall `json:"items"`
}

// StorageBucket contains details of the storage bucket for backups
type StorageBucket struct {
	// Name is the name of the storage bucket created to store Velero backup details
	// +kubebuilder:validation:MaxLength=63
	Name string `json:"name,omitempty"`

	// Provisioned is true once the bucket has been initially provisioned.
	Provisioned bool `json:"provisioned"`

	// LastSyncTimestamp is the time that the bucket policy was last synced.
	LastSyncTimestamp *metav1.Time `json:"lastSyncTimestamp,omitempty"`
}

func init() {
	SchemeBuilder.Register(&VeleroInstall{}, &VeleroInstallList{})
}
