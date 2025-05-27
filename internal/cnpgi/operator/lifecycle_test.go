package operator

import (
	"encoding/json"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cloudnative-pg/pkg/utils"
	"github.com/cloudnative-pg/cnpg-i/pkg/lifecycle"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cloudnative-pg/plugin-barman-cloud/internal/cnpgi/operator/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LifecycleImplementation", func() {
	var (
		lifecycleImpl       LifecycleImplementation
		pluginConfiguration *config.PluginConfiguration
		cluster             *cnpgv1.Cluster
		jobTypeMeta         = metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		}
		podTypeMeta = metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		}
	)

	BeforeEach(func() {
		pluginConfiguration = &config.PluginConfiguration{
			BarmanObjectName: "minio-store-dest",
		}
		cluster = &cnpgv1.Cluster{
			Spec: cnpgv1.ClusterSpec{
				Bootstrap: &cnpgv1.BootstrapConfiguration{
					Recovery: &cnpgv1.BootstrapRecovery{
						Source: "origin-server",
					},
				},
				ExternalClusters: []cnpgv1.ExternalCluster{
					{
						Name: "origin-server",
						PluginConfiguration: &cnpgv1.PluginConfiguration{
							Name: "barman-cloud.cloudnative-pg.io",
							Parameters: map[string]string{
								"barmanObjectName": "minio-store-source",
							},
						},
					},
				},
				Plugins: []cnpgv1.PluginConfiguration{
					{
						Name: "barman-cloud.cloudnative-pg.io",
						Parameters: map[string]string{
							"barmanObjectName": "minio-store-dest",
						},
					},
				},
			},
		}
	})

	Describe("GetCapabilities", func() {
		It("returns the correct capabilities", func(ctx SpecContext) {
			response, err := lifecycleImpl.GetCapabilities(ctx, &lifecycle.OperatorLifecycleCapabilitiesRequest{})
			Expect(err).NotTo(HaveOccurred())
			Expect(response).NotTo(BeNil())
			Expect(response.LifecycleCapabilities).To(HaveLen(2))
		})
	})

	Describe("LifecycleHook", func() {
		It("returns an error if object definition is invalid", func(ctx SpecContext) {
			request := &lifecycle.OperatorLifecycleRequest{
				ObjectDefinition: []byte("invalid-json"),
			}
			response, err := lifecycleImpl.LifecycleHook(ctx, request)
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
		})
	})

	Describe("reconcileJob", func() {
		It("returns a patch for a valid recovery job", func(ctx SpecContext) {
			job := &batchv1.Job{
				TypeMeta: jobTypeMeta,
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-job",
					Labels: map[string]string{},
				},
				Spec: batchv1.JobSpec{Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							utils.JobRoleLabelName: "full-recovery",
						},
					},
					Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "full-recovery"}}},
				}},
			}
			jobJSON, _ := json.Marshal(job)
			request := &lifecycle.OperatorLifecycleRequest{
				ObjectDefinition: jobJSON,
			}

			response, err := reconcileJob(ctx, cluster, request, sidecarConfiguration{})
			Expect(err).NotTo(HaveOccurred())
			Expect(response).NotTo(BeNil())
			Expect(response.JsonPatch).NotTo(BeEmpty())
		})

		It("skips non-recovery jobs", func(ctx SpecContext) {
			job := &batchv1.Job{
				TypeMeta: jobTypeMeta,
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-job",
					Labels: map[string]string{
						"job-role": "non-recovery",
					},
				},
			}
			jobJSON, _ := json.Marshal(job)
			request := &lifecycle.OperatorLifecycleRequest{
				ObjectDefinition: jobJSON,
			}

			response, err := reconcileJob(ctx, cluster, request, sidecarConfiguration{})
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(BeNil())
		})

		It("returns an error for invalid job definition", func(ctx SpecContext) {
			request := &lifecycle.OperatorLifecycleRequest{
				ObjectDefinition: []byte("invalid-json"),
			}

			response, err := reconcileJob(ctx, cluster, request, sidecarConfiguration{})
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
		})

		It("should not error out if backup object name is not set and the job isn't full recovery",
			func(ctx SpecContext) {
				job := &batchv1.Job{
					TypeMeta: jobTypeMeta,
					ObjectMeta: metav1.ObjectMeta{
						Name:   "test-job",
						Labels: map[string]string{},
					},
					Spec: batchv1.JobSpec{Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								utils.JobRoleLabelName: "non-recovery",
							},
						},
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "non-recovery"}}},
					}},
				}
				jobJSON, _ := json.Marshal(job)
				request := &lifecycle.OperatorLifecycleRequest{
					ObjectDefinition: jobJSON,
				}

				response, err := reconcileJob(ctx, cluster, request, sidecarConfiguration{})
				Expect(err).NotTo(HaveOccurred())
				Expect(response).To(BeNil())
			})
	})

	Describe("reconcilePod", func() {
		It("returns a patch for a valid pod", func(ctx SpecContext) {
			pod := &corev1.Pod{
				TypeMeta: podTypeMeta,
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "postgres"}}},
			}
			podJSON, _ := json.Marshal(pod)
			request := &lifecycle.OperatorLifecycleRequest{
				ObjectDefinition: podJSON,
			}

			response, err := reconcilePod(ctx, cluster, request, pluginConfiguration, sidecarConfiguration{})
			Expect(err).NotTo(HaveOccurred())
			Expect(response).NotTo(BeNil())
			Expect(response.JsonPatch).NotTo(BeEmpty())
			var patch []map[string]interface{}
			err = json.Unmarshal(response.JsonPatch, &patch)
			Expect(err).NotTo(HaveOccurred())
			Expect(patch).To(ContainElement(HaveKeyWithValue("op", "add")))
			Expect(patch).To(ContainElement(HaveKeyWithValue("path", "/spec/initContainers")))
			Expect(patch).To(ContainElement(
				HaveKey("value")))
		})

		It("returns an error for invalid pod definition", func(ctx SpecContext) {
			request := &lifecycle.OperatorLifecycleRequest{
				ObjectDefinition: []byte("invalid-json"),
			}

			response, err := reconcilePod(ctx, cluster, request, pluginConfiguration, sidecarConfiguration{})
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
		})
	})
})

