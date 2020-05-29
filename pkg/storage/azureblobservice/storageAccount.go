package azureblobservice

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/dchest/uniuri"
	"github.com/go-logr/logr"

	veleroInstallCR "github.com/openshift/managed-velero-operator/pkg/apis/managed/v1alpha2"
	storageConstants "github.com/openshift/managed-velero-operator/pkg/storage/constants"
)

func checkExistingStorageAccount(ctx context.Context, reqLogger logr.Logger, client *AzureClient) (storageAccount *storage.Account, err error) {
	storageAccountList, err := client.storageAccountsClient.ListByResourceGroup(ctx, client.resourceGroupName)
	if err != nil {
		return nil, fmt.Errorf("Error listing storage accounts. Error: %w", err)
	}

	var tagMatchesInfra, tagMatchesVelero bool

	for _, item := range *storageAccountList.Value {
		tagMatchesVelero = false
		tagMatchesInfra = false
		for key, value := range item.Tags {
			if key == storageConstants.AzureStorageAccountTagBackupLocation && value != nil &&
				*value == storageConstants.DefaultVeleroBackupStorageLocation {
				tagMatchesVelero = true
			}
			if key == storageConstants.AzureStorageAccountTagInfrastructureName && value != nil &&
				*value == client.infrastructureName {
				tagMatchesInfra = true
			}
		}
		if tagMatchesVelero && tagMatchesInfra {
			return &item, nil
		}
	}
	return nil, nil
}

func createStorageAccount(ctx context.Context, client *AzureClient) (*string, error) {

	proposedName := generateAccountName(storageConstants.AzureStorageAccountPrefix)

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
		return nil, fmt.Errorf("storage account name not available: %v because : %v", proposedName, *result.Message)
	}

	future, err := client.storageAccountsClient.Create(
		ctx,
		client.resourceGroupName,
		proposedName,
		storage.AccountCreateParameters{
			Sku: &storage.Sku{
				Name: storage.StandardGRS,
			},
			Kind:     storage.BlobStorage,
			Location: to.StringPtr(client.region),
			AccountPropertiesCreateParameters: &storage.AccountPropertiesCreateParameters{
				EnableHTTPSTrafficOnly: to.BoolPtr(true),
				AccessTier:             storage.Hot,
			},
			Tags: map[string]*string{
				storageConstants.AzureStorageAccountTagBackupLocation:     to.StringPtr(storageConstants.DefaultVeleroBackupStorageLocation),
				storageConstants.AzureStorageAccountTagInfrastructureName: &client.infrastructureName,
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
	return storageAccount.Name, err
}

func generateAccountName(prefix string) string {
	return prefix + uniuri.NewLenChars(8, []byte("abcdefghijklmnopqrstuvwxyz0123456789"))
}

func getAccountKeys(ctx context.Context, client *AzureClient, storageAccountName string) (storage.AccountListKeysResult, error) {
	return client.storageAccountsClient.ListKeys(ctx, client.resourceGroupName, storageAccountName, storage.Kerb)
}

func getAccountPrimaryKey(ctx context.Context, client *AzureClient, storageAccountName string) (string, error) {
	response, err := getAccountKeys(ctx, client, storageAccountName)
	if err != nil {
		return "", err
	}
	return *(((*response.Keys)[0]).Value), nil
}

func getOrCreateStorageAccount(d *driver, reqLogger logr.Logger, instance *veleroInstallCR.VeleroInstall) (*string, error) {
	storageAccount, err := checkExistingStorageAccount(d.Context, reqLogger, d.client)

	if err != nil {
		return nil, err
	}

	if storageAccount != nil {
		reqLogger.Info(fmt.Sprintf("Found existing storage account : %v", *storageAccount.Name))
		return storageAccount.Name, nil
	}

	reqLogger.Info("Existing Storage account cannot be found. Creating new storage account")
	return createStorageAccount(d.Context, d.client)
}

func getStorageAccount(ctx context.Context, client *AzureClient, storageAccountName string) (storage.Account, error) {
	return client.storageAccountsClient.GetProperties(ctx, client.resourceGroupName, storageAccountName, "")
}

func reconcileStorageAccount(d *driver, reqLogger logr.Logger, storageAccountName string) error {
	reqLogger.Info(fmt.Sprintf("Reconciling Storage Account : %v", storageAccountName))

	_, err := d.client.storageAccountsClient.Update(
		d.Context,
		d.client.resourceGroupName,
		storageAccountName,
		storage.AccountUpdateParameters{
			Sku: &storage.Sku{
				Name: storage.StandardGRS,
			},
			AccountPropertiesUpdateParameters: &storage.AccountPropertiesUpdateParameters{
				EnableHTTPSTrafficOnly: to.BoolPtr(true),
				AccessTier:             storage.Hot,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("Error reconciling storage account : %w", err)
	}
	return nil
}
