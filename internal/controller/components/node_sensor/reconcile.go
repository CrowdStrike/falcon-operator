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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

func (n *NodeSensor) Reconcile(ctx context.Context) (ctrl.Result, error) {
	log := n.r.GetLog()

	if n.cfg.Owner.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(n.cfg.Owner, pkgcommon.FalconFinalizer) {
			log.Info("FalconClusterGuard is being deleted, running finalization logic")
			if n.cfg.NodeSensor.NodeCleanup != nil && *n.cfg.NodeSensor.NodeCleanup {
				log.Info("Skipping node cleanup because it is disabled", "disableCleanup", *n.cfg.NodeSensor.NodeCleanup)
			} else {
				done, err := n.finalize(ctx)
				if err != nil {
					return ctrl.Result{}, err
				}
				if !done {
					return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
				}
			}
			controllerutil.RemoveFinalizer(n.cfg.Owner, pkgcommon.FalconFinalizer)
			if err := n.r.Update(ctx, n.cfg.Owner); err != nil {
				return ctrl.Result{}, err
			}
			log.Info("Successfully finalized FalconClusterGuard")
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(n.cfg.Owner, pkgcommon.FalconFinalizer) {
		controllerutil.AddFinalizer(n.cfg.Owner, pkgcommon.FalconFinalizer)
		if err := n.r.Update(ctx, n.cfg.Owner); err != nil {
			log.Error(err, "Unable to add finalizer to FalconClusterGuard")
			return ctrl.Result{}, err
		}
		log.Info("Added finalizer to FalconClusterGuard")
	}

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

	meta.SetStatusCondition(&n.cfg.Status.Conditions, metav1.Condition{
		Type:               falconv1alpha1.ConditionNodeSensorReady,
		Status:             metav1.ConditionTrue,
		Reason:             falconv1alpha1.ReasonInstallSucceeded,
		Message:            "Node sensor DaemonSet is ready",
		ObservedGeneration: n.cfg.Owner.GetGeneration(),
	})

	return ctrl.Result{}, nil
}

// Safe to call on every reconcile — delete and DaemonSet creation are idempotent.
func (n *NodeSensor) finalize(ctx context.Context) (bool, error) {
	dsCleanupName := n.prefix() + "-sensor-cleanup"

	n.r.GetLog().Info("Deleting main sensor DaemonSet")
	if err := n.r.Delete(ctx, &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: n.prefix() + "-sensor", Namespace: n.cfg.InstallNamespace},
	}); err != nil && !apierrors.IsNotFound(err) {
		n.r.GetLog().Error(err, "Failed to delete main sensor DaemonSet")
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
	if err := n.r.Delete(ctx, &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: dsCleanupName, Namespace: n.cfg.InstallNamespace},
	}); err != nil && !apierrors.IsNotFound(err) {
		return false, err
	}

	return true, nil
}
