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

func checkExistingStorageAccount(ctx context.Context, reqLogger logr.Logger, client *AzureClient) (storageAccount *storage.Account, err error) {
	storageAccountList, err := client.storageAccountsClient.ListByResourceGroup(ctx, client.resourceGroupName)
	if err != nil {
		reqLogger.Error(err, "Error listing storage accounts")
		return nil, err
	}

	for _, item := range *storageAccountList.Value {
		if *item.Tags[storageConstants.BucketTagBackupStorageLocation] == storageConstants.DefaultVeleroBackupStorageLocation &&
			*item.Tags[storageConstants.BucketTagInfrastructureName] == client.infrastructureName {
			return storageAccount, nil
		}
	}

	return nil, nil
}

func createStorageAccount(ctx context.Context, client *AzureClient) (*storage.Account, error) {

	proposedName := generateAccountName(storageConstants.StorageBucketPrefix)

	result, err := client.storageAccountsClient.CheckNameAvailability(ctx,
		storage.AccountCheckNameAvailabilityParameters{
			Name: to.StringPtr(proposedName),
			Type: to.StringPtr("Microsoft.Storage/storageAccounts"),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("storage account creation failed: %v", err)
	}

	if *result.NameAvailable != true {
		return nil, fmt.Errorf("storage account name not available: %v", err)
	}

	future, err := client.storageAccountsClient.Create(
		ctx,
		client.resourceGroupName,
		client.infrastructureName,
		storage.AccountCreateParameters{
			Sku: &storage.Sku{
				Name: storage.StandardLRS,
			},
			Kind:                              storage.Storage,
			AccountPropertiesCreateParameters: &storage.AccountPropertiesCreateParameters{},
			Tags: map[string]*string{
				storageConstants.BucketTagBackupStorageLocation: to.StringPtr(storageConstants.DefaultVeleroBackupStorageLocation),
				storageConstants.BucketTagInfrastructureName:    &client.infrastructureName,
			},
		})

	if err != nil {
		return nil, fmt.Errorf("cannot create storage account, reason: %v", err)
	}
	err = future.WaitForCompletionRef(ctx, client.storageAccountsClient.Client)
	if err != nil {
		return nil, fmt.Errorf("cannot get the storage account create future response: %v", err)
	}

	storageAccount, err := future.Result(client.storageAccountsClient)
	return &storageAccount, err
}

func generateAccountName(prefix string) string {
	id := uuid.New().String()
	return prefix + id
}

func getStorageAccount(ctx context.Context, client *AzureClient, storageAccountName string) (storage.Account, error) {
	return client.storageAccountsClient.GetProperties(ctx, client.resourceGroupName, storageAccountName, "")
}

func setInstanceStorageAccount(d *driver, reqLogger logr.Logger, instance *veleroInstallCR.VeleroInstall) error {
	storageAccount, err := checkExistingStorageAccount(d.Context, reqLogger, d.client)
	if err != nil {
		return err
	}

	if storageAccount == nil {
		reqLogger.Info("Existing Storage account cannot be found. Creating new storage account")
		storageAccount, err = createStorageAccount(d.Context, d.client)
		if err != nil {
			return err
		}
	}

	instance.Status.Azure.StorageAccount = storageAccount.Name

	return nil
}
