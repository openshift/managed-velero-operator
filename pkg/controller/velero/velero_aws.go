package velero

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// ReconcileVeleroAWS reconciles a Velero object on Amazon Web Services
type ReconcileVeleroAWS struct {
	ReconcileVeleroBase
}

func newReconcileVeleroAWS(ctx context.Context, mgr manager.Manager, config *configv1.InfrastructureStatus) VeleroReconciler {
	var r = &ReconcileVeleroAWS{
		ReconcileVeleroBase{
			client: mgr.GetClient(),
			scheme: mgr.GetScheme(),
			config: config,
		},
	}
	r.vtable = r

	return r
}
