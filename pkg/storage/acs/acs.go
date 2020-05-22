package acs

import (
	"context"
	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	veleroInstallCR "github.com/openshift/managed-velero-operator/pkg/apis/managed/v1alpha2"
	storageBase "github.com/openshift/managed-velero-operator/pkg/storage/base"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ACS struct {
	ResourceGroupName string
	InfraName string
}

type driver struct {
	storageBase.Driver
	Config *ACS
}

// NewDriver creates a new acs storage driver
// Used during bootstrapping
func NewDriver(ctx context.Context, cfg *configv1.InfrastructureStatus, clnt client.Client) *driver {
	drv := driver{
		Config: &ACS{
			ResourceGroupName: cfg.PlatformStatus.Azure.ResourceGroupName,
			InfraName: cfg.InfrastructureName,
		},
	}
	drv.Context = ctx
	drv.KubeClient = clnt
	return &drv
}

// GetPlatformType returns the platform type of this driver
func (d *driver) GetPlatformType() configv1.PlatformType {
	return configv1.AzurePlatformType
}


// CreateStorage attempts to create a ACS bucket and apply any provided tags
func (d *driver) CreateStorage(reqLogger logr.Logger, instance *veleroInstallCR.VeleroInstall) error {
	//TODO Implement method to create Azure bucket
	//https://github.com/vmware-tanzu/velero-plugin-for-microsoft-azure#Create-Azure-storage-account-and-blob-container

	acslient, err := NewAcsClient(d.KubeClient, d.Config.ResourceGroupName)
	if err != nil {
		return err
	}

	bucketLog := reqLogger.WithValues("StorageBucket.Name", instance.Status.StorageBucket.Name, "StorageBucket.ResourceGroupName", d.Config.ResourceGroupName)

	switch {
	// We don't yet have a bucket name selected
	case instance.Status.StorageBucket.Name == "":
		//TODO Implement method to create Azure bucket if no bucket name is provided
		break
	case instance.Status.StorageBucket.Name != "" && !instance.Status.StorageBucket.Provisioned:
		//TODO Implement method to create Azure bucket if bucket name is provided but no provisioned
		break
	}

	return instance.StatusUpdate(reqLogger, d.KubeClient)
}



// StorageExists checks that the bucket exists, and that we have access to it.
func (d *driver) StorageExists(bucketName string) (bool, error) {

	//create an Azure Client
	acslient, err := NewAcsClient(d.KubeClient, d.Config.ResourceGroupName)
	if err != nil {
		return false, err
	}

	_, err = acslient.blobContainersClient.Get(context.TODO(), d.Config.ResourceGroupName, acslient.accountName, bucketName);

	if err != nil {
		//TODO check the error if bucket/container exists

		return false, err
	}
	return true, nil
}