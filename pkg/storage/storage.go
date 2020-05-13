package storage

import (
	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	veleroInstallCR "github.com/openshift/managed-velero-operator/pkg/apis/managed/v1alpha2"
)

//Driver interface to be satisfied by all present and future storage cloud providers
type Driver interface {
	GetPlatformType() configv1.PlatformType
	CreateStorage(logr.Logger, *veleroInstallCR.VeleroInstall) error
	StorageExists(string) (bool, error)
}
