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

			response, err := reconcileJob(ctx, cluster, request, nil, nil)
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

			response, err := reconcileJob(ctx, cluster, request, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(BeNil())
		})

		It("returns an error for invalid job definition", func(ctx SpecContext) {
			request := &lifecycle.OperatorLifecycleRequest{
				ObjectDefinition: []byte("invalid-json"),
			}

			response, err := reconcileJob(ctx, cluster, request, nil, nil)
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

				response, err := reconcileJob(ctx, cluster, request, nil, nil)
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

			response, err := reconcilePod(ctx, cluster, request, pluginConfiguration, nil, nil)
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

			response, err := reconcilePod(ctx, cluster, request, pluginConfiguration, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
		})
	})
})
