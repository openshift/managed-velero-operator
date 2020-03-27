package v1alpha2

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStorageBucketReconcileRequired(t *testing.T) {
	var testcases = []struct {
		testName          string
		bucketName        string
		bucketProvisioned bool
		shouldReconcile   bool
		timestamp         time.Time
		reconcilePeriod   time.Duration
	}{
		{
			testName:          "default bucket, provisioned 30 mins ago, 60 minute period",
			bucketName:        "test-bucket",
			bucketProvisioned: true,
			timestamp:         time.Now().Add(-time.Minute * 30),
			reconcilePeriod:   time.Minute * 60,
			shouldReconcile:   false,
		},
		{
			testName:          "bucket name empty",
			bucketName:        "",
			bucketProvisioned: true,
			timestamp:         time.Now(),
			reconcilePeriod:   time.Minute * 60,
			shouldReconcile:   true,
		},
		{
			testName:          "bucket not provioned",
			bucketName:        "test-bucket",
			bucketProvisioned: false,
			timestamp:         time.Now(),
			reconcilePeriod:   time.Minute * 60,
			shouldReconcile:   true,
		},
		{
			testName:          "timestamp is epoch",
			bucketName:        "test-bucket",
			bucketProvisioned: true,
			timestamp:         time.Unix(0, 0),
			reconcilePeriod:   time.Minute * 60,
			shouldReconcile:   true,
		},
		{
			testName:          "timestamp is unset",
			bucketName:        "test-bucket",
			bucketProvisioned: true,
			reconcilePeriod:   time.Minute * 60,
			shouldReconcile:   true,
		},
	}

	for _, tc := range testcases {
		t.Logf("Running scenario %q", tc.testName)

		instance := &VeleroInstall{
			Spec: VeleroInstallSpec{},
			Status: VeleroInstallStatus{
				StorageBucket: StorageBucket{
					Name:        tc.bucketName,
					Provisioned: tc.bucketProvisioned,
				},
			},
		}

		if !tc.timestamp.IsZero() {
			instance.Status.StorageBucket.LastSyncTimestamp = &metav1.Time{Time: tc.timestamp}
		}

		reconcile := instance.StorageBucketReconcileRequired(tc.reconcilePeriod)

		if reconcile != tc.shouldReconcile {
			if tc.shouldReconcile {
				t.Errorf("did not reconcile when expecting one")
			} else {
				t.Errorf("reconciled when not expecting one")
			}
		}
	}
}
