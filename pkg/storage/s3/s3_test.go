package s3

import (
	"context"
	"testing"

	logrTesting "github.com/go-logr/logr/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	velerov1alpha2 "github.com/openshift/managed-velero-operator/pkg/apis/managed/v1alpha2"
)

func TestCreateStorage(t *testing.T) {
	t.Run("doesn't reclaim a bucket it can't access", func(t *testing.T) {
		instance := setUpInstance(t)
		driver := setUpDriver(t, instance)

		err := driver.CreateStorage(nullLogr, instance)
		if err != nil {
			t.Fatalf("got an unexpected error: %s", err)
		}
		t.Error("still need to create the test but the code doesn't do this yet")
	})
}

func TestSetInstanceBucketName(t *testing.T) {
	t.Run("sets bucket name in instance status", func(t *testing.T) {
		instance := setUpInstance(t)
		testDriver := setUpDriver(t, instance)

		err := setInstanceBucketName(testDriver, fakeClient, nullLogr, instance)
		if err != nil {
			t.Fatalf("got an unexpected error: %s", err)
		}

		if instance.Status.StorageBucket.Name == "" {
			t.Error("bucket name was empty in the instance")
		}
	})

	t.Run("buckets it can't access", func(t *testing.T) {
		t.Run("aren't reclaimed", func(t *testing.T) {
			instance := setUpInstance(t)
			testDriver := setUpDriver(t, instance)

			err := setInstanceBucketName(testDriver, fakeInconsistentClient, nullLogr, instance)
			if err != nil {
				t.Fatalf("got an unexpected error: %s", err)
			}

			if instance.Status.StorageBucket.Name != "" {
				t.Errorf("instance bucket name: %s, expected it to be unset", instance.Status.StorageBucket.Name)
			}
		})

		t.Run("are labeled do-not-reclaim", func(t *testing.T) {
			instance := setUpInstance(t)
			testDriver := setUpDriver(t, instance)

			err := setInstanceBucketName(testDriver, fakeInconsistentClient, nullLogr, instance)
			if err != nil {
				t.Fatalf("got an unexpected error: %s", err)
			}

			actual := ""
			for _, tag := range fakeInconsistentClient.BucketsTags["inconsistentBucket"].TagSet {
				if *tag.Key == bucketTagDoNotReclaim {
					actual = *tag.Value
				}
			}

			if actual != "true" {
				t.Errorf("expected tag %s to be true, was %s", bucketTagDoNotReclaim, actual)
			}
		})
	})
}

// utilities and variables
var nullLogr = &logrTesting.NullLogger{}

/*
NB: this file shares a packages with bucket_test.go and all the mock aws client
stuff is in there
*/

/*
setUpTestClient sets up a test kube client loaded with a VeleroInstall instance

Parameters:
*testing.T - a pointer to testing.T to run the helper function
*velerov1alpha2.VeleroInstall - a pointer to the VeleroInstall instance
*/
func setUpTestClient(t *testing.T, instance *velerov1alpha2.VeleroInstall) k8sClient.Client {
	s := scheme.Scheme
	s.AddKnownTypes(velerov1alpha2.SchemeGroupVersion, instance)
	objects := []runtime.Object{instance}

	return fake.NewFakeClientWithScheme(s, objects...)
}

/*
setUpInstance sets up a new VeleroInstall instance and returns a pointer to it.
This is to avoid cross-contamination between tests
*/
func setUpInstance(t *testing.T) *velerov1alpha2.VeleroInstall {
	t.Helper()

	return &velerov1alpha2.VeleroInstall{
		metav1.TypeMeta{
			Kind:       "VeleroInstall",
			APIVersion: "managed.openshift.io/v1alpha2v1alpha2",
		},
		metav1.ObjectMeta{
			Name:      "cluster",
			Namespace: "openshift-velero",
		},
		velerov1alpha2.VeleroInstallSpec{},
		velerov1alpha2.VeleroInstallStatus{},
	}
}

/*
setUpDriver creates a new driver and returns a pointer to it. This is to avoid
cross-contamination between tests.

Parameters:
*testing.T - a pointer to testing.T to run the helper funciton
*velerov1alpha2.VeleroInstall - a pointer to the VeleroInstall instance for the driver
*/
func setUpDriver(t *testing.T, instance *velerov1alpha2.VeleroInstall) *driver {
	t.Helper()

	return &driver{
		Config: &S3{
			Region:    region,
			InfraName: clusterInfraName,
		},
		Context:    context.TODO(),
		kubeClient: setUpTestClient(t, instance),
	}
}
