package s3

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/openshift/managed-velero-operator/pkg/controller/velero"
	"github.com/prometheus/common/log"
)

const (
	bucketPrefix                 = "managed-velero-backups-"
	defaultBackupStorageLocation = "default"
)

type S3 struct {
	Region string
}

type driver struct {
	Config  *S3
	Context context.Context
}

// NewDriver creates a new s3 storage driver
// Used during bootstrapping
func NewDriver(ctx context.Context, cfg *configv1.InfrastructureStatus) *driver {
	return &driver{
		Context: ctx,
		Config:  &S3{Region: cfg.Region},
	}
}

// CreateStorage attempts to create an s3 bucket
// and apply any provided tags
func (d *driver) CreateStorage(reqLogger logr.Logger, r *velero.ReconcileVelero, instance *mangedv1alpha1.Velero, infraName string) error {

	var err error

	// Create an S3 client based on the region we received
	s3Client, err := s3.NewS3Client(r.kubeClient, d.cfg.Region)
	if err != nil {
		return err
	}

	bucketLog := reqLogger.WithValues("S3Bucket.Name", instance.Status.S3Bucket.Name, "S3Bucket.Region", d.cfg.Region)

	// This switch handles the provisioning steps/checks
	switch {
	// We don't yet have a bucket name selected
	case instance.Status.S3Bucket.Name == "":

		// Use an existing bucket, if it exists.
		log.Info("No S3 bucket defined. Searching for existing bucket to use")
		bucketlist, err := s3.ListBuckets(s3Client)
		if err != nil {
			return err
		}

		bucketinfo, err := s3.ListBucketTags(s3Client, bucketlist)
		if err != nil {
			return err
		}

		existingBucket := s3.FindMatchingTags(bucketinfo, infraName)
		if existingBucket != "" {
			log.Info(fmt.Sprintf("Recovered existing bucket: %s", existingBucket))
			instance.Status.S3Bucket.Name = existingBucket
			instance.Status.S3Bucket.Provisioned = true
			return r.statusUpdate(reqLogger, instance)
		}

		// Prepare to create a new bucket, if none exist.
		proposedName := generateBucketName(bucketPrefix)
		proposedBucketExists, err := s3.StorageExists(r, proposedName)
		if err != nil {
			return err
		}
		if proposedBucketExists {
			return fmt.Errorf("proposed bucket %s already exists, retrying", proposedName)
		}

		log.Info("Setting proposed bucket name", "S3Bucket.Name", proposedName)
		instance.Status.S3Bucket.Name = proposedName
		instance.Status.S3Bucket.Provisioned = false
		return r.statusUpdate(reqLogger, instance)

	// We have a bucket name, but haven't kicked off provisioning of the bucket yet
	case instance.Status.S3Bucket.Name != "" && !instance.Status.S3Bucket.Provisioned:
		bucketLog.Info("S3 bucket defined, but not provisioned")

		// Create S3 bucket
		bucketLog.Info("Creating S3 Bucket")
		err = s3.CreateBucket(s3Client, instance.Status.S3Bucket.Name)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case awss3.ErrCodeBucketAlreadyExists:
					bucketLog.Info("Bucket exists, but is not owned by current user; retrying")
					instance.Status.S3Bucket.Name = ""
					return r.statusUpdate(reqLogger, instance)
				case awss3.ErrCodeBucketAlreadyOwnedByYou:
					bucketLog.Info("Bucket exists, and is owned by current user; continue")
				default:
					return fmt.Errorf("error occurred when creating bucket %v: %v", instance.Status.S3Bucket.Name, aerr.Error())
				}
			} else {
				return fmt.Errorf("error occurred when creating bucket %v: %v", instance.Status.S3Bucket.Name, err.Error())
			}
		}
		err = s3.TagBucket(s3Client, instance.Status.S3Bucket.Name, defaultBackupStorageLocation, infraName)
		if err != nil {
			return fmt.Errorf("error occurred when tagging bucket %v: %v", instance.Status.S3Bucket.Name, err.Error())
		}
	}

	// Verify S3 bucket exists
	bucketLog.Info("Verifing S3 Bucket exists")
	exists, err := s3.StorageExists(r, instance.Status.S3Bucket.Name)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("error occurred when verifying bucket %v: %v", instance.Status.S3Bucket.Name, aerr.Error())
		}
		return fmt.Errorf("error occurred when verifying bucket %v: %v", instance.Status.S3Bucket.Name, err.Error())
	}
	if !exists {
		bucketLog.Error(nil, "S3 bucket doesn't appear to exist")
		instance.Status.S3Bucket.Provisioned = false
		return r.statusUpdate(reqLogger, instance)
	}

	// Encrypt S3 bucket
	bucketLog.Info("Enforcing S3 Bucket encryption")
	err = s3.EncryptBucket(s3Client, instance.Status.S3Bucket.Name)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("error occurred when encrypting bucket %v: %v", instance.Status.S3Bucket.Name, aerr.Error())
		}
		return fmt.Errorf("error occurred when encrypting bucket %v: %v", instance.Status.S3Bucket.Name, err.Error())
	}

	// Block public access to S3 bucket
	bucketLog.Info("Enforcing S3 Bucket public access policy")
	err = s3.BlockBucketPublicAccess(s3Client, instance.Status.S3Bucket.Name)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("error occurred when blocking public access to bucket %v: %v", instance.Status.S3Bucket.Name, aerr.Error())
		}
		return fmt.Errorf("error occurred when blocking public access to bucket %v: %v", instance.Status.S3Bucket.Name, err.Error())
	}

	// Configure lifecycle rules on S3 bucket
	bucketLog.Info("Enforcing S3 Bucket lifecycle rules on S3 Bucket")
	err = s3.SetBucketLifecycle(s3Client, instance.Status.S3Bucket.Name)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("error occurred when configuring lifecycle rules on bucket %v: %v", instance.Status.S3Bucket.Name, aerr.Error())
		}
		return fmt.Errorf("error occurred when configuring lifecycle rules on bucket %v: %v", instance.Status.S3Bucket.Name, err.Error())
	}

	// Make sure that tags are applied to buckets
	bucketLog.Info("Enforcing S3 Bucket tags on S3 Bucket")
	err = s3.TagBucket(s3Client, instance.Status.S3Bucket.Name, defaultBackupStorageLocation, infraName)
	if err != nil {
		return fmt.Errorf("error occurred when tagging bucket %v: %v", instance.Status.S3Bucket.Name, err.Error())
	}

	instance.Status.S3Bucket.Provisioned = true
	instance.Status.S3Bucket.LastSyncTimestamp = &metav1.Time{
		Time: time.Now(),
	}
	return r.statusUpdate(reqLogger, instance)

}

// StorageExists checks that the bucket exists, and that we have access to it.
func (d *driver) StorageExists(client, r *velero.ReconcileVeleroReconcileVelero, bucketName string) (bool, error) {

	//create an S3 Client
	s3Client, err := s3.NewS3Client(r.kubeClient, d.cfg.Region)
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
