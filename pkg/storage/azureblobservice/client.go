package azureblobservice

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/openshift/managed-velero-operator/version"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	azureCredsSecretName = version.OperatorName + "-iam-credentials"
	azureCredsSecretKey  = "osServicePrincipal.json"
)

// AzureClient interact with Azure API
type AzureClient struct {
	resourceGroupName     string
	infrastructureName    string
	blobContainersClient  storage.BlobContainersClient
	storageAccountsClient storage.AccountsClient
}

func getCredentialsSecret(kubeClient client.Client) (secret *corev1.Secret, err error) {
	namespace, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		return nil, fmt.Errorf("failed to get operator namespace: %v", err)
	}

	secret = &corev1.Secret{}
	err = kubeClient.Get(context.TODO(),
		types.NamespacedName{
			Name:      azureCredsSecretName,
			Namespace: namespace,
		},
		secret)

	return secret, err
}

func getAzureCredentials(kubeClient client.Client) (clientID string, clientSecret string, tenantID string, subscriptionID string, err error) {

	secret, err := getCredentialsSecret(kubeClient)

	if err != nil {
		return "", "", "", "", err
	}

	if secret.Data[azureCredsSecretKey] == nil {
		return "", "", "", "", fmt.Errorf("secret %v does not contain required key %v", secret.Name, azureCredsSecretKey)
	}

	var authMap map[string]string

	if err := json.Unmarshal(secret.Data[azureCredsSecretKey], &authMap); err != nil {
		return "", "", "", "", fmt.Errorf("json unmarshlling failed key: %v, secret: %v, namespace: %v", azureCredsSecretKey, secret.Name, secret.Namespace)
	}

	clientID, ok := authMap["clientId"]
	if !ok {
		return "", "", "", "", fmt.Errorf("clientId is missing for Key: '%v', secret: '%v', namespace: '%v'", azureCredsSecretKey, secret.Name, secret.Namespace)
	}
	clientSecret, ok = authMap["clientSecret"]
	if !ok {
		return "", "", "", "", fmt.Errorf("clientSecret is missing for Key: '%v', secret: '%v', namespace: '%v'", azureCredsSecretKey, secret.Name, secret.Namespace)
	}
	tenantID, ok = authMap["tenantId"]
	if !ok {
		return "", "", "", "", fmt.Errorf("tenantId is missing for Key: '%v', secret: '%v', namespace: '%v'", azureCredsSecretKey, secret.Name, secret.Namespace)
	}
	subscriptionID, ok = authMap["subscriptionId"]
	if !ok {
		return "", "", "", "", fmt.Errorf("subscriptionId is missing for Key: '%v', secret: '%v', namespace: '%v'", azureCredsSecretKey, secret.Name, secret.Namespace)
	}

	return clientID, clientSecret, tenantID, subscriptionID, nil
}

// NewAzureClient reads the credentials secret in the operator's namespace and uses
// them to create a new azure client.
func NewAzureClient(kubeClient client.Client, cfg *configv1.InfrastructureStatus) (*AzureClient, error) {
	var err error

	clientID, clientSecret, tenantID, subscriptionID, err := getAzureCredentials(kubeClient)

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

	storageAccountsClient := storage.NewAccountsClientWithBaseURI(azure.PublicCloud.ResourceManagerEndpoint, subscriptionID)
	storageAccountsClient.Authorizer = authorizer
	return &AzureClient{
		resourceGroupName:     cfg.PlatformStatus.Azure.ResourceGroupName,
		infrastructureName:    cfg.InfrastructureName,
		blobContainersClient:  blobContainersClient,
		storageAccountsClient: storageAccountsClient,
	}, nil
}
