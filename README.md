# managed-velero-operator

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

#### Pushing to your personal Quay repo

To push to your personal Quay repo, use the following:
```shell
export IMAGE_REPOSITORY=<username>
make build
make push
```
