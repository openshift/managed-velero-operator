#!/bin/bash

# AppSRE team CD

set -exv

CURRENT_DIR=$(dirname "$0")
CRD_DIR="$CURRENT_DIR"/../deploy/crds

if [[ -d $CRD_DIR ]]; then
	python "$CURRENT_DIR"/validate_yaml.py $CRD_DIR

	if [ "$?" != "0" ]; then
	    exit 1
	fi

else
	echo "WARNING: No crds for validation"
fi

BASE_IMG="managed-velero-operator"
IMG="${BASE_IMG}:latest"

BUILD_CMD="docker build" IMG="$IMG" make docker-build
