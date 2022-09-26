package velero

import (
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	"time"

	minterv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/cblecker/platformutils"
	veleroInstallCR "github.com/openshift/managed-velero-operator/api/v1alpha2"
	"github.com/openshift/managed-velero-operator/pkg/storage"
)

var (
	log               = logf.Log.WithName("controller_velero")
	s3ReconcilePeriod = 60 * time.Minute
)

// VeleroInstallReconciler reconciles a Velero object
type VeleroInstallReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	driver storage.Driver
}

//+kubebuilder:rbac:groups=managed.openshift.io,resources=veleroinstalls,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=managed.openshift.io,resources=veleroinstalls/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=managed.openshift.io,resources=veleroinstalls/finalizers,verbs=update

// Reconcile reads that state of the cluster for a Velero object and makes changes based on the state read
// and what is in the Velero.Spec
func (r *VeleroInstallReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Velero Installation")
	var err error

	// Fetch the Velero instance
	instance := &veleroInstallCR.VeleroInstall{}
	err = r.Client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Grab infrastructureStatus to determine where OpenShift is installed.
	pc, err := platformutils.NewClient(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}
	infraStatus, err := pc.GetInfrastructureStatus()
	if err != nil {
		return reconcile.Result{}, err
	}

	// Create the Storage Driver
	if r.driver == nil {
		r.driver, err = storage.NewDriver(infraStatus, r.Client)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// Check if bucket needs to be reconciled
	if instance.StorageBucketReconcileRequired(s3ReconcilePeriod) {
		// Create storage using the storage driver
		// Always return from this, as we will either be updating the status *or* there will be an error.
		return reconcile.Result{}, r.driver.CreateStorage(reqLogger, instance)
	}

	// Now go provision Velero
	return r.provisionVelero(reqLogger, request.Namespace, infraStatus.PlatformStatus, instance)
}

// SetupWithManager sets up the controller with the Manager.
func (r *VeleroInstallReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&veleroInstallCR.VeleroInstall{}).
		Owns(&velerov1.BackupStorageLocation{}).
		Owns(&velerov1.VolumeSnapshotLocation{}).
		Owns(&minterv1.CredentialsRequest{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
