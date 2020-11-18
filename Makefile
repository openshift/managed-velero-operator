include boilerplate/generated-includes.mk

SHELL := /usr/bin/env bash

OPERATOR_DOCKERFILE = ./build/Dockerfile

# Include shared Makefiles
include project.mk
include standard.mk

default: gobuild

# Extend Makefile after here

.PHONY: docker-build
docker-build: build

.PHONY: generate
generate:
	operator-sdk generate k8s
	operator-sdk generate crds
	openapi-gen --logtostderr=true \
		-i ./pkg/apis/managed/v1alpha2 \
		-o "" \
		-O zz_generated.openapi \
		-p ./pkg/apis/managed/v1alpha2 \
		-h /dev/null \
		-r "-"

.PHONY: boilerplate-update
boilerplate-update:
	@boilerplate/update
