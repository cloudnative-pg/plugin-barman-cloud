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
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	internalClient "github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/client"
	internalCluster "github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/cluster"
	"github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/command"
	"github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/deployment"
	nmsp "github.com/cloudnative-pg/plugin-barman-cloud/test/e2e/internal/namespace"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	// spoolDirectory is where the plugin sidecar prefetches WAL segments and
	// records the end-of-wal-stream sentinel. It must match the SPOOL_DIRECTORY
	// the operator injects on the sidecar (internal/cnpgi/operator/lifecycle.go),
	// which lives on the CloudNativePG scratch-data volume shared with the
	// postgres container, so the path is visible from the postgres container too.
	spoolDirectory = "/controller/wal-restore-spool"
	// pgWalPath is PgDataPath + "/pg_wal" in the CloudNativePG operand image.
	pgWalPath = "/var/lib/postgresql/data/pgdata/pg_wal"
	// endOfWALStreamFlag is the sentinel file the barman-cloud restorer writes in
	// the spool to record that the archive ran out of segments.
	endOfWALStreamFlag = "end-of-wal-stream"
	// managerExecutable is the CloudNativePG instance manager. Its wal-restore
	// subcommand delegates to the plugin when the cluster uses it as WAL archiver.
	managerExecutable = "/controller/manager"
	// postgresContainer is the container we exec into on instance pods.
	postgresContainer = "postgres"
	// walLogDir is the wals sub-directory (timeline + log id) the forged segments
	// live under; a freshly bootstrapped, idle cluster stays within it.
	walLogDir = "0000000100000000"
	// bucket is the destination bucket of the minio ObjectStore.
	bucket = "backups"
)

// walFile returns the name of the n-th forged WAL segment (segment 0xF0+n on
// timeline 1, log 0). The high segment number keeps it out of the range an idle
// PostgreSQL would archive on its own. Hex formatting keeps the name a valid
// 24-character segment for any small n.
func walFile(n int) string {
	return fmt.Sprintf("0000000100000000%08X", 0xF0+n)
}

// walObjectURI returns the s3 URI of a wals object (by file name) in the store.
func walObjectURI(name string) string {
	return fmt.Sprintf("s3://%s/%s/wals/%s/%s", bucket, clusterName, walLogDir, name)
}

// execInPod runs a command in the given container and returns stdout, stderr
// and the error (non-nil for a non-zero exit code).
func execInPod(
	ctx context.Context,
	clientSet *kubernetes.Clientset,
	cfg *rest.Config,
	namespace, pod, container string,
	args ...string,
) (string, string, error) {
	return command.ExecuteInContainer(
		ctx,
		*clientSet,
		cfg,
		command.ContainerLocator{
			NamespaceName: namespace,
			PodName:       pod,
			ContainerName: container,
		},
		nil,
		args,
	)
}

