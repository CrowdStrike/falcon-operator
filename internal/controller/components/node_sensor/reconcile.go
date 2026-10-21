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
// The FalconFinalizer is managed by the FalconClusterGuard controller, which calls Cleanup on CR deletion.
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
	if err := n.reconcileProxyService(ctx); err != nil {
		return ctrl.Result{}, err
	}
	if err := n.reconcileDaemonSet(ctx); err != nil {
		return ctrl.Result{}, err
	}
	if err := n.reconcileCleanupServiceAccount(ctx); err != nil {
		return ctrl.Result{}, err
	}

	// A DaemonSet that is not ready yet is reported through the condition rather than a requeue, so the
	// remaining components still reconcile; the FalconClusterGuard controller requeues until it is ready.
	ds := n.sensorDaemonSetRef()
	status, err := k8sutils.DaemonSetStatus(ctx, n.r, client.ObjectKeyFromObject(ds))
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, k8sutils.ConditionsSet(n.r, ctx, n.cfg.Request, n.r.GetLog(), n.cfg.Owner, n.cfg.Status,
		status.Condition(falconv1alpha1.ConditionNodeSensorReady, "Node sensor DaemonSet is ready", n.cfg.Owner.GetGeneration()))
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

// cleanupTimeout bounds how long node cleanup waits for cleanup pods that neither finish nor report a start
// failure, for example when the DaemonSet controller cannot create them. Deletion then proceeds without them.
const cleanupTimeout = 5 * time.Minute

