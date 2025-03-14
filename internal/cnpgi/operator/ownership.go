package operator

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

// setOwnerReference explicitly set the owner reference between an
// owner object and a controller one.
//
// Important: this function won't use any registered scheme and will
// fail unless the metadata has been correctly set into the owner
// object.
func setOwnerReference(owner, controlled metav1.Object) error {
	ro, ok := owner.(runtime.Object)
	if !ok {
		return fmt.Errorf("%T is not a runtime.Object, cannot call setOwnerReference", owner)
	}

	if len(ro.DeepCopyObject().GetObjectKind().GroupVersionKind().Group) == 0 {
		return fmt.Errorf("%T metadata have not been set, cannot call setOwnerReference", owner)
	}

	controlled.SetOwnerReferences([]metav1.OwnerReference{
		{
			APIVersion:         ro.GetObjectKind().GroupVersionKind().GroupVersion().String(),
			Kind:               ro.GetObjectKind().GroupVersionKind().Kind,
			Name:               owner.GetName(),
			UID:                owner.GetUID(),
			BlockOwnerDeletion: ptr.To(true),
			Controller:         ptr.To(true),
		},
	})

	return nil
}
