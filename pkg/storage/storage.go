package storage

import (
	"context"

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	veleroCR "github.com/openshift/managed-velero-operator/pkg/apis/managed/v1alpha1"
	"github.com/openshift/managed-velero-operator/pkg/storage/s3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//Driver interface to be satisfied by all present and future storage cloud providers
type Driver interface {
	CreateStorage(logr.Logger, *veleroCR.Velero, string) error
	StorageExists(string) (bool, error)
}

//NewDriver will return a driver object
func NewDriver(cfg *configv1.InfrastructureStatus, clnt client.Client) Driver {

	ctx := context.Background()
	var driver Driver

	if cfg.Platform == "AWS" {
		driver = s3.NewDriver(ctx, cfg, clnt)
	}

	return driver
}
