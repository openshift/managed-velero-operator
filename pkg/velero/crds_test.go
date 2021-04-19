package velero

import (
	"context"
	"testing"

	veleroInstall "github.com/vmware-tanzu/velero/pkg/install"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestInstallVeleroCRDs(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()

	err := InstallVeleroCRDs(logf.Log, fakeClient)
	if err != nil {
		t.Errorf("unexpected error returned when installing CRDs: %v", err)
	}

	for _, unstructuredCrd := range veleroInstall.AllCRDs().Items {
		foundCrd := &apiextv1beta1.CustomResourceDefinition{}
		err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: unstructuredCrd.GetName()}, foundCrd)
		if err != nil {
			t.Errorf("error returned when looking for CRD: %v", err)
		}
	}

}
