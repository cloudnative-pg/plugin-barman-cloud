package replicacluster

import (
	cloudnativepgv1 "github.com/cloudnative-pg/api/pkg/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	pluginBarmanCloudV1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/objectstore"
)

type testCaseFactory interface {
	createReplicaClusterTestResources(namespace string) replicaClusterTestResources
}

const (
	// Size of the PVCs for the object stores and the cluster instances.
	size                   = "1Gi"
	srcObjectStoreName     = "source"
	srcClusterName         = "source"
	srcBackupName          = "source"
	replicaObjectStoreName = "replica"
	replicaClusterName     = "replica"
	replicaBackupName      = "replica"
	minioSrc               = "minio-src"
	minioReplica           = "minio-replica"
	gcsSrc                 = "fakegcs-src"
	azuriteSrc             = "azurite-src"
	azuriteReplica         = "azurite-replica"
)

type replicaClusterTestResources struct {
	SrcObjectStoreResources     *objectstore.Resources
	SrcObjectStore              *pluginBarmanCloudV1.ObjectStore
	SrcCluster                  *cloudnativepgv1.Cluster
	SrcBackup                   *cloudnativepgv1.Backup
	ReplicaObjectStoreResources *objectstore.Resources
	ReplicaObjectStore          *pluginBarmanCloudV1.ObjectStore
	ReplicaCluster              *cloudnativepgv1.Cluster
	ReplicaBackup               *cloudnativepgv1.Backup
}

type s3ReplicaClusterFactory struct{}

func (f s3ReplicaClusterFactory) createReplicaClusterTestResources(namespace string) replicaClusterTestResources {
	result := replicaClusterTestResources{}

	result.SrcObjectStoreResources = objectstore.NewMinioObjectStoreResources(namespace, minioSrc)
	result.SrcObjectStore = objectstore.NewMinioObjectStore(namespace, srcObjectStoreName, minioSrc)
	result.SrcCluster = newSrcCluster(namespace)
	result.SrcBackup = newSrcBackup(namespace)
	result.ReplicaObjectStoreResources = objectstore.NewMinioObjectStoreResources(namespace, minioReplica)
	result.ReplicaObjectStore = objectstore.NewMinioObjectStore(namespace, replicaObjectStoreName, minioReplica)
	result.ReplicaCluster = newReplicaCluster(namespace)
	result.ReplicaBackup = newReplicaBackup(namespace)

	return result
}

type gcsReplicaClusterFactory struct{}

func (f gcsReplicaClusterFactory) createReplicaClusterTestResources(namespace string) replicaClusterTestResources {
	result := replicaClusterTestResources{}

	result.SrcObjectStoreResources = objectstore.NewGCSObjectStoreResources(namespace, gcsSrc)
	result.SrcObjectStore = objectstore.NewGCSObjectStore(namespace, srcObjectStoreName, gcsSrc)
	result.SrcCluster = newSrcCluster(namespace)
	result.SrcCluster.Spec.ExternalClusters[1].PluginConfiguration.Parameters["barmanObjectName"] = srcObjectStoreName
	result.SrcBackup = newSrcBackup(namespace)
	// fake-gcs-server requires the STORAGE_EMULATOR_HOST environment variable to be set.
	// We would have to set that variable to different values to point to the different fake-gcs-server instances,
	// however the plugin does not support injecting in the sidecar variables with the same name and different values,
	// so we can only point to a single instance. However, this reflects the real-world scenario, since GCS always
	// points to the same endpoint.
	result.ReplicaObjectStoreResources = &objectstore.Resources{}
	result.ReplicaObjectStore = nil
	result.ReplicaCluster = newReplicaCluster(namespace)
	result.ReplicaCluster.Spec.Plugins[0].Parameters["barmanObjectName"] = srcObjectStoreName
	result.ReplicaCluster.Spec.ExternalClusters[1].PluginConfiguration.Parameters["barmanObjectName"] = srcObjectStoreName
	result.ReplicaBackup = newReplicaBackup(namespace)

	return result
}

type azuriteReplicaClusterFactory struct{}

func (f azuriteReplicaClusterFactory) createReplicaClusterTestResources(namespace string) replicaClusterTestResources {
	result := replicaClusterTestResources{}

	result.SrcObjectStoreResources = objectstore.NewAzuriteObjectStoreResources(namespace, azuriteSrc)
	result.SrcObjectStore = objectstore.NewAzuriteObjectStore(namespace, srcObjectStoreName, azuriteSrc)
	result.SrcCluster = newSrcCluster(namespace)
	result.SrcBackup = newSrcBackup(namespace)
	result.ReplicaObjectStoreResources = objectstore.NewAzuriteObjectStoreResources(namespace, azuriteReplica)
	result.ReplicaObjectStore = objectstore.NewAzuriteObjectStore(namespace, replicaObjectStoreName, azuriteReplica)
	result.ReplicaCluster = newReplicaCluster(namespace)
	result.ReplicaBackup = newReplicaBackup(namespace)

	return result
}

