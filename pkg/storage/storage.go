package storage

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	veleroInstallCR "github.com/openshift/managed-velero-operator/pkg/apis/managed/v1alpha2"
	"github.com/openshift/managed-velero-operator/pkg/storage/gcs"
	"github.com/openshift/managed-velero-operator/pkg/storage/s3"
	"github.com/openshift/managed-velero-operator/pkg/storage/acs"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//Driver interface to be satisfied by all present and future storage cloud providers
type Driver interface {
	GetPlatformType() configv1.PlatformType
	CreateStorage(logr.Logger, *veleroInstallCR.VeleroInstall) error
	StorageExists(string) (bool, error)
}

//NewDriver will return a driver object
func NewDriver(cfg *configv1.InfrastructureStatus, client client.Client) (Driver, error) {
	var driver Driver

	ctx := context.Background()

	// Verify that we have received the needed platform information
	switch cfg.PlatformStatus.Type {
	case configv1.AWSPlatformType:
		if cfg.PlatformStatus.AWS == nil ||
			len(cfg.PlatformStatus.AWS.Region) < 1 {
			return nil, fmt.Errorf("unable to determine AWS region")
		}
		driver = s3.NewDriver(ctx, cfg, client)
	case configv1.GCPPlatformType:
		if cfg.PlatformStatus.GCP == nil ||
			len(cfg.PlatformStatus.GCP.Region) < 1 ||
			len(cfg.PlatformStatus.GCP.ProjectID) < 1 {
			return nil, fmt.Errorf("unable to determine GCP region")
		}
		driver = gcs.NewDriver(ctx, cfg, client)
	case configv1.AzurePlatformType:
		if cfg.PlatformStatus.Azure == nil ||
			len(cfg.PlatformStatus.Azure.ResourceGroupName) < 1 {
			return nil, fmt.Errorf("unable to determine Azure region")
		}
		driver = acs.NewDriver(ctx, cfg, client)
	default:
		return nil, fmt.Errorf("unable to determine platform")
	}

	return driver, nil
}
