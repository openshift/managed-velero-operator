package velero

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	minterv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// MockManager mocks the controller-runtime Manager interface.
type MockManager struct {}

func (m *MockManager) Add(manager.Runnable) error {
	return nil
}

func (m *MockManager) Elected() <-chan struct{} {
	return nil
}

func (m *MockManager) SetFields(interface{}) error {
	return nil
}

func (m *MockManager) AddMetricsExtraHandler(path string, handler http.Handler) error {
	return nil
}

func (m *MockManager) AddHealthzCheck(name string, check healthz.Checker) error {
	return nil
}

func (m *MockManager) AddReadyzCheck(name string, check healthz.Checker) error {
	return nil
}

func (m *MockManager) Start(<-chan struct{}) error {
	return nil
}

func (m *MockManager) GetConfig() *rest.Config {
	return nil
}

func (m *MockManager) GetScheme() *runtime.Scheme {
	return nil
}

func (m *MockManager) GetClient() client.Client {
	return nil
}

func (m *MockManager) GetFieldIndexer() client.FieldIndexer {
	return nil
}

func (m *MockManager) GetCache() cache.Cache {
	return nil
}

func (m *MockManager) GetEventRecorderFor(name string) record.EventRecorder {
	return nil
}

func (m *MockManager) GetRESTMapper() meta.RESTMapper {
	return nil
}

func (m *MockManager) GetAPIReader() client.Reader {
	return nil
}

func (m *MockManager) GetWebhookServer() *webhook.Server {
	return nil
}

var (
)

