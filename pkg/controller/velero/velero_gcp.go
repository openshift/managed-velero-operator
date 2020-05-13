package velero

import (
	"context"

	"github.com/openshift/managed-velero-operator/pkg/storage/gcs"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1 "github.com/openshift/api/config/v1"
	minterv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"

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
