package common

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FalconResource defines the interface that all Falcon CRDs must implement
// to support generic status updates.
type FalconResource interface {
	client.Object
	GetGeneration() int64
	GetConditions() *[]metav1.Condition
}

// StatusUpdate updates the status of a Falcon resource with a condition.
// It uses retry logic to handle conflicts and is generic across all Falcon CRD types.
//
// Parameters:
//   - ctx: The context for the operation
//   - r: The Kubernetes client for reading resources
//   - statusWriter: The status subresource writer for updating status
//   - req: The reconcile request containing the resource name and namespace
//   - log: The logger for recording errors
//   - resource: The Falcon resource to update (must implement FalconResource interface)
//   - condType: The type of condition to set
//   - status: The status of the condition (True, False, Unknown)
//   - reason: A brief reason for the condition status
//   - message: A detailed message explaining the condition
//
// Returns an error if the status update fails after retries.
func StatusUpdate[T FalconResource](
	ctx context.Context,
	r client.Client,
	statusWriter client.StatusWriter,
	req ctrl.Request,
	log logr.Logger,
	resource T,
	condType string,
	status metav1.ConditionStatus,
	reason string,
	message string,
) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := r.Get(ctx, req.NamespacedName, resource); err != nil {
			return err
		}

		meta.SetStatusCondition(resource.GetConditions(), metav1.Condition{
			Status:             status,
			Reason:             reason,
			Message:            message,
			Type:               condType,
			ObservedGeneration: resource.GetGeneration(),
		})

		return statusWriter.Update(ctx, resource)
	})
	if err != nil {
		log.Error(err, "Failed to update status", "resourceType", resource.GetObjectKind().GroupVersionKind().Kind)
		return err
	}

	return nil
}
