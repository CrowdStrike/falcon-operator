package common

import (
	"context"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const readinessNamespace = "falcon-clusterguard"

type readinessReconciler struct{ client.Client }

func (f *readinessReconciler) GetK8sReader() client.Reader { return f.Client }
func (f *readinessReconciler) GetScheme() *runtime.Scheme  { return f.Client.Scheme() }
func (f *readinessReconciler) GetLog() logr.Logger         { return logr.Discard() }

func newReadinessReconciler(t *testing.T, objs ...client.Object) *readinessReconciler {
	t.Helper()
	scheme := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{corev1.AddToScheme, appsv1.AddToScheme, falconv1alpha1.AddToScheme} {
		if err := add(scheme); err != nil {
			t.Fatal(err)
		}
	}
	builder := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...)
	for _, o := range objs {
		if _, ok := o.(*falconv1alpha1.FalconClusterGuard); ok {
			builder = builder.WithStatusSubresource(o)
		}
	}
	return &readinessReconciler{Client: builder.Build()}
}

var readinessSelector = &metav1.LabelSelector{MatchLabels: map[string]string{"app": "workload"}}

// workloadPod returns a pod selected by readinessSelector whose first container is waiting with reason.
func workloadPod(name, reason, message string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: readinessNamespace, Labels: readinessSelector.MatchLabels},
		Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{
			Name:  "app",
			State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: reason, Message: message}},
		}}},
	}
}

func deployment(status appsv1.DeploymentStatus) *appsv1.Deployment {
	replicas := int32(2)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "workload", Namespace: readinessNamespace, Generation: 2},
		Spec:       appsv1.DeploymentSpec{Replicas: &replicas, Selector: readinessSelector},
		Status:     status,
	}
}

func TestDeploymentStatus(t *testing.T) {
	tests := []struct {
		name        string
		status      appsv1.DeploymentStatus
		pods        []client.Object
		wantReady   bool
		wantFailed  bool
		wantMessage string
	}{
		{
			name:      "rolled out and available",
			status:    appsv1.DeploymentStatus{ObservedGeneration: 2, Replicas: 2, UpdatedReplicas: 2, AvailableReplicas: 2},
			wantReady: true,
		},
		{
			name:        "spec change not observed yet",
			status:      appsv1.DeploymentStatus{ObservedGeneration: 1, Replicas: 2, UpdatedReplicas: 2, AvailableReplicas: 2},
			wantMessage: "Deployment workload: waiting for the rollout to start",
		},
		{
			name:        "replicas not updated",
			status:      appsv1.DeploymentStatus{ObservedGeneration: 2, Replicas: 2, UpdatedReplicas: 1, AvailableReplicas: 2},
			wantMessage: "Deployment workload: 1 of 2 replicas updated",
		},
		{
			name:        "old replicas still running",
			status:      appsv1.DeploymentStatus{ObservedGeneration: 2, Replicas: 3, UpdatedReplicas: 2, AvailableReplicas: 2},
			wantMessage: "Deployment workload: 1 old replicas pending termination",
		},
		{
			name:        "pods starting",
			status:      appsv1.DeploymentStatus{ObservedGeneration: 2, Replicas: 2, UpdatedReplicas: 2, AvailableReplicas: 1},
			pods:        []client.Object{workloadPod("workload-a", "ContainerCreating", "")},
			wantMessage: "Deployment workload: 1 of 2 updated replicas available; pod workload-a container app: ContainerCreating",
		},
		{
			name:   "pod failure takes precedence over a pod that is starting",
			status: appsv1.DeploymentStatus{ObservedGeneration: 2, Replicas: 2, UpdatedReplicas: 2},
			pods: []client.Object{
				workloadPod("workload-a", "ContainerCreating", ""),
				workloadPod("workload-b", "ImagePullBackOff", `Back-off pulling image "bad:tag"`),
			},
			wantFailed:  true,
			wantMessage: `Deployment workload: 0 of 2 updated replicas available; pod workload-b container app: ImagePullBackOff: Back-off pulling image "bad:tag"`,
		},
		{
			name: "progress deadline exceeded",
			status: appsv1.DeploymentStatus{ObservedGeneration: 2, Replicas: 2, UpdatedReplicas: 2, Conditions: []appsv1.DeploymentCondition{
				{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionFalse, Reason: "ProgressDeadlineExceeded"},
			}},
			wantFailed:  true,
			wantMessage: "Deployment workload: 0 of 2 updated replicas available",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newReadinessReconciler(t, append(tt.pods, deployment(tt.status))...)
			got, err := DeploymentStatus(context.Background(), r, types.NamespacedName{Name: "workload", Namespace: readinessNamespace})
			if err != nil {
				t.Fatal(err)
			}
			want := WorkloadStatus{Ready: tt.wantReady, Failed: tt.wantFailed, Message: tt.wantMessage}
			if got != want {
				t.Errorf("DeploymentStatus() = %+v, want %+v", got, want)
			}
		})
	}
}

