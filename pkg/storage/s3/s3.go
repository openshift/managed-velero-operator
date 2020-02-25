package s3

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"reflect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"golang.org/x/net/http/httpproxy"

	configapiv1 "github.com/openshift/api/config/v1"
	imageregistryv1 "github.com/openshift/api/imageregistry/v1"
	operatorapi "github.com/openshift/api/operator/v1"

	"github.com/openshift/managed-velero-operator/defaults"
	regopclient "github.com/openshift/managed-velero-operator/pkg/client"

	"github.com/openshift/managed-velero-operator/pkg/storage/util"
	"github.com/openshift/managed-velero-operator/version"
)

type S3 struct {
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
}

type driver struct {
	Context context.Context
	Config  *imageregistryv1.ImageRegistryConfigStorageS3
	Listers *regopclient.Listers
}

// NewDriver creates a new s3 storage driver
// Used during bootstrapping
func NewDriver(ctx context.Context, c *imageregistryv1.ImageRegistryConfigStorageS3, listers *regopclient.Listers) *driver {
	return &driver{
		Context: ctx,
		Config:  c,
		Listers: listers,
	}
}

// GetConfig reads configuration for the S3 cloud platform services.
func GetConfig(listers *regopclient.Listers) (*S3, error) {
	cfg := &S3{}

	infra, err := util.GetInfrastructure(listers)
	if err != nil {
		return nil, err
	}

	if infra.Status.PlatformStatus != nil && infra.Status.PlatformStatus.Type == configapiv1.AWSPlatformType {
		cfg.Region = infra.Status.PlatformStatus.AWS.Region
	}

	// Fall back to those provided by the credential minter if nothing is provided by the user
	sec, err = listers.Secrets.Get(defaults.AwsCredsSecretName)
	if err != nil {
		return nil, fmt.Errorf("unable to get cluster minted credentials %q: %v", fmt.Sprintf("%s/%s", defaults.ManagedVeleroOperatorNamespace, defaults.AwsCredsSecretName), err)
	}

	if v, ok := sec.Data["aws_access_key_id"]; ok {
		cfg.AccessKey = string(v)
	} else {
		return nil, fmt.Errorf("secret %q does not contain required key \"aws_access_key_id\"", fmt.Sprintf("%s/%s", defaults.ManagedVeleroOperatorNamespace, defaults.AwsCredsSecretName))
	}
	if v, ok := sec.Data["aws_secret_access_key"]; ok {
		cfg.SecretKey = string(v)
	} else {
		return nil, fmt.Errorf("secret %q does not contain required key \"aws_secret_access_key\"", fmt.Sprintf("%s/%s", defaults.ManagedVeleroOperatorNamespace, defaults.AwsCredsSecretName))
	}

	return cfg, nil
}

// getS3Service returns a client that allows us to interact
// with the aws S3 service
func (d *driver) getS3Service() (*s3.S3, error) {
	cfg, err := GetConfig(d.Listers)
	if err != nil {
		return nil, err
	}

	if len(d.Config.Region) == 0 {
		d.Config.Region = cfg.Region
	}

	// A custom HTTPClient is used here since the default HTTPClients ProxyFromEnvironment
	// uses a cache which won't let us update the proxy env vars
	awsConfig := &aws.Config{
		Credentials: credentials.NewStaticCredentials(cfg.AccessKey, cfg.SecretKey, ""),
		Region:      &d.Config.Region,
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				Proxy: func(req *http.Request) (*url.URL, error) {
					return httpproxy.FromEnvironment().ProxyFunc()(req.URL)
				},
			},
		},
	}
	awsConfig.WithUseDualStack(true)
	if d.Config.RegionEndpoint != "" {
		awsConfig.WithS3ForcePathStyle(true)
		awsConfig.WithEndpoint(d.Config.RegionEndpoint)
	}
	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, err
	}
	sess.Handlers.Build.PushBackNamed(request.NamedHandler{
		Name: "openshift.io/cluster-image-registry-operator",
		Fn:   request.MakeAddToUserAgentHandler("openshift.io cluster-image-registry-operator", version.Version),
	})

	return s3.New(sess), nil

}

func isBucketNotFound(err interface{}) bool {
	switch s3Err := err.(type) {
	case awserr.Error:
		if s3Err.Code() == "NoSuchBucket" {
			return true
		}
		origErr := s3Err.OrigErr()
		if origErr != nil {
			return isBucketNotFound(origErr)
		}
	case s3manager.Error:
		if s3Err.OrigErr != nil {
			return isBucketNotFound(s3Err.OrigErr)
		}
	case s3manager.Errors:
		if len(s3Err) == 1 {
			return isBucketNotFound(s3Err[0])
		}
	}
	return false
}

