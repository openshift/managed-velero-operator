package velero

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	veleroInstallCR "github.com/openshift/managed-velero-operator/pkg/apis/managed/v1alpha2"

	configv1 "github.com/openshift/api/config/v1"
	minterv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	veleroInstall "github.com/vmware-tanzu/velero/pkg/install"

	"github.com/go-logr/logr"
	"github.com/openshift/managed-velero-operator/version"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/intstr"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	veleroImageRegistry   = "docker.io/velero"
	veleroImageRegistryCN = "registry.docker-cn.com/velero"

	veleroImageTag    = "velero:v1.6.0"
	veleroAwsImageTag = "velero-plugin-for-aws:v1.2.0"
	veleroGcpImageTag = "velero-plugin-for-gcp:v1.2.0"

	credentialsRequestName = "velero-iam-credentials"
)

var (
	awsChinaRegions = []string{"cn-north-1", "cn-northwest-1"}
)

var (
	awsCredsSecretName = version.OperatorName + "-iam-credentials"
)

func (r *ReconcileVelero) provisionVelero(reqLogger logr.Logger, namespace string, platformStatus *configv1.PlatformStatus, instance *veleroInstallCR.VeleroInstall) (reconcile.Result, error) {
	var err error

	var locationConfig map[string]string
	switch r.driver.GetPlatformType() {
	case configv1.AWSPlatformType:
		locationConfig = map[string]string{
			"region": platformStatus.AWS.Region,
		}
	case configv1.GCPPlatformType:
		// No region configuration needed for GCP
	default:
		return reconcile.Result{}, fmt.Errorf("unable to determine platform")
	}

	provider := strings.ToLower(string(r.driver.GetPlatformType()))

	// Install BackupStorageLocation
	foundBsl := &velerov1.BackupStorageLocation{}
	var caCertData []byte
	bsl := veleroInstall.BackupStorageLocation(namespace, provider, instance.Status.StorageBucket.Name, "", locationConfig, caCertData)
	if err = r.client.Get(context.TODO(), runtimeClient.ObjectKeyFromObject(bsl), foundBsl); err != nil {
		if errors.IsNotFound(err) {
			// Didn't find BackupStorageLocation
			reqLogger.Info("Creating BackupStorageLocation")
			if err := controllerutil.SetControllerReference(instance, bsl, r.scheme); err != nil {
				return reconcile.Result{}, err
			}
			if err = r.client.Create(context.TODO(), bsl); err != nil {
				return reconcile.Result{}, err
			}
		} else {
			return reconcile.Result{}, err
		}
	} else {
		// BackupStorageLocation exists, check if it's updated.
		if !reflect.DeepEqual(foundBsl.Spec, bsl.Spec) {
			// Specs aren't equal, update and fix.
			reqLogger.Info("Updating BackupStorageLocation", "foundBsl.Spec", foundBsl.Spec, "bsl.Spec", bsl.Spec)
			foundBsl.Spec = *bsl.Spec.DeepCopy()
			if err = r.client.Update(context.TODO(), foundBsl); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	// Install VolumeSnapshotLocation
	foundVsl := &velerov1.VolumeSnapshotLocation{}
	vsl := veleroInstall.VolumeSnapshotLocation(namespace, provider, locationConfig)
	if err = r.client.Get(context.TODO(), runtimeClient.ObjectKeyFromObject(vsl), foundVsl); err != nil {
		if errors.IsNotFound(err) {
			// Didn't find VolumeSnapshotLocation
			reqLogger.Info("Creating VolumeSnapshotLocation")
			if err := controllerutil.SetControllerReference(instance, vsl, r.scheme); err != nil {
				return reconcile.Result{}, err
			}
			if err = r.client.Create(context.TODO(), vsl); err != nil {
				return reconcile.Result{}, err
			}
		} else {
			return reconcile.Result{}, err
		}
	} else {
		// VolumeSnapshotLocation exists, check if it's updated.
		if !reflect.DeepEqual(foundVsl.Spec, vsl.Spec) {
			// Specs aren't equal, update and fix.
			reqLogger.Info("Updating VolumeSnapshotLocation", "foundVsl.Spec", foundVsl.Spec, "vsl.Spec", vsl.Spec)
			foundVsl.Spec = *vsl.Spec.DeepCopy()
			if err = r.client.Update(context.TODO(), foundVsl); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	// Install CredentialsRequest
	foundCr := &minterv1.CredentialsRequest{}
	var cr *minterv1.CredentialsRequest
	switch r.driver.GetPlatformType() {
	case configv1.AWSPlatformType:
		// No credentialsRequest needed for aws
	case configv1.GCPPlatformType:
		cr = gcpCredentialsRequest(namespace, credentialsRequestName)
	default:
		return reconcile.Result{}, fmt.Errorf("unable to determine platform")
	}
	if err = r.client.Get(context.TODO(), runtimeClient.ObjectKeyFromObject(cr), foundCr); err != nil {
		if errors.IsNotFound(err) {
			// Didn't find CredentialsRequest
			reqLogger.Info("Creating CredentialsRequest")
			if err := controllerutil.SetControllerReference(instance, cr, r.scheme); err != nil {
				return reconcile.Result{}, err
			}
			if err = r.client.Create(context.TODO(), cr); err != nil {
				return reconcile.Result{}, err
			}
		} else {
			return reconcile.Result{}, err
		}
	} else {
		// CredentialsRequest exists, check if it's updated.
		crEqual, err := credentialsRequestSpecEqual(foundCr.Spec, cr.Spec)
		if err != nil {
			return reconcile.Result{}, err
		}
		if !crEqual {
			// Specs aren't equal, update and fix.
			reqLogger.Info("Updating CredentialsRequest", "foundCr.Spec", foundCr.Spec, "cr.Spec", cr.Spec)
			foundCr.Spec = *cr.Spec.DeepCopy()
			if err = r.client.Update(context.TODO(), foundCr); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	// Install Deployment
	foundDeployment := &appsv1.Deployment{}
	deployment := veleroDeployment(namespace, r.driver.GetPlatformType(), determineVeleroImageRegistry(r.driver.GetPlatformType(), locationConfig["region"]))
	if err = r.client.Get(context.TODO(), runtimeClient.ObjectKeyFromObject(deployment), foundDeployment); err != nil {
		if errors.IsNotFound(err) {
			// Didn't find Deployment
			reqLogger.Info("Creating Deployment")
			if err := controllerutil.SetControllerReference(instance, deployment, r.scheme); err != nil {
				return reconcile.Result{}, err
			}
			if err = r.client.Create(context.TODO(), deployment); err != nil {
				return reconcile.Result{}, err
			}
		} else {
			return reconcile.Result{}, err
		}
	} else {
		// Deployment exists, check if it's updated.
		if !reflect.DeepEqual(foundDeployment.Spec, deployment.Spec) {
			// Specs aren't equal, update and fix.
			reqLogger.Info("Updating Deployment", "foundDeployment.Spec", foundDeployment.Spec, "deployment.Spec", deployment.Spec)
			foundDeployment.Spec = *deployment.Spec.DeepCopy()
			if err = r.client.Update(context.TODO(), foundDeployment); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	// Install Metrics Service
	foundService := &corev1.Service{}
	service := metricsServiceFromDeployment(deployment)
	if err = r.client.Get(context.TODO(), runtimeClient.ObjectKeyFromObject(service), foundService); err != nil {
		if errors.IsNotFound(err) {
			// Didn't find Service
			reqLogger.Info("Creating Service")
			if err := controllerutil.SetControllerReference(instance, service, r.scheme); err != nil {
				return reconcile.Result{}, err
			}
			if err = r.client.Create(context.TODO(), service); err != nil {
				return reconcile.Result{}, err
			}
			// We need a populated foundService (with a UID) to generate the
			// ServiceMonitor below, so requeue and fetch it on the next pass.
			return reconcile.Result{Requeue: true}, nil
		}
		return reconcile.Result{}, err
	}
	// Service exists, check if it's updated.
	// Note: We leave Spec.ClusterIP unspecified for the master to set.
	//       Copy it from foundService to satisfy reflect.DeepEqual.
	service.Spec.ClusterIP = foundService.Spec.ClusterIP
	if !reflect.DeepEqual(foundService.Spec, service.Spec) {
		// Specs aren't equal, update and fix.
		reqLogger.Info("Updating Service", "foundService.Spec", foundService.Spec, "service.Spec", service.Spec)
		foundService.Spec = *service.Spec.DeepCopy()
		if err = r.client.Update(context.TODO(), foundService); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Install Metrics ServiceMonitor
	foundServiceMonitor := &monitoringv1.ServiceMonitor{}
	serviceMonitor := generateServiceMonitor(foundService)
	if err = r.client.Get(context.TODO(), runtimeClient.ObjectKeyFromObject(serviceMonitor), foundServiceMonitor); err != nil {
		if errors.IsNotFound(err) {
			// Didn't find ServiceMonitor
			reqLogger.Info("Creating ServiceMonitor")
			// Note, generateServiceMonitor already set an owner reference.
			if err = r.client.Create(context.TODO(), serviceMonitor); err != nil {
				return reconcile.Result{}, err
			}
		} else {
			return reconcile.Result{}, err
		}
	} else {
		// ServiceMonitor exists, check if it's updated.
		if !reflect.DeepEqual(foundServiceMonitor.Spec, serviceMonitor.Spec) {
			// Specs aren't equal, update and fix.
			reqLogger.Info("Updating ServiceMonitor", "foundServiceMonitor.Spec", foundServiceMonitor.Spec, "serviceMonitor.Spec", serviceMonitor.Spec)
			foundServiceMonitor.Spec = *serviceMonitor.Spec.DeepCopy()
			if err = r.client.Update(context.TODO(), foundServiceMonitor); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	return reconcile.Result{}, nil
}

func gcpCredentialsRequest(namespace, name string) *minterv1.CredentialsRequest {
	codec, _ := minterv1.NewCodec()
	provSpec, _ := codec.EncodeProviderSpec(
		&minterv1.GCPProviderSpec{
			TypeMeta: metav1.TypeMeta{
				Kind: "GCPProviderSpec",
			},
			PredefinedRoles: []string{
				"roles/compute.storageAdmin",
				"roles/iam.serviceAccountUser",
				"roles/cloudmigration.storageaccess",
			},
			SkipServiceCheck: true,
		})

	return &minterv1.CredentialsRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "CredentialsRequest",
			APIVersion: minterv1.SchemeGroupVersion.String(),
		},
		Spec: minterv1.CredentialsRequestSpec{
			SecretRef: corev1.ObjectReference{
				Name:      name,
				Namespace: namespace,
			},
			ProviderSpec: provSpec,
		},
	}
}

func veleroDeployment(namespace string, platform configv1.PlatformType, veleroImageRegistry string) *appsv1.Deployment {
	var deployment *appsv1.Deployment

	//TODO(cblecker): fix resources
	// veleroPodResources, _ := velerokubeutil.ParseResourceRequirements(veleroInstall.DefaultVeleroPodCPURequest, veleroInstall.DefaultVeleroPodMemRequest, veleroInstall.DefaultVeleroPodCPULimit, veleroInstall.DefaultVeleroPodMemLimit)

	switch platform {
	case configv1.AWSPlatformType:
		deployment = veleroInstall.Deployment(namespace,
			//TODO(cblecker): fix resources
			// veleroInstall.WithResources(veleroPodResources),
			veleroInstall.WithPlugins([]string{veleroImageRegistry + "/" + veleroAwsImageTag}),
			veleroInstall.WithImage(veleroImageRegistry+"/"+veleroImageTag),
		)
		defaultMode := int32(420)
		deployment.Spec.Template.Spec.Volumes = append(
			deployment.Spec.Template.Spec.Volumes,
			corev1.Volume{
				Name: "cloud-credentials",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  awsCredsSecretName,
						DefaultMode: &defaultMode,
					},
				},
			},
		)

		deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(
			deployment.Spec.Template.Spec.Containers[0].VolumeMounts,
			corev1.VolumeMount{
				Name:      "cloud-credentials",
				MountPath: "/.aws",
			},
		)
	case configv1.GCPPlatformType:
		deployment = veleroInstall.Deployment(namespace,
			//TODO(cblecker): fix resources
			// veleroInstall.WithResources(veleroPodResources),
			veleroInstall.WithPlugins([]string{veleroImageRegistry + "/" + veleroGcpImageTag}),
			veleroInstall.WithImage(veleroImageRegistry+"/"+veleroImageTag),
		)
		defaultMode := int32(420)
		deployment.Spec.Template.Spec.Volumes = append(
			deployment.Spec.Template.Spec.Volumes,
			corev1.Volume{
				Name: "cloud-credentials",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  credentialsRequestName,
						DefaultMode: &defaultMode,
					},
				},
			},
		)

		deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(
			deployment.Spec.Template.Spec.Containers[0].VolumeMounts,
			corev1.VolumeMount{
				Name:      "cloud-credentials",
				MountPath: "/credentials",
			},
		)

		deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env, []corev1.EnvVar{
			{
				Name:  "GOOGLE_APPLICATION_CREDENTIALS",
				Value: "/credentials/service_account.json",
			},
		}...)
	}

	replicas := int32(1)
	terminationGracePeriodSeconds := int64(30)
	revisionHistoryLimit := int32(2)
	progressDeadlineSeconds := int32(600)
	maxUnavailable := intstr.FromString("25%")
	maxSurge := intstr.FromString("25%")
	deployment.Spec.Replicas = &replicas
	deployment.Spec.RevisionHistoryLimit = &revisionHistoryLimit
	deployment.Spec.ProgressDeadlineSeconds = &progressDeadlineSeconds
	deployment.Spec.Template.Spec.InitContainers[0].TerminationMessagePath = "/dev/termination-log"
	deployment.Spec.Template.Spec.InitContainers[0].TerminationMessagePolicy = "File"
	deployment.Spec.Template.Spec.Containers[0].Env[1].ValueFrom.FieldRef.APIVersion = "v1"
	deployment.Spec.Template.Spec.Containers[0].Ports[0].Protocol = "TCP"
	deployment.Spec.Template.Spec.Containers[0].TerminationMessagePath = "/dev/termination-log"
	deployment.Spec.Template.Spec.Containers[0].TerminationMessagePolicy = "File"
	deployment.Spec.Template.Spec.DeprecatedServiceAccount = "velero"
	deployment.Spec.Template.Spec.DNSPolicy = "ClusterFirst"
	deployment.Spec.Template.Spec.SchedulerName = "default-scheduler"
	deployment.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{}
	deployment.Spec.Template.Spec.TerminationGracePeriodSeconds = &terminationGracePeriodSeconds
	deployment.Spec.Strategy = appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxUnavailable: &maxUnavailable,
			MaxSurge:       &maxSurge,
		},
	}
	deployment.Spec.Template.Spec.Tolerations = []corev1.Toleration{
		{
			Key:      "node-role.kubernetes.io/infra",
			Operator: corev1.TolerationOpExists,
			Effect:   corev1.TaintEffectNoSchedule,
		},
	}
	deployment.Spec.Template.Spec.Affinity = &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
				{
					Weight: 1,
					Preference: corev1.NodeSelectorTerm{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "node-role.kubernetes.io/infra",
								Operator: corev1.NodeSelectorOpExists,
							},
						},
					},
				},
			},
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "beta.kubernetes.io/arch",
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{"amd64"},
							},
						},
					},
				},
			},
		},
	}

	return deployment
}

