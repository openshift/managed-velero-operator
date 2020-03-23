package storage

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/openshift/managed-velero-operator/pkg/storage/s3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//Driver interface doc
type Driver interface {
	CreateStorage(logr.Logger, *mangedv1alpha1.Velero, string) error
	StorageExists(string) (bool, error)
}

//NewDriver doc
func NewDriver(cfg *configv1.InfrastructureStatus, clnt client.Client) Driver {

	ctx := context.Background()
	var driver Driver

	if cfg.Type == "AWS" {
		driver = s3.NewDriver(ctx, cfg, clnt)
	}

	return driver
}
