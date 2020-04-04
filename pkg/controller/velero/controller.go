package velero

import (
	"context"
	"fmt"
	"time"

	"github.com/openshift/managed-velero-operator/pkg/storage"

	veleroInstallCR "github.com/openshift/managed-velero-operator/pkg/apis/managed/v1alpha2"

	configv1 "github.com/openshift/api/config/v1"
	minterv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/cblecker/platformutils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileVelero{client: mgr.GetClient(), scheme: mgr.GetScheme()}
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

// blank assignment to verify that ReconcileVelero implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileVelero{}

// ReconcileVelero reconciles a Velero object
type ReconcileVelero struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Velero object and makes changes based on the state read
// and what is in the Velero.Spec
func (r *ReconcileVelero) Reconcile(request reconcile.Request) (reconcile.Result, error) {
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

	// Grab infrastructureStatus to determine where OpenShift is installed.
	pc, err := platformutils.NewClient(context.TODO())
	if err != nil {
		return reconcile.Result{}, err
	}
	infraStatus, err := pc.GetInfrastructureStatus()
	if err != nil {
		return reconcile.Result{}, err
	}

	// Verify that we have received the needed platform information
	switch infraStatus.PlatformStatus.Type {
	case configv1.AWSPlatformType:
		if infraStatus.PlatformStatus.AWS == nil ||
			len(infraStatus.PlatformStatus.AWS.Region) < 1 {
			return reconcile.Result{}, fmt.Errorf("unable to determine AWS region")
		}
	case configv1.GCPPlatformType:
		if infraStatus.PlatformStatus.GCP == nil ||
			len(infraStatus.PlatformStatus.GCP.Region) < 1 ||
			len(infraStatus.PlatformStatus.GCP.ProjectID) < 1 {
			return reconcile.Result{}, fmt.Errorf("unable to determine GCP region")
		}
	default:
		return reconcile.Result{}, fmt.Errorf("unable to determine platform")
	}

	// Create the Storage Driver
	drv := storage.NewDriver(infraStatus, r.client)

	// Check if bucket needs to be reconciled
	if instance.StorageBucketReconcileRequired(s3ReconcilePeriod) {
		// Create storage using the storage driver
		// Always return from this, as we will either be updating the status *or* there will be an error.
		return reconcile.Result{}, drv.CreateStorage(reqLogger, instance)
	}

	// Now go provision Velero
	return r.provisionVelero(reqLogger, request.Namespace, infraStatus.PlatformStatus, instance)
}
