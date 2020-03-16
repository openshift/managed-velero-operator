package storage

import (
	"context"

	"github.com/openshift/managed-velero-operator/pkg/storage/s3"
)

//Driver interface doc
type Driver interface {
	CreateStorage(logr.Logger, *ReconcileVelero, *mangedv1alpha1.Velero, string) error
	StorageExists(*ReconcileVelero, string) (bool, error)
}

//NewDriver doc
func NewDriver(cfg *configv1.InfrastructureStatus) Driver {

	ctx := context.Background()
	var driver Driver

	if cfg.Type == "AWS" {
		driver = s3.NewDriver(ctx, cfg)
	}

	return driver
}
