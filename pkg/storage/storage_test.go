package storage

import (
	"reflect"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	velerov1alpha2 "github.com/openshift/managed-velero-operator/pkg/apis/managed/v1alpha2"
)

func TestNewDriver(t *testing.T) {
	tests := []struct {
		name          string
		config        *configv1.InfrastructureStatus
		namespace     string
		exptectedType string
		wantError     bool
	}{
		{
			name: "aws driver",
			config: &configv1.InfrastructureStatus{
				InfrastructureName: "managed-velero-fake-cluster",
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "us-east-1",
					},
				},
			},
			namespace:     "openshift-velero",
			exptectedType: "*s3.driver",
			wantError:     false,
		},
		{
			name: "gcs driver",
			config: &configv1.InfrastructureStatus{
				InfrastructureName: "managed-velero-fake-cluster",
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.GCPPlatformType,
					GCP: &configv1.GCPPlatformStatus{
						ProjectID: "",
						Region:    "",
					},
				},
			},
			namespace:     "openshift-velero",
			exptectedType: "*gcs.driver",
			wantError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kubeClient := setUpTestClient(t, setUpInstance(t))

			actualDriver := NewDriver(tt.config, kubeClient, tt.namespace)

			// test that NewDriver actually returns the expected type
			if reflect.TypeOf(actualDriver).String() != tt.exptectedType {
				t.Fatalf("NewDriver(): expected %s got %s", tt.exptectedType, reflect.TypeOf(actualDriver))
			}
		})
	}
}

// utils and variables

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

// setUpTestClient sets up a test kube client
func setUpTestClient(t *testing.T, instance *velerov1alpha2.VeleroInstall) client.Client {
	s := scheme.Scheme
	s.AddKnownTypes(velerov1alpha2.SchemeGroupVersion, instance)
	objects := []runtime.Object{instance}

	return fake.NewFakeClientWithScheme(s, objects...)
}
