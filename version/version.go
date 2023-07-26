package version

var (
	Version = "0.2.0"
)

const (
	OperatorName      = "managed-velero-operator"
	VeleroImageTag    = "oadp-velero-rhel8@sha256:bdf6c3b957c8e4b02d83559c03a4cbbb08fd893eed6f3b04f9164cdddf9c3868"                // registry.redhat.io/oadp/oadp-velero-rhel8:1.2.0-37
	VeleroAwsImageTag = "oadp-velero-plugin-for-aws-rhel8@sha256:2ae11ce320b0383bb2fc98eaf8b70d8ffa3269b0c0b83d80598d65a25aa1190f" // registry.redhat.io/oadp/oadp-velero-plugin-for-aws-rhel8:1.2.0-21
	VeleroGcpImageTag = "oadp-velero-plugin-for-gcp-rhel8@sha256:42b3c8b0f027a96fef8aaa913ea73537a84667284f198309ae76c4693a564851" // registry.redhat.io/oadp/oadp-velero-plugin-for-gcp-rhel8:1.2.0-22
)
