package node_sensor

import (
	"context"
	"fmt"
	"slices"
	"time"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	"github.com/crowdstrike/falcon-operator/internal/controller/components"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Config struct {
	components.BaseConfig
	FalconAPI  *falconv1alpha1.FalconAPI
	NodeSensor falconv1alpha1.FalconClusterGuardNodeSpec
}

const nodeSensorDefaultPrefix = "falcon"

func (n *NodeSensor) prefix() string {
	if n.cfg.NamePrefix != "" {
		return n.cfg.NamePrefix
	}
	return nodeSensorDefaultPrefix
}

type NodeSensor struct {
	r   k8sutils.Reconciler
	cfg Config
}

func New(r k8sutils.Reconciler, cfg Config) *NodeSensor {
	return &NodeSensor{r: r, cfg: cfg}
}

// Reconcile runs all node sensor reconciliation steps in order.
// The FalconFinalizer is managed by the FalconClusterGuard controller, which calls Finalize on CR deletion.
func (n *NodeSensor) Reconcile(ctx context.Context) (ctrl.Result, error) {
	if err := n.reconcileServiceAccount(ctx); err != nil {
		return ctrl.Result{}, err
	}
	if err := n.reconcilePriorityClass(ctx); err != nil {
		return ctrl.Result{}, err
	}
	if err := n.reconcileConfigMap(ctx); err != nil {
		return ctrl.Result{}, err
	}
	if err := n.reconcileClusterRoleBinding(ctx); err != nil {
		return ctrl.Result{}, err
	}
	if err := n.reconcileDaemonSet(ctx); err != nil {
		return ctrl.Result{}, err
	}
	if err := n.reconcileCleanupServiceAccount(ctx); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, k8sutils.ConditionsUpdate(n.r, ctx, n.cfg.Request, n.r.GetLog(), n.cfg.Owner, n.cfg.Status, metav1.Condition{
		Type:               falconv1alpha1.ConditionNodeSensorReady,
		Status:             metav1.ConditionTrue,
		Reason:             falconv1alpha1.ReasonInstallSucceeded,
		Message:            "Node sensor DaemonSet is ready",
		ObservedGeneration: n.cfg.Owner.GetGeneration(),
	})
}

// skipNodeCleanup reports whether the user disabled removal of /opt/CrowdStrike from the nodes.
func (n *NodeSensor) skipNodeCleanup() bool {
	return n.cfg.NodeSensor.NodeCleanup != nil && *n.cfg.NodeSensor.NodeCleanup
}

func (n *NodeSensor) sensorDaemonSetRef() *appsv1.DaemonSet {
	return &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: n.prefix() + "-sensor", Namespace: n.cfg.InstallNamespace}}
}

func (n *NodeSensor) cleanupDaemonSetRef() *appsv1.DaemonSet {
	return &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: n.prefix() + "-sensor-cleanup", Namespace: n.cfg.InstallNamespace}}
}

// Finalize runs node cleanup while the FalconClusterGuard is being deleted.
// It requeues until the cleanup DaemonSet has run on every node; the remaining resources are garbage collected.
func (n *NodeSensor) Finalize(ctx context.Context) (ctrl.Result, error) {
	if n.skipNodeCleanup() {
		n.r.GetLog().Info("Skipping node cleanup because it is disabled", "disableCleanup", true)
		return ctrl.Result{}, nil
	}
	done, err := n.finalize(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !done {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}
	return ctrl.Result{}, nil
}

// Cleanup removes the node sensor when the component is disabled.
// If the sensor DaemonSet (or an in-progress cleanup DaemonSet) exists, node cleanup runs first and
// Cleanup requeues until it finishes. It is a no-op once the resources are gone.
func (n *NodeSensor) Cleanup(ctx context.Context) (ctrl.Result, error) {
	sensorDS, err := components.GetOwned(ctx, n.r, n.cfg.Owner, n.sensorDaemonSetRef())
	if err != nil {
		return ctrl.Result{}, err
	}
	cleanupDS, err := components.GetOwned(ctx, n.r, n.cfg.Owner, n.cleanupDaemonSetRef())
	if err != nil {
		return ctrl.Result{}, err
	}
	if sensorDS || cleanupDS {
		if n.skipNodeCleanup() {
			if err := components.DeleteOwned(ctx, n.r, n.cfg.Owner, n.sensorDaemonSetRef(), n.cleanupDaemonSetRef()); err != nil {
				return ctrl.Result{}, err
			}
		} else if result, err := n.Finalize(ctx); err != nil || result.RequeueAfter > 0 {
			return result, err
		}
	}

	ns := n.cfg.InstallNamespace
	objs := []client.Object{
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.ClusterGuardNodeSensorConfigMapName, Namespace: ns}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.GKEAutoPilotConfigMapName, Namespace: ns}},
		&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.ClusterGuardNodeSensorClusterRoleBindingName}},
		&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.ClusterGuardNodeSensorServiceAccountName, Namespace: ns}},
		&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.ClusterGuardNodeSensorCleanupServiceAccountName, Namespace: ns}},
		&schedulingv1.PriorityClass{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.ClusterGuardNodeSensorPriorityClassName}},
	}
	if pc := n.cfg.NodeSensor.PriorityClass.Name; pc != "" && pc != pkgcommon.ClusterGuardNodeSensorPriorityClassName {
		objs = append(objs, &schedulingv1.PriorityClass{ObjectMeta: metav1.ObjectMeta{Name: pc}})
	}
	if err := components.DeleteOwned(ctx, n.r, n.cfg.Owner, objs...); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, k8sutils.ConditionsRemove(n.r, ctx, n.cfg.Request, n.r.GetLog(), n.cfg.Owner, n.cfg.Status,
		falconv1alpha1.ConditionNodeSensorReady)
}