func newSrcCluster(namespace string) *cloudnativepgv1.Cluster {
	cluster := &cloudnativepgv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "postgresql.cnpg.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      srcClusterName,
			Namespace: namespace,
		},
		Spec: cloudnativepgv1.ClusterSpec{
			Instances:       2,
			ImagePullPolicy: corev1.PullAlways,
			Plugins: []cloudnativepgv1.PluginConfiguration{
				{
					Name: "barman-cloud.cloudnative-pg.io",
					Parameters: map[string]string{
						"barmanObjectName": srcObjectStoreName,
					},
					IsWALArchiver: ptr.To(true),
				},
			},
			PostgresConfiguration: cloudnativepgv1.PostgresConfiguration{
				Parameters: map[string]string{
					"log_min_messages": "DEBUG4",
				},
			},
			StorageConfiguration: cloudnativepgv1.StorageConfiguration{
				Size: size,
			},
			ReplicaCluster: &cloudnativepgv1.ReplicaClusterConfiguration{
				Primary: "source",
				Source:  "replica",
			},
			ExternalClusters: []cloudnativepgv1.ExternalCluster{
				{
					Name: "source",
					PluginConfiguration: &cloudnativepgv1.PluginConfiguration{
						Name: "barman-cloud.cloudnative-pg.io",
						Parameters: map[string]string{
							"barmanObjectName": srcObjectStoreName,
							"serverName":       srcClusterName,
						},
					},
				},
				{
					Name: "replica",
					PluginConfiguration: &cloudnativepgv1.PluginConfiguration{
						Name: "barman-cloud.cloudnative-pg.io",
						Parameters: map[string]string{
							"barmanObjectName": replicaObjectStoreName,
							"serverName":       replicaObjectStoreName,
						},
					},
				},
			},
		},
	}

	return cluster
}

func newSrcBackup(namespace string) *cloudnativepgv1.Backup {
	return &cloudnativepgv1.Backup{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Backup",
			APIVersion: "postgresql.cnpg.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      srcBackupName,
			Namespace: namespace,
		},
		Spec: cloudnativepgv1.BackupSpec{
			Cluster: cloudnativepgv1.LocalObjectReference{
				Name: srcClusterName,
			},
			Method: "plugin",
			PluginConfiguration: &cloudnativepgv1.BackupPluginConfiguration{
				Name: "barman-cloud.cloudnative-pg.io",
			},
			Target: "primary",
		},
	}
}

func newReplicaBackup(namespace string) *cloudnativepgv1.Backup {
	return &cloudnativepgv1.Backup{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Backup",
			APIVersion: "postgresql.cnpg.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      replicaBackupName,
			Namespace: namespace,
		},
		Spec: cloudnativepgv1.BackupSpec{
			Cluster: cloudnativepgv1.LocalObjectReference{
				Name: replicaClusterName,
			},
			Method: "plugin",
			PluginConfiguration: &cloudnativepgv1.BackupPluginConfiguration{
				Name: "barman-cloud.cloudnative-pg.io",
			},
		},
	}
}

func newReplicaCluster(namespace string) *cloudnativepgv1.Cluster {
	cluster := &cloudnativepgv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "postgresql.cnpg.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      replicaClusterName,
			Namespace: namespace,
		},
		Spec: cloudnativepgv1.ClusterSpec{
			Instances:       2,
			ImagePullPolicy: corev1.PullAlways,
			Bootstrap: &cloudnativepgv1.BootstrapConfiguration{
				Recovery: &cloudnativepgv1.BootstrapRecovery{
					Source: "source",
				},
			},
			Plugins: []cloudnativepgv1.PluginConfiguration{
				{
					Name: "barman-cloud.cloudnative-pg.io",
					Parameters: map[string]string{
						"barmanObjectName": replicaObjectStoreName,
					},
					IsWALArchiver: ptr.To(true),
				},
			},
			PostgresConfiguration: cloudnativepgv1.PostgresConfiguration{
				Parameters: map[string]string{
					"log_min_messages": "DEBUG4",
				},
			},
			ExternalClusters: []cloudnativepgv1.ExternalCluster{
				{
					Name: "source",
					PluginConfiguration: &cloudnativepgv1.PluginConfiguration{
						Name: "barman-cloud.cloudnative-pg.io",
						Parameters: map[string]string{
							"barmanObjectName": srcObjectStoreName,
							"serverName":       srcClusterName,
						},
					},
				},
				{
					Name: "replica",
					PluginConfiguration: &cloudnativepgv1.PluginConfiguration{
						Name: "barman-cloud.cloudnative-pg.io",
						Parameters: map[string]string{
							"barmanObjectName": replicaObjectStoreName,
							"serverName":       replicaObjectStoreName,
						},
					},
				},
			},
			ReplicaCluster: &cloudnativepgv1.ReplicaClusterConfiguration{
				Primary: "source",
				Source:  "source",
			},
			StorageConfiguration: cloudnativepgv1.StorageConfiguration{
				Size: size,
			},
		},
	}

	return cluster
}