// bucketExists checks whether or not the s3 bucket exists
func (d *driver) bucketExists(bucketName string) error {
	if len(bucketName) == 0 {
		return nil
	}

	svc, err := d.getS3Service()
	if err != nil {
		return err
	}

	_, err = svc.HeadBucketWithContext(d.Context, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})

	return err
}

// StorageExists checks if an S3 bucket with the given name exists
// and we can access it
func (d *driver) StorageExists(cr *imageregistryv1.Config) (bool, error) {
	if len(d.Config.Bucket) == 0 {
		return false, nil
	}

	err := d.bucketExists(d.Config.Bucket)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket, "Forbidden", "NotFound":
				util.UpdateCondition(cr, defaults.StorageExists, operatorapi.ConditionFalse, aerr.Code(), aerr.Error())
				return false, nil
			}
		}
		util.UpdateCondition(cr, defaults.StorageExists, operatorapi.ConditionUnknown, "Unknown Error Occurred", err.Error())
		return false, err
	}

	util.UpdateCondition(cr, defaults.StorageExists, operatorapi.ConditionTrue, "S3 Bucket Exists", "")
	return true, nil

}

// StorageChanged checks to see if the name of the storage medium
// has changed
func (d *driver) StorageChanged(cr *imageregistryv1.Config) bool {
	if !reflect.DeepEqual(cr.Status.Storage.S3, cr.Spec.Storage.S3) {
		util.UpdateCondition(cr, defaults.StorageExists, operatorapi.ConditionUnknown, "S3 Configuration Changed", "S3 storage is in an unknown state")
		return true
	}

	return false
}

