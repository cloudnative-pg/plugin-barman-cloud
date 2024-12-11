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

package replicacluster

import (
	"fmt"
	"strings"
	"time"

	cloudnativepgv1 "github.com/cloudnative-pg/api/pkg/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	internalClient "github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/client"
	cluster2 "github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/cluster"
	"github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/command"
	nmsp "github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/namespace"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Replica cluster", func() {
	var namespace *corev1.Namespace
	var cl client.Client
	BeforeEach(func(ctx SpecContext) {
		var err error
		cl, _, err = internalClient.NewClient()
		Expect(err).NotTo(HaveOccurred())
		namespace, err = nmsp.CreateUniqueNamespace(ctx, cl, "replica-cluster")
		Expect(err).NotTo(HaveOccurred())
	})
	AfterEach(func(ctx SpecContext) {
		Expect(cl.Delete(ctx, namespace)).To(Succeed())
	})
	DescribeTable("can switchover to a replica cluster",
		func(
			ctx SpecContext,
			factory testCaseFactory,
		) {
			testResources := factory.createReplicaClusterTestResources(namespace.Name)

			By("starting the ObjectStore deployments")
			Expect(testResources.SrcObjectStoreResources.Create(ctx, cl)).To(Succeed())
			Expect(testResources.ReplicaObjectStoreResources.Create(ctx, cl)).To(Succeed())

			By("creating the ObjectStores")
			Expect(cl.Create(ctx, testResources.SrcObjectStore)).To(Succeed())
			// We do not need to create the replica object store if we are using the same object store for both clusters.
			if testResources.ReplicaObjectStore != nil {
				Expect(cl.Create(ctx, testResources.ReplicaObjectStore)).To(Succeed())
			}

			By("Creating a CloudNativePG cluster")
			src := testResources.SrcCluster
			Expect(cl.Create(ctx, testResources.SrcCluster)).To(Succeed())

			By("Having the Cluster ready")
			Eventually(func(g Gomega) {
				g.Expect(cl.Get(
					ctx,
					types.NamespacedName{
						Name:      src.Name,
						Namespace: src.Namespace,
					},
					src)).To(Succeed())
				g.Expect(cluster2.IsReady(*src)).To(BeTrue())
			}).WithTimeout(10 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())

			By("Adding data to PostgreSQL")
			clientSet, cfg, err := internalClient.NewClientSet()
			Expect(err).NotTo(HaveOccurred())
			_, _, err = command.ExecuteInContainer(ctx,
				*clientSet,
				cfg,
				command.ContainerLocator{
					NamespaceName: src.Namespace,
					PodName:       fmt.Sprintf("%v-1", src.Name),
					ContainerName: "postgres",
				},
				nil,
				[]string{"psql", "-tAc", "CREATE TABLE test (i int); INSERT INTO test VALUES (1);"})
			Expect(err).NotTo(HaveOccurred())

			By("Creating a backup")
			backup := testResources.SrcBackup
			Expect(cl.Create(ctx, backup)).To(Succeed())

			By("Waiting for the backup to complete")
			Eventually(func(g Gomega) {
				g.Expect(cl.Get(ctx, types.NamespacedName{Name: backup.Name, Namespace: backup.Namespace},
					backup)).To(Succeed())
				g.Expect(backup.Status.Phase).To(BeEquivalentTo(cloudnativepgv1.BackupPhaseCompleted))
			}).Within(3 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())

			By("Creating a replica cluster")
			replica := testResources.ReplicaCluster
			Expect(cl.Create(ctx, testResources.ReplicaCluster)).To(Succeed())

			By("Having the replica cluster ready")
			Eventually(func(g Gomega) {
				g.Expect(cl.Get(
					ctx,
					types.NamespacedName{
						Name:      replica.Name,
						Namespace: replica.Namespace,
					},
					replica)).To(Succeed())
				g.Expect(cluster2.IsReady(*replica)).To(BeTrue())
			}).WithTimeout(10 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())

			By("Checking the data in the replica cluster")
			output, _, err := command.ExecuteInContainer(ctx,
				*clientSet,
				cfg,
				command.ContainerLocator{
					NamespaceName: replica.Namespace,
					PodName:       fmt.Sprintf("%v-1", replica.Name),
					ContainerName: "postgres",
				},
				nil,
				[]string{"psql", "-tAc", "SELECT count(*) FROM test;"})
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(BeEquivalentTo("1\n"))

			// We want to check if the WALs archived by the operator outside the standard PostgreSQL archive_command
			// are correctly archived.
			By("Demoting the source to a replica")
			err = cl.Get(ctx, types.NamespacedName{Name: src.Name, Namespace: src.Namespace}, src)
			Expect(err).ToNot(HaveOccurred())
			oldSrc := src.DeepCopy()
			src.Spec.ReplicaCluster.Primary = replicaClusterName
			Expect(cl.Patch(ctx, src, client.MergeFrom(oldSrc))).To(Succeed())

			By("Waiting for all the source pods to be in recovery")
			for i := 1; i <= src.Spec.Instances; i++ {
				Eventually(func() (string, error) {
					stdOut, stdErr, err := command.ExecuteInContainer(
						ctx,
						*clientSet,
						cfg,
						command.ContainerLocator{
							NamespaceName: src.Namespace,
							PodName:       fmt.Sprintf("%v-%v", src.Name, i),
							ContainerName: "postgres",
						},
						ptr.To(5*time.Second),
						[]string{"psql", "-tAc", "SELECT pg_is_in_recovery();"})
					if err != nil {
						GinkgoWriter.Printf("stdout: %v\ntderr: %v\n", stdOut, stdErr)
					}

					return strings.Trim(stdOut, "\n"), err
				}, 300, 10).Should(BeEquivalentTo("t"))
			}

			By("Getting the demotion token")
			err = cl.Get(ctx, types.NamespacedName{Name: src.Name, Namespace: src.Namespace}, src)
			Expect(err).ToNot(HaveOccurred())
			token := src.Status.DemotionToken

			By("Promoting the replica")
			err = cl.Get(ctx, types.NamespacedName{Name: replica.Name, Namespace: replica.Namespace}, replica)
			Expect(err).ToNot(HaveOccurred())
			oldReplica := replica.DeepCopy()
			replica.Spec.ReplicaCluster.PromotionToken = token
			replica.Spec.ReplicaCluster.Primary = replica.Name
			Expect(cl.Patch(ctx, replica, client.MergeFrom(oldReplica))).To(Succeed())

			By("Waiting for the replica to be promoted")
			Eventually(func() (string, error) {
				stdOut, stdErr, err := command.ExecuteInContainer(
					ctx,
					*clientSet,
					cfg,
					command.ContainerLocator{
						NamespaceName: replica.Namespace,
						PodName:       fmt.Sprintf("%v-1", replica.Name),
						ContainerName: "postgres",
					},
					ptr.To(5*time.Second),
					[]string{"psql", "-tAc", "SELECT pg_is_in_recovery();"})
				if err != nil {
					GinkgoWriter.Printf("stdout: %v\ntderr: %v\n", stdOut, stdErr)
				}

				return strings.Trim(stdOut, "\n"), err
			}, 300, 10).Should(BeEquivalentTo("f"))

			By("Adding new data to PostgreSQL")
			clientSet, cfg, err = internalClient.NewClientSet()
			Expect(err).NotTo(HaveOccurred())
			_, _, err = command.ExecuteInContainer(ctx,
				*clientSet,
				cfg,
				command.ContainerLocator{
					NamespaceName: replica.Namespace,
					PodName:       fmt.Sprintf("%v-1", replica.Name),
					ContainerName: "postgres",
				},
				nil,
				[]string{"psql", "-tAc", "INSERT INTO test VALUES (2);"})
			Expect(err).NotTo(HaveOccurred())
			_, _, err = command.ExecuteInContainer(ctx,
				*clientSet,
				cfg,
				command.ContainerLocator{
					NamespaceName: replica.Namespace,
					PodName:       fmt.Sprintf("%v-1", replica.Name),
					ContainerName: "postgres",
				},
				nil,
				[]string{"psql", "-tAc", "SELECT pg_switch_wal();"})
			Expect(err).NotTo(HaveOccurred())

			By("Creating a backup in the replica cluster")
			replicaBackup := testResources.ReplicaBackup
			Expect(cl.Create(ctx, replicaBackup)).To(Succeed())

			By("Waiting for the backup to complete")
			Eventually(func(g Gomega) {
				g.Expect(cl.Get(ctx, types.NamespacedName{Name: replicaBackup.Name, Namespace: replicaBackup.Namespace},
					replicaBackup)).To(Succeed())
				g.Expect(replicaBackup.Status.Phase).To(BeEquivalentTo(cloudnativepgv1.BackupPhaseCompleted))
			}).Within(3 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())

			By("Checking the data in the former primary cluster")
			Eventually(func(g Gomega) {
				output, _, err = command.ExecuteInContainer(ctx,
					*clientSet,
					cfg,
					command.ContainerLocator{
						NamespaceName: src.Namespace,
						PodName:       fmt.Sprintf("%v-1", src.Name),
						ContainerName: "postgres",
					},
					nil,
					[]string{"psql", "-tAc", "SELECT count(*) FROM test;"})

				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(BeEquivalentTo("2\n"))
			}).Within(2 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())
		},
		Entry(
			"with MinIO",
			s3ReplicaClusterFactory{},
		),
		Entry(
			"with Azurite",
			azuriteReplicaClusterFactory{},
		),
		Entry(
			"with fake-gcs-server",
			gcsReplicaClusterFactory{},
		),
	)
})
