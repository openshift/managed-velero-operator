package azureblobservice

import (
	"context"

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	veleroInstallCR "github.com/openshift/managed-velero-operator/pkg/apis/managed/v1alpha2"
	storageBase "github.com/openshift/managed-velero-operator/pkg/storage/base"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type driver struct {
	storageBase.Driver
	client *AzureClient
}

// NewDriver creates a new AzureBlobService driver
// Used during bootstrapping
func NewDriver(ctx context.Context, cfg *configv1.InfrastructureStatus, kubeClient client.Client) (*driver, error) {
	client, err := NewAzureClient(kubeClient, cfg)

	if err != nil {
		return nil, err
	}
	drv := driver{
		client: client,
	}
	drv.Context = ctx
	drv.KubeClient = kubeClient

	return &drv, nil
}

// GetPlatformType returns the platform type of this driver
func (d *driver) GetPlatformType() configv1.PlatformType {
	return configv1.AzurePlatformType
}

// CreateStorage attempts to create a Azure Blob Service Container with relevant tags
func (d *driver) CreateStorage(reqLogger logr.Logger, instance *veleroInstallCR.VeleroInstall) error {

	if *instance.Status.Azure.StorageAccount == "" {
		err := setInstanceStorageAccount(d, reqLogger, instance)
		if err != nil {
			reqLogger.Error(err, "Error setting storage account")
			return err
		}
	}

	if instance.Status.Azure.StorageBucket.Name == "" {
		err := setInstanceStorageBucket(d, reqLogger, instance)
		if err != nil {
			reqLogger.Error(err, "Error setting storage bucket")
			return err
		}
	}

	return instance.StatusUpdate(reqLogger, d.KubeClient)
}

// StorageExists checks that the blob exists, and that we have access to it.
func (d *driver) StorageExists(status *veleroInstallCR.VeleroInstallStatus) (bool, error) {
	_, err := getStorageAccount(d.Context, d.client, *status.Azure.StorageAccount)

	if err != nil {
		return false, err
	}

	_, err = getBlobContainer(d.Context, d.client, *status.Azure.StorageAccount, status.AWS.StorageBucket.Name)

	if err != nil {
		return false, err
	}

	return true, nil
}
