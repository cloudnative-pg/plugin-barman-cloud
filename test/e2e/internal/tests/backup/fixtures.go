/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package backup

import (
	"fmt"
	"net"

	cloudnativepgv1 "github.com/cloudnative-pg/api/pkg/api/v1"
	barmanapi "github.com/cloudnative-pg/barman-cloud/pkg/api"
	"github.com/cloudnative-pg/machinery/pkg/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pluginBarmanCloudV1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/objectstore"
)

const (
	minio   = "minio"
	azurite = "azurite"
	gcs     = "gcs"
	// Size of the PVCs for the object stores and the cluster instances.
	size               = "1Gi"
	srcClusterName     = "source"
	srcBackupName      = "source"
	objectStoreName    = "source"
	dstBackupName      = "restore"
	restoreClusterName = "restore"
)

type testCaseFactory interface {
	createBackupRestoreTestResources(namespace string) backupRestoreTestResources
}

type backupRestoreTestResources struct {
	ObjectStoreResources *objectstore.Resources
	ObjectStore          *pluginBarmanCloudV1.ObjectStore
	SrcCluster           *cloudnativepgv1.Cluster
	SrcBackup            *cloudnativepgv1.Backup
	DstCluster           *cloudnativepgv1.Cluster
	DstBackup            *cloudnativepgv1.Backup
}

type s3BackupPluginBackupPluginRestore struct{}

func (s s3BackupPluginBackupPluginRestore) createBackupRestoreTestResources(
	namespace string,
) backupRestoreTestResources {
	result := backupRestoreTestResources{}

	result.ObjectStoreResources = objectstore.NewMinioObjectStoreResources(namespace, minio)
	result.ObjectStore = objectstore.NewMinioObjectStore(namespace, objectStoreName, minio)
	result.SrcCluster = newSrcClusterWithPlugin(namespace)
	result.SrcBackup = newSrcPluginBackup(namespace)
	result.DstCluster = newDstClusterWithPlugin(namespace)
	result.DstBackup = newDstPluginBackup(namespace)

	return result
}

type s3BackupPluginBackupInTreeRestore struct{}

func (s s3BackupPluginBackupInTreeRestore) createBackupRestoreTestResources(
	namespace string,
) backupRestoreTestResources {
	result := backupRestoreTestResources{}

	result.ObjectStoreResources = objectstore.NewMinioObjectStoreResources(namespace, minio)
	result.ObjectStore = objectstore.NewMinioObjectStore(namespace, objectStoreName, minio)
	result.SrcCluster = newSrcClusterWithPlugin(namespace)
	result.SrcBackup = newSrcPluginBackup(namespace)
	result.DstCluster = newDstClusterInTreeS3(namespace)
	result.DstBackup = newDstPluginBackup(namespace)

	return result
}

type s3BackupPluginInTreeBackupPluginRestore struct{}

func (s s3BackupPluginInTreeBackupPluginRestore) createBackupRestoreTestResources(
	namespace string,
) backupRestoreTestResources {
	result := backupRestoreTestResources{}

	result.ObjectStoreResources = objectstore.NewMinioObjectStoreResources(namespace, minio)
	result.ObjectStore = objectstore.NewMinioObjectStore(namespace, objectStoreName, minio)
	result.SrcCluster = newSrcClusterInTreeS3(namespace)
	result.SrcBackup = newSrcInTreeBackup(namespace)
	result.DstCluster = newDstClusterWithPlugin(namespace)
	result.DstBackup = newDstPluginBackup(namespace)

	return result
}

type azureBackupPluginBackupPluginRestore struct{}

func (a azureBackupPluginBackupPluginRestore) createBackupRestoreTestResources(
	namespace string,
) backupRestoreTestResources {
	result := backupRestoreTestResources{}

	result.ObjectStoreResources = objectstore.NewAzuriteObjectStoreResources(namespace, azurite)
	result.ObjectStore = objectstore.NewAzuriteObjectStore(namespace, objectStoreName, azurite)
	result.SrcCluster = newSrcClusterWithPlugin(namespace)
	result.SrcBackup = newSrcPluginBackup(namespace)
	result.DstCluster = newDstClusterWithPlugin(namespace)
	result.DstBackup = newDstPluginBackup(namespace)

	return result
}

