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
	"time"

	v1 "github.com/cloudnative-pg/api/pkg/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	internalClient "github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/client"
	cluster2 "github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/cluster"
	"github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/command"
	nmsp "github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/namespace"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Backup and restore", func() {
	var namespace *corev1.Namespace
	var cl client.Client
	BeforeEach(func(ctx SpecContext) {
		var err error
		cl, _, err = internalClient.NewClient()
		Expect(err).NotTo(HaveOccurred())
		namespace, err = nmsp.CreateUniqueNamespace(ctx, cl, "backup-restore")
		Expect(err).NotTo(HaveOccurred())
	})
	AfterEach(func(ctx SpecContext) {
		Expect(cl.Delete(ctx, namespace)).To(Succeed())
	})

	DescribeTable("should backup and restore a cluster",
		func(
			ctx SpecContext,
			factory testCaseFactory,
		) {
			testResources := factory.createBackupRestoreTestResources(namespace.Name)

			By("starting the ObjectStore deployment")
			Expect(testResources.ObjectStoreResources.Create(ctx, cl)).To(Succeed())

			By("creating the ObjectStore")
			Expect(cl.Create(ctx, testResources.ObjectStore)).To(Succeed())

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
			}).WithTimeout(15 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())

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
				g.Expect(backup.Status.Phase).To(BeEquivalentTo(v1.BackupPhaseCompleted))
			}).Within(2 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())

			By("Adding data after the backup")
			_, _, err = command.ExecuteInContainer(ctx,
				*clientSet,
				cfg,
				command.ContainerLocator{
					NamespaceName: src.Namespace,
					PodName:       fmt.Sprintf("%v-1", src.Name),
					ContainerName: "postgres",
				},
				nil,
				[]string{
					"psql", "-tAc",
					"SELECT pg_switch_wal()" +
						"; INSERT INTO test VALUES (2)",
				})
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
				[]string{
					"psql", "-tAc",
					"SELECT pg_switch_wal()",
				})
			Expect(err).NotTo(HaveOccurred())

			By("Restoring the backup")
			dst := testResources.DstCluster
			Expect(cl.Create(ctx, dst)).To(Succeed())

			By("Having the Cluster ready")
			Eventually(func(g Gomega) {
				g.Expect(cl.Get(ctx,
					types.NamespacedName{Name: dst.Name, Namespace: dst.Namespace},
					dst)).To(Succeed())
				g.Expect(cluster2.IsReady(*dst)).To(BeTrue())
			}).WithTimeout(15 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())

			By("Verifying the data exists in the restored instance")
			output, _, err := command.ExecuteInContainer(ctx,
				*clientSet,
				cfg,
				command.ContainerLocator{
					NamespaceName: dst.Namespace,
					PodName:       fmt.Sprintf("%v-1", dst.Name),
					ContainerName: "postgres",
				},
				nil,
				[]string{"psql", "-tAc", "SELECT count(*) FROM test;"})
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(BeEquivalentTo("2\n"))

			By("taking a backup from the restored cluster")
			backup = testResources.DstBackup
			Expect(cl.Create(ctx, backup)).To(Succeed())

			By("Waiting for the backup to complete")
			Eventually(func(g Gomega) {
				g.Expect(cl.Get(ctx, types.NamespacedName{Name: backup.Name, Namespace: backup.Namespace},
					backup)).To(Succeed())
				g.Expect(backup.Status.Phase).To(BeEquivalentTo(v1.BackupPhaseCompleted))
			}).Within(2 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())
		},
		Entry(
			"using the plugin for backup and restore on S3",
			&s3BackupPluginBackupPluginRestore{},
		),
		Entry(
			"using the plugin for backup and in-tree for restore on S3",
			&s3BackupPluginBackupInTreeRestore{},
		),
		Entry(
			"using in-tree for backup and the plugin for restore on S3",
			&s3BackupPluginInTreeBackupPluginRestore{},
		),
		Entry(
			"using the plugin for backup and restore on Azure",
			&azureBackupPluginBackupPluginRestore{},
		),
		Entry(
			"using the plugin for backup and in-tree for restore on Azure",
			&azureBackupPluginBackupInTreeRestore{},
		),
		Entry(
			"using in-tree for backup and the plugin for restore on Azure",
			&azureBackupPluginInTreeBackupPluginRestore{},
		),
		Entry("using the plugin for backup and restore on GCS",
			&gcsBackupPluginBackupPluginRestore{},
		),
		Entry("using the plugin for backup and in-tree for restore on GCS",
			&gcsBackupPluginBackupInTreeRestore{},
		),
		Entry(
			"using in-tree for backup and the plugin for restore on GCS",
			&gcsBackupPluginInTreeBackupPluginRestore{},
		),
	)
})