// CreateStorage attempts to create an s3 bucket
// and apply any provided tags
func (d *driver) CreateStorage(cr *imageregistryv1.Config) error {
	svc, err := d.getS3Service()
	if err != nil {
		return err
	}

	infra, err := util.GetInfrastructure(d.Listers)
	if err != nil {
		return err
	}

	// If a bucket name is supplied, and it already exists and we can access it
	// just update the config
	var bucketExists bool
	if len(d.Config.Bucket) != 0 {
		err = d.bucketExists(d.Config.Bucket)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case s3.ErrCodeNoSuchBucket, "Forbidden", "NotFound":
					// If the bucket doesn't exist that's ok, we'll try to create it
					util.UpdateCondition(cr, defaults.StorageExists, operatorapi.ConditionFalse, aerr.Code(), aerr.Error())
				default:
					util.UpdateCondition(cr, defaults.StorageExists, operatorapi.ConditionUnknown, "Unknown Error Occurred", err.Error())
					return err
				}
			} else {
				util.UpdateCondition(cr, defaults.StorageExists, operatorapi.ConditionUnknown, "Unknown Error Occurred", err.Error())
				return err
			}
		} else {
			bucketExists = true
		}

	}
	if len(d.Config.Bucket) != 0 && bucketExists {
		cr.Status.Storage = imageregistryv1.ImageRegistryConfigStorage{
			S3: d.Config.DeepCopy(),
		}
		util.UpdateCondition(cr, defaults.StorageExists, operatorapi.ConditionTrue, "S3 Bucket Exists", "User supplied S3 bucket exists and is accessible")

	} else {
		generatedName := false
		// Retry up to 5000 times if we get a naming conflict
		const numRetries = 5000
		for i := 0; i < numRetries; i++ {
			// If the bucket name is blank, let's generate one
			if len(d.Config.Bucket) == 0 {
				if d.Config.Bucket, err = util.GenerateStorageName(d.Listers, d.Config.Region); err != nil {
					return err
				}
				generatedName = true
			}

			_, err := svc.CreateBucketWithContext(d.Context, &s3.CreateBucketInput{
				Bucket: aws.String(d.Config.Bucket),
			})
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok {
					switch aerr.Code() {
					case s3.ErrCodeBucketAlreadyExists:
						if d.Config.Bucket != "" && !generatedName {
							util.UpdateCondition(cr, defaults.StorageExists, operatorapi.ConditionFalse, "Unable to Access Bucket", "The bucket exists, but we do not have permission to access it")
							break
						}
						d.Config.Bucket = ""
						continue
					default:
						util.UpdateCondition(cr, defaults.StorageExists, operatorapi.ConditionFalse, aerr.Code(), aerr.Error())
						return err
					}
				}
			}
			cr.Status.StorageManaged = true
			cr.Status.Storage = imageregistryv1.ImageRegistryConfigStorage{
				S3: d.Config.DeepCopy(),
			}
			cr.Spec.Storage.S3 = d.Config.DeepCopy()

			util.UpdateCondition(cr, defaults.StorageExists, operatorapi.ConditionTrue, "Creation Successful", "S3 bucket was successfully created")

			break
		}

		if len(d.Config.Bucket) == 0 {
			util.UpdateCondition(cr, defaults.StorageExists, operatorapi.ConditionFalse, "Unable to Generate Unique Bucket Name", "")
			return fmt.Errorf("unable to generate a unique s3 bucket name")
		}
	}

	// Wait until the bucket exists
	if err := svc.WaitUntilBucketExistsWithContext(d.Context, &s3.HeadBucketInput{
		Bucket: aws.String(d.Config.Bucket),
	}); err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			util.UpdateCondition(cr, defaults.StorageExists, operatorapi.ConditionFalse, aerr.Code(), aerr.Error())
		}

		return err
	}

	// Block public access to the s3 bucket and its objects by default
	if cr.Status.StorageManaged {
		_, err := svc.PutPublicAccessBlockWithContext(d.Context, &s3.PutPublicAccessBlockInput{
			Bucket: aws.String(d.Config.Bucket),
			PublicAccessBlockConfiguration: &s3.PublicAccessBlockConfiguration{
				BlockPublicAcls:       aws.Bool(true),
				BlockPublicPolicy:     aws.Bool(true),
				IgnorePublicAcls:      aws.Bool(true),
				RestrictPublicBuckets: aws.Bool(true),
			},
		})

		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				util.UpdateCondition(cr, defaults.StoragePublicAccessBlocked, operatorapi.ConditionFalse, aerr.Code(), aerr.Error())
			} else {
				util.UpdateCondition(cr, defaults.StoragePublicAccessBlocked, operatorapi.ConditionFalse, "Unknown Error Occurred", err.Error())
			}
		} else {
			util.UpdateCondition(cr, defaults.StoragePublicAccessBlocked, operatorapi.ConditionTrue, "Public Access Block Successful", "Public access to the S3 bucket and its contents have been successfully blocked.")
			cr.Status.Storage = imageregistryv1.ImageRegistryConfigStorage{
				S3: d.Config.DeepCopy(),
			}
			cr.Spec.Storage.S3 = d.Config.DeepCopy()
		}
	}

	// Tag the bucket with the openshiftClusterID
	// along with any user defined tags from the cluster configuration
	if cr.Status.StorageManaged {
		_, err := svc.PutBucketTaggingWithContext(d.Context, &s3.PutBucketTaggingInput{
			Bucket: aws.String(d.Config.Bucket),
			Tagging: &s3.Tagging{

				TagSet: []*s3.Tag{
					{
						Key:   aws.String("kubernetes.io/cluster/" + infra.Status.InfrastructureName),
						Value: aws.String("owned"),
					},
					{
						Key:   aws.String("Name"),
						Value: aws.String(infra.Status.InfrastructureName + "-image-registry"),
					},
				},
			},
		})
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				util.UpdateCondition(cr, defaults.StorageTagged, operatorapi.ConditionFalse, aerr.Code(), aerr.Error())
			} else {
				util.UpdateCondition(cr, defaults.StorageTagged, operatorapi.ConditionFalse, "Unknown Error Occurred", err.Error())
			}
		} else {
			util.UpdateCondition(cr, defaults.StorageTagged, operatorapi.ConditionTrue, "Tagging Successful", "Tags were successfully applied to the S3 bucket")
		}
	}

	// Enable default encryption on the bucket
	if cr.Status.StorageManaged {
		var encryption *s3.ServerSideEncryptionByDefault
		var encryptionType string

		if len(d.Config.KeyID) != 0 {
			encryption = &s3.ServerSideEncryptionByDefault{
				SSEAlgorithm:   aws.String(s3.ServerSideEncryptionAwsKms),
				KMSMasterKeyID: aws.String(d.Config.KeyID),
			}
			encryptionType = s3.ServerSideEncryptionAwsKms
		} else {
			encryption = &s3.ServerSideEncryptionByDefault{
				SSEAlgorithm: aws.String(s3.ServerSideEncryptionAes256),
			}
			encryptionType = s3.ServerSideEncryptionAes256
		}

		_, err = svc.PutBucketEncryptionWithContext(d.Context, &s3.PutBucketEncryptionInput{
			Bucket: aws.String(d.Config.Bucket),
			ServerSideEncryptionConfiguration: &s3.ServerSideEncryptionConfiguration{
				Rules: []*s3.ServerSideEncryptionRule{
					{
						ApplyServerSideEncryptionByDefault: encryption,
					},
				},
			},
		})
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				util.UpdateCondition(cr, defaults.StorageEncrypted, operatorapi.ConditionFalse, aerr.Code(), aerr.Error())
			} else {
				util.UpdateCondition(cr, defaults.StorageEncrypted, operatorapi.ConditionFalse, "Unknown Error Occurred", err.Error())
			}
		} else {
			util.UpdateCondition(cr, defaults.StorageEncrypted, operatorapi.ConditionTrue, "Encryption Successful", fmt.Sprintf("Default %s encryption was successfully enabled on the S3 bucket", encryptionType))
			d.Config.Encrypt = true
			cr.Status.Storage = imageregistryv1.ImageRegistryConfigStorage{
				S3: d.Config.DeepCopy(),
			}
			cr.Spec.Storage.S3 = d.Config.DeepCopy()
		}
	} else {
		if !reflect.DeepEqual(cr.Status.Storage.S3, d.Config) {
			cr.Status.Storage = imageregistryv1.ImageRegistryConfigStorage{
				S3: d.Config.DeepCopy(),
			}
		}
	}

	// Enable default incomplete multipart upload cleanup after one (1) day
	if cr.Status.StorageManaged {
		_, err = svc.PutBucketLifecycleConfigurationWithContext(d.Context, &s3.PutBucketLifecycleConfigurationInput{
			Bucket: aws.String(d.Config.Bucket),
			LifecycleConfiguration: &s3.BucketLifecycleConfiguration{
				Rules: []*s3.LifecycleRule{
					{
						ID:     aws.String("cleanup-incomplete-multipart-registry-uploads"),
						Status: aws.String("Enabled"),
						Filter: &s3.LifecycleRuleFilter{
							Prefix: aws.String(""),
						},
						AbortIncompleteMultipartUpload: &s3.AbortIncompleteMultipartUpload{
							DaysAfterInitiation: aws.Int64(1),
						},
					},
				},
			},
		})
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				util.UpdateCondition(cr, defaults.StorageIncompleteUploadCleanupEnabled, operatorapi.ConditionFalse, aerr.Code(), aerr.Error())
			} else {
				util.UpdateCondition(cr, defaults.StorageIncompleteUploadCleanupEnabled, operatorapi.ConditionFalse, "Unknown Error Occurred", err.Error())
			}
		} else {
			util.UpdateCondition(cr, defaults.StorageIncompleteUploadCleanupEnabled, operatorapi.ConditionTrue, "Enable Cleanup Successful", "Default cleanup of incomplete multipart uploads after one (1) day was successfully enabled")
		}
	}

	return nil
}

