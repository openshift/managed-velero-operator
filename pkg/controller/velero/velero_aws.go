package velero

import (
	"context"

	"github.com/openshift/managed-velero-operator/pkg/storage/s3"

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

	r.driver = s3.NewDriver(ctx, r.config, r.client)

	return r
}

func (r *ReconcileVeleroAWS) RegionInChina() bool {
	for _, region := range awsChinaRegions {
		if r.config.PlatformStatus.AWS.Region == region {
			return true
		}
	}
	return false
}

func (r *ReconcileVeleroAWS) GetLocationConfig() map[string]string {
	return map[string]string{"region": r.config.PlatformStatus.AWS.Region}
}