func metricsServiceFromDeployment(deployment *appsv1.Deployment) *corev1.Service {
	// Build a list of ServicePorts from the container ports of the
	// deployment's pod template having "metrics" in the port name.
	var servicePorts []corev1.ServicePort
	for _, container := range deployment.Spec.Template.Spec.Containers {
		for _, port := range container.Ports {
			if strings.Contains(port.Name, "metrics") {
				servicePorts = append(servicePorts, corev1.ServicePort{
					Name:       port.Name,
					Protocol:   port.Protocol,
					Port:       port.ContainerPort,
					TargetPort: intstr.FromInt(int(port.ContainerPort)),
				})
			}
		}
	}

	// Copy labels from the deployment's pod template.
	serviceSelector := make(map[string]string)
	// XXX Looping in lieu of an ObjectMeta.CopyLabels() method.
	for k, v := range deployment.Spec.Template.ObjectMeta.Labels {
		serviceSelector[k] = v
	}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployment.ObjectMeta.Name + "-metrics",
			Namespace: deployment.ObjectMeta.Namespace,
			Labels:    map[string]string{"name": deployment.ObjectMeta.Name},
		},
		Spec: corev1.ServiceSpec{
			Ports:           servicePorts,
			Selector:        serviceSelector,
			Type:            corev1.ServiceTypeClusterIP,
			SessionAffinity: corev1.ServiceAffinityNone,
		},
	}
}

