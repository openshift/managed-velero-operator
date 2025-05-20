package velero

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/sets"
)

var exampleService = &corev1.Service{
	TypeMeta: metav1.TypeMeta{
		Kind:       "Service",
		APIVersion: corev1.SchemeGroupVersion.String(),
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "example",
		Namespace: "default",
		Labels: map[string]string{
			"app": "web",
			"env": "production",
		},
	},
	Spec: corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name:     "https",
				Protocol: corev1.ProtocolTCP,
				Port:     443,
			},
		},
		Selector: map[string]string{
			"app": "web",
		},
		Type: corev1.ServiceTypeClusterIP,
	},
}

func TestGenerateServiceMonitor(t *testing.T) {
	input := exampleService
	output := generateServiceMonitor(input)

	if !reflect.DeepEqual(input.Labels, output.Labels) {
		t.Errorf("Metadata label sets don't match: got %v, want %v", output.Labels, input.Labels)
	}
	if !reflect.DeepEqual(input.Labels, output.Spec.Selector.MatchLabels) {
		t.Errorf("Selector label sets don't match: got %v, want %v", output.Labels, input.Labels)
	}
}

func TestPopulateEndpointsFromServicePorts(t *testing.T) {
	tests := []struct {
		name  string
		ports []corev1.ServicePort
	}{
		{
			name: "Single port",
			ports: []corev1.ServicePort{
				{
					Name: "https",
					Port: 443,
				},
			},
		},
		{
			name: "Two ports",
			ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
				{
					Name: "https",
					Port: 443,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := exampleService
			input.Spec.Ports = tt.ports // overwrite example with test case
			output := populateEndpointsFromServicePorts(input)

			// create sets to more easily anylize results
			inputSet := sets.NewString()
			outputSet := sets.NewString()
			for _, port := range input.Spec.Ports {
				inputSet.Insert(port.Name)
			}
			for _, endpoint := range output {
				outputSet.Insert(endpoint.Port)
			}

			if !inputSet.Equal(outputSet) {
				t.Errorf("Received endpoint list doesn't match: got %v, want %v", outputSet.List(), inputSet.List())
			}
		})
	}
}