type azureBackupPluginBackupInTreeRestore struct{}

func (a azureBackupPluginBackupInTreeRestore) createBackupRestoreTestResources(
	namespace string,
) backupRestoreTestResources {
	result := backupRestoreTestResources{}

	result.ObjectStoreResources = objectstore.NewAzuriteObjectStoreResources(namespace, azurite)
	result.ObjectStore = objectstore.NewAzuriteObjectStore(namespace, objectStoreName, azurite)
	result.SrcCluster = newSrcClusterWithPlugin(namespace)
	result.SrcBackup = newSrcPluginBackup(namespace)
	result.DstCluster = newDstClusterInTreeAzure(namespace)
	result.DstBackup = newDstPluginBackup(namespace)

	return result
}

type azureBackupPluginInTreeBackupPluginRestore struct{}

func (a azureBackupPluginInTreeBackupPluginRestore) createBackupRestoreTestResources(
	namespace string,
) backupRestoreTestResources {
	result := backupRestoreTestResources{}

	result.ObjectStoreResources = objectstore.NewAzuriteObjectStoreResources(namespace, azurite)
	result.ObjectStore = objectstore.NewAzuriteObjectStore(namespace, objectStoreName, azurite)
	result.SrcCluster = newSrcClusterInTreeAzure(namespace)
	result.SrcBackup = newSrcInTreeBackup(namespace)
	result.DstCluster = newDstClusterWithPlugin(namespace)
	result.DstBackup = newDstPluginBackup(namespace)

	return result
}

type gcsBackupPluginBackupPluginRestore struct{}

func (g gcsBackupPluginBackupPluginRestore) createBackupRestoreTestResources(
	namespace string,
) backupRestoreTestResources {
	result := backupRestoreTestResources{}

	result.ObjectStoreResources = objectstore.NewGCSObjectStoreResources(namespace, gcs)
	result.ObjectStore = objectstore.NewGCSObjectStore(namespace, objectStoreName, gcs)
	result.SrcCluster = newSrcClusterWithPlugin(namespace)
	result.SrcBackup = newSrcPluginBackup(namespace)
	result.DstCluster = newDstClusterWithPlugin(namespace)
	result.DstBackup = newDstPluginBackup(namespace)

	return result
}

type gcsBackupPluginBackupInTreeRestore struct{}

func (g gcsBackupPluginBackupInTreeRestore) createBackupRestoreTestResources(
	namespace string,
) backupRestoreTestResources {
	result := backupRestoreTestResources{}

	result.ObjectStoreResources = objectstore.NewGCSObjectStoreResources(namespace, gcs)
	result.ObjectStore = objectstore.NewGCSObjectStore(namespace, objectStoreName, gcs)
	result.SrcCluster = newSrcClusterWithPlugin(namespace)
	result.SrcBackup = newSrcPluginBackup(namespace)
	result.DstCluster = newDstClusterInTreeGCS(namespace)
	result.DstBackup = newDstPluginBackup(namespace)

	return result
}

type gcsBackupPluginInTreeBackupPluginRestore struct{}

func (g gcsBackupPluginInTreeBackupPluginRestore) createBackupRestoreTestResources(
	namespace string,
) backupRestoreTestResources {
	result := backupRestoreTestResources{}

	result.ObjectStoreResources = objectstore.NewGCSObjectStoreResources(namespace, gcs)
	result.ObjectStore = objectstore.NewGCSObjectStore(namespace, objectStoreName, gcs)
	result.SrcCluster = newSrcClusterInTreeGCS(namespace)
	result.SrcBackup = newSrcInTreeBackup(namespace)
	result.DstCluster = newDstClusterWithPlugin(namespace)
	result.DstBackup = newDstPluginBackup(namespace)

	return result
}

func newSrcPluginBackup(namespace string) *cloudnativepgv1.Backup {
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
		},
	}
}

func newSrcInTreeBackup(namespace string) *cloudnativepgv1.Backup {
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
			Method: "barmanObjectStore",
		},
	}
}

