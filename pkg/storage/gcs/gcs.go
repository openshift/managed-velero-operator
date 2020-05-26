package gcs

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gstorage "cloud.google.com/go/storage"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	configv1 "github.com/openshift/api/config/v1"
	veleroInstallCR "github.com/openshift/managed-velero-operator/pkg/apis/managed/v1alpha2"
	storageBase "github.com/openshift/managed-velero-operator/pkg/storage/base"
	storageConstants "github.com/openshift/managed-velero-operator/pkg/storage/constants"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GCS struct {
	Region    string
	Project   string
	InfraName string
}

type driver struct {
	storageBase.Driver
	Config *GCS
}

// NewDriver creates a new gcs storage driver
// Used during bootstrapping
func NewDriver(ctx context.Context, cfg *configv1.InfrastructureStatus, clnt client.Client) *driver {
	drv := driver{
		Config: &GCS{
			Region:    cfg.PlatformStatus.GCP.Region,
			Project:   cfg.PlatformStatus.GCP.ProjectID,
			InfraName: cfg.InfrastructureName,
		},
	}
	drv.Context = ctx
	drv.KubeClient = clnt
	return &drv
}

// GetPlatformType returns the platform type of this driver
func (d *driver) GetPlatformType() configv1.PlatformType {
	return configv1.GCPPlatformType
}

// CreateStorage attempts to create a GCS bucket and apply any provided tags
func (d *driver) CreateStorage(reqLogger logr.Logger, instance *veleroInstallCR.VeleroInstall) error {
	var err error

	// Create a GCS client
	gcsClient, err := NewGcsClient(d.KubeClient)
	if err != nil {
		return err
	}

	bucketLog := reqLogger.WithValues("StorageBucket.Name", instance.Status.GCP.StorageBucket.Name, "StorageBucket.Region", d.Config.Region)

	// This switch handles the provisioning steps/checks
	switch {
	// We don't yet have a bucket name selected
	case instance.Status.GCP.StorageBucket.Name == "":

		// Use an existing bucket, if it exists.
		bucketLog.Info("No GCS bucket defined. Searching for existing bucket to use")
		bucketlist, err := d.listBuckets(gcsClient)
		if err != nil {
			return err
		}

		existingBucket := d.findVeleroBucket(bucketlist)
		if existingBucket != "" {
			bucketLog.Info("Recovered existing bucket", "StorageBucket.Name", existingBucket)
			instance.Status.GCP.StorageBucket.Name = existingBucket
			instance.Status.GCP.StorageBucket.Provisioned = true
			return instance.StatusUpdate(reqLogger, d.KubeClient)
		}

		// Prepare to create a new bucket, if none exist.
		proposedName := generateBucketName(storageConstants.StorageBucketPrefix)
		proposedBucketExists, err := d.StorageExists(proposedName)
		if err != nil {
			return err
		}
		if proposedBucketExists {
			return fmt.Errorf("proposed bucket %s already exists, retrying", proposedName)
		}

		bucketLog.Info("Setting proposed bucket name", "StorageBucket.Name", proposedName)
		instance.Status.GCP.StorageBucket.Name = proposedName
		instance.Status.GCP.StorageBucket.Provisioned = false
		return instance.StatusUpdate(reqLogger, d.KubeClient)

	// We have a bucket name, but haven't kicked off provisioning of the bucket yet
	case instance.Status.GCP.StorageBucket.Name != "" && !instance.Status.GCP.StorageBucket.Provisioned:
		bucketLog.Info("GCS bucket defined, but not provisioned")

		// Create GCS bucket
		bucketLog.Info("Creating GCS Bucket")
		err = d.createBucket(gcsClient, instance.Status.GCP.StorageBucket.Name)
		if err != nil {
			return fmt.Errorf("error occurred when creating bucket %v: %v", instance.Status.GCP.StorageBucket.Name, err.Error())
		}
	}

	// Verify GCS bucket exists
	bucketLog.Info("Verifing GCS Bucket exists")
	exists, err := d.StorageExists(instance.Status.GCP.StorageBucket.Name)
	if err != nil {
		return fmt.Errorf("error occurred when verifying bucket %v: %v", instance.Status.GCP.StorageBucket.Name, err.Error())
	}
	if !exists {
		bucketLog.Error(nil, "GCS bucket doesn't appear to exist")
		instance.Status.GCP.StorageBucket.Provisioned = false
		return instance.StatusUpdate(reqLogger, d.KubeClient)
	}

	//TODO(cblecker): ACL enforcement

	//TODO(cblecker): Lifecycle enforcement

	// Make sure that tags are applied to buckets
	bucketLog.Info("Enforcing GCS Bucket tags on GCS Bucket")
	err = d.enforceBucketLabels(gcsClient, instance.Status.GCP.StorageBucket.Name)
	if err != nil {
		return fmt.Errorf("error occurred when tagging bucket %v: %v", instance.Status.GCP.StorageBucket.Name, err.Error())
	}

	instance.Status.GCP.StorageBucket.Provisioned = true
	instance.Status.GCP.StorageBucket.LastSyncTimestamp = &metav1.Time{
		Time: time.Now(),
	}
	return instance.StatusUpdate(reqLogger, d.KubeClient)

}

// StorageExists checks that the bucket exists, and that we have access to it.
func (d *driver) StorageExists(bucketName string) (bool, error) {
	var err error

	//create an GCS Client
	gcsClient, err := NewGcsClient(d.KubeClient)
	if err != nil {
		return false, err
	}

	_, err = gcsClient.Bucket(bucketName).Attrs(context.TODO())
	if err != nil {
		if err == gstorage.ErrBucketNotExist {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

//generateBucketName generates a proposed name for the GCS Bucket
func generateBucketName(prefix string) string {
	id := uuid.New().String()
	return prefix + id
}
