package v1alpha1

import (
	"time"
)

func (i *Velero) S3BucketReconcileRequired(reconcilePeriod time.Duration) bool {
	// If any of the following are true, reconcile the S3 bucket:
	// - Name is empty
	// - Provisioned is false
	// - The LastSyncTimestamp is unset
	// - It's been longer than 1 hour since last sync
	if i.Status.S3Bucket.Name == "" ||
		!i.Status.S3Bucket.Provisioned ||
		i.Status.S3Bucket.LastSyncTimestamp.IsZero() ||
		time.Since(i.Status.S3Bucket.LastSyncTimestamp.Time) > reconcilePeriod {
		return true
	}

	return false
}
