package v1alpha2

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// StorageBucketReconcileRequired determines if bucket reconsile is required
func (i *VeleroInstall) StorageBucketReconcileRequired(platformType configv1.PlatformType, reconcilePeriod time.Duration) bool {
	// If any of the following are true, reconcile the storage bucket:
	// - Name is empty
	// - Provisioned is false
	// - The LastSyncTimestamp is unset
	// - It's been longer than 1 hour since last sync
	// - StorageAccount is empty(only for Azure)
	switch platformType {
	case configv1.AWSPlatformType:
		if i.Status.AWS.StorageBucket.Name == "" ||
			!i.Status.AWS.StorageBucket.Provisioned ||
			i.Status.AWS.StorageBucket.LastSyncTimestamp.IsZero() ||
			time.Since(i.Status.AWS.StorageBucket.LastSyncTimestamp.Time) > reconcilePeriod {
			return true
		}
	case configv1.GCPPlatformType:
		if i.Status.GCP.StorageBucket.Name == "" ||
			!i.Status.GCP.StorageBucket.Provisioned ||
			i.Status.GCP.StorageBucket.LastSyncTimestamp.IsZero() ||
			time.Since(i.Status.GCP.StorageBucket.LastSyncTimestamp.Time) > reconcilePeriod {
			return true
		}
	case configv1.AzurePlatformType:
		if *i.Status.Azure.StorageAccount == "" ||
			i.Status.Azure.StorageBucket.Name == "" ||
			!i.Status.Azure.StorageBucket.Provisioned ||
			i.Status.Azure.StorageBucket.LastSyncTimestamp.IsZero() ||
			time.Since(i.Status.Azure.StorageBucket.LastSyncTimestamp.Time) > reconcilePeriod {
			return true
		}
	}

	return false
}

func (i *VeleroInstall) StatusUpdate(reqLogger logr.Logger, kubeClient client.Client) error {
	err := kubeClient.Status().Update(context.TODO(), i)
	if err != nil {
		reqLogger.Error(err, fmt.Sprintf("Status update for %s failed", i.Name))
	} else {
		reqLogger.Info(fmt.Sprintf("Status updated for %s", i.Name))
	}
	return err
}
