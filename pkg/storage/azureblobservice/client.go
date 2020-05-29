package azureblobservice

import (
	"context"
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
	region                string
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

func getAzureCredentials(kubeClient client.Client) (clientID string, clientSecret string, tenantID string, subscriptionID string, region string, err error) {

	secret, err := getCredentialsSecret(kubeClient)

	if err != nil {
		return "", "", "", "", "", err
	}

	clientID, ok := getStringFromSecret(secret, "azure_client_id")
	if !ok {
		return "", "", "", "", "", fmt.Errorf("azure_client_id is missing for secret: '%v', namespace: '%v'", secret.Name, secret.Namespace)
	}
	clientSecret, ok = getStringFromSecret(secret, "azure_client_secret")
	if !ok {
		return "", "", "", "", "", fmt.Errorf("azure_client_secret is missing for secret: '%v', namespace: '%v'", secret.Name, secret.Namespace)
	}
	tenantID, ok = getStringFromSecret(secret, "azure_tenant_id")
	if !ok {
		return "", "", "", "", "", fmt.Errorf("azure_tenant_id is missing for secret: '%v', namespace: '%v'", secret.Name, secret.Namespace)
	}
	subscriptionID, ok = getStringFromSecret(secret, "azure_subscription_id")
	if !ok {
		return "", "", "", "", "", fmt.Errorf("azure_subscription_id is missing for secret: '%v', namespace: '%v'", secret.Name, secret.Namespace)
	}
	region, ok = getStringFromSecret(secret, "azure_region")
	if !ok {
		return "", "", "", "", "", fmt.Errorf("azure_region is missing for secret: '%v', namespace: '%v'", secret.Name, secret.Namespace)
	}

	return clientID, clientSecret, tenantID, subscriptionID, region, nil
}

func getStringFromSecret(secret *corev1.Secret, key string) (string, bool) {
	bytesVal, ok := secret.Data[key]
	if !ok {
		return "", false
	}
	return string(bytesVal), true
}

// NewAzureClient reads the credentials secret in the operator's namespace and uses
// them to create a new azure client.
func NewAzureClient(kubeClient client.Client, cfg *configv1.InfrastructureStatus) (*AzureClient, error) {
	var err error

	clientID, clientSecret, tenantID, subscriptionID, region, err := getAzureCredentials(kubeClient)

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
		region:                region,
		blobContainersClient:  blobContainersClient,
		storageAccountsClient: storageAccountsClient,
	}, nil
}
