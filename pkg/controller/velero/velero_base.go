package velero

import (
	"github.com/openshift/managed-velero-operator/pkg/storage"

	configv1 "github.com/openshift/api/config/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcileVeleroBase reconciles a Velero object.  It serves as an "abstract"
// base struct for embedding in other cloud-platform-specific structs.
type ReconcileVeleroBase struct {
	// virtual method table
	vtable VeleroReconciler

	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	config *configv1.InfrastructureStatus
	driver storage.Driver
}

func (r *ReconcileVeleroBase) RegionInChina() bool {
	return false
}
