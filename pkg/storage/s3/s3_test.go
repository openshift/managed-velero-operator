package s3

import (
	"context"
	"testing"

	logrTesting "github.com/go-logr/logr/testing"

	velerov1alpha2 "github.com/openshift/managed-velero-operator/pkg/apis/managed/v1alpha2"
)

func TestSetInstanceBucketName(t *testing.T) {
	tests := []struct {
		name            string
		awsClient       *mockAWSClient
		bucketName      string
		matchBucketName bool
	}{
		{
			name:            "set new bucket name in instance status",
			awsClient:       fakeEmptyClient,
			bucketName:      "",
			matchBucketName: false,
		},
		{
			name:            "set recovered bucket name in instance status",
			awsClient:       fakeClient,
			bucketName:      "testBucket",
			matchBucketName: true,
		},
		{
			name:            "don't reclaim inaccessible bucket",
			awsClient:       fakeInconsistentClient,
			bucketName:      "inconsistentBucket",
			matchBucketName: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := setUpInstance(t)
			testDriver := setUpDriver(t, instance)

			err := setInstanceBucketName(testDriver, tt.awsClient, nullLogr, instance)
			if err != nil {
				t.Fatalf("got an unexpected error: %s", err)
			}

			// if the instace status' bucket name doesn't match the specified bucket name but is supposed to
			if (instance.Status.StorageBucket.Name != tt.bucketName) && tt.matchBucketName {
				t.Errorf("setInstanceBucketName() bucket name: %s, expected %s", instance.Status.StorageBucket.Name, tt.bucketName)
			}

			// if the instance status' bucket name matches the specified bucket name but isn't supposed to
			if (instance.Status.StorageBucket.Name == tt.bucketName) && !tt.matchBucketName {
				t.Errorf("setInstanceBucketName() bucket name: %s, didn't expect %s", instance.Status.StorageBucket.Name, tt.bucketName)
			}
		})
	}
}

// utilities and variables
var nullLogr = &logrTesting.NullLogger{}

// NB: this file shares a packages with bucket_test.go and all the mock aws client
// stuff is in there

// setUpDriver creates a new driver and returns a pointer to it. This is to avoid
// cross-contamination between tests.
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
