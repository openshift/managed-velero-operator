package velero

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws/endpoints"

	"github.com/openshift/managed-velero-operator/pkg/storage/s3"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1 "github.com/openshift/api/config/v1"
	minterv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	veleroInstall "github.com/vmware-tanzu/velero/pkg/install"
)

// ReconcileVeleroAWS reconciles a Velero object on Amazon Web Services
type ReconcileVeleroAWS struct {
	ReconcileVeleroBase
}

func newReconcileVeleroAWS(ctx context.Context, mgr manager.Manager, config *configv1.InfrastructureStatus) VeleroReconciler {
	var r = &ReconcileVeleroAWS{
		ReconcileVeleroBase{
			client: mgr.GetClient(),
			scheme: mgr.GetScheme(),
			config: config,
		},
	}
	r.vtable = r

	r.driver = s3.NewDriver(ctx, r.config, r.client)

	return r
}

func (r *ReconcileVeleroAWS) RegionInChina() bool {
	for _, region := range awsChinaRegions {
		if r.config.PlatformStatus.AWS.Region == region {
			return true
		}
	}
	return false
}

func (r *ReconcileVeleroAWS) GetLocationConfig() map[string]string {
	return map[string]string{"region": r.config.PlatformStatus.AWS.Region}
}

func (r *ReconcileVeleroAWS) CredentialsRequest(namespace, bucketName string) (*minterv1.CredentialsRequest, error) {
	region := r.config.PlatformStatus.AWS.Region
	partition, ok := endpoints.PartitionForRegion(endpoints.DefaultPartitions(), region)
	if !ok {
		return nil, fmt.Errorf("no partition found for region %q", region)
	}

	resource := fmt.Sprintf("arn:%s:s3:::%s", partition.ID(), bucketName)

	codec, _ := minterv1.NewCodec()
	providerSpec, _ := codec.EncodeProviderSpec(
		&minterv1.AWSProviderSpec{
			TypeMeta: metav1.TypeMeta{
				Kind: "AWSProviderSpec",
			},
			StatementEntries: []minterv1.StatementEntry{
				{
					Effect: "Allow",
					Action: []string{
						"ec2:DescribeVolumes",
						"ec2:DescribeSnapshots",
						"ec2:CreateTags",
						"ec2:CreateVolume",
						"ec2:CreateSnapshot",
						"ec2:DeleteSnapshot",
					},
					Resource: "*",
				},
				{
					Effect: "Allow",
					Action: []string{
						"s3:GetObject",
						"s3:DeleteObject",
						"s3:PutObject",
						"s3:AbortMultipartUpload",
						"s3:ListMultipartUploadParts",
					},
					Resource: resource + "/*",
				},
				{
					Effect: "Allow",
					Action: []string{
						"s3:ListBucket",
					},
					Resource: resource,
				},
			},
		})

	return &minterv1.CredentialsRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      credentialsRequestName,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "CredentialsRequest",
			APIVersion: minterv1.SchemeGroupVersion.String(),
		},
		Spec: minterv1.CredentialsRequestSpec{
			SecretRef: corev1.ObjectReference{
				Name:      credentialsRequestName,
				Namespace: namespace,
			},
			ProviderSpec: providerSpec,
		},
	}, nil
}

func (r *ReconcileVeleroAWS) VeleroDeployment(namespace string) *appsv1.Deployment {
	imageRegistry := r.GetImageRegistry()

	deployment := veleroInstall.Deployment(
		namespace,
		veleroInstall.WithEnvFromSecretKey(
			strings.ToUpper(awsCredsSecretIDKey),
			credentialsRequestName,
			awsCredsSecretIDKey),
		veleroInstall.WithEnvFromSecretKey(
			strings.ToUpper(awsCredsSecretAccessKey),
			credentialsRequestName,
			awsCredsSecretAccessKey),
		veleroInstall.WithPlugins(
			[]string{imageRegistry + "/" + veleroAwsImageTag}),
		veleroInstall.WithImage(
			imageRegistry + "/" + veleroImageTag))

	return deployment
}
