package environment

import (
	"fmt"

	ackcondition "github.com/aws-controllers-k8s/runtime/pkg/condition"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/mwaa/types"
	corev1 "k8s.io/api/core/v1"

	"github.com/aws-controllers-k8s/mwaa-controller/pkg/tags"
)

var syncTags = tags.SyncTags

// handleUpdateFailed inspects the resource's Status.LastUpdate and, if MWAA
// reported a failed update, sets ACK.ResourceSynced=False on the resource
// with the MWAA error details and returns a non-terminal requeue error so
// the condition is refreshed on future reconciles.
//
// MWAA rolls back after a failed update and the environment returns to
// AVAILABLE, but Status.LastUpdate.Status stays "FAILED" and
// Status.LastUpdate.Error carries the reason. Surfacing this as
// ACK.ResourceSynced=False is the only signal the user has that their patch
// silently failed — otherwise the CR would appear Synced with stale config.
//
// Using a non-terminal requeue error (not a terminal error) so that:
//   - The requeue timer still drives re-polling of GetEnvironment.
//   - The next user CR patch triggers a new Update attempt; on success,
//     MWAA overwrites LastUpdate.Status to SUCCESS and this branch no
//     longer fires, clearing the condition automatically.
//
// Returns (nil, nil) when the resource is not in a failed-update state and
// the caller should proceed normally. Returns (r, requeueErr) when the
// failed state is detected.
func handleUpdateFailed(r *resource) (*resource, error) {
	ko := r.ko
	if ko.Status.LastUpdate == nil ||
		ko.Status.LastUpdate.Status == nil ||
		*ko.Status.LastUpdate.Status != string(svcsdktypes.UpdateStatusFailed) {
		return nil, nil
	}
	code := "Unknown"
	msg := "update failed with no error details"
	if ko.Status.LastUpdate.Error != nil {
		if ko.Status.LastUpdate.Error.ErrorCode != nil {
			code = *ko.Status.LastUpdate.Error.ErrorCode
		}
		if ko.Status.LastUpdate.Error.ErrorMessage != nil {
			msg = *ko.Status.LastUpdate.Error.ErrorMessage
		}
	}
	errMsg := fmt.Sprintf("last UpdateEnvironment failed: %s: %s; patch the spec to retry", code, msg)
	ackcondition.SetSynced(r, corev1.ConditionFalse, &errMsg, nil)
	return r, ackrequeue.NeededAfter(
		fmt.Errorf("%s", errMsg),
		ackrequeue.DefaultRequeueAfterDuration,
	)
}
