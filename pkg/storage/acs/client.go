package acs

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/openshift/managed-velero-operator/version"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	actCredsSecretName = version.OperatorName + "-iam-credentials"
)


// client implements the Client interface
type azureClient struct {
	resourceGroupName string
	accountName string
	//TODO Add relevant storage client
	blobContainersClient storage.BlobContainersClient
	accountsClient storage.AccountsClient
}

func getAzureCredentialsFromSecret(secret corev1.Secret) (clientId string, clientSecret string, tenantId string, subscriptionId string, accountName string, err error) {

	var acsCredsKey = "osServicePrincipal.json"
	var authMap map[string]string
	if secret.Data[acsCredsKey] == nil {
		return "", "", "", "", "", fmt.Errorf("secret %v does not contain required key %v", secret.Name, acsCredsKey)
	}

	if err := json.Unmarshal(secret.Data[acsCredsKey], &authMap); err != nil {
		return "", "", "", "", "", fmt.Errorf("json unmarshlling failed key: %v, secret: %v, namespace: %v", acsCredsKey, secret.Name, secret.Namespace)
	}

	clientId, ok := authMap["clientId"]
	if !ok {
		return "", "", "", "", "", fmt.Errorf("clientId is missing for Key: '%v', secret: '%v', namespace: '%v'", acsCredsKey, secret.Name, secret.Namespace)
	}
	clientSecret, ok = authMap["clientSecret"]
	if !ok {
		return "", "", "", "", "", fmt.Errorf("clientSecret is missing for Key: '%v', secret: '%v', namespace: '%v'", acsCredsKey, secret.Name, secret.Namespace)
	}
	tenantId, ok = authMap["tenantId"]
	if !ok {
		return "", "", "", "", "", fmt.Errorf("tenantId is missing for Key: '%v', secret: '%v', namespace: '%v'", acsCredsKey, secret.Name, secret.Namespace)
	}
	subscriptionId, ok = authMap["subscriptionId"]
	if !ok {
		return "", "", "", "", "", fmt.Errorf("subscriptionId is missing for Key: '%v', secret: '%v', namespace: '%v'", acsCredsKey, secret.Name, secret.Namespace)
	}
	accountName, ok = authMap["accountName"]

	return clientId, clientSecret, tenantId, subscriptionId, accountName, nil
}

// NewAcsClient reads the acs secrets in the operator's namespace and uses
// them to create a new client for accessing the ACS API.
func NewAcsClient(kubeClient client.Client, resourceGroupName string) (*azureClient, error) {
	var err error

	namespace, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		return nil, fmt.Errorf("failed to get operator namespace: %v", err)
	}

	secret := &corev1.Secret{}
	err = kubeClient.Get(context.TODO(),
		types.NamespacedName{
			Name:      actCredsSecretName,
			Namespace: namespace,
		},
		secret)
	if err != nil {
		return nil, err
	}

	clientID, clientSecret, tenantID, subscriptionID, accountName, err := getAzureCredentialsFromSecret(*secret)

	if err != nil {
		return nil, err
	}

	config := auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)

	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, err
	}

	blobContainersClient := storage.NewBlobContainersClientWithBaseURI(azure.PublicCloud.ResourceManagerEndpoint, subscriptionID)
	blobContainersClient.Authorizer = authorizer

	accountsClient := storage.NewAccountsClientWithBaseURI(azure.PublicCloud.ResourceManagerEndpoint, subscriptionID)
	accountsClient.Authorizer = authorizer
	return &azureClient{
		resourceGroupName: resourceGroupName,
		accountName: accountName,
		// TODO Add storage client properties
		blobContainersClient: blobContainersClient,
		accountsClient: accountsClient,
	}, nil
}
