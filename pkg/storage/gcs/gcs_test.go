package gcs

import (
	"context"
	"reflect"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
)

func TestNewDriver(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		region    string
	}{
		{
			name:      "create a gcs driver",
			namespace: "openshift-velero",
			region:    "us-east-1",
		},
	}

	expectedType := "*gcs.driver"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kubeClient := setUpTestClient(t, setUpInstance(t))

			infraStatus := &configv1.InfrastructureStatus{
				InfrastructureName: "managed-velero-fake-cluster",
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.GCPPlatformType,
					GCP: &configv1.GCPPlatformStatus{
						ProjectID: "",
						Region:    "",
					},
				},
			}

			actualDriver := NewDriver(context.Background(), infraStatus, kubeClient, tt.namespace)

			// test that NewDriver actually returns the expected type
			if reflect.TypeOf(actualDriver).String() != expectedType {
				t.Fatalf("NewDriver(): expected %s got %s", expectedType, reflect.TypeOf(actualDriver))
			}

			// test that the new driver has all the expected fields
			if reflect.TypeOf(actualDriver.Context).String() != "*context.emptyCtx" {
				t.Errorf("NewDriver(): driver.Context expected Context got %s", reflect.TypeOf(actualDriver.Context).String())
			}

			expectedConfig := &GCS{
				Region:    infraStatus.PlatformStatus.GCP.Region,
				Project:   infraStatus.PlatformStatus.GCP.ProjectID,
				InfraName: infraStatus.InfrastructureName,
			}
			if !reflect.DeepEqual(actualDriver.Config, expectedConfig) {
				t.Errorf("NewDriver(): driver.Config expected %v got %v", expectedConfig, actualDriver.Config)
			}

			if actualDriver.kubeClient != kubeClient {
				t.Errorf("NewDriver(): driver.kubeClient expected %v got %v", kubeClient, actualDriver.kubeClient)
			}

			if actualDriver.Namespace != tt.namespace {
				t.Errorf("NewDriver(): driver.Namespace expected %s got %s", tt.namespace, actualDriver.Namespace)
			}
		})
	}
}
