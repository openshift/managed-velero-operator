package constants

const (
	// StorageAccountPrefix must only contain lower case letter or numbers
	AzureStorageAccountPrefix = "velerobackups"
	// AzureStorageAccountTagBackupLocation is Azure specific tag on storageAccount
	// Azure tags cannot contain <>%&\?/ characters
	AzureStorageAccountTagBackupLocation = "velero.io_backup-location"
	// AzureStorageAccountTagInfrastructureName is Azure specific tag on storageAccount
	AzureStorageAccountTagInfrastructureName = "velero.io_infrastructureName"

	StorageBucketPrefix                = "managed-velero-backups-"
	DefaultVeleroBackupStorageLocation = "default"
	BucketTagBackupStorageLocation     = "velero.io/backup-location"
	BucketTagInfrastructureName        = "velero.io/infrastructureName"
)
