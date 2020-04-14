# Managed Velero Operator

## Summary

The Managed Velero Operator is used for backups in OpenShift Dedicated v4. It is based on an open source project [Velero]( https://velero.io) that uses a controller model where it monitors custom resources and takes actions based on their states. The Managed Velero Operator dynamically creates and configures prerequisites for Velero and it then deploys the Velero software on the OpenShift Dedicated cluster. Velero backs up the Kubernetes object store: all the deployments, pods, config maps, secrets, etc. In addition to Kubernetes configuration pieces, Velero also handles backing up and any persistent volumes that are attached to the cluster. For OpenShift Dedicated v4, we snapshot the cloud provider's persistent storage volumes.

## What the Managed Velero Operator Does

1. When the Managed Velero Operator starts, it checks whether it is installed on a supported platform. For OpenShift Dedicated v4 environment it validates that 
	+ it is installed on AWS or GCP
	+ it has been installed with installer provisioned infrastructure
	+ it has all the needed details in the cluster's infrastructure configuration to provision Velero

2. Next, the Managed Velero Operator begins the **Reconcile loop**. It checks whether the Velero custom resources are created/installed and it ensures that they are created/installed before it takes any further action.

3. Next, the Managed Velero Operator starts up the manager and controller and waits for the initial configuration.

4. Once the Managed Velero Operator detects its custom resources and understands that it's ready to provision Velero in this cluster, it will check if an object storage bucket already has been provisioned to store the Kubernetes object store, the metadata and the details of the backup. If the Managed Velero Operator detects that there isn't an object storage bucket defined in its custom resources, it will provision an object storage bucket for that use. **The Managed Velero Operator will ensure the following settings when the bucket is created and it will check periodically to ensure that the bucket settings are re-enforced.**
	+ the bucket is encrypted
	+ the public access to the bucket is enforced turned off
	+ the permissions and the life cycle settings on the bucket are correct

5. Next, the Managed Velero Operator will configure and install the Velero software. This includes ensuring that setup manifests are installed, and Velero custom resources such as the volume storage location and the backup storage location are specified. This step also provisions credentials for Velero to access the object storage bucket through the cluster credentials operator and a credentials request custom resource that is part of OpenShift v4.

6. Finally, the Managed Velero Operator completes the **Reconcile loop**. 

The Managed Velero Operator will listen to changes in settings and custom resources and periodically run the Reconcile loop to change the settings back to what it expects.

## Requirements

+ Access to OpenShift version 4.1 or later.

## How to run unit tests

Launch the tests locally by running

```shell
make test
```
Once the local tests passed, submit your Pull Request and wait for the automated tests to complete.



## Restoring from a Backup
### Assumptions

This document assumes some familiarity with Velero and it's concepts. Many terms and concepts from Velero's own documentation will be used. Please visit the Velero documentation for additional help: https://velero.io/docs/v1.1.0/.

Contrary to the standard Velero documentation OSD runs Velero and the Managed Velero Operator in the namespace 'openshift-velero' rather than 'velero'.

### Process
https://velero.io/docs/v1.1.0/disaster-case/

This requires there to be a running Velero server and Velero CRDs in the running cluster.

1. First you must find the backup that you want to restore

	`oc get backup -n openshift-velero`

1. Choose the one you want to restore and issue the restore command

	`velero client config set namespace=openshift-velero`
	`velero restore create --from-backup <backup-name>`

## Building the Docker Image

To build a Docker image run the following command:

```shell
make docker-build
```

### Pushing to your personal Quay repo

To push to your personal Quay repo, use the following:
```shell
export IMAGE_REPOSITORY=<username>
make build
make push
```
