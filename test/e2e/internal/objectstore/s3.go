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

// The S3-compatible object store used by the e2e tests is RustFS. It runs as
// a non-root user and writes its data and logs to these subdirectories, which
// an init container creates and makes writable.
const (
	s3DataDir = "/data/rustfs"
	s3LogDir  = "/logs/rustfs"
)

// NewS3ObjectStoreResources creates the resources required to run an
// S3-compatible object store.
func NewS3ObjectStoreResources(namespace, name string) *Resources {
	return &Resources{
		Deployment: newS3Deployment(namespace, name),
		Service:    newS3Service(namespace, name),
		PVC:        newS3PVC(namespace, name),
		Secret:     newS3Secret(namespace, name),
	}
}

func newS3Deployment(namespace, name string) *appsv1.Deployment {
	seccompProfile := &corev1.SeccompProfile{
		Type: corev1.SeccompProfileTypeRuntimeDefault,
	}

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
					// RustFS runs as a non-root user but the PVC is root-owned, and
					// a non-root init container cannot chown it on OpenShift.
					// Instead the init creates a subdirectory it owns and makes it
					// world-writable, which works as root (kind, cloud) or as the
					// SCC-assigned UID (OpenShift).
					InitContainers: []corev1.Container{
						{
							Name: "init-permissions",
							// renovate: datasource=docker depName=busybox versioning=docker
							// Version: 1.38.0
							Image: "docker.io/library/busybox@sha256:dc2d74b28e4cf8984fa52af1f39bc7c3d9c73760b41a74d629f5d11b1ab28616",
							Command: []string{
								"sh", "-c",
								fmt.Sprintf("mkdir -p %[1]s %[2]s && chmod 0777 %[1]s %[2]s", s3DataDir, s3LogDir),
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "data", MountPath: "/data"},
								{Name: "logs", MountPath: "/logs"},
							},
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: ptr.To(false),
								SeccompProfile:           seccompProfile,
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name: name,
							// The glibc build is used because the musl default is
							// noticeably slower (rustfs/rustfs#1662).
							// renovate: datasource=docker depName=rustfs/rustfs versioning=docker
							// Version: 1.0.0-glibc
							Image:   "docker.io/rustfs/rustfs@sha256:bffcab0c9d647aab0055d1c69d340b202d0909966b385932d4ead1aeb7602858",
							Command: []string{"/usr/bin/rustfs"},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 9000,
									Name:          name,
								},
							},
							Env: []corev1.EnvVar{
								{Name: "RUSTFS_ADDRESS", Value: ":9000"},
								{Name: "RUSTFS_VOLUMES", Value: s3DataDir},
								{Name: "RUSTFS_REGION", Value: "us-east-1"},
								{Name: "RUSTFS_CONSOLE_ENABLE", Value: "false"},
								{Name: "RUSTFS_OBS_LOG_DIRECTORY", Value: s3LogDir},
								{
									Name: "RUSTFS_ACCESS_KEY",
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
									Name: "RUSTFS_SECRET_KEY",
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
								{Name: "data", MountPath: "/data"},
								{Name: "logs", MountPath: "/logs"},
							},
							LivenessProbe: newHealthProbe(30, 10),
							// RustFS is up within a few seconds; poll early so
							// each spec does not wait the liveness grace period.
							ReadinessProbe: newHealthProbe(5, 5),
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: ptr.To(false),
								SeccompProfile:           seccompProfile,
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
						{
							Name: "logs",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					// No fsGroup/runAsUser: let OpenShift's restricted SCC assign
					// the UID; elsewhere the server runs as its image default user.
					SecurityContext: &corev1.PodSecurityContext{
						SeccompProfile: seccompProfile,
					},
				},
			},
		},
	}
}

// newHealthProbe returns a probe hitting the RustFS health endpoint.
func newHealthProbe(initialDelaySeconds, periodSeconds int32) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/health",
				Port: intstr.FromInt32(9000),
			},
		},
		InitialDelaySeconds: initialDelaySeconds,
		PeriodSeconds:       periodSeconds,
	}
}

func newS3Service(namespace, name string) *corev1.Service {
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

func newS3Secret(namespace, name string) *corev1.Secret {
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
			"ACCESS_KEY_ID":     []byte("s3accesskey"),
			"ACCESS_SECRET_KEY": []byte("s3secretkey123"),
		},
	}
}

func newS3PVC(namespace, name string) *corev1.PersistentVolumeClaim {
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

// NewS3ObjectStore creates a new ObjectStore pointing at the S3-compatible
// object store created by NewS3ObjectStoreResources with the given name.
func NewS3ObjectStore(namespace, name, s3Name string) *pluginBarmanCloudV1.ObjectStore {
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
								Name: s3Name,
							},
							Key: "ACCESS_KEY_ID",
						},
						SecretAccessKeyReference: &api.SecretKeySelector{
							LocalObjectReference: api.LocalObjectReference{
								Name: s3Name,
							},
							Key: "ACCESS_SECRET_KEY",
						},
					},
				},
				EndpointURL:     "http://" + net.JoinHostPort(s3Name, "9000"),
				DestinationPath: "s3://backups/",
			},
		},
	}
}
