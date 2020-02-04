package platform

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
)

// GetInfrastructureClient provides a k8s client that is capable of retrieving
// the items necessary to determine the platform status.
func GetInfrastructureClient() (client.Client, error) {
	var err error
	scheme := runtime.NewScheme()

	// Set up platform status client
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	// Add OpenShift config apis to scheme
	if err := configv1.Install(scheme); err != nil {
		return nil, err
	}

	// Add Core apis to scheme
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	// Create client
	return client.New(cfg, client.Options{Scheme: scheme})
}

// GetInfrastructureStatus fetches the InfrastructureStatus for the cluster.
func GetInfrastructureStatus(client client.Client) (*configv1.InfrastructureStatus, error) {
	var err error

	// Retrieve the cluster infrastructure config.
	infra := &configv1.Infrastructure{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: "cluster"}, infra)
	if err != nil {
		return nil, err
	}

	return &infra.Status, nil
}

// IsPlatformSupported checks if specified platform is in a slice of supported
// platforms
func IsPlatformSupported(platform configv1.PlatformType, supportedPlatforms []configv1.PlatformType) bool {
	for _, p := range supportedPlatforms {
		if p == platform {
			return true
		}
	}
	return false
}
