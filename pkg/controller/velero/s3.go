package velero

import (
	"fmt"
	"time"

	veleroCR "github.com/openshift/managed-velero-operator/pkg/apis/managed/v1alpha1"
	"github.com/openshift/managed-velero-operator/pkg/s3"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/aws/aws-sdk-go/aws/awserr"
	awss3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
)

const (
	bucketPrefix = "managed-velero-backups-"
)

func (r *ReconcileVelero) provisionS3(reqLogger logr.Logger, s3Client *awss3.S3, instance *veleroCR.Velero) (reconcile.Result, error) {
	var err error
	bucketLog := reqLogger.WithValues("S3Bucket.Name", instance.Status.S3Bucket.Name, "S3Bucket.Region", s3Client.Client.Config.Region)

	// This switch handles the provisioning steps/checks
	switch {

	// We don't yet have a bucket name selected
	case instance.Status.S3Bucket.Name == "":
		log.Info("No S3 bucket defined")
		proposedName := generateBucketName(bucketPrefix)
		proposedBucketExists, err := s3.DoesBucketExist(s3Client, proposedName)
		if err != nil {
			return reconcile.Result{}, err
		}
		if proposedBucketExists {
			return reconcile.Result{}, fmt.Errorf("proposed bucket %s already exists, retrying", proposedName)
		}

		log.Info("Setting proposed bucket name", "S3Bucket.Name", proposedName)
		instance.Status.S3Bucket.Name = proposedName
		instance.Status.S3Bucket.Provisioned = false
		return reconcile.Result{}, r.statusUpdate(reqLogger, instance)

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
					return reconcile.Result{}, r.statusUpdate(reqLogger, instance)
				case awss3.ErrCodeBucketAlreadyOwnedByYou:
					bucketLog.Info("Bucket exists, and is owned by current user; continue")
				default:
					return reconcile.Result{}, fmt.Errorf("error occurred when creating bucket %v: %v", instance.Status.S3Bucket.Name, aerr.Error())
				}
			} else {
				return reconcile.Result{}, fmt.Errorf("error occurred when creating bucket %v: %v", instance.Status.S3Bucket.Name, err.Error())
			}
		}

	}

	// Verify S3 bucket exists
	bucketLog.Info("Verifing S3 Bucket exists")
	exists, err := s3.DoesBucketExist(s3Client, instance.Status.S3Bucket.Name)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			return reconcile.Result{}, fmt.Errorf("error occurred when verifying bucket %v: %v", instance.Status.S3Bucket.Name, aerr.Error())
		}
		return reconcile.Result{}, fmt.Errorf("error occurred when verifying bucket %v: %v", instance.Status.S3Bucket.Name, err.Error())
	}
	if !exists {
		bucketLog.Error(nil, "S3 bucket doesn't appear to exist")
		instance.Status.S3Bucket.Provisioned = false
		return reconcile.Result{}, r.statusUpdate(reqLogger, instance)
	}

	// Encrypt S3 bucket
	bucketLog.Info("Enforcing S3 Bucket encryption")
	err = s3.EncryptBucket(s3Client, instance.Status.S3Bucket.Name)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			return reconcile.Result{}, fmt.Errorf("error occurred when encrypting bucket %v: %v", instance.Status.S3Bucket.Name, aerr.Error())
		}
		return reconcile.Result{}, fmt.Errorf("error occurred when encrypting bucket %v: %v", instance.Status.S3Bucket.Name, err.Error())
	}

	// Block public access to S3 bucket
	bucketLog.Info("Enforcing S3 Bucket public access policy")
	err = s3.BlockBucketPublicAccess(s3Client, instance.Status.S3Bucket.Name)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			return reconcile.Result{}, fmt.Errorf("error occurred when blocking public access to bucket %v: %v", instance.Status.S3Bucket.Name, aerr.Error())
		}
		return reconcile.Result{}, fmt.Errorf("error occurred when blocking public access to bucket %v: %v", instance.Status.S3Bucket.Name, err.Error())
	}

	// Configure lifecycle rules on S3 bucket
	bucketLog.Info("Enforcing S3 Bucket lifecycle rules on S3 Bucket")
	err = s3.SetBucketLifecycle(s3Client, instance.Status.S3Bucket.Name)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			return reconcile.Result{}, fmt.Errorf("error occurred when configuring lifecycle rules on bucket %v: %v", instance.Status.S3Bucket.Name, aerr.Error())
		}
		return reconcile.Result{}, fmt.Errorf("error occurred when configuring lifecycle rules on bucket %v: %v", instance.Status.S3Bucket.Name, err.Error())
	}

	instance.Status.S3Bucket.Provisioned = true
	instance.Status.S3Bucket.LastSyncTimestamp.Time = time.Now()
	return reconcile.Result{}, r.statusUpdate(reqLogger, instance)
}

func generateBucketName(prefix string) string {
	id := uuid.New().String()
	return prefix + id
}
