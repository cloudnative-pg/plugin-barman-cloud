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

package objectstore

import (
	"net"

	barmanapi "github.com/cloudnative-pg/barman-cloud/pkg/api"
	"github.com/cloudnative-pg/machinery/pkg/api"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	pluginBarmanCloudV1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
)

// NewMinioObjectStoreResources creates the resources required to create a Minio object store.
func NewMinioObjectStoreResources(namespace, name string) *Resources {
	return &Resources{
		Deployment: newMinioDeployment(namespace, name),
		Service:    newMinioService(namespace, name),
		PVC:        newMinioPVC(namespace, name),
		Secret:     newMinioSecret(namespace, name),
	}
}

func newMinioDeployment(namespace, name string) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: name,
							// TODO: renovate the image
							Image: "minio/minio:latest",
							Args:  []string{"server", "/data"},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 9000,
									Name:          name,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "MINIO_ACCESS_KEY",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: name,
											},
											Key: "ACCESS_KEY_ID",
										},
									},
								},
								{
									Name: "MINIO_SECRET_KEY",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: name,
											},
											Key: "ACCESS_SECRET_KEY",
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/data",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: name,
								},
							},
						},
					},
				},
			},
		},
	}
}

func newMinioService(namespace, name string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": name,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       9000,
					TargetPort: intstr.FromInt32(9000),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

func newMinioSecret(namespace, name string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"ACCESS_KEY_ID":     []byte("minio"),
			"ACCESS_SECRET_KEY": []byte("minio123"),
		},
	}
}

func newMinioPVC(namespace, name string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(DefaultSize),
				},
			},
		},
	}
}

// NewMinioObjectStore creates a new Minio object store.
func NewMinioObjectStore(namespace, name, minioOSName string) *pluginBarmanCloudV1.ObjectStore {
	return &pluginBarmanCloudV1.ObjectStore{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ObjectStore",
			APIVersion: "barmancloud.cnpg.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: pluginBarmanCloudV1.ObjectStoreSpec{
			Configuration: barmanapi.BarmanObjectStoreConfiguration{
				BarmanCredentials: barmanapi.BarmanCredentials{
					AWS: &barmanapi.S3Credentials{
						AccessKeyIDReference: &api.SecretKeySelector{
							LocalObjectReference: api.LocalObjectReference{
								Name: minioOSName,
							},
							Key: "ACCESS_KEY_ID",
						},
						SecretAccessKeyReference: &api.SecretKeySelector{
							LocalObjectReference: api.LocalObjectReference{
								Name: minioOSName,
							},
							Key: "ACCESS_SECRET_KEY",
						},
					},
				},
				EndpointURL:     "http://" + net.JoinHostPort(minioOSName, "9000"),
				DestinationPath: "s3://backups/",
			},
		},
	}
}
