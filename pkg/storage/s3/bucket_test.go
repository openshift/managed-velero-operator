package s3

import (
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"

	storageConstants "github.com/openshift/managed-velero-operator/pkg/storage/constants"
)

const (
	clusterInfraName = "fakeCluster"
	region           = "us-east-1"
)

var awsConfig = &aws.Config{Region: aws.String(region)}

var s, _ = session.NewSession(awsConfig)

// Create a fake AWS client for mocking API responses.
var fakeClient = mockAWSClient{
	s3Client: s3.New(s),
	Config:   awsConfig,
}

// mockAWSClient implements the Client interface.
type mockAWSClient struct {
	s3Client s3iface.S3API
	Config   *aws.Config
}

// CreateBucket implements the CreateBucket method for mockAWSClient.
func (c *mockAWSClient) CreateBucket(input *s3.CreateBucketInput) (*s3.CreateBucketOutput, error) {
	return &s3.CreateBucketOutput{
		Location: aws.String(region),
	}, nil
}

// DeleteBucketTagging implements the DeleteBucketTagging method for mockAWSClient.
func (c *mockAWSClient) DeleteBucketTagging(input *s3.DeleteBucketTaggingInput) (*s3.DeleteBucketTaggingOutput, error) {
	return c.s3Client.DeleteBucketTagging(input)
}

// GetAWSClientConfig returns a copy of the AWS Client Config for the mockAWSClient.
func (c *mockAWSClient) GetAWSClientConfig() *aws.Config {
	return c.Config
}

// HeadBucket implements the HeadBucket method for mockAWSClient.
// This mocks the AWS API response of having access to a single bucket named "testBucket".
func (c *mockAWSClient) HeadBucket(input *s3.HeadBucketInput) (*s3.HeadBucketOutput, error) {
	if *input.Bucket == "testBucket" {
		return &s3.HeadBucketOutput{}, nil
	}
	return &s3.HeadBucketOutput{}, awserr.New("NotFound", "Not Found", nil)
}

// GetBucketLocation implements the GetBucketLocation method for mockAWSClient.
// This mocks the AWS API response of having access to a single bucket named "testBucket".
func (c *mockAWSClient) GetBucketLocation(input *s3.GetBucketLocationInput) (*s3.GetBucketLocationOutput, error) {
	if *input.Bucket == "testBucket" {
		return &s3.GetBucketLocationOutput{LocationConstraint: aws.String(region)}, nil
	}
	return &s3.GetBucketLocationOutput{}, awserr.New("NotFound", "Not Found", nil)
}

// GetBucketTagging implements the GetBucketTagging method for mockAWSClient.
func (c *mockAWSClient) GetBucketTagging(input *s3.GetBucketTaggingInput) (*s3.GetBucketTaggingOutput, error) {
	if *input.Bucket == "testBucket" {
		return &s3.GetBucketTaggingOutput{
			TagSet: []*s3.Tag{
				{
					Key:   aws.String(bucketTagBackupLocation),
					Value: aws.String(storageConstants.DefaultVeleroBackupStorageLocation),
				},
				{
					Key:   aws.String(bucketTagInfraName),
					Value: aws.String(clusterInfraName),
				},
			},
		}, nil
	}
	return &s3.GetBucketTaggingOutput{
		TagSet: []*s3.Tag{},
	}, nil
}

// GetPublicAccessBlock implements the GetPublicAccessBlock method for mockAWSClient.
func (c *mockAWSClient) GetPublicAccessBlock(input *s3.GetPublicAccessBlockInput) (*s3.GetPublicAccessBlockOutput, error) {
	return c.s3Client.GetPublicAccessBlock(input)
}

// ListBuckets implements the ListBuckets method for mockAWSClient.
func (c *mockAWSClient) ListBuckets(input *s3.ListBucketsInput) (*s3.ListBucketsOutput, error) {
	return &s3.ListBucketsOutput{
		Buckets: []*s3.Bucket{
			{
				CreationDate: &time.Time{},
				Name: aws.String("testBucket"),
			},
		},
		Owner: &s3.Owner{},
	}, nil
}

// PutBucketEncryption implements the PutBucketEncryption method for mockAWSClient.
func (c *mockAWSClient) PutBucketEncryption(input *s3.PutBucketEncryptionInput) (*s3.PutBucketEncryptionOutput, error) {
	return c.s3Client.PutBucketEncryption(input)
}

// PutBucketLifecycleConfiguration implements the PutBucketLifecycleConfiguration method for mockAWSClient.
func (c *mockAWSClient) PutBucketLifecycleConfiguration(
	input *s3.PutBucketLifecycleConfigurationInput) (*s3.PutBucketLifecycleConfigurationOutput, error) {
	return c.s3Client.PutBucketLifecycleConfiguration(input)
}

// PutBucketTagging implements the PutBucketTagging method for mockAWSClient.
func (c *mockAWSClient) PutBucketTagging(input *s3.PutBucketTaggingInput) (*s3.PutBucketTaggingOutput, error) {
	return c.s3Client.PutBucketTagging(input)
}

// PutPublicAccessBlock implements the PutPublicAccessBlock method for mockAWSClient.
func (c *mockAWSClient) PutPublicAccessBlock(input *s3.PutPublicAccessBlockInput) (*s3.PutPublicAccessBlockOutput, error) {
	return c.s3Client.PutPublicAccessBlock(input)
}

