package s3

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

const clusterInfraName = "fakeCluster"

func TestFindMatchingTags(t *testing.T) {

	tests := []struct {
		name       string
		bucketinfo map[string]*s3.GetBucketTaggingOutput
		infraName  string
		want       string
	}{
		// This tests the case of having buckets that don't match our cluster's name.
		// Since this bucket belongs to a different cluster, we want the function to return "",
		// indicating that there is no matching bucket name.
		{
			name:      "Bucket infraName doesn't match tag.",
			infraName: "wrongClusterName",
			bucketinfo: map[string]*s3.GetBucketTaggingOutput{
				"bucket1": {
					TagSet: []*s3.Tag{
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
			},
			want: "",
		},
		// This tests the case of having a bucket with a matching infraName, indicating that
		// the bucket belongs to our cluster. We expect the name of the bucket returned.
		{
			name:      "Bucket infraName matches tag.",
			infraName: clusterInfraName,
			bucketinfo: map[string]*s3.GetBucketTaggingOutput{
				"bucket1": {
					TagSet: []*s3.Tag{
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
			},
			want: "bucket1",
		},
		// This tests the case of two buckets. The first bucket should not match.
		// The name of the second bucket should be returned.
		{
			name:      "Two buckets; second bucket should match.",
			infraName: clusterInfraName,
			bucketinfo: map[string]*s3.GetBucketTaggingOutput{
				"bucket1": {
					TagSet: []*s3.Tag{
						{
							Key:   aws.String("kubernetes.io/cluster/testCluster"),
							Value: aws.String("owned"),
						},
						{
							Key:   aws.String("Name"),
							Value: aws.String("testCluster-image-registry"),
						},
					},
				},
				"bucket2": {
					TagSet: []*s3.Tag{
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
