package platform

import (
	"context"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type installConfig struct {
	Platform struct {
		AWS struct {
			Region string `json:"region"`
		} `json:"aws"`
	} `json:"platform"`
}

// GetPlatformStatus provides a backwards-compatible way to look up platform
// status. AWS is the special case. 4.1 clusters on AWS expose the region config
// only through install-config. New AWS clusters and all other 4.2+ platforms
// are configured via platform status.
func GetPlatformStatus(client client.Client) (*configv1.PlatformStatus, error) {
	var err error

	// Retrieve the cluster infrastructure config.
	infra := &configv1.Infrastructure{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: "cluster"}, infra)
	if err != nil {
		return nil, err
	}

	if status := infra.Status.PlatformStatus; status != nil {
		// Only AWS needs backwards compatibility with install-config
		if status.Type != configv1.AWSPlatformType {
			return status, nil
		}

		// Check whether the cluster config is already migrated
		if status.AWS != nil && len(status.AWS.Region) > 0 {
			return status, nil
		}
	}

	// Otherwise build a platform status from the deprecated install-config
	clusterConfigName := types.NamespacedName{Namespace: "kube-system", Name: "cluster-config-v1"}
	clusterConfig := &corev1.ConfigMap{}
	if err := client.Get(context.TODO(), clusterConfigName, clusterConfig); err != nil {
		return nil, fmt.Errorf("failed to get configmap %s: %v", clusterConfigName, err)
	}
	data, ok := clusterConfig.Data["install-config"]
	if !ok {
		return nil, fmt.Errorf("missing install-config in configmap")
	}
	var ic installConfig
	if err := yaml.Unmarshal([]byte(data), &ic); err != nil {
		return nil, fmt.Errorf("invalid install-config: %v\njson:\n%s", err, data)
	}
	return &configv1.PlatformStatus{
		//lint:ignore SA1019 ignore deprecation, as this function is specifically
		// designed for backwards compatibility
		Type: infra.Status.Platform,
		AWS: &configv1.AWSPlatformStatus{
			Region: ic.Platform.AWS.Region,
		},
	}, nil
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
