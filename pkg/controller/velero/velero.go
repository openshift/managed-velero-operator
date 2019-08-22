package velero

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	veleroCR "github.com/openshift/managed-velero-operator/pkg/apis/managed/v1alpha1"

	velerov1 "github.com/heptio/velero/pkg/apis/velero/v1"
	veleroInstall "github.com/heptio/velero/pkg/install"
	minterv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	awsCredsSecretIDKey     = "aws_access_key_id"     // #nosec G101
	awsCredsSecretAccessKey = "aws_secret_access_key" // #nosec G101
	credentialsRequestName  = "velero-iam-credentials"
)

func (r *ReconcileVelero) provisionVelero(reqLogger logr.Logger, namespace string, platformStatus *configv1.PlatformStatus, instance *veleroCR.Velero) (reconcile.Result, error) {
	var err error

	// Install CRDs
	for _, crd := range veleroInstall.CRDs() {
		found := &apiextv1beta1.CustomResourceDefinition{}
		if err = r.client.Get(context.TODO(), types.NamespacedName{Name: crd.ObjectMeta.Name}, found); err != nil {
			if errors.IsNotFound(err) {
				// Didn't find CRD, we should create it.
				reqLogger.Info("Creating CRD", "CRD.Name", crd.ObjectMeta.Name)
				if err = r.client.Create(context.TODO(), crd); err != nil {
					return reconcile.Result{}, err
				}
			} else {
				// Return other errors
				return reconcile.Result{}, err
			}
		}
	}

	locationConfig := make(map[string]string)
	locationConfig["region"] = platformStatus.AWS.Region

	// Install BackupStorageLocation
	foundBsl := &velerov1.BackupStorageLocation{}
	bsl := veleroInstall.BackupStorageLocation(namespace, strings.ToLower(string(platformStatus.Type)), instance.Status.S3Bucket.Name, "", locationConfig)
	if err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: "default"}, foundBsl); err != nil {
		if errors.IsNotFound(err) {
			// Didn't find BackupStorageLocation
			reqLogger.Info("Creating BackupStorageLocation")
			if err := controllerutil.SetControllerReference(instance, bsl, r.scheme); err != nil {
				return reconcile.Result{}, err
			}
			if err = r.client.Create(context.TODO(), bsl); err != nil {
				return reconcile.Result{}, err
			}
		} else {
			return reconcile.Result{}, err
		}
	} else {
		// BackupStorageLocation exists, check if it's updated.
		if !reflect.DeepEqual(foundBsl.Spec, bsl.Spec) {
			// Specs aren't equal, update and fix.
			reqLogger.Info("Updating BackupStorageLocation")
			foundBsl.Spec = *bsl.Spec.DeepCopy()
			if err = r.client.Update(context.TODO(), foundBsl); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	// Install VolumeSnapshotLocation
	foundVsl := &velerov1.VolumeSnapshotLocation{}
	vsl := veleroInstall.VolumeSnapshotLocation(namespace, strings.ToLower(string(platformStatus.Type)), locationConfig)
	if err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: "default"}, foundVsl); err != nil {
		if errors.IsNotFound(err) {
			// Didn't find VolumeSnapshotLocation
			reqLogger.Info("Creating VolumeSnapshotLocation")
			if err := controllerutil.SetControllerReference(instance, vsl, r.scheme); err != nil {
				return reconcile.Result{}, err
			}
			if err = r.client.Create(context.TODO(), vsl); err != nil {
				return reconcile.Result{}, err
			}
		} else {
			return reconcile.Result{}, err
		}
	} else {
		// VolumeSnapshotLocation exists, check if it's updated.
		if !reflect.DeepEqual(foundVsl.Spec, vsl.Spec) {
			// Specs aren't equal, update and fix.
			reqLogger.Info("Updating VolumeSnapshotLocation")
			foundVsl.Spec = *vsl.Spec.DeepCopy()
			if err = r.client.Update(context.TODO(), foundVsl); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	// Install CredentialsRequest
	foundCr := &minterv1.CredentialsRequest{}
	cr := credentialsRequest(namespace, credentialsRequestName, instance.Status.S3Bucket.Name)
	if err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: credentialsRequestName}, foundCr); err != nil {
		if errors.IsNotFound(err) {
			// Didn't find CredentialsRequest
			reqLogger.Info("Creating CredentialsRequest")
			if err := controllerutil.SetControllerReference(instance, cr, r.scheme); err != nil {
				return reconcile.Result{}, err
			}
			if err = r.client.Create(context.TODO(), cr); err != nil {
				return reconcile.Result{}, err
			}
		} else {
			return reconcile.Result{}, err
		}
	} else {
		// CredentialsRequest exists, check if it's updated.
		if !reflect.DeepEqual(foundCr.Spec, cr.Spec) {
			// Specs aren't equal, update and fix.
			reqLogger.Info("Updating CredentialsRequest")
			foundCr.Spec = *cr.Spec.DeepCopy()
			if err = r.client.Update(context.TODO(), foundCr); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	// Install Deployment
	foundDeployment := &appsv1beta1.Deployment{}
	deployment := veleroInstall.Deployment(namespace, veleroInstall.WithoutCredentialsVolume(),
		veleroInstall.WithEnvFromSecretKey(strings.ToUpper(awsCredsSecretIDKey), credentialsRequestName, awsCredsSecretIDKey),
		veleroInstall.WithEnvFromSecretKey(strings.ToUpper(awsCredsSecretAccessKey), credentialsRequestName, awsCredsSecretAccessKey),
		veleroInstall.WithImage("gcr.io/heptio-images/velero:v1.0.0"),
	)
	if err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: "velero"}, foundDeployment); err != nil {
		if errors.IsNotFound(err) {
			// Didn't find Deployment
			reqLogger.Info("Creating Deployment")
			if err := controllerutil.SetControllerReference(instance, deployment, r.scheme); err != nil {
				return reconcile.Result{}, err
			}
			if err = r.client.Create(context.TODO(), deployment); err != nil {
				return reconcile.Result{}, err
			}
		} else {
			return reconcile.Result{}, err
		}
	} else {
		// Deployment exists, check if it's updated.
		if !reflect.DeepEqual(foundDeployment.Spec, deployment.Spec) {
			// Specs aren't equal, update and fix.
			reqLogger.Info("Updating Deployment")
			foundDeployment.Spec = *deployment.Spec.DeepCopy()
			if err = r.client.Update(context.TODO(), foundDeployment); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	return reconcile.Result{}, nil
}

func credentialsRequest(namespace, name, bucketName string) *minterv1.CredentialsRequest {
	codec, _ := minterv1.NewCodec()
	awsProvSpec, _ := codec.EncodeProviderSpec(
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
					Resource: fmt.Sprintf("arn:aws:s3:::%s/*", bucketName),
				},
				{
					Effect: "Allow",
					Action: []string{
						"s3:ListBucket",
					},
					Resource: fmt.Sprintf("arn:aws:s3:::%s", bucketName),
				},
			},
		})

	return &minterv1.CredentialsRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "CredentialsRequest",
			APIVersion: minterv1.SchemeGroupVersion.String(),
		},
		Spec: minterv1.CredentialsRequestSpec{
			SecretRef: corev1.ObjectReference{
				Name:      name,
				Namespace: namespace,
			},
			ProviderSpec: awsProvSpec,
		},
	}
}
