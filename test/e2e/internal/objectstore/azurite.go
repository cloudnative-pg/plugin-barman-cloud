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
	"fmt"

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

// NewAzuriteObjectStoreResources creates the resources required to create an Azurite object store.
func NewAzuriteObjectStoreResources(namespace, name string) *Resources {
	return &Resources{
		Deployment: newAzuriteDeployment(namespace, name),
		Service:    newAzuriteService(namespace, name),
		PVC:        newAzuritePVC(namespace, name),
		Secret:     newAzuriteSecret(namespace, name),
	}
}

func newAzuriteDeployment(namespace, name string) *appsv1.Deployment {
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
							// renovate: datasource=docker depName=mcr.microsoft.com/azure-storage/azurite versioning=docker
							// Version: 3.35.0
							Image: "mcr.microsoft.com/azure-storage/azurite@sha256:647c63a91102a9d8e8000aab803436e1fc85fbb285e7ce830a82ee5d6661cf37",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 10000,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "AZURITE_ACCOUNTS",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: name,
											},
											Key: "AZURITE_ACCOUNTS",
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

func newAzuriteService(namespace, name string) *corev1.Service {
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
					Port:       10000,
					TargetPort: intstr.FromInt32(10000),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

func newAzuriteSecret(namespace, name string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		StringData: map[string]string{
			"AZURITE_ACCOUNTS": "storageaccountname:c3RvcmFnZWFjY291bnRrZXk=",
			"AZURE_CONNECTION_STRING": "DefaultEndpointsProtocol=http;AccountName=storageaccountname;" +
				"AccountKey=c3RvcmFnZWFjY291bnRrZXk=;" +
				fmt.Sprintf("BlobEndpoint=http://%v:10000/storageaccountname;", name),
		},
	}
}

func newAzuritePVC(namespace, name string) *corev1.PersistentVolumeClaim {
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

// NewAzuriteObjectStore creates a new ObjectStore object for Azurite.
func NewAzuriteObjectStore(namespace, name, azuriteOSName string) *pluginBarmanCloudV1.ObjectStore {
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
					Azure: &barmanapi.AzureCredentials{
						ConnectionString: &api.SecretKeySelector{
							LocalObjectReference: api.LocalObjectReference{
								Name: azuriteOSName,
							},
							Key: "AZURE_CONNECTION_STRING",
						},
					},
				},
				DestinationPath: fmt.Sprintf("http://%v:10000/storageaccountname/backups/", azuriteOSName),
			},
		},
	}
}
