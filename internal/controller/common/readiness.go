package common

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WorkloadNotReadyRequeue is how often a controller re-checks workloads that are not ready yet.
// Pod-level failures such as ImagePullBackOff do not always change the Deployment or DaemonSet status,
// so the owned-object watches alone would leave the reported reason stale.
const WorkloadNotReadyRequeue = 10 * time.Second

// podFailureReasons are container waiting reasons that do not clear without a change to the spec or environment.
var podFailureReasons = []string{
	"ErrImagePull",
	"ErrImageNeverPull",
	"ImagePullBackOff",
	"InvalidImageName",
	"CrashLoopBackOff",
	"CreateContainerConfigError",
	"CreateContainerError",
	"RunContainerError",
}

// WorkloadStatus reports whether a Deployment or DaemonSet has finished rolling out and is available.
type WorkloadStatus struct {
	Ready bool
	// Failed is set when a pod is stuck in a state that will not clear on its own, such as ImagePullBackOff.
	Failed  bool
	Message string
}

// Condition returns the component condition of type condType that reflects w.
func (w WorkloadStatus) Condition(condType, readyMessage string, generation int64) metav1.Condition {
	cond := metav1.Condition{
		Type:               condType,
		Status:             metav1.ConditionTrue,
		Reason:             falconv1alpha1.ReasonInstallSucceeded,
		Message:            readyMessage,
		ObservedGeneration: generation,
	}
	if !w.Ready {
		cond.Status = metav1.ConditionFalse
		cond.Reason = falconv1alpha1.ReasonProgressing
		if w.Failed {
			cond.Reason = falconv1alpha1.ReasonFailed
		}
		cond.Message = w.Message
	}
	return cond
}

// DeploymentStatus reports the rollout state of the Deployment at key, using the same checks as kubectl rollout status.
func DeploymentStatus(ctx context.Context, r Reconciler, key types.NamespacedName) (WorkloadStatus, error) {
	dep := &appsv1.Deployment{}
	if err := pkgcommon.GetNamespacedObject(ctx, r, r.GetK8sReader(), key, dep); err != nil {
		return WorkloadStatus{}, err
	}

	desired := int32(1)
	if dep.Spec.Replicas != nil {
		desired = *dep.Spec.Replicas
	}
	st := dep.Status

	var waiting string
	switch {
	case st.ObservedGeneration < dep.Generation:
		waiting = "waiting for the rollout to start"
	case st.UpdatedReplicas < desired:
		waiting = fmt.Sprintf("%d of %d replicas updated", st.UpdatedReplicas, desired)
	case st.Replicas > st.UpdatedReplicas:
		waiting = fmt.Sprintf("%d old replicas pending termination", st.Replicas-st.UpdatedReplicas)
	case st.AvailableReplicas < st.UpdatedReplicas:
		waiting = fmt.Sprintf("%d of %d updated replicas available", st.AvailableReplicas, st.UpdatedReplicas)
	default:
		return WorkloadStatus{Ready: true}, nil
	}

	ws, err := workloadNotReady(ctx, r.GetK8sReader(), "Deployment", dep.Namespace, dep.Name, dep.Spec.Selector, waiting)
	if err != nil {
		return WorkloadStatus{}, err
	}
	for _, c := range st.Conditions {
		if c.Type == appsv1.DeploymentProgressing && c.Reason == "ProgressDeadlineExceeded" {
			ws.Failed = true
		}
	}
	return ws, nil
}

