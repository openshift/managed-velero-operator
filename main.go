package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	corev1 "k8s.io/api/core/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	opmetrics "github.com/openshift/operator-custom-metrics/pkg/metrics"
	"github.com/operator-framework/operator-lib/leader"

	managedv1alpha2 "github.com/openshift/managed-velero-operator/api/v1alpha2"
	veleroctrl "github.com/openshift/managed-velero-operator/controllers/velero"
	"github.com/openshift/managed-velero-operator/pkg/velero"
	"github.com/openshift/managed-velero-operator/version"
	//+kubebuilder:scaffold:imports

	configv1 "github.com/openshift/api/config/v1"
	minterv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monclientv1 "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/typed/monitoring/v1"

	"github.com/cblecker/platformutils"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8383
	operatorMetricsPort int32 = 8686
)

var (
	scheme   = k8sruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

var log = logf.Log.WithName(version.OperatorName)

const (
	WatchNamespaceEnvVar           = "WATCH_NAMESPACE"
	ManagedVeleroOperatorNamespace = "openshift-velero"
	OperatorName                   = "managed-velero-operator"
)

// supportedPlatforms is the list of platform supported by the operator
var supportedPlatforms = []configv1.PlatformType{
	configv1.AWSPlatformType,
	configv1.GCPPlatformType,
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(managedv1alpha2.AddToScheme(scheme))
	utilruntime.Must(configv1.Install(scheme))
	utilruntime.Must(monitoringv1.AddToScheme(scheme))
	utilruntime.Must(minterv1.Install(scheme))
	utilruntime.Must(apiextv1beta1.AddToScheme(scheme))
	utilruntime.Must(velerov1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func printVersion() {
	log.Info(fmt.Sprintf("Operator Version: %s", version.Version))
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8383", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	printVersion()

	namespace, err := getWatchNamespace()
	if err != nil {
		log.Error(err, "Failed to get watch namespace")
		os.Exit(1)
	}

	// The operator makes assumptions about the namespace to configure Velero in.
	// If the operator is deployed in a different namespace than expected, error.
	if namespace != ManagedVeleroOperatorNamespace {
		log.Error(fmt.Errorf("unexpected operator namespace: expected %s, got %s", ManagedVeleroOperatorNamespace, namespace), "")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "fcfdbe85.openshift.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	ctx := context.TODO()
	// Become the leader before proceeding
	err = leader.Become(ctx, "managed-velero-operator-lock")
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Create k8s clients to perform startup tasks
	startupClient, err := crclient.New(cfg, crclient.Options{Scheme: mgr.GetScheme()})
	if err != nil {
		log.Error(err, "Unable to create operator startup client")
		os.Exit(1)
	}
	pc, err := platformutils.NewClient(context.TODO())
	if err != nil {
		log.Error(err, "Unable to create platformutils client")
		os.Exit(1)
	}

	// Get infrastructureStatus so we can discover the platform we are running on
	infraStatus, err := pc.GetInfrastructureStatus()
	if err != nil {
		log.Error(err, "Failed to retrieve infrastructure status")
		os.Exit(1)
	}

	// Verify platform is in support platforms list
	// TODO: expand support to other platforms
	if !platformutils.IsPlatformSupported(infraStatus.PlatformStatus.Type, supportedPlatforms) {
		log.Error(fmt.Errorf("expected %v got %v", supportedPlatforms, infraStatus.PlatformStatus.Type), "Unsupported platform")
		os.Exit(1)
	}

	// Verify all velero CRDs are installed
	if err = velero.InstallVeleroCRDs(log, startupClient); err != nil {
		log.Error(err, "Failed to install Velero CRDs")
		os.Exit(1)
	}

	if err = (&veleroctrl.VeleroInstallReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VeleroInstall")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Add the Metrics Service and ServiceMonitor
	if err := addMetrics(ctx, startupClient, cfg); err != nil {
		log.Error(err, "Metrics service is not added.")
		os.Exit(1)
	}

	log.Info("Starting the Cmd.")

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func getWatchNamespace() (string, error) {
	ns, found := os.LookupEnv(WatchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", WatchNamespaceEnvVar)
	}
	return ns, nil
}

// addMetrics will create the Services and Service Monitors to allow the operator export the metrics by using
// the Prometheus operator
func addMetrics(ctx context.Context, cl crclient.Client, cfg *rest.Config) error {
	service, err := opmetrics.GenerateService(metricsPort, "http-metrics", OperatorName+"-metrics", ManagedVeleroOperatorNamespace, map[string]string{"name": OperatorName})
	if err != nil {
		log.Info("Could not create metrics Service", "error", err.Error())
		return err
	}

	log.Info(fmt.Sprintf("Attempting to create service %s", service.Name))
	err = cl.Create(ctx, service)
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			log.Error(err, "Could not create metrics service")
			return err
		} else {
			log.Info("Metrics service already exists, will not create")
		}
	}

	services := []*corev1.Service{service}
	mclient := monclientv1.NewForConfigOrDie(cfg)
	copts := metav1.CreateOptions{}

	for _, s := range services {
		if s == nil {
			continue
		}

		sm := opmetrics.GenerateServiceMonitor(s)

		// ErrSMMetricsExists is used to detect if the -metrics ServiceMonitor already exists
		var ErrSMMetricsExists = fmt.Sprintf("servicemonitors.monitoring.coreos.com \"%s-metrics\" already exists", OperatorName)

		log.Info(fmt.Sprintf("Attempting to create service monitor %s", sm.Name))
		// TODO: Get SM and compare to see if an UPDATE is required
		_, err := mclient.ServiceMonitors(ManagedVeleroOperatorNamespace).Create(ctx, sm, copts)
		if err != nil {
			if err.Error() != ErrSMMetricsExists {
				return err
			}
			log.Info("ServiceMonitor already exists")
		}
		log.Info(fmt.Sprintf("Successfully configured service monitor %s", sm.Name))
	}
	return nil
}
