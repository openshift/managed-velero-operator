package s3

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
)

func CreateBucket(s3Client *s3.S3, bucketName string) error {
	createBucketInput := &s3.CreateBucketInput{
		ACL:    aws.String(s3.BucketCannedACLPrivate),
		Bucket: aws.String(bucketName),
	}
	// Only set a location constraint if the cluster isn't in us-east-1
	// https://github.com/boto/boto3/issues/125
	if *s3Client.Client.Config.Region != "us-east-1" {
		createBucketConfiguation := &s3.CreateBucketConfiguration{
			LocationConstraint: s3Client.Client.Config.Region,
		}
		createBucketInput.SetCreateBucketConfiguration(createBucketConfiguation)
	}
	if err := createBucketInput.Validate(); err != nil {
		return fmt.Errorf("unable to validate %v bucket creation configuration: %v", bucketName, err)
	}

	_, err := s3Client.CreateBucket(createBucketInput)

	return err
}

func DoesBucketExist(s3Client *s3.S3, bucketName string) (bool, error) {
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

func EncryptBucket(s3Client *s3.S3, bucketName string) error {
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

func BlockBucketPublicAccess(s3Client *s3.S3, bucketName string) error {
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

func SetBucketLifecycle(s3Client *s3.S3, bucketName string) error {
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
