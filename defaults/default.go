package defaults

import "github.com/openshift/managed-velero-operator/version"

var (
	// AwsCredsSecretName is the credentials given to allow certain operatios to
	// managed-velero-operator
	AwsCredsSecretName = version.OperatorName + "-iam-credentials"
)

const (
	// ManagedVeleroOperatorNamespace is de default namespace where Manged-velero-operator
	// is deployed
	ManagedVeleroOperatorNamespace = "openshift-velero"

	// StorageExists denotes whether or not the storage medium exists
	StorageExists = "StorageExists"

	// StorageTagged denotes whether or not the storage medium
	// that we created was tagged correctly
	StorageTagged = "StorageTagged"

	// StorageLabeled denotes whether or not the storage medium
	// that we created was labeled correctly
	StorageLabeled = "StorageLabeled"

	// StorageEncrypted denotes whether or not the storage medium
	// that we created has encryption enabled
	StorageEncrypted = "StorageEncrypted"

	// StoragePublicAccessBlocked denotes whether or not the storage medium
	// that we created has had public access to itself and its objects blocked
	StoragePublicAccessBlocked = "StoragePublicAccessBlocked"

	// StorageIncompleteUploadCleanupEnabled denotes whether or not the storage
	// medium is configured to automatically cleanup incomplete uploads
	StorageIncompleteUploadCleanupEnabled = "StorageIncompleteUploadCleanupEnabled"
)
