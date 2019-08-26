FROM registry.svc.ci.openshift.org/openshift/release:golang-1.12 AS builder

# Allow specifying a GOPROXY cache during build to speed up dependency resolution
ARG GOPROXY
ENV GOPROXY=$GOPROXY

ENV OPERATOR_PATH=/go/src/github.com/openshift/managed-velero-operator
COPY . ${OPERATOR_PATH}
WORKDIR ${OPERATOR_PATH}
RUN make gobuild

####
FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

ENV OPERATOR_PATH=/go/src/github.com/openshift/managed-velero-operator \
    OPERATOR_BIN=managed-velero-operator \
    USER_UID=1001 \
    USER_NAME=managed-velero-operator

# install operator binary
COPY --from=builder ${OPERATOR_PATH}/build/_output/bin/${OPERATOR_BIN} /usr/local/bin/${OPERATOR_BIN}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}

LABEL io.openshift.managed.name="managed-velero-operator" \
      io.openshift.managed.description="Operator to manage installation of Velero in managed OpenShift environments"
