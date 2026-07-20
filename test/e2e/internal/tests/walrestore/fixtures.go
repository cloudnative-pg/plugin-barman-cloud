/*
Copyright © contributors to CloudNativePG, established as
CloudNativePG a Series of LF Projects, LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

SPDX-License-Identifier: Apache-2.0
*/

package walrestore

import (
	cloudnativepgv1 "github.com/cloudnative-pg/api/pkg/api/v1"
	barmanapi "github.com/cloudnative-pg/barman-cloud/pkg/api"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	pluginBarmanCloudV1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
	"github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/objectstore"
)

const (
	minioName       = "minio"
	objectStoreName = "source"
	clusterName     = "source"
	s3ClientName    = "s3-client"
	storageSize     = "1Gi"
	// walMaxParallel is the prefetch parallelism under test: for a regular WAL
	// request the plugin restores the requested segment and prefetches the next
	// ones, up to this many segments in total.
	walMaxParallel = 3
)

// newObjectStoreResources returns the minio server Deployment/Service/Secret/PVC.
func newObjectStoreResources(namespace string) *objectstore.Resources {
	return objectstore.NewMinioObjectStoreResources(namespace, minioName)
}

// newObjectStore returns a minio-backed ObjectStore configured with the WAL
// prefetch parallelism (maxParallel) under test. Archiving with gzip makes the
// archived segments carry the ".gz" suffix that forged segments are copied from.
func newObjectStore(namespace string) *pluginBarmanCloudV1.ObjectStore {
	store := objectstore.NewMinioObjectStore(namespace, objectStoreName, minioName)
	store.Spec.Configuration.Wal = &barmanapi.WalBackupConfiguration{
		MaxParallel: walMaxParallel,
		Compression: barmanapi.CompressionTypeGzip,
	}
	return store
}

// newCluster returns a 2-instance cluster that uses the plugin as its WAL
// archiver, so the standby drives WAL restore (and its prefetch/spool/
// end-of-wal-stream state machine) through the plugin.
func newCluster(namespace string) *cloudnativepgv1.Cluster {
	return &cloudnativepgv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "postgresql.cnpg.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
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
					IsWALArchiver: ptr.To(true),
				},
			},
			PostgresConfiguration: cloudnativepgv1.PostgresConfiguration{
				Parameters: map[string]string{
					"log_min_messages": "DEBUG4",
				},
			},
			StorageConfiguration: cloudnativepgv1.StorageConfiguration{
				Size: storageSize,
			},
		},
	}
}

// newS3ClientDeployment returns a deployment running the AWS CLI configured to
// talk to the in-namespace minio service. The test execs `aws s3` commands in
// it to forge WAL segments on the object store and to check their presence.
func newS3ClientDeployment(namespace string) *appsv1.Deployment {
	labels := map[string]string{"app": s3ClientName}
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      s3ClientName,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(int32(1)),
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: s3ClientName,
							// renovate: datasource=docker depName=amazon/aws-cli versioning=docker
							// Version: 2.36.3
							Image:   "docker.io/amazon/aws-cli@sha256:bdd02067a00c354684086071b475955c54caa7bd88b851aac99a51326fe19652",
							Command: []string{"sleep", "infinity"},
							Env: []corev1.EnvVar{
								{
									Name:  "AWS_ENDPOINT_URL",
									Value: "http://" + minioName + ":9000",
								},
								{
									Name: "AWS_ACCESS_KEY_ID",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: minioName},
											Key:                  "ACCESS_KEY_ID",
										},
									},
								},
								{
									Name: "AWS_SECRET_ACCESS_KEY",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: minioName},
											Key:                  "ACCESS_SECRET_KEY",
										},
									},
								},
								{
									Name:  "AWS_DEFAULT_REGION",
									Value: "us-east-1",
								},
								// The CRC-based default checksums introduced in AWS
								// CLI 2.23 are not supported by every S3-compatible
								// object store, minio included.
								{
									Name:  "AWS_REQUEST_CHECKSUM_CALCULATION",
									Value: "when_required",
								},
								{
									Name:  "AWS_RESPONSE_CHECKSUM_VALIDATION",
									Value: "when_required",
								},
							},
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: ptr.To(false),
								SeccompProfile: &corev1.SeccompProfile{
									Type: corev1.SeccompProfileTypeRuntimeDefault,
								},
							},
						},
					},
				},
			},
		},
	}
}
