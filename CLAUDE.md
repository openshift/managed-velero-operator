# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

The Managed Velero Operator is a Kubernetes operator for OpenShift Dedicated v4 that automates the deployment and configuration of Velero backup software. It manages cloud storage buckets (AWS S3/GCP) and ensures proper security settings for backups.

## Development Commands

### Build and Test
```bash
# Run tests
make test

# Build the operator binary
make go-build

# Build Docker image
make docker-build

# Build and push Docker image
make docker-build-push-one

# Clean build artifacts
make clean
```

### Linting and Code Quality
```bash
# Run Go linter (uses golangci-lint)
make go-check

# The linter configuration is in .golangci.yml with specific disabled checks:
# - depguard, gochecknoglobals, gochecknoinits, lll
```

### Development
```bash
# Default target runs: go-check, go-test, go-build
make

# Push images to personal registry (set IMAGE_REPOSITORY=<username> first)
export IMAGE_REPOSITORY=<username>
make build
make push
```

## Architecture

### Core Components

**API Layer (`api/v1alpha2/`)**
- `VeleroInstall` - Main custom resource that triggers operator reconciliation
- `VeleroInstallSpec` - Currently empty, configuration is inferred from cluster
- `VeleroInstallStatus` - Tracks storage bucket state and provisioning status
- `StorageBucket` - Contains bucket name, provisioning status, and sync timestamps

**Controller Layer (`controllers/velero/`)**
- `VeleroInstallController` - Main reconciliation logic for VeleroInstall resources
- Orchestrates bucket creation, Velero deployment, and credential management

**Storage Layer (`pkg/storage/`)**
- **S3 (`pkg/storage/s3/`)** - AWS S3 bucket management, encryption, lifecycle policies
- **GCS (`pkg/storage/gcs/`)** - Google Cloud Storage bucket management
- **Base (`pkg/storage/base/`)** - Common storage interfaces and utilities
- **Constants (`pkg/storage/constants/`)** - Shared storage configuration values

**Velero Integration (`pkg/velero/`)**
- CRD management and installation
- Velero-specific configuration and deployment logic

### Key Reconciliation Flow

1. **Platform Validation** - Ensures operator runs on supported platforms (AWS/GCP with IPI)
2. **Storage Bucket Provisioning** - Creates encrypted buckets with proper lifecycle and access policies
3. **Credential Management** - Uses OpenShift's Cloud Credential Operator for secure access
4. **Velero Deployment** - Installs Velero CRDs, configures BackupStorageLocation and VolumeSnapshotLocation
5. **Ongoing Enforcement** - Periodically validates and re-enforces bucket security settings

### Important Design Patterns

- **Cloud Provider Abstraction** - Storage layer abstracts AWS S3 vs GCP differences
- **Security-First** - All buckets are encrypted, public access blocked, proper IAM policies
- **Credential Operator Integration** - Leverages OpenShift's credential management rather than manual secrets
- **Reconcile Loop** - Continuously monitors and maintains desired state
- **FIPS Compliance** - Built with FIPS_ENABLED=true for security requirements

### Dependencies

- **Velero**: Uses forked version (`github.com/openshift/velero`) instead of upstream
- **OpenShift APIs**: Heavy integration with OpenShift-specific resources (config/v1, cloud credentials)
- **Controller Runtime**: Standard Kubernetes controller patterns
- **Cloud SDKs**: AWS SDK and Google Cloud APIs for bucket management

### Testing Strategy

- Unit tests for storage providers (`pkg/storage/*/`)
- Controller tests (`controllers/velero/velero_test.go`)
- Mock testing with `googleapis/google-cloud-go-testing` for GCP

### Deployment

- Operator runs in `openshift-velero` namespace (not standard `velero` namespace)
- Uses boilerplate makefiles from OpenShift golang-osd-operator conventions
- Container images built for OpenShift registry with specific versioning scheme