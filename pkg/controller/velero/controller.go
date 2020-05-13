package velero

import (
	"context"
	"fmt"
	"time"

	veleroInstallCR "github.com/openshift/managed-velero-operator/pkg/apis/managed/v1alpha2"

	appsv1 "k8s.io/api/apps/v1"
	configv1 "github.com/openshift/api/config/v1"
	minterv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	log               = logf.Log.WithName("controller_velero")
	s3ReconcilePeriod = 60 * time.Minute
)

// Add creates a new Velero Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, config *configv1.InfrastructureStatus) error {
	r, err := newVeleroReconciler(mgr, config)
	if err != nil {
		return err
	}

	return add(mgr, r)
}

// newVeleroReconciler returns a new VeleroReconciler for the detected platform.
func newVeleroReconciler(mgr manager.Manager, config *configv1.InfrastructureStatus) (VeleroReconciler, error) {
	var r VeleroReconciler

	ctx := context.Background()

	switch config.PlatformStatus.Type {
	case configv1.AWSPlatformType:
		r = newReconcileVeleroAWS(ctx, mgr, config)

	case configv1.GCPPlatformType:
		r = newReconcileVeleroGCP(ctx, mgr, config)

	default:
		return nil, fmt.Errorf("unable to determine platform")
	}

	return r, nil
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("velero-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource, Velero
	err = c.Watch(&source.Kind{Type: &veleroInstallCR.VeleroInstall{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to BackupStorageLocation
	err = c.Watch(&source.Kind{Type: &velerov1.BackupStorageLocation{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &veleroInstallCR.VeleroInstall{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to VolumeSnapshotLocation
	err = c.Watch(&source.Kind{Type: &velerov1.VolumeSnapshotLocation{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &veleroInstallCR.VeleroInstall{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to CredentialsRequest
	err = c.Watch(&source.Kind{Type: &minterv1.CredentialsRequest{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &veleroInstallCR.VeleroInstall{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to Deployments
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &veleroInstallCR.VeleroInstall{},
	})
	if err != nil {
		return err
	}

	return nil
}

type VeleroReconciler interface {
	reconcile.Reconciler

	// RegionInChina returns whether the cloud platform region is in China.
	RegionInChina() bool

	// GetImageRegistry returns the Velero image registry location.
	GetImageRegistry() string

	// GetLocationConfig returns the Velero BackupStorageLocationSpec.Config field.
	GetLocationConfig() map[string]string

	// CredentialsRequest creates an appropriate CredentialsRequest object for the
	// cloud platform.
	CredentialsRequest(namespace, bucketName string) (*minterv1.CredentialsRequest, error)

	// VeleroDeployment creates a base Deployment object with which to deploy Velero
	// on the cloud platform.
	VeleroDeployment(namespace string) *appsv1.Deployment
}

// blank assignment to verify that ReconcileVeleroBase implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileVeleroBase{}

// Reconcile reads that state of the cluster for a Velero object and makes changes based on the state read
// and what is in the Velero.Spec
func (r *ReconcileVeleroBase) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Velero Installation")
	var err error

	// Fetch the Velero instance
	instance := &veleroInstallCR.VeleroInstall{}
	err = r.client.Get(context.TODO(), request.NamespacedName, instance)
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

	// Check if bucket needs to be reconciled
	if instance.StorageBucketReconcileRequired(s3ReconcilePeriod) {
		// Create storage using the storage driver
		// Always return from this, as we will either be updating the status *or* there will be an error.
		return reconcile.Result{}, r.driver.CreateStorage(reqLogger, instance)
	}

	// Now go provision Velero
	return r.provisionVelero(reqLogger, request.Namespace, instance)
}
