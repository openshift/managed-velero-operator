package base

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Driver holds common fields for storage drivers
type Driver struct {
	Context    context.Context
	KubeClient client.Client
}

// GetPlatformType returns the platform type of this driver
func (d *Driver) GetPlatformType() configv1.PlatformType {
	return configv1.NonePlatformType
}
