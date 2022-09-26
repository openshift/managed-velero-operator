/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VeleroInstallSpec defines the desired state of Velero
type VeleroInstallSpec struct {
}

// VeleroInstallStatus defines the observed state of VeleroInstall
type VeleroInstallStatus struct {
	// StorageBucket contains details of the storage bucket for backups
	// +optional
	StorageBucket StorageBucket `json:"storageBucket,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

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

// VeleroInstall is the Schema for the veleroinstalls API
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

func init() {
	SchemeBuilder.Register(&VeleroInstall{}, &VeleroInstallList{})
}