// DaemonSetStatus reports the rollout state of the DaemonSet at key, using the same checks as kubectl rollout status.
// DaemonSets with the OnDelete strategy only update pods when they are deleted, so they are not required to be up to date.
func DaemonSetStatus(ctx context.Context, r Reconciler, key types.NamespacedName) (WorkloadStatus, error) {
	ds := &appsv1.DaemonSet{}
	if err := pkgcommon.GetNamespacedObject(ctx, r, r.GetK8sReader(), key, ds); err != nil {
		return WorkloadStatus{}, err
	}
	st := ds.Status

	var waiting string
	switch {
	case st.ObservedGeneration < ds.Generation:
		waiting = "waiting for the rollout to start"
	case ds.Spec.UpdateStrategy.Type != appsv1.OnDeleteDaemonSetStrategyType && st.UpdatedNumberScheduled < st.DesiredNumberScheduled:
		waiting = fmt.Sprintf("%d of %d pods updated", st.UpdatedNumberScheduled, st.DesiredNumberScheduled)
	case st.NumberAvailable < st.DesiredNumberScheduled:
		waiting = fmt.Sprintf("%d of %d pods available", st.NumberAvailable, st.DesiredNumberScheduled)
	default:
		return WorkloadStatus{Ready: true}, nil
	}

	return workloadNotReady(ctx, r.GetK8sReader(), "DaemonSet", ds.Namespace, ds.Name, ds.Spec.Selector, waiting)
}

// workloadNotReady builds the status of a workload that is not ready, adding the first pod problem found.
func workloadNotReady(ctx context.Context, reader client.Reader, kind, namespace, name string, selector *metav1.LabelSelector, waiting string) (WorkloadStatus, error) {
	ws := WorkloadStatus{Message: fmt.Sprintf("%s %s: %s", kind, name, waiting)}
	if selector == nil {
		return ws, nil
	}

	problem, failed, err := podProblem(ctx, reader, namespace, selector)
	if err != nil {
		return WorkloadStatus{}, err
	}
	if problem != "" {
		ws.Message += "; " + problem
	}
	ws.Failed = failed
	return ws, nil
}

// podProblem describes why the pods matching selector are not ready. Failures listed in podFailureReasons take
// precedence over transient states such as ContainerCreating. Pods are checked in name order so the result is stable.
func podProblem(ctx context.Context, reader client.Reader, namespace string, selector *metav1.LabelSelector) (string, bool, error) {
	sel, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return "", false, err
	}
	pods := &corev1.PodList{}
	if err := reader.List(ctx, pods, client.InNamespace(namespace), client.MatchingLabelsSelector{Selector: sel}); err != nil {
		return "", false, fmt.Errorf("unable to list pods: %w", err)
	}
	slices.SortFunc(pods.Items, func(a, b corev1.Pod) int { return strings.Compare(a.Name, b.Name) })

	var transient string
	for _, pod := range pods.Items {
		if pod.DeletionTimestamp != nil {
			continue
		}
		if failure := PodStartFailure(&pod); failure != "" {
			return fmt.Sprintf("pod %s %s", pod.Name, failure), true, nil
		}
		if transient != "" {
			continue
		}
		for _, cs := range slices.Concat(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses) {
			if w := cs.State.Waiting; w != nil && w.Reason != "" {
				transient = fmt.Sprintf("pod %s %s", pod.Name, containerWaiting(cs))
				break
			}
		}
		for _, c := range pod.Status.Conditions {
			if transient == "" && c.Type == corev1.PodScheduled && c.Status == corev1.ConditionFalse {
				transient = fmt.Sprintf("pod %s: %s: %s", pod.Name, c.Reason, c.Message)
			}
		}
	}
	return transient, false, nil
}

// PodStartFailure describes the first container of pod that is waiting for one of podFailureReasons, such as
// ErrImageNeverPull or CrashLoopBackOff, or returns "" if none is.
func PodStartFailure(pod *corev1.Pod) string {
	for _, cs := range slices.Concat(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses) {
		if w := cs.State.Waiting; w != nil && slices.Contains(podFailureReasons, w.Reason) {
			return containerWaiting(cs)
		}
	}
	return ""
}

// containerWaiting describes a waiting container as "container <name>: <reason>[: <message>]".
func containerWaiting(cs corev1.ContainerStatus) string {
	s := fmt.Sprintf("container %s: %s", cs.Name, cs.State.Waiting.Reason)
	if cs.State.Waiting.Message != "" {
		s += ": " + cs.State.Waiting.Message
	}
	return s
}
