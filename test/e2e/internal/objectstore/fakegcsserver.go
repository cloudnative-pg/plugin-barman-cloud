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

// NewGCSObjectStoreResources creates the resources required to create a GCS object store.
func NewGCSObjectStoreResources(namespace, name string) *Resources {
	return &Resources{
		Deployment: newGCSDeployment(namespace, name),
		Service:    newGCSService(namespace, name),
		Secret:     newGCSSecret(namespace, name),
		PVC:        newGCSPVC(namespace, name),
	}
}

func newGCSDeployment(namespace, name string) *appsv1.Deployment {
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
							Name:  name,
							Image: "fsouza/fake-gcs-server:latest",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 4443,
								},
							},
							Command: []string{"fake-gcs-server"},
							Args: []string{
								"-scheme",
								"http",
								"-port",
								"4443",
								"-external-url",
								fmt.Sprintf("http://%v:4443", name),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "storage",
									MountPath: "/storage",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "storage",
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

func newGCSService(namespace, name string) *corev1.Service {
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
					Port:       4443,
					TargetPort: intstr.FromInt32(4443),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

func newGCSSecret(namespace, name string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		StringData: map[string]string{
			"application_credentials": "",
		},
	}
}

func newGCSPVC(namespace, name string) *corev1.PersistentVolumeClaim {
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

// NewGCSObjectStore creates a new GCS object store.
func NewGCSObjectStore(namespace, name, gcsOSName string) *pluginBarmanCloudV1.ObjectStore {
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
					Google: &barmanapi.GoogleCredentials{
						ApplicationCredentials: &api.SecretKeySelector{
							LocalObjectReference: api.LocalObjectReference{
								Name: gcsOSName,
							},
							Key: "application_credentials",
						},
					},
				},
				DestinationPath: "gs://backups/",
			},
			InstanceSidecarConfiguration: pluginBarmanCloudV1.InstanceSidecarConfiguration{
				Env: []corev1.EnvVar{
					{
						Name:  "STORAGE_EMULATOR_HOST",
						Value: fmt.Sprintf("http://%v:4443", gcsOSName),
					},
				},
			},
		},
	}
}