func TestVeleroReconciler(t *testing.T) {
	var mgr manager.Manager = &MockManager{}

	// AWS Regions: https://docs.aws.amazon.com/general/latest/gr/rande.html
	// GCP Regions: https://cloud.google.com/compute/docs/regions-zones/#locations
	var tests = []struct {
		config             *configv1.InfrastructureStatus
		wantErr            bool
		wantRegionInChina  bool
		wantLocationConfig map[string]string
	}{
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.NonePlatformType,
				},
			},
			wantErr: true,
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "us-east-2",
					},
				},
			},
			wantLocationConfig: map[string]string {"region": "us-east-2"},
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "us-east-1",
					},
				},
			},
			wantLocationConfig: map[string]string {"region": "us-east-1"},
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "us-west-1",
					},
				},
			},
			wantLocationConfig: map[string]string {"region": "us-west-1"},
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "us-west-2",
					},
				},
			},
			wantLocationConfig: map[string]string {"region": "us-west-2"},
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "af-south-1",
					},
				},
			},
			wantLocationConfig: map[string]string {"region": "af-south-1"},
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "ap-east-1",
					},
				},
			},
			wantLocationConfig: map[string]string {"region": "ap-east-1"},
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "ap-south-1",
					},
				},
			},
			wantLocationConfig: map[string]string {"region": "ap-south-1"},
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "ap-northeast-3",
					},
				},
			},
			wantLocationConfig: map[string]string {"region": "ap-northeast-3"},
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "ap-northeast-2",
					},
				},
			},
			wantLocationConfig: map[string]string {"region": "ap-northeast-2"},
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "ap-southeast-1",
					},
				},
			},
			wantLocationConfig: map[string]string {"region": "ap-southeast-1"},
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "ap-southeast-2",
					},
				},
			},
			wantLocationConfig: map[string]string {"region": "ap-southeast-2"},
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "ap-northeast-1",
					},
				},
			},
			wantLocationConfig: map[string]string {"region": "ap-northeast-1"},
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "ca-central-1",
					},
				},
			},
			wantLocationConfig: map[string]string {"region": "ca-central-1"},
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "cn-north-1",
					},
				},
			},
			wantRegionInChina: true,
			wantLocationConfig: map[string]string {"region": "cn-north-1"},
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "cn-northwest-1",
					},
				},
			},
			wantRegionInChina: true,
			wantLocationConfig: map[string]string {"region": "cn-northwest-1"},
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "eu-central-1",
					},
				},
			},
			wantLocationConfig: map[string]string {"region": "eu-central-1"},
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "eu-west-1",
					},
				},
			},
			wantLocationConfig: map[string]string {"region": "eu-west-1"},
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "eu-west-2",
					},
				},
			},
			wantLocationConfig: map[string]string {"region": "eu-west-2"},
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "eu-south-1",
					},
				},
			},
			wantLocationConfig: map[string]string {"region": "eu-south-1"},
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "eu-west-3",
					},
				},
			},
			wantLocationConfig: map[string]string {"region": "eu-west-3"},
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "eu-north-1",
					},
				},
			},
			wantLocationConfig: map[string]string {"region": "eu-north-1"},
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "me-south-1",
					},
				},
			},
			wantLocationConfig: map[string]string {"region": "me-south-1"},
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.AWSPlatformType,
					AWS: &configv1.AWSPlatformStatus{
						Region: "sa-east-1",
					},
				},
			},
			wantLocationConfig: map[string]string {"region": "sa-east-1"},
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.GCPPlatformType,
					GCP: &configv1.GCPPlatformStatus{
						Region: "asia-east1",
					},
				},
			},
			wantLocationConfig: nil,
			// FIXME wantRegionInChina: ??? (Taiwan)
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.GCPPlatformType,
					GCP: &configv1.GCPPlatformStatus{
						Region: "asia-east2",
					},
				},
			},
			wantLocationConfig: nil,
			// FIXME wantRegionInChina: true (Hong Kong)
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.GCPPlatformType,
					GCP: &configv1.GCPPlatformStatus{
						Region: "asia-northeast1",
					},
				},
			},
			wantLocationConfig: nil,
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.GCPPlatformType,
					GCP: &configv1.GCPPlatformStatus{
						Region: "asia-northeast2",
					},
				},
			},
			wantLocationConfig: nil,
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.GCPPlatformType,
					GCP: &configv1.GCPPlatformStatus{
						Region: "asia-northeast3",
					},
				},
			},
			wantLocationConfig: nil,
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.GCPPlatformType,
					GCP: &configv1.GCPPlatformStatus{
						Region: "asia-south1",
					},
				},
			},
			wantLocationConfig: nil,
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.GCPPlatformType,
					GCP: &configv1.GCPPlatformStatus{
						Region: "asia-southeast1",
					},
				},
			},
			wantLocationConfig: nil,
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.GCPPlatformType,
					GCP: &configv1.GCPPlatformStatus{
						Region: "australia-southeast1",
					},
				},
			},
			wantLocationConfig: nil,
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.GCPPlatformType,
					GCP: &configv1.GCPPlatformStatus{
						Region: "europe-north1",
					},
				},
			},
			wantLocationConfig: nil,
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.GCPPlatformType,
					GCP: &configv1.GCPPlatformStatus{
						Region: "europe-west1",
					},
				},
			},
			wantLocationConfig: nil,
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.GCPPlatformType,
					GCP: &configv1.GCPPlatformStatus{
						Region: "europe-west2",
					},
				},
			},
			wantLocationConfig: nil,
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.GCPPlatformType,
					GCP: &configv1.GCPPlatformStatus{
						Region: "europe-west3",
					},
				},
			},
			wantLocationConfig: nil,
		},
		{
			config: &configv1.InfrastructureStatus{
				PlatformStatus: &configv1.PlatformStatus{
					Type: configv1.GCPPlatformType,
					GCP: &configv1.GCPPlatformStatus{
						Region: "europe-west4",
					},
				},
			},
			wantLocationConfig: nil,
		},
	}

	for _, tt := range tests {
		var name string

		switch tt.config.PlatformStatus.Type {
		case configv1.AWSPlatformType:
			name = fmt.Sprintf("aws-%s", tt.config.PlatformStatus.AWS.Region)
		case configv1.GCPPlatformType:
			name = fmt.Sprintf("gcp-%s", tt.config.PlatformStatus.GCP.Region)
		default:
			name = fmt.Sprintf("Unsupported platform (%s)", tt.config.PlatformStatus.Type)
		}

		t.Run(name, func(t *testing.T) {
			r, err := newVeleroReconciler(mgr, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("newVeleroReconciler() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}

			switch tt.config.PlatformStatus.Type {
			case configv1.AWSPlatformType:
				if _, ok := r.(*ReconcileVeleroAWS); !ok {
					t.Errorf("newVeleroReconciler() produced a %T, wanted *velero.ReconcileVeleroAWS", r)
				}
			case configv1.GCPPlatformType:
				if _, ok := r.(*ReconcileVeleroGCP); !ok {
					t.Errorf("newVeleroReconciler() produced a %t, wanted *velero.ReconcileVeleroGCP", r)
				}
			}

			regionInChina := r.RegionInChina()
			if regionInChina != tt.wantRegionInChina {
				t.Errorf("RegionInChina() returned %t, wanted %t", regionInChina, tt.wantRegionInChina)
			}

			imageRegistry := r.GetImageRegistry()
			if !tt.wantRegionInChina && imageRegistry != veleroImageRegistry {
				t.Errorf("GetImageRegistry() returned %s, wanted %s", imageRegistry, veleroImageRegistry)
			} else if tt.wantRegionInChina && imageRegistry != veleroImageRegistryCN {
				t.Errorf("GetImageRegistry() returned %s, wanted %s", imageRegistry, veleroImageRegistryCN)
			}

			locationConfig := r.GetLocationConfig()
			if !reflect.DeepEqual(locationConfig, tt.wantLocationConfig) {
				t.Errorf("GetLocationConfig() returned %v, wanted %v", locationConfig, tt.wantLocationConfig)
			}

			credentialsRequest, err := r.CredentialsRequest("namespace", "bucket")
			if err != nil {
				t.Errorf("CredentialsRequest() failed: %v", err)
			} else {
				var object runtime.Object

				// Test that we can decode the ProviderSpec.
				codec, _ := minterv1.NewCodec()
				switch tt.config.PlatformStatus.Type {
				case configv1.AWSPlatformType:
					object = &minterv1.AWSProviderSpec{}
				case configv1.GCPPlatformType:
					object = &minterv1.GCPProviderSpec{}
				}
				if err = codec.DecodeProviderSpec(credentialsRequest.Spec.ProviderSpec, object); err != nil {
					t.Errorf("Unable to decode ProviderSpec for CredentialsRequest: %v", err)
				}
			}
		})
	}
}
