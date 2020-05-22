// TODO Add methods for bucket creation

package acs

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/go-autorest/autorest/to"
)

// CreateStorageAccount creates a new storage account.
func (d *driver)CreateStorageAccount(ctx context.Context, client *azureClient, accountName string) (s storage.Account, err error) {
	storageAccountsClient := client.accountsClient

	result, err := storageAccountsClient.CheckNameAvailability(ctx,
		storage.AccountCheckNameAvailabilityParameters{
			Name: to.StringPtr(accountName),
			Type: to.StringPtr("Microsoft.Storage/storageAccounts")})
	if err != nil {
		return s, fmt.Errorf("storage account creation failed: %v", err)
	}
	if *result.NameAvailable != true {
		return s, fmt.Errorf("storage account name not available: %v", err)
	}

	future, err := storageAccountsClient.Create(
		ctx,
		client.resourceGroupName,
		accountName,
		storage.AccountCreateParameters{
			Sku: &storage.Sku{
				Name: storage.StandardLRS},
			Kind:     storage.Storage,
			AccountPropertiesCreateParameters: &storage.AccountPropertiesCreateParameters{},
		})

	if err != nil {
		return s, fmt.Errorf("cannot create storage account, reason: %v", err)
	}
	err = future.WaitForCompletionRef(ctx, storageAccountsClient.Client)
	if err != nil {
		return s, fmt.Errorf("cannot get the storage account create future response: %v", err)
	}
	return future.Result(storageAccountsClient)
}

func (d *driver)CreateStorageContainer(ctx context.Context, client *azureClient, containerName string, accountName string) (container storage.BlobContainer, err error) {
	blobContainersClient := client.blobContainersClient

	//TODO add the blob container properties
	container, err = blobContainersClient.Create(ctx, client.resourceGroupName, accountName, containerName, storage.BlobContainer{
		// ContainerProperties - Properties of the blob container.
		ContainerProperties: &storage.ContainerProperties{
			PublicAccess: storage.PublicAccessNone,
		},
		Name: to.StringPtr(containerName),
		Type: to.StringPtr("Microsoft.Storage/storageAccounts"),
	})
	if err != nil {
		return container, fmt.Errorf("cannot create blob container: %v", err)
	}
	return container, nil;
}

func (d *driver)ListStorageContainer(ctx context.Context, client *azureClient, accountName string) (containerList storage.ListContainerItemsPage, err error) {
	blobContainersClient := client.blobContainersClient
	containerList, err = blobContainersClient.List(ctx, client.resourceGroupName, accountName, "", "");
	if err != nil {
		return containerList, fmt.Errorf("cannot list blob containers: %v", err)
	}
	return containerList, nil;
}

func (d *driver)GetStorageContainer(ctx context.Context, client *azureClient, containerName string, accountName string) (container storage.BlobContainer, err error) {
	blobContainersClient := client.blobContainersClient
	container, err = blobContainersClient.Get(ctx, client.resourceGroupName, accountName, containerName);
	if err != nil {
		return container, fmt.Errorf("cannot get blob container: %v", err)
	}
	return container, nil;
}


func (d *driver)GetContainerAccount(ctx context.Context, client *azureClient, accountName string) (container storage.Account, err error) {
	accountsClient := client.accountsClient
	container, err = accountsClient.GetProperties(ctx, client.resourceGroupName, accountName, "");
	if err != nil {
		return container, fmt.Errorf("cannot get container account: %v", err)
	}
	return container, nil;
}

func (d *driver) findVeleroBucket(containerList storage.ListContainerItemsPage) string {
	//TODO find the existing velero bucket
}