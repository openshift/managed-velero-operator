package velero

import (
	"context"

	"github.com/openshift/managed-velero-operator/pkg/storage/gcs"

	configv1 "github.com/openshift/api/config/v1"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// ReconcileVeleroGCP reconciles a Velero object on Google Cloud Platform
type ReconcileVeleroGCP struct {
	ReconcileVeleroBase
}

func newReconcileVeleroGCP(ctx context.Context, mgr manager.Manager, config *configv1.InfrastructureStatus) VeleroReconciler {
	var r = &ReconcileVeleroGCP{
		ReconcileVeleroBase{
			client: mgr.GetClient(),
			scheme: mgr.GetScheme(),
			config: config,
		},
	}
	r.vtable = r

	r.driver = gcs.NewDriver(ctx, r.config, r.client)

	return r
}