func TestFindMatchingTags(t *testing.T) {

	tests := []struct {
		name       string
		bucketinfo map[string][]*s3.Tag
		infraName  string
		want       string
	}{
		// This tests the case of having buckets that don't match our cluster's name.
		// Since this bucket belongs to a different cluster, we want the function to return "",
		// indicating that there is no matching bucket name.
		{
			name:      "Bucket infraName doesn't match tag.",
			infraName: "wrongClusterName",
			bucketinfo: map[string][]*s3.Tag{
				"bucket1": {
					{
						Key:   aws.String(bucketTagBackupLocation),
						Value: aws.String("default"),
					},
					{
						Key:   aws.String(bucketTagInfraName),
						Value: aws.String(clusterInfraName),
					},
				},
			},
			want: "",
		},
		// This tests the case of having a bucket with a matching infraName, indicating that
		// the bucket belongs to our cluster. We expect the name of the bucket returned.
		{
			name:      "Bucket infraName matches tag.",
			infraName: clusterInfraName,
			bucketinfo: map[string][]*s3.Tag{
				"bucket1": {
					{
						Key:   aws.String(bucketTagBackupLocation),
						Value: aws.String("default"),
					},
					{
						Key:   aws.String(bucketTagInfraName),
						Value: aws.String(clusterInfraName),
					},
				},
			},
			want: "bucket1",
		},
		// This tests the case of two buckets. The first bucket should not match.
		// The name of the second bucket should be returned.
		{
			name:      "Two buckets; second bucket should match.",
			infraName: clusterInfraName,
			bucketinfo: map[string][]*s3.Tag{
				"bucket1": {
					{
						Key:   aws.String("kubernetes.io/cluster/testCluster"),
						Value: aws.String("owned"),
					},
					{
						Key:   aws.String("Name"),
						Value: aws.String("testCluster-image-registry"),
					},
				},
				"bucket2": {
					{
						Key:   aws.String(bucketTagBackupLocation),
						Value: aws.String(storageConstants.DefaultVeleroBackupStorageLocation),
					},
					{
						Key:   aws.String(bucketTagInfraName),
						Value: aws.String(clusterInfraName),
					},
				},
			},
			want: "bucket2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindMatchingTags(tt.bucketinfo, tt.infraName)
			if got != tt.want {
				t.Errorf("FindMatchingTags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateBucket(t *testing.T) {
	type args struct {
		s3Client   Client
		bucketName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Create a bucket named 'testBucket'",
			args: args{
				s3Client:   &fakeClient,
				bucketName: "testBucket",
			},
			wantErr: false,
		},
		{
			name: "Create a bucket with an empty name",
			args: args{
				s3Client:   &fakeClient,
				bucketName: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateBucket(tt.args.s3Client, tt.args.bucketName); (err != nil) != tt.wantErr {
				t.Errorf("CreateBucket() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDoesBucketExist(t *testing.T) {
	type args struct {
		s3Client   Client
		bucketName string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "Ensure that bucket named 'testBucket' exists",
			args: args{
				s3Client:   &fakeClient,
				bucketName: "testBucket",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Ensure that bucket named 'nonExistentBucket' does not exist",
			args: args{
				s3Client:   &fakeClient,
				bucketName: "nonExistentBucket",
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DoesBucketExist(tt.args.s3Client, tt.args.bucketName)
			if (err != nil) != tt.wantErr {
				t.Errorf("DoesBucketExist() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DoesBucketExist() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListBucketsInRegion(t *testing.T) {
	type args struct {
		s3Client Client
		region   string
	}
	tests := []struct {
		name    string
		args    args
		want    *s3.ListBucketsOutput
		wantErr bool
	}{
		{
			name: "List buckets in the region of 'testBucket'",
			args: args{
				s3Client: &fakeClient,
				region: region,
			},
			want: &s3.ListBucketsOutput{
				Buckets: []*s3.Bucket{
					{
						CreationDate: &time.Time{},
						Name: aws.String("testBucket"),
					},
				},
				Owner: &s3.Owner{},
			},
			wantErr: false,
		},
		{
			name: "List buckets in a different region than 'testBucket'",
			args: args{
				s3Client: &fakeClient,
				region: "ap-northeast-1",
			},
			want: &s3.ListBucketsOutput{
				Buckets: []*s3.Bucket{},
				Owner: &s3.Owner{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ListBucketsInRegion(tt.args.s3Client, tt.args.region)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListBucketsInRegion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListBucketsInRegion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListBucketTags(t *testing.T) {
	type args struct {
		s3Client Client
		buckets  []*s3.Bucket
	}
	tests := []struct {
		name    string
		args    args
		want    map[string][]*s3.Tag
		wantErr bool
	}{
		{
			name: "Ensure that bucket named 'testBucket' has expected tags",
			args: args{
				s3Client: &fakeClient,
				buckets: []*s3.Bucket{
					{
						Name: aws.String("testBucket"),
					},
				},
			},
			want: map[string][]*s3.Tag{
				"testBucket": []*s3.Tag{
					{
						Key:   aws.String(bucketTagBackupLocation),
						Value: aws.String(storageConstants.DefaultVeleroBackupStorageLocation),
					},
					{
						Key:   aws.String(bucketTagInfraName),
						Value: aws.String(clusterInfraName),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Ensure that bucket named 'nonTaggedBucket' returns no tags",
			args: args{
				s3Client: &fakeClient,
				buckets: []*s3.Bucket{
					{
						Name: aws.String("nonTaggedBucket"),
					},
				},
			},
			want: map[string][]*s3.Tag{
				"nonTaggedBucket": []*s3.Tag{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ListBucketTags(tt.args.s3Client, tt.args.buckets)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListBucketTags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListBucketTags() = %v, want %v", got, tt.want)
			}
		})
	}
}