// This test drives the plugin's parallel WAL restore directly: it invokes the
// instance-manager wal-restore command on the standby (which delegates to the
// plugin) and asserts the prefetch/spool/end-of-wal-stream state machine with
// maxParallel = 3. To control the archive deterministically it forges WAL
// segments on the object store by copying a real archived segment under new
// names.
var _ = Describe("Parallel WAL restore", func() {
	var (
		namespace *corev1.Namespace
		cl        client.Client
		clientSet *kubernetes.Clientset
		cfg       *rest.Config
	)

	BeforeEach(func(ctx SpecContext) {
		var err error
		cl, _, err = internalClient.NewClient()
		Expect(err).NotTo(HaveOccurred())
		clientSet, cfg, err = internalClient.NewClientSet()
		Expect(err).NotTo(HaveOccurred())
		namespace, err = nmsp.CreateUniqueNamespace(ctx, cl, "wal-restore-parallel")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func(ctx SpecContext) {
		Expect(cl.Delete(ctx, namespace)).To(Succeed())
	})

	It("prefetches segments, serves them from the spool and tracks end-of-wal-stream",
		func(ctx SpecContext) {
			ns := namespace.Name

			By("creating the object store backing resources")
			Expect(newObjectStoreResources(ns).Create(ctx, cl)).To(Succeed())

			By("creating the ObjectStore with WAL maxParallel")
			Expect(cl.Create(ctx, newObjectStore(ns))).To(Succeed())

			By("deploying the S3 client used to forge and inspect WAL segments")
			Expect(cl.Create(ctx, newS3ClientDeployment(ns))).To(Succeed())

			By("creating the cluster using the plugin as WAL archiver")
			cluster := newCluster(ns)
			Expect(cl.Create(ctx, cluster)).To(Succeed())

			By("waiting for the cluster to become ready")
			Eventually(func(g Gomega) {
				g.Expect(cl.Get(ctx,
					apitypes.NamespacedName{Name: clusterName, Namespace: ns},
					cluster)).To(Succeed())
				g.Expect(internalCluster.IsReady(*cluster)).To(BeTrue())
			}).WithTimeout(10 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())

			By("waiting for the S3 client to become ready")
			Eventually(func(g Gomega) {
				ready, err := deployment.IsReady(ctx, cl,
					apitypes.NamespacedName{Name: s3ClientName, Namespace: ns})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(ready).To(BeTrue())
			}).WithTimeout(2 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())

			primary := cluster.Status.CurrentPrimary
			Expect(primary).NotTo(BeEmpty())
			standby := clusterName + "-2"
			if primary == standby {
				standby = clusterName + "-1"
			}

			var s3ClientPods corev1.PodList
			Expect(cl.List(ctx, &s3ClientPods,
				client.InNamespace(ns),
				client.MatchingLabels{"app": s3ClientName})).To(Succeed())
			Expect(s3ClientPods.Items).NotTo(BeEmpty())
			s3Client := s3ClientPods.Items[0].Name

			// Operations scoped to the fixed pods/clients, kept as closures so the
			// step assertions below read like the original state-machine table.
			restore := func(name string) error {
				_, _, err := execInPod(ctx, clientSet, cfg, ns, standby, postgresContainer,
					managerExecutable, "wal-restore", name, pgWalPath+"/"+name)
				return err
			}
			existsIn := func(dir, name string) bool {
				_, _, err := execInPod(ctx, clientSet, cfg, ns, standby, postgresContainer,
					"test", "-f", dir+"/"+name)
				return err == nil
			}
			flagSet := func() bool { return existsIn(spoolDirectory, endOfWALStreamFlag) }
			spoolSegments := func() int {
				out, _, _ := execInPod(ctx, clientSet, cfg, ns, standby, postgresContainer,
					"sh", "-c",
					"ls -1 "+spoolDirectory+" 2>/dev/null | grep -Ec '^[0-9A-F]{24}$' || true")
				n, _ := strconv.Atoi(strings.TrimSpace(out))
				return n
			}
			purgeSpool := func() {
				_, _, _ = execInPod(ctx, clientSet, cfg, ns, standby, postgresContainer,
					"sh", "-c", "rm -f "+spoolDirectory+"/* 2>/dev/null; true")
			}
			forge := func(src, dst string) {
				// ExecuteInContainer folds a non-zero exit (and its output) into the
				// returned error, so we surface that rather than the empty stderr.
				_, _, err := execInPod(ctx, clientSet, cfg, ns, s3Client, s3ClientName,
					"aws", "s3", "cp", walObjectURI(src), walObjectURI(dst))
				Expect(err).NotTo(HaveOccurred(), "forging %s -> %s", src, dst)
			}
			objectExists := func(name string) bool {
				out, _, err := execInPod(ctx, clientSet, cfg, ns, s3Client, s3ClientName,
					"aws", "s3", "ls", walObjectURI(name))
				return err == nil && strings.TrimSpace(out) != ""
			}

			var latestWAL string
			By("archiving a real WAL on the primary and learning its name")
			_, _, err := execInPod(ctx, clientSet, cfg, ns, primary, postgresContainer,
				"psql", "-tAc", "CHECKPOINT")
			Expect(err).NotTo(HaveOccurred(), "CHECKPOINT on the primary failed")
			out, _, err := execInPod(ctx, clientSet, cfg, ns, primary, postgresContainer,
				"psql", "-tAc", "SELECT pg_walfile_name(pg_switch_wal())")
			Expect(err).NotTo(HaveOccurred(), "switching WAL on the primary failed")
			latestWAL = strings.TrimSpace(out)
			Expect(latestWAL).To(HavePrefix(walLogDir),
				"the freshly bootstrapped cluster should still be on the first WAL log")

			By("waiting for the archived WAL to land on the object store")
			Eventually(func() bool {
				return objectExists(latestWAL + ".gz")
			}).WithTimeout(2 * time.Minute).WithPolling(5 * time.Second).Should(BeTrue())

			By("forging WAL segments #1 to #5 from the archived WAL")
			for n := 1; n <= 5; n++ {
				forge(latestWAL+".gz", walFile(n))
			}

			By("ensuring the spool directory is empty on the standby")
			purgeSpool()

			// #1: served fresh; #2 and #3 prefetched into the spool; flag unset.
			By("requesting WAL #1: #1 restored, #2 and #3 prefetched")
			Expect(restore(walFile(1))).To(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(existsIn(pgWalPath, walFile(1))).To(BeTrue(), "#1 in pg_wal")
				g.Expect(existsIn(spoolDirectory, walFile(2))).To(BeTrue(), "#2 in spool")
				g.Expect(existsIn(spoolDirectory, walFile(3))).To(BeTrue(), "#3 in spool")
				g.Expect(flagSet()).To(BeFalse(), "end-of-wal-stream unset")
			}).WithTimeout(time.Minute).WithPolling(2 * time.Second).Should(Succeed())

			// #2: served from the spool; #3 stays prefetched; no new prefetch; flag unset.
			By("requesting WAL #2: served from the spool, #3 still prefetched")
			Expect(restore(walFile(2))).To(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(existsIn(pgWalPath, walFile(2))).To(BeTrue(), "#2 in pg_wal")
				g.Expect(existsIn(spoolDirectory, walFile(3))).To(BeTrue(), "#3 in spool")
				g.Expect(flagSet()).To(BeFalse(), "end-of-wal-stream unset")
			}).WithTimeout(time.Minute).WithPolling(2 * time.Second).Should(Succeed())

			// #3: served from the spool; spool now empty; flag unset.
			By("requesting WAL #3: served from the spool, spool now empty")
			Expect(restore(walFile(3))).To(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(existsIn(pgWalPath, walFile(3))).To(BeTrue(), "#3 in pg_wal")
				g.Expect(spoolSegments()).To(Equal(0), "no WAL segments in spool")
				g.Expect(flagSet()).To(BeFalse(), "end-of-wal-stream unset")
			}).WithTimeout(time.Minute).WithPolling(2 * time.Second).Should(Succeed())

			// #4: served fresh; #5 prefetched; #6 absent so end-of-wal-stream is set.
			By("requesting WAL #4: #4 restored, #5 prefetched, end-of-wal-stream set")
			Expect(restore(walFile(4))).To(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(existsIn(pgWalPath, walFile(4))).To(BeTrue(), "#4 in pg_wal")
				g.Expect(existsIn(spoolDirectory, walFile(5))).To(BeTrue(), "#5 in spool")
				g.Expect(flagSet()).To(BeTrue(), "end-of-wal-stream set (#6 absent)")
			}).WithTimeout(time.Minute).WithPolling(2 * time.Second).Should(Succeed())

			By("forging WAL segment #6 on the object store")
			forge(latestWAL+".gz", walFile(6))

			// #5: served from the spool; flag untouched (served before it is checked).
			By("requesting WAL #5: served from the spool, end-of-wal-stream still set")
			Expect(restore(walFile(5))).To(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(existsIn(pgWalPath, walFile(5))).To(BeTrue(), "#5 in pg_wal")
				g.Expect(spoolSegments()).To(Equal(0), "no WAL segments in spool")
				g.Expect(flagSet()).To(BeTrue(), "end-of-wal-stream still set")
			}).WithTimeout(time.Minute).WithPolling(2 * time.Second).Should(Succeed())

			// #6 (first): flag is set, so the request fails fast (exit 1) and the
			// flag is consumed, leaving an empty spool.
			By("requesting WAL #6: fails fast on the end-of-wal-stream flag, spool cleared")
			Expect(restore(walFile(6))).To(HaveOccurred(), "exit code should be 1")
			Eventually(func(g Gomega) {
				g.Expect(existsIn(pgWalPath, walFile(6))).To(BeFalse(), "#6 not restored")
				g.Expect(spoolSegments()).To(Equal(0), "no WAL segments in spool")
				g.Expect(flagSet()).To(BeFalse(), "end-of-wal-stream consumed")
			}).WithTimeout(time.Minute).WithPolling(2 * time.Second).Should(Succeed())

			// #6 (second): now present, restored; #7 and #8 absent so the flag is
			// set again.
			By("requesting WAL #6 again: #6 restored, end-of-wal-stream set again")
			Expect(restore(walFile(6))).To(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(existsIn(pgWalPath, walFile(6))).To(BeTrue(), "#6 in pg_wal")
				g.Expect(spoolSegments()).To(Equal(0), "no WAL segments in spool")
				g.Expect(flagSet()).To(BeTrue(), "end-of-wal-stream set (#7/#8 absent)")
			}).WithTimeout(time.Minute).WithPolling(2 * time.Second).Should(Succeed())
		})
})