func newDstPluginBackup(namespace string) *cloudnativepgv1.Backup {
	return &cloudnativepgv1.Backup{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Backup",
			APIVersion: "postgresql.cnpg.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dstBackupName,
			Namespace: namespace,
		},
		Spec: cloudnativepgv1.BackupSpec{
			Cluster: cloudnativepgv1.LocalObjectReference{
				Name: restoreClusterName,
			},
			Method: "plugin",
			PluginConfiguration: &cloudnativepgv1.BackupPluginConfiguration{
				Name: "barman-cloud.cloudnative-pg.io",
			},
		},
	}
}

func newSrcClusterWithPlugin(namespace string) *cloudnativepgv1.Cluster {
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
						"barmanObjectName": objectStoreName,
					},
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
		},
	}

	return cluster
}

func newDstClusterWithPlugin(namespace string) *cloudnativepgv1.Cluster {
	cluster := &cloudnativepgv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "postgresql.cnpg.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      restoreClusterName,
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
						"barmanObjectName": objectStoreName,
					},
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
							"barmanObjectName": objectStoreName,
							"serverName":       srcClusterName,
						},
					},
				},
			},
			StorageConfiguration: cloudnativepgv1.StorageConfiguration{
				Size: size,
			},
		},
	}

	return cluster
}

func newSrcClusterInTreeS3(namespace string) *cloudnativepgv1.Cluster {
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
			StorageConfiguration: cloudnativepgv1.StorageConfiguration{
				Size: size,
			},
			PostgresConfiguration: cloudnativepgv1.PostgresConfiguration{
				Parameters: map[string]string{
					"log_min_messages": "DEBUG4",
				},
			},
			Backup: &cloudnativepgv1.BackupConfiguration{
				BarmanObjectStore: &cloudnativepgv1.BarmanObjectStoreConfiguration{
					BarmanCredentials: barmanapi.BarmanCredentials{
						AWS: &barmanapi.S3Credentials{
							AccessKeyIDReference: &api.SecretKeySelector{
								LocalObjectReference: api.LocalObjectReference{
									Name: minio,
								},
								Key: "ACCESS_KEY_ID",
							},
							SecretAccessKeyReference: &api.SecretKeySelector{
								LocalObjectReference: api.LocalObjectReference{
									Name: minio,
								},
								Key: "ACCESS_SECRET_KEY",
							},
						},
					},
					EndpointURL:     "http://" + net.JoinHostPort(minio, "9000"),
					DestinationPath: "s3://backups/",
				},
			},
		},
	}

	return cluster
}

func newDstClusterInTreeS3(namespace string) *cloudnativepgv1.Cluster {
	cluster := &cloudnativepgv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "postgresql.cnpg.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      restoreClusterName,
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
			PostgresConfiguration: cloudnativepgv1.PostgresConfiguration{
				Parameters: map[string]string{
					"log_min_messages": "DEBUG4",
				},
			},
			Plugins: []cloudnativepgv1.PluginConfiguration{
				{
					Name: "barman-cloud.cloudnative-pg.io",
					Parameters: map[string]string{
						"barmanObjectName": objectStoreName,
					},
				},
			},
			ExternalClusters: []cloudnativepgv1.ExternalCluster{
				{
					Name: "source",
					BarmanObjectStore: &cloudnativepgv1.BarmanObjectStoreConfiguration{
						BarmanCredentials: barmanapi.BarmanCredentials{
							AWS: &barmanapi.S3Credentials{
								AccessKeyIDReference: &api.SecretKeySelector{
									LocalObjectReference: api.LocalObjectReference{
										Name: minio,
									},
									Key: "ACCESS_KEY_ID",
								},
								SecretAccessKeyReference: &api.SecretKeySelector{
									LocalObjectReference: api.LocalObjectReference{
										Name: minio,
									},
									Key: "ACCESS_SECRET_KEY",
								},
							},
						},
						EndpointURL:     "http://" + net.JoinHostPort(minio, "9000"),
						DestinationPath: "s3://backups/",
					},
				},
			},
			StorageConfiguration: cloudnativepgv1.StorageConfiguration{
				Size: size,
			},
		},
	}

	return cluster
}

