package version

var (
	Version = "0.2.0"
)

const (
	OperatorName      = "managed-velero-operator"
	VeleroImageTag    = "oadp-velero-rhel8@sha256:035f48844600bd3beebd6740bf85cf54d98a9232f01c31621d4e995ff366690a"                // registry.redhat.io/oadp/oadp-velero-rhel8:1.2.5-3
	VeleroAwsImageTag = "oadp-velero-plugin-for-aws-rhel8@sha256:317149aaba6bbe1600330a381ba2f8a7c2aba36db4f7cbd68545e037cfeed9db" // registry.redhat.io/oadp/oadp-velero-plugin-for-aws-rhel8:1.2.5-3
	VeleroGcpImageTag = "oadp-velero-plugin-for-gcp-rhel8@sha256:1556f9a9d3cf8920ecda5f2a568f7277d339ec2725d2fd4a844d590c483a3bd6" // registry.redhat.io/oadp/oadp-velero-plugin-for-gcp-rhel8:1.2.5-3
)
