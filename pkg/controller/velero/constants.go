package velero

const (
	awsCredsSecretIDKey     = "aws_access_key_id"     // #nosec G101
	awsCredsSecretAccessKey = "aws_secret_access_key" // #nosec G101

	veleroImageRegistry   = "docker.io/velero"
	veleroImageRegistryCN = "registry.docker-cn.com/velero"

	veleroImageTag    = "velero:v1.3.1"
	veleroAwsImageTag = "velero-plugin-for-aws:v1.0.1"
	veleroGcpImageTag = "velero-plugin-for-gcp:v1.0.1"

	credentialsRequestName = "velero-iam-credentials"
)

var (
	// Treat as a constant, even though arrays are mutable.
	awsChinaRegions = []string{"cn-north-1", "cn-northwest-1"}
)
