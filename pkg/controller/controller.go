package controller

import (
	"sigs.k8s.io/controller-runtime/pkg/manager"
	configv1 "github.com/openshift/api/config/v1"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager, *configv1.InfrastructureStatus) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, cfg *configv1.InfrastructureStatus) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m, cfg); err != nil {
			return err
		}
	}
	return nil
}
