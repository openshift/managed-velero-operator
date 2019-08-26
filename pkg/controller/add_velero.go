package controller

import (
	"github.com/openshift/managed-velero-operator/pkg/controller/velero"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, velero.Add)
}