// Safe to call on every reconcile — delete and DaemonSet creation are idempotent.
func (n *NodeSensor) finalize(ctx context.Context) (bool, error) {
	dsCleanupName := n.prefix() + "-sensor-cleanup"

	if err := components.DeleteOwned(ctx, n.r, n.cfg.Owner, n.sensorDaemonSetRef()); err != nil {
		return false, err
	}

	if err := n.reconcileCleanupServiceAccount(ctx); err != nil {
		return false, err
	}
	if err := n.reconcileCleanupDaemonSet(ctx); err != nil {
		return false, err
	}

	daemonset := &appsv1.DaemonSet{}
	if err := pkgcommon.GetNamespacedObject(ctx, n.r, n.r.GetK8sReader(),
		types.NamespacedName{Name: dsCleanupName, Namespace: n.cfg.InstallNamespace}, daemonset); err != nil {
		if apierrors.IsNotFound(err) {
			n.r.GetLog().Info("Cleanup DaemonSet not found yet, requeueing...")
			return false, nil
		}
		return false, err
	}

	pods := corev1.PodList{}
	cleanupListOptions := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{"app": dsCleanupName}),
		Namespace:     n.cfg.InstallNamespace,
	}
	if err := n.r.List(ctx, &pods, cleanupListOptions); err != nil {
		if err = n.r.GetK8sReader().List(ctx, &pods, cleanupListOptions); err != nil {
			return false, err
		}
	}

	nodeCount := daemonset.Status.DesiredNumberScheduled
	if nodeCount == 0 || len(pods.Items) == 0 {
		n.r.GetLog().Info("Waiting for cleanup pods to be scheduled...")
		return false, nil
	}

	var runningCount int32
	var crashloopingPodNodes []string
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			runningCount++
		}
		if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodPending {
			for _, status := range pod.Status.ContainerStatuses {
				if status.State.Waiting != nil && status.State.Waiting.Reason == "CrashLoopBackOff" {
					crashloopingPodNodes = append(crashloopingPodNodes, pod.Spec.NodeName)
				}
			}
			for _, status := range pod.Status.InitContainerStatuses {
				if status.State.Waiting != nil && status.State.Waiting.Reason == "CrashLoopBackOff" {
					crashloopingPodNodes = append(crashloopingPodNodes, pod.Spec.NodeName)
				}
			}
		}
	}

	if len(crashloopingPodNodes) > 0 {
		slices.Sort(crashloopingPodNodes)
		crashloopingPodNodes = slices.Compact(crashloopingPodNodes)
		n.r.GetLog().Info(fmt.Sprintf("Some cleanup pods are in CrashLoopBackOff on nodes: %v", crashloopingPodNodes))
	}

	crashloopingCount := int32(len(crashloopingPodNodes))
	n.r.GetLog().Info(fmt.Sprintf("Cleanup progress: %d/%d pods running, %d crashlooping", runningCount, nodeCount, crashloopingCount))

	if runningCount+crashloopingCount < nodeCount {
		return false, nil
	}

	n.r.GetLog().Info("All cleanup pods completed, deleting cleanup DaemonSet")
	if err := components.DeleteOwned(ctx, n.r, n.cfg.Owner, n.cleanupDaemonSetRef()); err != nil {
		return false, err
	}

	return true, nil
}