func TestDaemonSetStatus(t *testing.T) {
	tests := []struct {
		name        string
		strategy    appsv1.DaemonSetUpdateStrategyType
		status      appsv1.DaemonSetStatus
		pods        []client.Object
		wantReady   bool
		wantFailed  bool
		wantMessage string
	}{
		{
			name:      "rolled out and available",
			status:    appsv1.DaemonSetStatus{ObservedGeneration: 2, DesiredNumberScheduled: 3, UpdatedNumberScheduled: 3, NumberAvailable: 3},
			wantReady: true,
		},
		{
			name:        "spec change not observed yet",
			status:      appsv1.DaemonSetStatus{ObservedGeneration: 1, DesiredNumberScheduled: 3, UpdatedNumberScheduled: 3, NumberAvailable: 3},
			wantMessage: "DaemonSet workload: waiting for the rollout to start",
		},
		{
			name:        "pods not updated",
			status:      appsv1.DaemonSetStatus{ObservedGeneration: 2, DesiredNumberScheduled: 3, UpdatedNumberScheduled: 1, NumberAvailable: 3},
			wantMessage: "DaemonSet workload: 1 of 3 pods updated",
		},
		{
			name:      "OnDelete does not wait for pods to be updated",
			strategy:  appsv1.OnDeleteDaemonSetStrategyType,
			status:    appsv1.DaemonSetStatus{ObservedGeneration: 2, DesiredNumberScheduled: 3, UpdatedNumberScheduled: 1, NumberAvailable: 3},
			wantReady: true,
		},
		{
			name:        "pod crash looping",
			status:      appsv1.DaemonSetStatus{ObservedGeneration: 2, DesiredNumberScheduled: 3, UpdatedNumberScheduled: 3, NumberAvailable: 2},
			pods:        []client.Object{workloadPod("workload-a", "CrashLoopBackOff", "")},
			wantFailed:  true,
			wantMessage: "DaemonSet workload: 2 of 3 pods available; pod workload-a container app: CrashLoopBackOff",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{Name: "workload", Namespace: readinessNamespace, Generation: 2},
				Spec: appsv1.DaemonSetSpec{
					Selector:       readinessSelector,
					UpdateStrategy: appsv1.DaemonSetUpdateStrategy{Type: tt.strategy},
				},
				Status: tt.status,
			}
			r := newReadinessReconciler(t, append(tt.pods, ds)...)
			got, err := DaemonSetStatus(context.Background(), r, types.NamespacedName{Name: "workload", Namespace: readinessNamespace})
			if err != nil {
				t.Fatal(err)
			}
			want := WorkloadStatus{Ready: tt.wantReady, Failed: tt.wantFailed, Message: tt.wantMessage}
			if got != want {
				t.Errorf("DaemonSetStatus() = %+v, want %+v", got, want)
			}
		})
	}
}

func TestWorkloadStatusCondition(t *testing.T) {
	tests := []struct {
		name   string
		status WorkloadStatus
		want   metav1.Condition
	}{
		{
			name:   "ready",
			status: WorkloadStatus{Ready: true},
			want:   metav1.Condition{Status: metav1.ConditionTrue, Reason: falconv1alpha1.ReasonInstallSucceeded, Message: "component is ready"},
		},
		{
			name:   "progressing",
			status: WorkloadStatus{Message: "starting"},
			want:   metav1.Condition{Status: metav1.ConditionFalse, Reason: falconv1alpha1.ReasonProgressing, Message: "starting"},
		},
		{
			name:   "failed",
			status: WorkloadStatus{Failed: true, Message: "stuck"},
			want:   metav1.Condition{Status: metav1.ConditionFalse, Reason: falconv1alpha1.ReasonFailed, Message: "stuck"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.want.Type = "ComponentReady"
			tt.want.ObservedGeneration = 3
			if got := tt.status.Condition("ComponentReady", "component is ready", 3); got != tt.want {
				t.Errorf("Condition() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestConditionsSet checks that a condition that stays False is rewritten when its cause changes, which
// ConditionsUpdate skips, and that an unchanged condition is not written again.
func TestConditionsSet(t *testing.T) {
	ctx := context.Background()
	owner := &falconv1alpha1.FalconClusterGuard{ObjectMeta: metav1.ObjectMeta{Name: "fcg"}}
	r := newReadinessReconciler(t, owner)
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: owner.Name}}
	set := func(message string) *falconv1alpha1.FalconClusterGuard {
		t.Helper()
		condition := metav1.Condition{Type: "ComponentReady", Status: metav1.ConditionFalse, Reason: falconv1alpha1.ReasonProgressing, Message: message}
		if err := ConditionsSet(r, ctx, req, logr.Discard(), owner, &owner.Status, condition); err != nil {
			t.Fatal(err)
		}
		persisted := &falconv1alpha1.FalconClusterGuard{}
		if err := r.Get(ctx, req.NamespacedName, persisted); err != nil {
			t.Fatal(err)
		}
		return persisted
	}

	first := set("starting")
	changed := set("ImagePullBackOff")
	if got := meta.FindStatusCondition(changed.Status.Conditions, "ComponentReady").Message; got != "ImagePullBackOff" {
		t.Errorf("message = %q, want the new message", got)
	}
	if changed.ResourceVersion == first.ResourceVersion {
		t.Error("expected a status write when the message changes")
	}
	if unchanged := set("ImagePullBackOff"); unchanged.ResourceVersion != changed.ResourceVersion {
		t.Errorf("unchanged condition was written again: resourceVersion %s -> %s", changed.ResourceVersion, unchanged.ResourceVersion)
	}
}
