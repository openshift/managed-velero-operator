package s3

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	bucketTagBackupLocation = "velero.io/backup-location"
	bucketTagInfraName      = "velero.io/infrastructureName"
)

// CreateBucket creates a new S3 bucket.
func CreateBucket(s3Client Client, bucketName string) error {
	createBucketInput := &s3.CreateBucketInput{
		ACL:    aws.String(s3.BucketCannedACLPrivate),
		Bucket: aws.String(bucketName),
	}
	// Only set a location constraint if the cluster isn't in us-east-1
	// https://github.com/boto/boto3/issues/125
	config := s3Client.GetAWSClientConfig()

	if *config.Region != "us-east-1" {
		createBucketConfiguation := &s3.CreateBucketConfiguration{
			LocationConstraint: config.Region,
		}
		createBucketInput.SetCreateBucketConfiguration(createBucketConfiguation)
	}
	if err := createBucketInput.Validate(); err != nil {
		return fmt.Errorf("unable to validate %v bucket creation configuration: %v", bucketName, err)
	}

	_, err := s3Client.CreateBucket(createBucketInput)

	return err
}

// DoesBucketExist checks that the bucket exists, and that we have access to it.
func DoesBucketExist(s3Client Client, bucketName string) (bool, error) {
	input := &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err := s3Client.HeadBucket(input)
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

// EncryptBucket sets the encryption configuration for the bucket.
func EncryptBucket(s3Client Client, bucketName string) error {
	bucketEncryptionInput := &s3.PutBucketEncryptionInput{
		Bucket: aws.String(bucketName),
		ServerSideEncryptionConfiguration: &s3.ServerSideEncryptionConfiguration{
			Rules: []*s3.ServerSideEncryptionRule{
				{
					ApplyServerSideEncryptionByDefault: &s3.ServerSideEncryptionByDefault{
						SSEAlgorithm: aws.String(s3.ServerSideEncryptionAes256),
					},
				},
			},
		},
	}

	if err := bucketEncryptionInput.Validate(); err != nil {
		return fmt.Errorf("unable to validate %v bucket encryption configuration: %v", bucketName, err)
	}

	_, err := s3Client.PutBucketEncryption(bucketEncryptionInput)

	return err
}

// BlockBucketPublicAccess blocks public access to the bucket's contents.
func BlockBucketPublicAccess(s3Client Client, bucketName string) error {
	publicAccessBlockInput := &s3.PutPublicAccessBlockInput{
		Bucket: aws.String(bucketName),
		PublicAccessBlockConfiguration: &s3.PublicAccessBlockConfiguration{
			BlockPublicAcls:       aws.Bool(true),
			BlockPublicPolicy:     aws.Bool(true),
			IgnorePublicAcls:      aws.Bool(true),
			RestrictPublicBuckets: aws.Bool(true),
		},
	}

	if err := publicAccessBlockInput.Validate(); err != nil {
		return fmt.Errorf("unable to validate %v bucket public access configuration: %v", bucketName, err)
	}

	_, err := s3Client.PutPublicAccessBlock(publicAccessBlockInput)

	return err
}

// SetBucketLifecycle sets a lifecycle on the specified bucket.
func SetBucketLifecycle(s3Client Client, bucketName string) error {
	bucketLifecycleConfigurationInput := &s3.PutBucketLifecycleConfigurationInput{
		Bucket: aws.String(bucketName),
		LifecycleConfiguration: &s3.BucketLifecycleConfiguration{
			Rules: []*s3.LifecycleRule{
				{
					ID:     aws.String("Backup Expiry"),
					Status: aws.String("Enabled"),
					Filter: &s3.LifecycleRuleFilter{
						Prefix: aws.String("backups/"),
					},
					Expiration: &s3.LifecycleExpiration{
						Days: aws.Int64(90),
					},
				},
			},
		},
	}

	if err := bucketLifecycleConfigurationInput.Validate(); err != nil {
		return fmt.Errorf("unable to validate %v bucket lifecycle configuration: %v", bucketName, err)
	}

	_, err := s3Client.PutBucketLifecycleConfiguration(bucketLifecycleConfigurationInput)

	return err
}

// CreateBucketTaggingInput creates an S3 PutBucketTaggingInput object,
// which is used to associate a list of tags with a bucket.
func CreateBucketTaggingInput(bucketname string, tags map[string]string) *s3.PutBucketTaggingInput {
	putInput := &s3.PutBucketTaggingInput{
		Bucket: aws.String(bucketname),
		Tagging: &s3.Tagging{
			TagSet: []*s3.Tag{},
		},
	}
	for key, value := range tags {
		newTag := s3.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		}
		putInput.Tagging.TagSet = append(putInput.Tagging.TagSet, &newTag)
	}
	return putInput
}