func determineVeleroImageRegistry(platform configv1.PlatformType, region string) string {
	if platform == configv1.AWSPlatformType {
		// Use the image in Chinese mirror if running on AWS China
		for _, v := range awsChinaRegions {
			if region == v {
				return veleroImageRegistryCN
			}
		}
	}

	// Use global image by default
	return veleroImageRegistry
}

func credentialsRequestSpecEqual(x, y minterv1.CredentialsRequestSpec) (bool, error) {
	var err error

	// Create new scheme for CredentialsRequest
	credentialsRequestScheme := runtime.NewScheme()

	// Add Cloud Credential apis to scheme
	if err := minterv1.AddToScheme(credentialsRequestScheme); err != nil {
		return false, err
	}

	// Create decoder to allow us to read CredentialsRequest API types
	credentialsRequestDecoder := serializer.NewCodecFactory(credentialsRequestScheme).UniversalDecoder(minterv1.SchemeGroupVersion)

	// Decode the ProviderSpecs for both objects
	xps, err := runtime.Decode(credentialsRequestDecoder, x.ProviderSpec.Raw)
	if err != nil {
		return false, err
	}
	yps, err := runtime.Decode(credentialsRequestDecoder, y.ProviderSpec.Raw)
	if err != nil {
		return false, err
	}

	// Check ProviderSpec matches
	if !reflect.DeepEqual(xps, yps) {
		return false, nil
	}

	// nil out the ProviderSpec and check everyhing else matches
	x.ProviderSpec = nil
	y.ProviderSpec = nil
	if !reflect.DeepEqual(x, y) {
		return false, nil
	}

	return true, nil
}

// generateServiceMonitor generates a prometheus-operator ServiceMonitor object
// based on the passed Service object.
func generateServiceMonitor(s *corev1.Service) *monitoringv1.ServiceMonitor {
	labels := make(map[string]string)
	for k, v := range s.ObjectMeta.Labels {
		labels[k] = v
	}
	endpoints := populateEndpointsFromServicePorts(s)
	boolTrue := true

	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.ObjectMeta.Name,
			Namespace: s.ObjectMeta.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "v1",
					BlockOwnerDeletion: &boolTrue,
					Controller:         &boolTrue,
					Kind:               "Service",
					Name:               s.Name,
					UID:                s.UID,
				},
			},
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: labels,
			},
			Endpoints: endpoints,
		},
	}
}

func populateEndpointsFromServicePorts(s *corev1.Service) []monitoringv1.Endpoint {
	var endpoints []monitoringv1.Endpoint
	for _, port := range s.Spec.Ports {
		endpoints = append(endpoints, monitoringv1.Endpoint{Port: port.Name})
	}
	return endpoints
}
