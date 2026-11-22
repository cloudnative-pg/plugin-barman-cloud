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

package instance

import (
	"context"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("deleteBackupsNotInCatalog", func() {
	const (
		clusterName = "test-cluster"
		clusterUID  = "test-cluster-uid"
		namespace   = "test-ns"
	)

	var (
		ctx     context.Context
		cluster *cnpgv1.Cluster
	)

	newTestBackup := func(name, backupID string, phase cnpgv1.BackupPhase) *cnpgv1.Backup {
		return &cnpgv1.Backup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: cnpgv1.BackupSpec{
				Cluster: cnpgv1.LocalObjectReference{
					Name: clusterName,
				},
			},
			Status: cnpgv1.BackupStatus{
				Phase:          phase,
				BackupID:       backupID,
				Method:         cnpgv1.BackupMethodPlugin,
				PluginMetadata: newBackupResultMetadata(types.UID(clusterUID), 0).toMap(),
			},
		}
	}

	newFakeClient := func(backups ...*cnpgv1.Backup) client.Client {
		scheme := runtime.NewScheme()
		utilruntime.Must(cnpgv1.AddToScheme(scheme))

		objects := make([]client.Object, 0, len(backups))
		for _, backup := range backups {
			objects = append(objects, backup)
		}

		return fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(objects...).
			Build()
	}

	backupExists := func(cli client.Client, name string) bool {
		backup := &cnpgv1.Backup{}
		return cli.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, backup) == nil
	}

	BeforeEach(func() {
		ctx = context.Background()
		cluster = &cnpgv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: namespace,
				UID:       types.UID(clusterUID),
			},
		}
	})

	Context("when the catalog lists the backup", func() {
		It("should keep the Backup object", func() {
			cli := newFakeClient(newTestBackup("backup-a", "20261101T020000", cnpgv1.BackupPhaseCompleted))

			err := deleteBackupsNotInCatalog(ctx, cli, cluster, []string{"20261101T020000"})
			Expect(err).NotTo(HaveOccurred())
			Expect(backupExists(cli, "backup-a")).To(BeTrue())
		})
	})

	Context("when the catalog is empty", func() {
		It("should keep every completed Backup object", func() {
			cli := newFakeClient(
				newTestBackup("backup-a", "20261101T020000", cnpgv1.BackupPhaseCompleted),
				newTestBackup("backup-b", "20261102T020000", cnpgv1.BackupPhaseCompleted),
			)

			err := deleteBackupsNotInCatalog(ctx, cli, cluster, []string{})
			Expect(err).NotTo(HaveOccurred())
			Expect(backupExists(cli, "backup-a")).To(BeTrue())
			Expect(backupExists(cli, "backup-b")).To(BeTrue())
		})
	})

	Context("when a backup is missing from a short catalog listing", func() {
		It("should keep a backup newer than the oldest catalog entry", func() {
			// The catalog still holds backups from November 1st, so the
			// November 20th backup is still inside the retention window.
			// Its absence means the listing came back short.
			cli := newFakeClient(newTestBackup("backup-new", "20261120T020000", cnpgv1.BackupPhaseCompleted))

			err := deleteBackupsNotInCatalog(ctx, cli, cluster, []string{"20261101T020000", "20261110T020000"})
			Expect(err).NotTo(HaveOccurred())
			Expect(backupExists(cli, "backup-new")).To(BeTrue())
		})

		It("should keep a backup with an unparsable backup ID", func() {
			cli := newFakeClient(newTestBackup("backup-bad-id", "not-a-backup-id", cnpgv1.BackupPhaseCompleted))

			err := deleteBackupsNotInCatalog(ctx, cli, cluster, []string{"20261101T020000"})
			Expect(err).NotTo(HaveOccurred())
			Expect(backupExists(cli, "backup-bad-id")).To(BeTrue())
		})
	})

	Context("when a backup was pruned by the retention policy", func() {
		It("should delete a backup older than the oldest catalog entry", func() {
			cli := newFakeClient(newTestBackup("backup-old", "20261001T020000", cnpgv1.BackupPhaseCompleted))

			err := deleteBackupsNotInCatalog(ctx, cli, cluster, []string{"20261101T020000", "20261110T020000"})
			Expect(err).NotTo(HaveOccurred())
			Expect(backupExists(cli, "backup-old")).To(BeFalse())
		})

		It("should delete a backup older than the oldest catalog entry with the dateless ID format", func() {
			cli := newFakeClient(newTestBackup("backup-old", "20261001020000", cnpgv1.BackupPhaseCompleted))

			err := deleteBackupsNotInCatalog(ctx, cli, cluster, []string{"20261101020000"})
			Expect(err).NotTo(HaveOccurred())
			Expect(backupExists(cli, "backup-old")).To(BeFalse())
		})
	})

	Context("when the backup should not be considered", func() {
		It("should keep a backup that is not completed", func() {
			cli := newFakeClient(newTestBackup("backup-pending", "20261001T020000", cnpgv1.BackupPhasePending))

			err := deleteBackupsNotInCatalog(ctx, cli, cluster, []string{"20261101T020000"})
			Expect(err).NotTo(HaveOccurred())
			Expect(backupExists(cli, "backup-pending")).To(BeTrue())
		})
	})
})

var _ = Describe("parseBackupIDTime", func() {
	It("should parse IDs with and without the T separator", func() {
		withT, err := parseBackupIDTime("20261120T020000")
		Expect(err).NotTo(HaveOccurred())

		withoutT, err := parseBackupIDTime("20261120020000")
		Expect(err).NotTo(HaveOccurred())

		Expect(withT).To(Equal(withoutT))
	})

	It("should reject IDs that are not timestamps", func() {
		_, err := parseBackupIDTime("not-a-backup-id")
		Expect(err).To(HaveOccurred())
	})
})
