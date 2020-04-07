package gcs

import (
	"context"
	"fmt"

	"github.com/openshift/managed-velero-operator/version"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gstorage "cloud.google.com/go/storage"
	goauth2 "golang.org/x/oauth2/google"
	goption "google.golang.org/api/option"
)

var (
	storageCredsSecretName = version.OperatorName + "-iam-credentials"
)

// NewGcsClient reads the gcp secrets in the operator's namespace and uses
// them to create a new client for accessing the GCS API.
func NewGcsClient(kubeClient client.Client) (*gstorage.Client, error) {
	var err error

	namespace, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		return nil, fmt.Errorf("failed to get operator namespace: %v", err)
	}

	secret := &corev1.Secret{}
	err = kubeClient.Get(context.TODO(),
		types.NamespacedName{
			Name:      storageCredsSecretName,
			Namespace: namespace,
		},
		secret)
	if err != nil {
		return nil, err
	}
	keyFileData, ok := secret.Data["service_account.json"]
	if !ok {
		return nil, fmt.Errorf("secret %q does not contain required key \"service_account.json\"", fmt.Sprintf("%s/%s", namespace, storageCredsSecretName))
	}

	credentials, err := goauth2.CredentialsFromJSON(context.TODO(), []byte(string(keyFileData)), gstorage.ScopeFullControl)
	if err != nil {
		return nil, err
	}

	gcsClient, err := gstorage.NewClient(context.TODO(), goption.WithCredentials(credentials))
	if err != nil {
		return nil, err
	}

	return gcsClient, nil
}
