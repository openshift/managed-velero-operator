package gcs

import (
	"regexp"
	"strings"

	storageConstants "github.com/openshift/managed-velero-operator/pkg/storage/constants"

	gstorage "cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

var (
	UniformBucketLevelAccessEnabled = gstorage.UniformBucketLevelAccess{Enabled: true}
)

// CreateBucket creates a new GCS bucket.
func (d *driver) createBucket(gcsClient *gstorage.Client, bucketName string) error {
	return gcsClient.Bucket(bucketName).Create(d.Context, d.Config.Project, &gstorage.BucketAttrs{
		Location:                 strings.ToUpper(d.Config.Region),
		UniformBucketLevelAccess: UniformBucketLevelAccessEnabled,
		Labels:                   buildLabelMap(d.Config.InfraName),
	})
}

// enforceBucketLabels enforces labels on an GCS bucket. The tags are used to indicate that velero backups
// are stored in the bucket, and to identify the associated cluster.
func (d *driver) enforceBucketLabels(gcsClient *gstorage.Client, bucketName string) error {
	bucketAttrs := &gstorage.BucketAttrsToUpdate{}
	labels := buildLabelMap(d.Config.InfraName)
	for k, v := range labels {
		bucketAttrs.SetLabel(k, v)
	}
	_, err := gcsClient.Bucket(bucketName).Update(d.Context, *bucketAttrs)
	return err
}

// listBuckets lists all buckets in the GCP account.
func (d *driver) listBuckets(gcsClient *gstorage.Client) ([]*gstorage.BucketAttrs, error) {
	var results []*gstorage.BucketAttrs

	list := gcsClient.Buckets(d.Context, d.Config.Project)
	for {
		bucket, err := list.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return results, err
		}

		results = append(results, bucket)
	}

	return results, nil
}

// FindVeleroBucket looks through the Labels for all GCS buckets and determines if
// any of the buckets are tagged for velero updates for the cluster.
// If matching tags are found, the bucket name is returned.
func (d *driver) findVeleroBucket(buckets []*gstorage.BucketAttrs) string {
	for _, bucket := range buckets {
		tagMatchesCluster := false
		tagMatchesVelero := false
		for k, v := range bucket.Labels {
			if k == sanitizeBucketLabel(storageConstants.BucketTagInfrastructureName) && v == sanitizeBucketLabel(d.Config.InfraName) {
				tagMatchesCluster = true
			}
			if k == sanitizeBucketLabel(storageConstants.BucketTagBackupStorageLocation) && v == sanitizeBucketLabel(storageConstants.DefaultVeleroBackupStorageLocation) {
				tagMatchesVelero = true
			}
		}

		if tagMatchesCluster && tagMatchesVelero {
			return bucket.Name
		}
	}

	// No matching buckets found.
	return ""
}

func sanitizeBucketLabel(input string) string {
	// https://cloud.google.com/storage/docs/key-terms#bucket-labels
	allowedRegEx := regexp.MustCompile("[^a-z0-9-_]+")

	return allowedRegEx.ReplaceAllString(strings.ToLower(input), "-")
}

func buildLabelMap(infraName string) map[string]string {
	return map[string]string{
		sanitizeBucketLabel(storageConstants.BucketTagBackupStorageLocation): sanitizeBucketLabel(storageConstants.DefaultVeleroBackupStorageLocation),
		sanitizeBucketLabel(storageConstants.BucketTagInfrastructureName):    sanitizeBucketLabel(infraName),
	}
}
