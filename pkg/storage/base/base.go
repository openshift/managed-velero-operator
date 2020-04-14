package base

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Driver holds common fields for storage drivers
type Driver struct {
	Context    context.Context
	KubeClient client.Client
}
