module github.com/openshift/managed-velero-operator

go 1.13

require (
	github.com/aws/aws-sdk-go v1.30.29
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/spec v0.19.8
	github.com/google/uuid v1.1.1
	github.com/openshift/api v3.9.1-0.20190924102528-32369d4db2ad+incompatible
	github.com/openshift/cloud-credential-operator v0.0.0-20200422160855-c442add7ccef
	github.com/vmware-tanzu/velero v1.4.2
)

require (
	cloud.google.com/go v0.57.0 // indirect
	cloud.google.com/go/storage v1.6.0
	github.com/cblecker/platformutils v0.0.0-20200321191645-443abe7fea11
	github.com/coreos/prometheus-operator v0.38.0
	github.com/googleapis/google-cloud-go-testing v0.0.0-20191008195207-8e1d251e947d
	github.com/operator-framework/operator-sdk v0.17.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	google.golang.org/api v0.22.0
	k8s.io/api v0.17.4
	k8s.io/apiextensions-apiserver v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a
	sigs.k8s.io/controller-runtime v0.5.2
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.17.4 // Required by prometheus-operator
)
