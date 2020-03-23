package velero

import (
	"context"
	"reflect"

	veleroInstall "github.com/vmware-tanzu/velero/pkg/install"

	"github.com/go-logr/logr"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// InstallVeleroCRDs ensures that operator dependencies are installed at runtime.
func InstallVeleroCRDs(log logr.Logger, client client.Client) error {
	var err error

	// Install CRDs
	for _, unstructuredCrd := range veleroInstall.AllCRDs().Items {
		foundCrd := &apiextv1beta1.CustomResourceDefinition{}
		crd := &apiextv1beta1.CustomResourceDefinition{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredCrd.Object, crd); err != nil {
			return err
		}
		// Add Conversion to the spec, as this will be returned in the founcCrd
		crd.Spec.Conversion = &apiextv1beta1.CustomResourceConversion{
			Strategy: apiextv1beta1.NoneConverter,
		}
		if err = client.Get(context.TODO(), types.NamespacedName{Name: crd.ObjectMeta.Name}, foundCrd); err != nil {
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
		} else {
			// CRD exists, check if it's updated.
			if !reflect.DeepEqual(foundCrd.Spec, crd.Spec) {
				// Specs aren't equal, update and fix.
				log.Info("Updating CRD", "CRD.Name", crd.ObjectMeta.Name, "foundCrd.Spec", foundCrd.Spec, "crd.Spec", crd.Spec)
				foundCrd.Spec = *crd.Spec.DeepCopy()
				if err = client.Update(context.TODO(), foundCrd); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
