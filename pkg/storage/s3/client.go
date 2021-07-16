package s3

import (
	"context"
	"fmt"
	"os"

	"github.com/openshift/managed-velero-operator/version"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
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
	GetBucketLocation(*s3.GetBucketLocationInput) (*s3.GetBucketLocationOutput, error)
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

// GetBucketLocation implements the GetBucketLocation method for awsClient.
func (c *awsClient) GetBucketLocation(input *s3.GetBucketLocationInput) (*s3.GetBucketLocationOutput, error) {
	return c.s3Client.GetBucketLocation(input)
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

	sessionOptions := session.Options{
		Config: aws.Config{
			Region: aws.String(region),
		},
	}

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

	// get sharedCredsFile from secret
	sharedCredsFile, err := SharedCredentialsFileFromSecret(secret)
	if err != nil {
		return nil, err
	}

	sessionOptions.SharedConfigState = session.SharedConfigEnable // Force enable Shared Config support
	sessionOptions.SharedConfigFiles = []string{sharedCredsFile}  // Ordered list of files the session will load configuration from.

	s, err := session.NewSessionWithOptions(sessionOptions)
	if err != nil {
		return nil, err
	}

	// Remove temporary shared credentials token at end of func after creating session
	defer os.Remove(sharedCredsFile)

	// Load the actual AWS client into the awsClient interface.
	return &awsClient{
		s3Client: s3.New(s),
		Config:   awsConfig,
	}, nil
}
