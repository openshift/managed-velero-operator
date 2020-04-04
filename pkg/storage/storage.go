package storage

import (
	"context"

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	veleroInstallCR "github.com/openshift/managed-velero-operator/pkg/apis/managed/v1alpha2"
	"github.com/openshift/managed-velero-operator/pkg/storage/gcs"
	"github.com/openshift/managed-velero-operator/pkg/storage/s3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//Driver interface to be satisfied by all present and future storage cloud providers
type Driver interface {
	CreateStorage(logr.Logger, *veleroInstallCR.VeleroInstall) error
	StorageExists(string) (bool, error)
}

//NewDriver will return a driver object
func NewDriver(cfg *configv1.InfrastructureStatus, clnt client.Client) Driver {

	ctx := context.Background()
	var driver Driver

	if cfg.PlatformStatus.Type == "AWS" {
		driver = s3.NewDriver(ctx, cfg, clnt)
	}

	if cfg.PlatformStatus.Type == "GCP" {
		driver = gcs.NewDriver(ctx, cfg, clnt)
	}

	return driver
}
