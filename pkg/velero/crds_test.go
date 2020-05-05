package velero

// func TestInstallVeleroCRDs(t *testing.T) {
// 	fakeClient := fake.NewFakeClient()

// 	err := InstallVeleroCRDs(logf.Log, fakeClient)
// 	if err != nil {
// 		t.Errorf("unexpected error returned when installing CRDs: %v", err)
// 	}

// 	for _, unstructuredCrd := range veleroInstall.AllCRDs().Items {
// 		foundCrd := &apiextv1beta1.CustomResourceDefinition{}
// 		err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: unstructuredCrd.GetName()}, foundCrd)
// 		if err != nil {
// 			t.Errorf("error returned when looking for CRD: %v", err)
// 		}
// 	}

// }