// ClearBucketTags wipes all existing tags from a bucket so that velero-specific
// tags can be applied to the bucket instead.
func ClearBucketTags(s3Client Client, bucketName string) (err error) {
	deleteInput := &s3.DeleteBucketTaggingInput{Bucket: aws.String(bucketName)}
	_, err = s3Client.DeleteBucketTagging(deleteInput)
	return err
}

// TagBucket adds tags to an S3 bucket. The tags are used to indicate that velero backups
// are stored in the bucket, and to identify the associated cluster.
func TagBucket(s3Client Client, bucketName string, backUpLocation string, infraName string) error {
	err := ClearBucketTags(s3Client, bucketName)
	if err != nil {
		return fmt.Errorf("unable to clear %v bucket tags: %v", bucketName, err)
	}
	input := CreateBucketTaggingInput(bucketName, map[string]string{
		bucketTagBackupLocation: backUpLocation,
		bucketTagInfraName:      infraName,
	})
	_, err = s3Client.PutBucketTagging(input)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	return nil
}

// ListBuckets lists all buckets in the AWS account.
func ListBuckets(s3Client Client) (*s3.ListBucketsOutput, error) {
	input := &s3.ListBucketsInput{}
	result, err := s3Client.ListBuckets(input)
	if err != nil {
		fmt.Println(err.Error())
		return result, err
	}
	return result, nil
}

// ListBucketTags returns a list of s3.GetBucketTagging objects, one for each bucket.
// If the bucket is not readable, or has no tags, the bucket name is omitted from the taglist.
// So taglist only contains the list of buckets that have tags.
func ListBucketTags(s3Client Client, bucketlist *s3.ListBucketsOutput) (map[string]*s3.GetBucketTaggingOutput, error) {
	taglist := make(map[string]*s3.GetBucketTaggingOutput)
	for _, bucket := range bucketlist.Buckets {
		// Sometimes deleted buckets will show up in this list.
		// In case they are in the process of being deleted, exit gracefully.
		bucketReadable, err := DoesBucketExist(s3Client, *bucket.Name)
		if err != nil {
			return taglist, err
		}
		if bucketReadable {
			request := &s3.GetBucketTaggingInput{
				Bucket: aws.String(*bucket.Name),
			}
			response, err := s3Client.GetBucketTagging(request)
			if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "NoSuchTagSet" {
				// If there is no tag set, exit this function without error.
				return taglist, nil
			} else if err != nil {
				return taglist, err
			}
			taglist[*bucket.Name] = response
		}
	}
	return taglist, nil
}

// FindMatchingTags looks through the TagSets for all AWS buckets and determines if
// any of the buckets are tagged for velero updates for the cluster.
// If matching tags are found, the bucket name is returned.
func FindMatchingTags(buckets map[string]*s3.GetBucketTaggingOutput, infraName string) string {
	var tagMatchesCluster, tagMatchesVelero bool
	var possiblematch string
	for bucket, tags := range buckets {
		for _, tag := range tags.TagSet {
			if *tag.Key == bucketTagInfraName && *tag.Value == infraName {
				tagMatchesCluster = true
				possiblematch = bucket
			}
			if *tag.Key == bucketTagBackupLocation {
				tagMatchesVelero = true
				possiblematch = bucket
			}
		}
	}

	// If these two conditions are true, the match is confirmed.
	if tagMatchesCluster && tagMatchesVelero {
		return possiblematch
	}

	// No matching buckets found.
	return ""
}
