package gcs

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	velerov1alpha2 "github.com/openshift/managed-velero-operator/pkg/apis/managed/v1alpha2"
)

func TestNewGcsClient(t *testing.T) {
	instance := setUpInstance(t)

	tests := []struct {
		name        string
		kubeClient  k8sClient.Client
		namespace   string
		region      string
		expectError bool
	}{
		{
			name:        "create a Gcs client",
			kubeClient:  setUpTestClient(t, instance, testSecret),
			namespace:   "openshift-velero",
			expectError: false,
		},
		{
			name:        "errors without Gcs credentials secret",
			kubeClient:  setUpEmptyTestClient(t, instance),
			namespace:   "openshift-velero",
			expectError: true,
		},
		{
			name:        "errors with unauthed Gcs credentials secret",
			kubeClient:  setUpTestClient(t, instance, emptySecret),
			namespace:   "openshift-velero",
			expectError: true,
		},
		{
			name:        "errors with broken Gcs credentials secret",
			kubeClient:  setUpTestClient(t, instance, brokenSecret),
			namespace:   "openshift-velero",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualClient, err := NewGcsClient(tt.kubeClient, tt.namespace)

			if (err != nil) && !tt.expectError {
				t.Fatalf("got an unexpected error: %s\n", err)
			}

			if !tt.expectError && (reflect.TypeOf(actualClient).String() != "stiface.client") {
				t.Errorf("expected stiface.client got %s", reflect.TypeOf(actualClient))
			}
		})
	}
}

// utils and variables
var testSecret = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "openshift-velero",
		Name:      storageCredsSecretName,
	},
	Data: map[string][]byte{
		"service_account.json": []byte(`
		{
			"type": "service_account",
			"project_id": "",
			"private_key_id": "",
			"private_key": "",
			"client_email": "",
			"client_id": "",
			"auth_uri": "https://accounts.google.com/o/oauth2/auth",
			"token_uri": "https://oauth2.googleapis.com/token",
			"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
			"client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/osd-v4-zlqlq-velero-iam--48xnl%40o-3e859ad4.iam.gserviceaccount.com"
		}
		`),
	},
}

var emptySecret = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "openshift-velero",
		Name:      storageCredsSecretName,
	},
	Data: map[string][]byte{},
}

var brokenSecret = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "openshift-velero",
		Name:      storageCredsSecretName,
	},
	Data: map[string][]byte{
		"service_account.json": []byte(" { } "),
	},
}

// setUpInstance sets up a new VeleroInstall instance and returns a pointer to it.
// This is to avoid cross-contamination between tests
func setUpInstance(t *testing.T) *velerov1alpha2.VeleroInstall {
	t.Helper()

	return &velerov1alpha2.VeleroInstall{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VeleroInstall",
			APIVersion: "managed.openshift.io/v1alpha2v1alpha2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster",
			Namespace: "openshift-velero",
		},
		Spec:   velerov1alpha2.VeleroInstallSpec{},
		Status: velerov1alpha2.VeleroInstallStatus{},
	}
}

// setUpTestClient sets up a test kube client loaded with a VeleroInstall instance
// and a specified secret
func setUpTestClient(t *testing.T, instance *velerov1alpha2.VeleroInstall, secret *corev1.Secret) k8sClient.Client {
	s := scheme.Scheme
	s.AddKnownTypes(velerov1alpha2.SchemeGroupVersion, instance)
	objects := []runtime.Object{instance, secret}

	return fake.NewFakeClientWithScheme(s, objects...)
}

// setUpEmptyTestClient sets up a test kube client that doesn't have a GCP secret
func setUpEmptyTestClient(t *testing.T, instance *velerov1alpha2.VeleroInstall) k8sClient.Client {
	s := scheme.Scheme
	s.AddKnownTypes(velerov1alpha2.SchemeGroupVersion, instance)
	objects := []runtime.Object{instance}

	return fake.NewFakeClientWithScheme(s, objects...)
}
