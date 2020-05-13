package velero

import (
	"context"

	"github.com/openshift/managed-velero-operator/pkg/storage/gcs"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1 "github.com/openshift/api/config/v1"
	minterv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	veleroInstall "github.com/vmware-tanzu/velero/pkg/install"
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

func (r *ReconcileVeleroGCP) CredentialsRequest(namespace, bucketName string) (*minterv1.CredentialsRequest, error) {
	codec, _ := minterv1.NewCodec()
	providerSpec, _ := codec.EncodeProviderSpec(
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
			Name:      credentialsRequestName,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "CredentialsRequest",
			APIVersion: minterv1.SchemeGroupVersion.String(),
		},
		Spec: minterv1.CredentialsRequestSpec{
			SecretRef: corev1.ObjectReference{
				Name:      credentialsRequestName,
				Namespace: namespace,
			},
			ProviderSpec: providerSpec,
		},
	}, nil
}

func (r *ReconcileVeleroGCP) VeleroDeployment(namespace string) *appsv1.Deployment {
	imageRegistry := r.GetImageRegistry()

	deployment := veleroInstall.Deployment(
		namespace,
		veleroInstall.WithPlugins(
			[]string{imageRegistry + "/" + veleroGcpImageTag}),
		veleroInstall.WithImage(
			imageRegistry + "/" + veleroImageTag))

	defaultMode := int32(40)
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

	deployment.Spec.Template.Spec.Containers[0].Env = append(
		deployment.Spec.Template.Spec.Containers[0].Env,
		[]corev1.EnvVar{
			{
				Name:  "GOOGLE_APPLICATION_CREDENTIALS",
				Value: "/credentials/service_account.json",
			},
		}...
	)

	return finishVeleroDeployment(deployment)
}
