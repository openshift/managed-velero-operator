# Project specific values
OPERATOR_NAME?=managed-velero-operator
OPERATOR_NAMESPACE?=openshift-velero

IMAGE_REGISTRY?=quay.io
IMAGE_REPOSITORY?=${USER}
IMAGE_NAME?=${OPERATOR_NAME}
CATALOG_REGISTRY_ORGANIZATION?=app-sre

VERSION_MAJOR?=0
VERSION_MINOR?=1

YAML_DIRECTORY?=manifests
SELECTOR_SYNC_SET_TEMPLATE_DIR?=scripts/templates/
GIT_ROOT?=$(shell git rev-parse --show-toplevel 2>&1)

# WARNING: REPO_NAME will default to the current directory if there are no remotes
REPO_NAME?=$(shell basename $$((git config --get-regex remote\.*\.url 2>/dev/null | cut -d ' ' -f2 || pwd) | head -n1 | sed 's|.git||g'))

SELECTOR_SYNC_SET_DESTINATION?=${GIT_ROOT}/build/templates/olm-artifacts-template.yaml.tmpl

IN_DOCKER_CONTAINER?=false

GEN_SYNCSET=scripts/generate_syncset.py -t ${SELECTOR_SYNC_SET_TEMPLATE_DIR} -y ${YAML_DIRECTORY} -d ${SELECTOR_SYNC_SET_DESTINATION} -r ${REPO_NAME}
