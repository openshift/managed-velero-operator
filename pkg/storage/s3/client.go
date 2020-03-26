package s3

import (
	"context"
	"fmt"

	"github.com/openshift/managed-velero-operator/version"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

const (
	awsCredsSecretIDKey     = "aws_access_key_id"     // #nosec G101
	awsCredsSecretAccessKey = "aws_secret_access_key" // #nosec G101
	bucketTagBackupLocation = "velero.io/backup-location"
	bucketTagInfraName      = "velero.io/infrastructureName"
)

var (
	awsCredsSecretName = version.OperatorName + "-iam-credentials"
)

// awsClient implements the Client interface.
type awsClient struct {
	s3Client s3iface.S3API
	Config   *aws.Config
}

// Client is a wrapper object for the actual AWS SDK client to allow for easier testing.
type Client interface {
	CreateBucket(*s3.CreateBucketInput) (*s3.CreateBucketOutput, error)
	DeleteBucketTagging(*s3.DeleteBucketTaggingInput) (*s3.DeleteBucketTaggingOutput, error)
	HeadBucket(*s3.HeadBucketInput) (*s3.HeadBucketOutput, error)
	GetAWSClientConfig() *aws.Config
	GetBucketTagging(*s3.GetBucketTaggingInput) (*s3.GetBucketTaggingOutput, error)
	GetPublicAccessBlock(*s3.GetPublicAccessBlockInput) (*s3.GetPublicAccessBlockOutput, error)
	ListBuckets(*s3.ListBucketsInput) (*s3.ListBucketsOutput, error)
	PutBucketEncryption(*s3.PutBucketEncryptionInput) (*s3.PutBucketEncryptionOutput, error)
	PutBucketLifecycleConfiguration(*s3.PutBucketLifecycleConfigurationInput) (*s3.PutBucketLifecycleConfigurationOutput, error)
	PutBucketTagging(*s3.PutBucketTaggingInput) (*s3.PutBucketTaggingOutput, error)
	PutPublicAccessBlock(*s3.PutPublicAccessBlockInput) (*s3.PutPublicAccessBlockOutput, error)
}

// When all of the above Client methods are implemented for awsClient, awsClient becomes a kind of Client.

// CreateBucket implements the CreateBucket method for awsClient.
func (c *awsClient) CreateBucket(input *s3.CreateBucketInput) (*s3.CreateBucketOutput, error) {
	return c.s3Client.CreateBucket(input)
}

// DeleteBucketTagging implements the DeleteBucketTagging method for awsClient.
func (c *awsClient) DeleteBucketTagging(input *s3.DeleteBucketTaggingInput) (*s3.DeleteBucketTaggingOutput, error) {
	return c.s3Client.DeleteBucketTagging(input)
}

// GetAWSClientConfig returns a copy of the AWS Client Config for the awsClient.
func (c *awsClient) GetAWSClientConfig() *aws.Config {
	return c.Config
}

// HeadBucket implements the HeadBucket method for awsClient.
func (c *awsClient) HeadBucket(input *s3.HeadBucketInput) (*s3.HeadBucketOutput, error) {
	return c.s3Client.HeadBucket(input)
}

// GetBucketTagging implements the GetBucketTagging method for awsClient.
func (c *awsClient) GetBucketTagging(input *s3.GetBucketTaggingInput) (*s3.GetBucketTaggingOutput, error) {
	return c.s3Client.GetBucketTagging(input)
}

// GetPublicAccessBlock implements the GetPublicAccessBlock method for awsClient.
func (c *awsClient) GetPublicAccessBlock(input *s3.GetPublicAccessBlockInput) (*s3.GetPublicAccessBlockOutput, error) {
	return c.s3Client.GetPublicAccessBlock(input)
}

// ListBuckets implements the ListBuckets method for awsClient.
func (c *awsClient) ListBuckets(input *s3.ListBucketsInput) (*s3.ListBucketsOutput, error) {
	return c.s3Client.ListBuckets(input)
}

// PutBucketEncryption implements the PutBucketEncryption method for awsClient.
func (c *awsClient) PutBucketEncryption(input *s3.PutBucketEncryptionInput) (*s3.PutBucketEncryptionOutput, error) {
	return c.s3Client.PutBucketEncryption(input)
}

// PutBucketLifecycleConfiguration implements the PutBucketLifecycleConfiguration method for awsClient.
func (c *awsClient) PutBucketLifecycleConfiguration(
	input *s3.PutBucketLifecycleConfigurationInput) (*s3.PutBucketLifecycleConfigurationOutput, error) {
	return c.s3Client.PutBucketLifecycleConfiguration(input)
}

// PutBucketTagging implements the PutBucketTagging method for awsClient.
func (c *awsClient) PutBucketTagging(input *s3.PutBucketTaggingInput) (*s3.PutBucketTaggingOutput, error) {
	return c.s3Client.PutBucketTagging(input)
}

// PutPublicAccessBlock implements the PutPublicAccessBlock method for awsClient.
func (c *awsClient) PutPublicAccessBlock(input *s3.PutPublicAccessBlockInput) (*s3.PutPublicAccessBlockOutput, error) {
	return c.s3Client.PutPublicAccessBlock(input)
}

// NewS3Client reads the aws secrets in the operator's namespace and uses
// them to create a new client for accessing the S3 API.
func NewS3Client(kubeClient client.Client, region string) (Client, error) {
	var err error

	awsConfig := &aws.Config{Region: aws.String(region)}
	namespace, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		return nil, fmt.Errorf("failed to get operator namespace: %v", err)
	}

	secret := &corev1.Secret{}
	err = kubeClient.Get(context.TODO(),
		types.NamespacedName{
			Name:      awsCredsSecretName,
			Namespace: namespace,
		},
		secret)
	if err != nil {
		return nil, err
	}
	accessKeyID, ok := secret.Data[awsCredsSecretIDKey]
	if !ok {
		return nil, fmt.Errorf("AWS credentials secret %v did not contain key %v",
			awsCredsSecretName, awsCredsSecretIDKey)
	}
	secretAccessKey, ok := secret.Data[awsCredsSecretAccessKey]
	if !ok {
		return nil, fmt.Errorf("AWS credentials secret %v did not contain key %v",
			awsCredsSecretName, awsCredsSecretAccessKey)
	}

	awsConfig.Credentials = credentials.NewStaticCredentials(
		string(accessKeyID), string(secretAccessKey), "")

	s, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, err
	}

	// Load the actual AWS client into the awsClient interface.
	return &awsClient{
		s3Client: s3.New(s),
		Config:   awsConfig,
	}, nil
}

// CreateBucket creates a new S3 bucket.
func CreateBucket(s3Client Client, bucketName string) error {
	createBucketInput := &s3.CreateBucketInput{
		ACL:    aws.String(s3.BucketCannedACLPrivate),
		Bucket: aws.String(bucketName),
	}
	// Only set a location constraint if the cluster isn't in us-east-1
	// https://github.com/boto/boto3/issues/125
	config := s3Client.GetAWSClientConfig()

	//TODO: This region should not be hard coded.
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
		request := &s3.GetBucketTaggingInput{
			Bucket: aws.String(*bucket.Name),
		}
		response, err := s3Client.GetBucketTagging(request)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case "NoSuchTagSet":
					// There are no tags on this bucket, continue.
					continue
				case "NoSuchBucket":
					// The bucket specified no longer exists (can be due to delays in AWS API), continue.
					continue
				default:
					return taglist, err
				}
			} else {
				return taglist, err
			}
		}
		taglist[*bucket.Name] = response
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
