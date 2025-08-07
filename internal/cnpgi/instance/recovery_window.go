package instance

import (
	"context"
	"time"

	"github.com/cloudnative-pg/barman-cloud/pkg/catalog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	barmancloudv1 "github.com/cloudnative-pg/plugin-barman-cloud/api/v1"
)

// updateRecoveryWindow updates the recovery window inside the object
// store status subresource
func updateRecoveryWindow(
	ctx context.Context,
	c client.Client,
	backupList *catalog.Catalog,
	objectStore *barmancloudv1.ObjectStore,
	serverName string,
) error {
	// Set the recovery window inside the barman object store object
	convertTime := func(t *time.Time) *metav1.Time {
		if t == nil {
			return nil
		}
		return ptr.To(metav1.NewTime(*t))
	}

	recoveryWindow := barmancloudv1.RecoveryWindow{
		FirstRecoverabilityPoint: convertTime(backupList.GetFirstRecoverabilityPoint()),
		LastSuccessfulBackupTime: convertTime(backupList.GetLastSuccessfulBackupTime()),
	}

	if objectStore.Status.ServerRecoveryWindow == nil {
		objectStore.Status.ServerRecoveryWindow = make(map[string]barmancloudv1.RecoveryWindow)
	}
	objectStore.Status.ServerRecoveryWindow[serverName] = recoveryWindow

	return c.Status().Update(ctx, objectStore)
}
