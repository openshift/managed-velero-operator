package s3

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	velerov1alpha2 "github.com/openshift/managed-velero-operator/pkg/apis/managed/v1alpha2"
)

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

// setUpTestClient sets up a test kube client loaded with a VeleroInstall instance
func setUpTestClient(t *testing.T, instance *velerov1alpha2.VeleroInstall) k8sClient.Client {
	s := scheme.Scheme
	s.AddKnownTypes(velerov1alpha2.SchemeGroupVersion, instance)
	objects := []runtime.Object{instance}

	return fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objects...).Build()
}
