package s3

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
	"github.com/openshift/managed-velero-operator/version"
)

func TestNewS3Client(t *testing.T) {
	instance := setUpInstance(t)

	tests := []struct {
		name        string
		kubeClient  k8sClient.Client
		region      string
		expectError bool
	}{
		{
			name:        "create an S3 client",
			kubeClient:  setUpTestClient(t, instance),
			region:      "us-east-1",
			expectError: false,
		},
		{
			name:        "errors without S3 credentials secret",
			kubeClient:  setUpUnauthedTestClient(t, instance),
			region:      "us-east-1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualClient, err := NewS3Client(tt.kubeClient, tt.region)
			if (err != nil) && (tt.expectError == false) {
				t.Fatalf("got an unexpected error: %s\n", err)
			}

			if (tt.expectError == false) && (reflect.TypeOf(actualClient).String() != "*s3.awsClient") {
				t.Errorf("expected *awsClient got %s", reflect.TypeOf(actualClient))
			}
		})
	}
}

// utils and variables
var testSecret = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: version.OperatorNamespace,
		Name:      awsCredsSecretName,
	},
	Data: map[string][]byte{
		awsCredsSecretIDKey:     []byte("fakeSecretIDKey"),
		awsCredsSecretAccessKey: []byte("fakeSecretAccessKey"),
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
func setUpTestClient(t *testing.T, instance *velerov1alpha2.VeleroInstall) k8sClient.Client {
	s := scheme.Scheme
	s.AddKnownTypes(velerov1alpha2.SchemeGroupVersion, instance)
	objects := []runtime.Object{instance, testSecret}

	return fake.NewFakeClientWithScheme(s, objects...)
}

// setUpUnauthedTestClient sets up a test kube client that is missing the AWS credentials secret
func setUpUnauthedTestClient(t *testing.T, instance *velerov1alpha2.VeleroInstall) k8sClient.Client {
	s := scheme.Scheme
	s.AddKnownTypes(velerov1alpha2.SchemeGroupVersion, instance)
	objects := []runtime.Object{instance}

	return fake.NewFakeClientWithScheme(s, objects...)
}