func newSrcClusterInTreeAzure(namespace string) *cloudnativepgv1.Cluster {
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
			StorageConfiguration: cloudnativepgv1.StorageConfiguration{
				Size: size,
			},
			Backup: &cloudnativepgv1.BackupConfiguration{
				BarmanObjectStore: &cloudnativepgv1.BarmanObjectStoreConfiguration{
					BarmanCredentials: barmanapi.BarmanCredentials{
						Azure: &barmanapi.AzureCredentials{
							ConnectionString: &api.SecretKeySelector{
								LocalObjectReference: api.LocalObjectReference{
									Name: azurite,
								},
								Key: "AZURE_CONNECTION_STRING",
							},
						},
					},
					DestinationPath: fmt.Sprintf("http://%v:10000/storageaccountname/backups/", azurite),
				},
			},
		},
	}

	return cluster
}

func newDstClusterInTreeAzure(namespace string) *cloudnativepgv1.Cluster {
	cluster := &cloudnativepgv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "postgresql.cnpg.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      restoreClusterName,
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
						"barmanObjectName": objectStoreName,
					},
				},
			},
			ExternalClusters: []cloudnativepgv1.ExternalCluster{
				{
					Name: "source",
					BarmanObjectStore: &cloudnativepgv1.BarmanObjectStoreConfiguration{
						BarmanCredentials: barmanapi.BarmanCredentials{
							Azure: &barmanapi.AzureCredentials{
								ConnectionString: &api.SecretKeySelector{
									LocalObjectReference: api.LocalObjectReference{
										Name: azurite,
									},
									Key: "AZURE_CONNECTION_STRING",
								},
							},
						},
						DestinationPath: fmt.Sprintf("http://%v:10000/storageaccountname/backups/", azurite),
					},
				},
			},
			StorageConfiguration: cloudnativepgv1.StorageConfiguration{
				Size: size,
			},
		},
	}

	return cluster
}

func newSrcClusterInTreeGCS(namespace string) *cloudnativepgv1.Cluster {
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
			StorageConfiguration: cloudnativepgv1.StorageConfiguration{
				Size: size,
			},
			Env: []corev1.EnvVar{
				{
					Name:  "STORAGE_EMULATOR_HOST",
					Value: fmt.Sprintf("http://%v:4443", gcs),
				},
			},
			Backup: &cloudnativepgv1.BackupConfiguration{
				BarmanObjectStore: &cloudnativepgv1.BarmanObjectStoreConfiguration{
					BarmanCredentials: barmanapi.BarmanCredentials{
						Google: &barmanapi.GoogleCredentials{
							ApplicationCredentials: &api.SecretKeySelector{
								LocalObjectReference: api.LocalObjectReference{
									Name: gcs,
								},
								Key: "application_credentials",
							},
						},
					},
					DestinationPath: "gs://backups/",
				},
			},
		},
	}

	return cluster
}

func newDstClusterInTreeGCS(namespace string) *cloudnativepgv1.Cluster {
	cluster := &cloudnativepgv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "postgresql.cnpg.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      restoreClusterName,
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
						"barmanObjectName": objectStoreName,
					},
				},
			},
			Env: []corev1.EnvVar{
				{
					Name:  "STORAGE_EMULATOR_HOST",
					Value: fmt.Sprintf("http://%v:4443", gcs),
				},
			},
			ExternalClusters: []cloudnativepgv1.ExternalCluster{
				{
					Name: "source",
					BarmanObjectStore: &cloudnativepgv1.BarmanObjectStoreConfiguration{
						BarmanCredentials: barmanapi.BarmanCredentials{
							Google: &barmanapi.GoogleCredentials{
								ApplicationCredentials: &api.SecretKeySelector{
									LocalObjectReference: api.LocalObjectReference{
										Name: gcs,
									},
									Key: "application_credentials",
								},
							},
						},
						DestinationPath: "gs://backups/",
					},
				},
			},
			StorageConfiguration: cloudnativepgv1.StorageConfiguration{
				Size: size,
			},
		},
	}

	return cluster
}