// Cleanup removes the node sensor when the component is disabled or the FalconClusterGuard is being deleted.
// If the sensor DaemonSet (or an in-progress cleanup DaemonSet) exists, node cleanup runs first and
// Cleanup requeues until it finishes. It is a no-op once the resources are gone.
func (n *NodeSensor) Cleanup(ctx context.Context) (ctrl.Result, error) {
	sensorDS := n.sensorDaemonSetRef()
	sensorFound, err := components.GetOwned(ctx, n.r, n.cfg.Owner, sensorDS)
	if err != nil {
		return ctrl.Result{}, err
	}
	cleanupDS := n.cleanupDaemonSetRef()
	cleanupFound, err := components.GetOwned(ctx, n.r, n.cfg.Owner, cleanupDS)
	if err != nil {
		return ctrl.Result{}, err
	}
	if sensorFound || cleanupFound {
		existing := cleanupDS
		if sensorFound {
			existing = sensorDS
		}
		if result, err := n.runNodeCleanup(ctx, existing); err != nil || result.RequeueAfter > 0 {
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

// runNodeCleanup deletes the sensor DaemonSet and runs node cleanup, requeueing until every cleanup pod has finished
// or failed to start. The cleanup DaemonSet takes its image, pull policy and pull secrets from existing, the sensor
// DaemonSet or, once that is gone, the cleanup DaemonSet. Cleanup then uses the image already on the nodes, and does
// not depend on resolving the image from the FalconClusterGuard spec, which can fail while it is being deleted.
func (n *NodeSensor) runNodeCleanup(ctx context.Context, existing *appsv1.DaemonSet) (ctrl.Result, error) {
	if n.skipNodeCleanup() {
		n.r.GetLog().Info("Skipping node cleanup because it is disabled", "disableCleanup", true)
		return ctrl.Result{}, components.DeleteOwned(ctx, n.r, n.cfg.Owner, n.sensorDaemonSetRef(), n.cleanupDaemonSetRef())
	}
	if err := n.useImageOf(ctx, existing); err != nil {
		return ctrl.Result{}, err
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

// useImageOf sets the image, pull policy and pull secrets in the config from the pods of ds. The sensor DaemonSet
// gets its pull secrets through its ServiceAccount rather than the pod spec, so those are read from there.
func (n *NodeSensor) useImageOf(ctx context.Context, ds *appsv1.DaemonSet) error {
	spec := ds.Spec.Template.Spec
	if len(spec.Containers) == 0 {
		return nil
	}
	n.cfg.Image = spec.Containers[0].Image
	n.cfg.ImagePullPolicy = spec.Containers[0].ImagePullPolicy
	n.cfg.ImagePullSecrets = spec.ImagePullSecrets
	if len(spec.ImagePullSecrets) > 0 || spec.ServiceAccountName == "" {
		return nil
	}
	sa := &corev1.ServiceAccount{}
	if err := n.r.GetK8sReader().Get(ctx, types.NamespacedName{Name: spec.ServiceAccountName, Namespace: ds.Namespace}, sa); err != nil {
		return client.IgnoreNotFound(err)
	}
	n.cfg.ImagePullSecrets = sa.ImagePullSecrets
	return nil
}

// finalize deletes the sensor DaemonSet whatever its state, then runs the cleanup DaemonSet. A cleanup pod counts as
// finished once it is running (its init container has removed /opt/CrowdStrike) or once it fails to start, for
// example with ErrImageNeverPull, so a cleanup that cannot run never blocks deletion. Safe to call on every
// reconcile — delete and DaemonSet creation are idempotent.
func (n *NodeSensor) finalize(ctx context.Context) (bool, error) {
	log := n.r.GetLog()
	dsCleanupName := n.prefix() + "-sensor-cleanup"

	// The ServiceAccount is created first so that, once the sensor DaemonSet is deleted, only creating the cleanup
	// DaemonSet can fail before it exists to carry the image into later passes.
	if err := n.reconcileCleanupServiceAccount(ctx); err != nil {
		return false, err
	}
	if err := components.DeleteOwned(ctx, n.r, n.cfg.Owner, n.sensorDaemonSetRef()); err != nil {
		return false, err
	}
	if err := n.reconcileCleanupDaemonSet(ctx); err != nil {
		return false, err
	}

	daemonset := &appsv1.DaemonSet{}
	if err := pkgcommon.GetNamespacedObject(ctx, n.r, n.r.GetK8sReader(),
		types.NamespacedName{Name: dsCleanupName, Namespace: n.cfg.InstallNamespace}, daemonset); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Cleanup DaemonSet not found yet, requeueing...")
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

	var finished int32
	var failedNodes []string
	for _, pod := range pods.Items {
		switch pod.Status.Phase {
		case corev1.PodRunning, corev1.PodSucceeded:
			finished++
		case corev1.PodFailed:
			finished++
			failedNodes = append(failedNodes, fmt.Sprintf("%s: pod %s failed", pod.Spec.NodeName, pod.Name))
		default:
			if failure := k8sutils.PodStartFailure(&pod); failure != "" {
				finished++
				failedNodes = append(failedNodes, fmt.Sprintf("%s: pod %s %s", pod.Spec.NodeName, pod.Name, failure))
			}
		}
	}
	slices.Sort(failedNodes)

	nodeCount := daemonset.Status.DesiredNumberScheduled
	if nodeCount == 0 || finished < nodeCount {
		if daemonset.CreationTimestamp.IsZero() || time.Since(daemonset.CreationTimestamp.Time) < cleanupTimeout {
			log.Info("Waiting for cleanup pods", "finished", finished, "nodes", nodeCount, "failed", failedNodes)
			return false, nil
		}
		log.Info("Timed out waiting for cleanup pods, proceeding with deletion", "timeout", cleanupTimeout,
			"finished", finished, "nodes", nodeCount)
	}
	if len(failedNodes) > 0 {
		log.Info("Some cleanup pods failed to start, so /opt/CrowdStrike may not have been removed from their nodes. "+
			"See the troubleshooting section of the node sensor documentation for more information.", "failed", failedNodes)
	}

	log.Info("Node cleanup finished, deleting cleanup DaemonSet")
	if err := components.DeleteOwned(ctx, n.r, n.cfg.Owner, n.cleanupDaemonSetRef()); err != nil {
		return false, err
	}

	return true, nil
}
