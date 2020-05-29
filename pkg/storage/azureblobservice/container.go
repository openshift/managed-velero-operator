package azureblobservice

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/go-logr/logr"
	veleroInstallCR "github.com/openshift/managed-velero-operator/pkg/apis/managed/v1alpha2"
)

var (
	blobFormatString        = `https://%s.blob.core.windows.net`
	veleroBlobContainerName = "managed-velero-backup-container"
)

func checkExistingBlobContainer(ctx context.Context, reqLogger logr.Logger, client *AzureClient,
	storageAccountName string, containerName string) *storage.BlobContainer {

	blobContainer, err := client.blobContainersClient.Get(ctx, client.resourceGroupName, storageAccountName, containerName)

	if err != nil {
		return nil
	}

	return &blobContainer
}

func createBlobContainer(ctx context.Context, client *AzureClient, storageAccountName string) (*string, error) {
	c, err := getContainerURL(ctx, client, storageAccountName, veleroBlobContainerName)
	if err != nil {
		return nil,
			fmt.Errorf(
				"Error fetching container URL for account: %v, container: %v . Error: %w",
				storageAccountName,
				veleroBlobContainerName,
				err,
			)
	}

	_, err = c.Create(ctx, azblob.Metadata{}, azblob.PublicAccessNone)
	if err != nil {
		return nil, fmt.Errorf(
			"Error creating blob container for account: %v, container: %v . Error: %w",
			storageAccountName,
			veleroBlobContainerName,
			err,
		)
	}

	return &veleroBlobContainerName, nil
}

func getBlobContainer(ctx context.Context, client *AzureClient, storageAccountName string, blobContainerName string) (*storage.BlobContainer, error) {
	container, err := client.blobContainersClient.Get(ctx, client.resourceGroupName, storageAccountName, blobContainerName)
	if err != nil {
		return nil, fmt.Errorf("cannot get blob container: %v", err)
	}
	return &container, nil
}

func getContainerURL(ctx context.Context, client *AzureClient, storageAccountName string, containerName string) (*azblob.ContainerURL, error) {
	key, err := getAccountPrimaryKey(ctx, client, storageAccountName)

	if err != nil {
		return nil, err
	}

	c, err := azblob.NewSharedKeyCredential(storageAccountName, key)
	if err != nil {
		return nil, err
	}

	p := azblob.NewPipeline(c, azblob.PipelineOptions{})
	u, _ := url.Parse(fmt.Sprintf(blobFormatString, storageAccountName))
	service := azblob.NewServiceURL(*u, p)
	container := service.NewContainerURL(containerName)
	return &container, nil
}

func getOrCreateContainer(d *driver, reqLogger logr.Logger, instance *veleroInstallCR.VeleroInstall) (blobContainerName *string, err error) {
	blobContainer := checkExistingBlobContainer(d.Context, reqLogger, d.client, instance.Status.Azure.StorageAccount, veleroBlobContainerName)

	if blobContainer != nil {
		return blobContainer.Name, nil
	}

	reqLogger.Info("Existing Blob Container cannot be found. Creating new blob container")
	return createBlobContainer(d.Context, d.client, instance.Status.Azure.StorageAccount)
}

func reconcileBlobContainer(d *driver, reqLogger logr.Logger, storageAccountName string) error {
	reqLogger.Info("Reconciling blob container")
	c, err := getContainerURL(d.Context, d.client, storageAccountName, veleroBlobContainerName)

	if err != nil {
		return fmt.Errorf(
			"Error fetching container URL for account: %v, container: %v . Error: %w",
			storageAccountName,
			veleroBlobContainerName,
			err,
		)
	}

	_, err = c.SetAccessPolicy(d.Context, azblob.PublicAccessNone, []azblob.SignedIdentifier{}, azblob.ContainerAccessConditions{})

	if err != nil {
		return fmt.Errorf(
			"Error reconciling container: %v , for account: %v . Error: %w",
			storageAccountName,
			veleroBlobContainerName,
			err,
		)
	}

	return nil
}
