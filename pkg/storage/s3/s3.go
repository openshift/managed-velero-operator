package s3

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	configv1 "github.com/openshift/api/config/v1"
	veleroInstallCR "github.com/openshift/managed-velero-operator/pkg/apis/managed/v1alpha2"
	storageConstants "github.com/openshift/managed-velero-operator/pkg/storage/constants"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type S3 struct {
	Region    string
	InfraName string
}

type driver struct {
	Config     *S3
	Context    context.Context
	kubeClient client.Client
}

// NewDriver creates a new s3 storage driver
// Used during bootstrapping
func NewDriver(ctx context.Context, cfg *configv1.InfrastructureStatus, clnt client.Client) *driver {
	return &driver{
		Context: ctx,
		Config: &S3{
			Region:    cfg.PlatformStatus.AWS.Region,
			InfraName: cfg.InfrastructureName,
		},
		kubeClient: clnt,
	}
}

// CreateStorage attempts to create an s3 bucket
// and apply any provided tags
func (d *driver) CreateStorage(reqLogger logr.Logger, instance *veleroInstallCR.VeleroInstall) error {

	var err error

	// Create an S3 client based on the region we received
	s3Client, err := NewS3Client(d.kubeClient, d.Config.Region)
	if err != nil {
		return err
	}

	bucketLog := reqLogger.WithValues("StorageBucket.Name", instance.Status.StorageBucket.Name, "StorageBucket.Region", d.Config.Region)

	// This switch handles the provisioning steps/checks
	switch {
	// We don't yet have a bucket name selected
	case instance.Status.StorageBucket.Name == "":

		// Use an existing bucket, if it exists.
		bucketLog.Info("No S3 bucket defined. Searching for existing bucket to use")
		bucketlist, err := ListBucketsInRegion(s3Client, d.Config.Region)
		if err != nil {
			return err
		}

		bucketinfo, err := ListBucketTags(s3Client, bucketlist.Buckets)
		if err != nil {
			return err
		}

		existingBucket := FindMatchingTags(bucketinfo, d.Config.InfraName)
		if existingBucket != "" {
			bucketLog.Info("Recovered existing bucket", "StorageBucket.Name", existingBucket)
			instance.Status.StorageBucket.Name = existingBucket
			instance.Status.StorageBucket.Provisioned = true
			return instance.StatusUpdate(reqLogger, d.kubeClient)
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
		instance.Status.StorageBucket.Name = proposedName
		instance.Status.StorageBucket.Provisioned = false
		return instance.StatusUpdate(reqLogger, d.kubeClient)

	// We have a bucket name, but haven't kicked off provisioning of the bucket yet
	case instance.Status.StorageBucket.Name != "" && !instance.Status.StorageBucket.Provisioned:
		bucketLog.Info("S3 bucket defined, but not provisioned")

		// Create S3 bucket
		bucketLog.Info("Creating S3 Bucket")
		err = CreateBucket(s3Client, instance.Status.StorageBucket.Name)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case s3.ErrCodeBucketAlreadyExists:
					bucketLog.Info("Bucket exists, but is not owned by current user; retrying")
					instance.Status.StorageBucket.Name = ""
					return instance.StatusUpdate(reqLogger, d.kubeClient)
				case s3.ErrCodeBucketAlreadyOwnedByYou:
					bucketLog.Info("Bucket exists, and is owned by current user; continue")
				default:
					return fmt.Errorf("error occurred when creating bucket %v: %v", instance.Status.StorageBucket.Name, aerr.Error())
				}
			} else {
				return fmt.Errorf("error occurred when creating bucket %v: %v", instance.Status.StorageBucket.Name, err.Error())
			}
		}
		err = TagBucket(s3Client, instance.Status.StorageBucket.Name, storageConstants.DefaultVeleroBackupStorageLocation, d.Config.InfraName)
		if err != nil {
			return fmt.Errorf("error occurred when tagging bucket %v: %v", instance.Status.StorageBucket.Name, err.Error())
		}
	}

	// Verify S3 bucket exists
	bucketLog.Info("Verifing S3 Bucket exists")
	exists, err := d.StorageExists(instance.Status.StorageBucket.Name)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("error occurred when verifying bucket %v: %v", instance.Status.StorageBucket.Name, aerr.Error())
		}
		return fmt.Errorf("error occurred when verifying bucket %v: %v", instance.Status.StorageBucket.Name, err.Error())
	}
	if !exists {
		bucketLog.Error(nil, "S3 bucket doesn't appear to exist")
		instance.Status.StorageBucket.Provisioned = false
		return instance.StatusUpdate(reqLogger, d.kubeClient)
	}

	// Encrypt S3 bucket
	bucketLog.Info("Enforcing S3 Bucket encryption")
	err = EncryptBucket(s3Client, instance.Status.StorageBucket.Name)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("error occurred when encrypting bucket %v: %v", instance.Status.StorageBucket.Name, aerr.Error())
		}
		return fmt.Errorf("error occurred when encrypting bucket %v: %v", instance.Status.StorageBucket.Name, err.Error())
	}

	// Block public access to S3 bucket
	bucketLog.Info("Enforcing S3 Bucket public access policy")
	err = BlockBucketPublicAccess(s3Client, instance.Status.StorageBucket.Name)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("error occurred when blocking public access to bucket %v: %v", instance.Status.StorageBucket.Name, aerr.Error())
		}
		return fmt.Errorf("error occurred when blocking public access to bucket %v: %v", instance.Status.StorageBucket.Name, err.Error())
	}

	// Configure lifecycle rules on S3 bucket
	bucketLog.Info("Enforcing S3 Bucket lifecycle rules on S3 Bucket")
	err = SetBucketLifecycle(s3Client, instance.Status.StorageBucket.Name)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("error occurred when configuring lifecycle rules on bucket %v: %v", instance.Status.StorageBucket.Name, aerr.Error())
		}
		return fmt.Errorf("error occurred when configuring lifecycle rules on bucket %v: %v", instance.Status.StorageBucket.Name, err.Error())
	}

	// Make sure that tags are applied to buckets
	bucketLog.Info("Enforcing S3 Bucket tags on S3 Bucket")
	err = TagBucket(s3Client, instance.Status.StorageBucket.Name, storageConstants.DefaultVeleroBackupStorageLocation, d.Config.InfraName)
	if err != nil {
		return fmt.Errorf("error occurred when tagging bucket %v: %v", instance.Status.StorageBucket.Name, err.Error())
	}

	instance.Status.StorageBucket.Provisioned = true
	instance.Status.StorageBucket.LastSyncTimestamp = &metav1.Time{
		Time: time.Now(),
	}
	return instance.StatusUpdate(reqLogger, d.kubeClient)

}

// StorageExists checks that the bucket exists, and that we have access to it.
func (d *driver) StorageExists(bucketName string) (bool, error) {

	//create an S3 Client
	s3Client, err := NewS3Client(d.kubeClient, d.Config.Region)
	if err != nil {
		return false, err
	}

	input := &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err = s3Client.HeadBucket(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			// This is supposed to say "NoSuchBucket", but actually emits "NotFound"
			// https://github.com/aws/aws-sdk-go/issues/2593
			case s3.ErrCodeNoSuchBucket, "NotFound":
				return false, nil
			default:
				return false, fmt.Errorf("unable to determine bucket %v status: %v", bucketName, aerr.Error())
			}
		} else {
			return false, fmt.Errorf("unable to determine bucket %v status: %v", bucketName, aerr.Error())
		}
	}

	return true, nil
}

//generateBucketName generates a proposed name for the S3 Bucket
func generateBucketName(prefix string) string {
	id := uuid.New().String()
	return prefix + id
}
