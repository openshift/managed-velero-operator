package s3

import (
	"context"
	"testing"

	logrTesting "github.com/go-logr/logr/testing"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	velerov1alpha2 "github.com/openshift/managed-velero-operator/pkg/apis/managed/v1alpha2"
)

func TestCreateStorage(t *testing.T) {
	t.Run("doesn't reclaim a bucket it can't access", func(t *testing.T) {
		t.Error("still need to create the test but the code doesn't do this yet")
	})
}

func TestSetInstanceBucketName(t *testing.T) {
	t.Run("sets bucket name in instance status", func(t *testing.T) {
		err := setInstanceBucketName(testDriver, fakeClient, nullLogr, instance)
		if err != nil {
			t.Fatalf("got an unexpected error: %s", err)
		}

		if instance.Status.StorageBucket.Name == "" {
			t.Error("bucket name was empty in the instance")
		}
	})

	t.Run("doesn't reclaim a bucket it can't access", func(t *testing.T) {
		t.Error("still need to create the test but the code doesn't do this yet")
	})
}

// utilities and variables
var nullLogr = &logrTesting.NullLogger{}
var instance = &velerov1alpha2.VeleroInstall{}
var testDriver = &driver{
	Config: &S3{
		Region:    region,
		InfraName: clusterInfraName,
	},
	Context:    context.TODO(),
	kubeClient: setUpTestClient(),
}

/*
NB: this file shares a packages with bucket_test.go and all the mock aws client
stuff is in there
*/

/*
setUpTestClient sets up a test kube client loaded with a specified let's
encrypt account secret or aws platformsecret (in the certificaterequest)

Parameters:
t *testing.T - Testing framework hookup. the argument should always be `t` from
the calling function.
*/
func setUpTestClient() (testClient k8sClient.Client) {
	s := scheme.Scheme
	s.AddKnownTypes(velerov1alpha2.SchemeGroupVersion, instance)
	objects := []runtime.Object{instance}

	testClient = fake.NewFakeClientWithScheme(s, objects...)
	return
}
