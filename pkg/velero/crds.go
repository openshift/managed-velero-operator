package velero

import (
	"context"

	veleroInstall "github.com/heptio/velero/pkg/install"

	"github.com/go-logr/logr"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// InstallVeleroCRDs ensures that operator dependencies are installed at runtime.
func InstallVeleroCRDs(log logr.Logger, client client.Client) error {
	var err error

	// Install CRDs
	for _, crd := range veleroInstall.CRDs() {
		found := &apiextv1beta1.CustomResourceDefinition{}
		if err = client.Get(context.TODO(), types.NamespacedName{Name: crd.ObjectMeta.Name}, found); err != nil {
			if errors.IsNotFound(err) {
				// Didn't find CRD, we should create it.
				log.Info("Creating CRD", "CRD.Name", crd.ObjectMeta.Name)
				if err = client.Create(context.TODO(), crd); err != nil {
					return err
				}
			} else {
				// Return other errors
				return err
			}
		}
	}

	return nil
}