// RemoveStorage deletes the storage medium that we created
// The s3 bucket must be empty before it can be removed
func (d *driver) RemoveStorage(cr *imageregistryv1.Config) (bool, error) {
	if !cr.Status.StorageManaged || len(d.Config.Bucket) == 0 {
		return false, nil
	}

	svc, err := d.getS3Service()
	if err != nil {
		return false, err
	}

	iter := s3manager.NewDeleteListIterator(svc, &s3.ListObjectsInput{
		Bucket: aws.String(d.Config.Bucket),
	})

	err = s3manager.NewBatchDeleteWithClient(svc).Delete(d.Context, iter)
	if err != nil && !isBucketNotFound(err) {
		return false, err
	}

	_, err = svc.DeleteBucketWithContext(d.Context, &s3.DeleteBucketInput{
		Bucket: aws.String(d.Config.Bucket),
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == s3.ErrCodeNoSuchBucket {
				util.UpdateCondition(cr, defaults.StorageExists, operatorapi.ConditionFalse, "S3 Bucket Deleted", "The S3 bucket did not exist.")
				return false, nil
			}
			util.UpdateCondition(cr, defaults.StorageExists, operatorapi.ConditionUnknown, aerr.Code(), aerr.Error())
			return false, err
		}
		return true, err
	}

	// Wait until the bucket does not exist
	if err := svc.WaitUntilBucketNotExistsWithContext(d.Context, &s3.HeadBucketInput{
		Bucket: aws.String(d.Config.Bucket),
	}); err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			util.UpdateCondition(cr, defaults.StorageExists, operatorapi.ConditionTrue, aerr.Code(), aerr.Error())
		}

		return false, err
	}

	if len(cr.Spec.Storage.S3.Bucket) != 0 {
		cr.Spec.Storage.S3.Bucket = ""
	}

	d.Config.Bucket = ""

	if !reflect.DeepEqual(cr.Status.Storage.S3, d.Config) {
		cr.Status.Storage = imageregistryv1.ImageRegistryConfigStorage{
			S3: d.Config.DeepCopy(),
		}
	}

	util.UpdateCondition(cr, defaults.StorageExists, operatorapi.ConditionFalse, "S3 Bucket Deleted", "The S3 bucket has been removed.")

	return false, nil
}

// ID return the underlying storage identificator, on this case the bucket name.
func (d *driver) ID() string {
	return d.Config.Bucket
}
