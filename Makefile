SHELL := /usr/bin/env bash

# Include shared Makefiles
include project.mk
include standard.mk
include functions.mk

default: generate-syncset gobuild

# Extend Makefile after here

# Build the docker image
.PHONY: docker-build
docker-build:
	$(MAKE) build

# Push the docker image
.PHONY: docker-push
docker-push:
	$(MAKE) push

.PHONY: generate-syncset
generate-syncset:
	if [ "${IN_DOCKER_CONTAINER}" == "true" ]; then \
		docker run --rm -v `pwd -P`:`pwd -P` python:2.7.15 /bin/sh -c "cd `pwd`; pip install oyaml; `pwd`/${GEN_SYNCSET}"; \
	else \
		${GEN_SYNCSET}; \
	fi

