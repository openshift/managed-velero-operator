package azureblobservice

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	veleroInstallCR "github.com/openshift/managed-velero-operator/pkg/apis/managed/v1alpha2"
	storageConstants "github.com/openshift/managed-velero-operator/pkg/storage/constants"
)

func checkExistingBlobContainer(ctx context.Context, reqLogger logr.Logger, client *AzureClient, storageAccountName string) (*storage.BlobContainer, error) {
	blobContainersPage, err := client.blobContainersClient.List(ctx, client.resourceGroupName, storageAccountName, "50", "")

	if err != nil {
		return nil, err
	}

	for blobContainersPage.NotDone() {
		for _, blobContainerItem := range blobContainersPage.Values() {
			if *blobContainerItem.Metadata[storageConstants.BucketTagBackupStorageLocation] == storageConstants.DefaultVeleroBackupStorageLocation &&
				*blobContainerItem.Metadata[storageConstants.BucketTagInfrastructureName] == client.infrastructureName {
				return getBlobContainer(ctx, client, storageAccountName, *blobContainerItem.Name)
			}
		}

		if err = blobContainersPage.NextWithContext(ctx); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func createBlobContainer(ctx context.Context, client *AzureClient, storageAccountName string) (*storage.BlobContainer, error) {

	proposedName := generateBlobContainerName(storageConstants.StorageBucketPrefix)

	container, err := client.blobContainersClient.Create(
		ctx,
		client.resourceGroupName,
		storageAccountName,
		proposedName,
		storage.BlobContainer{
			ContainerProperties: &storage.ContainerProperties{
				PublicAccess: storage.PublicAccessNone,
				Metadata: map[string]*string{
					storageConstants.BucketTagBackupStorageLocation: to.StringPtr(storageConstants.DefaultVeleroBackupStorageLocation),
					storageConstants.BucketTagInfrastructureName:    &client.infrastructureName,
				},
			},
			Name: to.StringPtr(proposedName),
			Type: to.StringPtr("Microsoft.Storage/storageAccounts"),
		})

	if err != nil {
		return nil, fmt.Errorf("cannot create blob container: %v", err)
	}
	return &container, nil
}

func generateBlobContainerName(prefix string) string {
	id := uuid.New().String()
	return prefix + id
}

func getBlobContainer(ctx context.Context, client *AzureClient, storageAccountName string, blobContainerName string) (*storage.BlobContainer, error) {
	container, err := client.blobContainersClient.Get(ctx, client.resourceGroupName, storageAccountName, blobContainerName)
	if err != nil {
		return nil, fmt.Errorf("cannot get blob container: %v", err)
	}
	return &container, nil
}

func setInstanceStorageBucket(d *driver, reqLogger logr.Logger, instance *veleroInstallCR.VeleroInstall) error {
	blobContainer, err := checkExistingBlobContainer(d.Context, reqLogger, d.client, *instance.Status.Azure.StorageAccount)
	if err != nil {
		return err
	}

	if blobContainer == nil {
		reqLogger.Info("Existing Blob Container cannot be found. Creating new blob container")
		blobContainer, err = createBlobContainer(d.Context, d.client, *instance.Status.Azure.StorageAccount)
		if err != nil {
			return err
		}
	}

	instance.Status.Azure.StorageBucket.Name = *blobContainer.Name
	instance.Status.Azure.StorageBucket.Provisioned = true
	return nil
}

// func (d *driver) findVeleroBucket(containerList storage.ListContainerItemsPage) string {
// 	//TODO find the existing velero bucket
// }