var _ = Describe("Volume utilities", func() {
	Describe("ensureVolume", func() {
		It("adds a new volume if not present", func() {
			volumes := []corev1.Volume{{Name: "vol1"}}
			newVol := corev1.Volume{Name: "vol2"}
			result := ensureVolume(volumes, newVol)
			Expect(result).To(HaveLen(2))
			Expect(result[1]).To(Equal(newVol))
		})

		It("updates an existing volume", func() {
			volumes := []corev1.Volume{{Name: "vol1"}}
			updatedVol := corev1.Volume{Name: "vol1", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}
			result := ensureVolume(volumes, updatedVol)
			Expect(result).To(HaveLen(1))
			Expect(result[0]).To(Equal(updatedVol))
		})
	})

	Describe("removeVolume", func() {
		It("removes the specified volume", func() {
			volumes := []corev1.Volume{
				{Name: "vol1"},
				{Name: "vol2"},
				{Name: "vol3"},
			}
			result := removeVolume(volumes, "vol2")
			Expect(result).To(HaveLen(2))
			for _, v := range result {
				Expect(v.Name).NotTo(Equal("vol2"))
			}
		})

		It("returns the same list if the volume is not present", func() {
			volumes := []corev1.Volume{{Name: "vol1"}, {Name: "vol2"}}
			result := removeVolume(volumes, "vol3")
			Expect(result).To(Equal(volumes))
		})
	})
})

var _ = Describe("removeVolumeMount", func() {
	It("removes the specified volume mount", func() {
		mounts := []corev1.VolumeMount{
			{Name: "mount1"},
			{Name: "mount2"},
			{Name: "mount3"},
		}
		result := removeVolumeMount(mounts, "mount2")
		Expect(result).To(HaveLen(2))
		for _, m := range result {
			Expect(m.Name).NotTo(Equal("mount2"))
		}
	})

	It("returns the same list if the volume mount is not present", func() {
		mounts := []corev1.VolumeMount{{Name: "mount1"}, {Name: "mount2"}}
		result := removeVolumeMount(mounts, "mount3")
		Expect(result).To(HaveLen(2))
		Expect(result[0].Name).To(Equal("mount1"))
		Expect(result[1].Name).To(Equal("mount2"))
	})

	It("handles empty input slice", func() {
		var mounts []corev1.VolumeMount
		result := removeVolumeMount(mounts, "mount1")
		Expect(result).To(BeEmpty())
	})
})

var _ = Describe("ensureVolumeMount", func() {
	It("adds a new volume mount to an empty list", func() {
		var mounts []corev1.VolumeMount
		newMount := corev1.VolumeMount{Name: "mount1", MountPath: "/path1"}
		result := ensureVolumeMount(mounts, newMount)
		Expect(result).To(HaveLen(1))
		Expect(result[0]).To(Equal(newMount))
	})

	It("adds a new volume mount to a non-empty list", func() {
		mounts := []corev1.VolumeMount{{Name: "mount1", MountPath: "/path1"}}
		newMount := corev1.VolumeMount{Name: "mount2", MountPath: "/path2"}
		result := ensureVolumeMount(mounts, newMount)
		Expect(result).To(HaveLen(2))
		Expect(result[0].Name).To(Equal("mount1"))
		Expect(result[1].Name).To(Equal("mount2"))
	})

	It("updates an existing volume mount", func() {
		mounts := []corev1.VolumeMount{{Name: "mount1", MountPath: "/path1"}}
		updatedMount := corev1.VolumeMount{Name: "mount1", MountPath: "/new-path"}
		result := ensureVolumeMount(mounts, updatedMount)
		Expect(result).To(HaveLen(1))
		Expect(result[0].MountPath).To(Equal("/new-path"))
	})

	It("adds multiple new volume mounts", func() {
		mounts := []corev1.VolumeMount{{Name: "mount1", MountPath: "/path1"}}
		newMount1 := corev1.VolumeMount{Name: "mount2", MountPath: "/path2"}
		newMount2 := corev1.VolumeMount{Name: "mount3", MountPath: "/path3"}
		result := ensureVolumeMount(mounts, newMount1, newMount2)
		Expect(result).To(HaveLen(3))
		Expect(result[0].Name).To(Equal("mount1"))
		Expect(result[1].Name).To(Equal("mount2"))
		Expect(result[2].Name).To(Equal("mount3"))
	})

	It("handles a mix of new and existing volume mounts", func() {
		mounts := []corev1.VolumeMount{
			{Name: "mount1", MountPath: "/path1"},
			{Name: "mount2", MountPath: "/path2"},
		}
		updatedMount := corev1.VolumeMount{Name: "mount1", MountPath: "/new-path"}
		newMount := corev1.VolumeMount{Name: "mount3", MountPath: "/path3"}
		result := ensureVolumeMount(mounts, updatedMount, newMount)
		Expect(result).To(HaveLen(3))
		Expect(result[0].Name).To(Equal("mount1"))
		Expect(result[0].MountPath).To(Equal("/new-path"))
		Expect(result[1].Name).To(Equal("mount2"))
		Expect(result[2].Name).To(Equal("mount3"))
	})
})
