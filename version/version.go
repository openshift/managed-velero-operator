package version

var (
	Version = "0.2.0"
)

const (
	OperatorName      = "managed-velero-operator"
	VeleroImageTag    = "velero@sha256:a60096c63ed34621a3d6fc69a02a25b1e1edb4396af891de98b0bcc91120231e"                // quay.io/konveyor/velero:oadp-1.2-amd64
	VeleroAwsImageTag = "velero-plugin-for-aws@sha256:a9259c6fb71a7ac7d50845cf3d79dfc683700156ab8011c40d5f092c85818f64" // quay.io/konveyor/velero-plugin-for-aws:oadp-1.2-amd64
	VeleroGcpImageTag = "velero-plugin-for-gcp@sha256:85a9d667ce44855bd1955ba1438c394e44c926d827520914f160d288f48a2525" // quay.io/konveyor/velero-plugin-for-gcp:oadp-1.2-amd64
)
