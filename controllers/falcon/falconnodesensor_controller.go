package falcon

import (
	"context"
	"reflect"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/assets/node"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/k8s_utils"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	clog "sigs.k8s.io/controller-runtime/pkg/log"
)

// FalconNodeSensorReconciler reconciles a FalconNodeSensor object
type FalconNodeSensorReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *FalconNodeSensorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&falconv1alpha1.FalconNodeSensor{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.DaemonSet{}).
		Complete(r)
}

//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconnodesensors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconnodesensors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconnodesensors/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,resourceNames=privileged,verbs=use
//+kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *FalconNodeSensorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := clog.FromContext(ctx)
	logger := log.WithValues("DaemonSet", req.NamespacedName)
	logger.Info("reconciling FalconNodeSensor")

	// Fetch the FalconNodeSensor instance.
	nodesensor := &falconv1alpha1.FalconNodeSensor{}

	err := r.Get(ctx, req.NamespacedName, nodesensor)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get FalconNodeSensor")
		return ctrl.Result{}, err
	}

	sensorConf, updated, err := r.handleConfigMaps(ctx, nodesensor, logger)
	if err != nil {
		logger.Error(err, "error handling configmap")
		return ctrl.Result{}, err
	}
	if sensorConf == nil {
		// this just got created, so re-queue.
		logger.Info("Configmap was just created. Re-queuing")
		return ctrl.Result{Requeue: true}, nil
	}
	if updated {
		logger.Info("Configmap was updated")
	}

	// Check if the daemonset already exists, if not create a new one
	daemonset := &appsv1.DaemonSet{}

	err = r.Get(ctx, types.NamespacedName{Name: nodesensor.Name, Namespace: nodesensor.Namespace}, daemonset)
	if err != nil && errors.IsNotFound(err) {
		// Define a new daemonset
		ds := r.nodeSensorDaemonset(nodesensor.Name, nodesensor, logger)

		err = r.Create(ctx, ds)
		if err != nil {
			logger.Error(err, "Failed to create new DaemonSet", "DaemonSet.Namespace", ds.Namespace, "DaemonSet.Name", ds.Name)
			return ctrl.Result{}, err
		}

		logger.Info("Created a new DaemonSet", "DaemonSet.Namespace", ds.Namespace, "DaemonSet.Name", ds.Name)
		// Daemonset created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil

	} else if err != nil {
		logger.Error(err, "error getting DaemonSet")
		return ctrl.Result{}, err
	} else {
		// Copy Daemonset for updates
		dsUpdate := daemonset.DeepCopy()

		// Objects to check for updates to re-spin pods
		imgUpdate := updateDaemonSetImages(dsUpdate, nodesensor, logger)
		tolsUpdate := updateDaemonSetTolerations(dsUpdate, nodesensor, logger)
		containerVolUpdate := updateDaemonSetContainerVolumes(dsUpdate, nodesensor, logger)
		volumeUpdates := updateDaemonSetVolumes(dsUpdate, nodesensor, logger)

		// Update the daemonset and re-spin pods with changes
		if imgUpdate || tolsUpdate || containerVolUpdate || volumeUpdates || updated {
			err = r.Update(ctx, dsUpdate)
			if err != nil {
				logger.Error(err, "Failed to update DaemonSet", "DaemonSet.Namespace", dsUpdate.Namespace, "DaemonSet.Name", dsUpdate.Name)
				return ctrl.Result{}, err
			}

			err := k8s_utils.RestartDeamonSet(ctx, r.Client, dsUpdate)
			if err != nil {
				logger.Error(err, "Failed to restart pods after DaemonSet configuration changed.")
				return ctrl.Result{}, err
			}
			logger.Info("FalconNodeSensor DaemonSet configuration changed. Pods have been restarted.")
		}
	}

	return ctrl.Result{}, nil
}

// handleConfigMaps creates and updates the node sensor configmap
func (r *FalconNodeSensorReconciler) handleConfigMaps(ctx context.Context, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) (*corev1.ConfigMap, bool, error) {
	var updated bool
	cmName := nodesensor.Name + "-config"
	confCm := &corev1.ConfigMap{}

	err := r.Get(ctx, types.NamespacedName{Name: cmName, Namespace: nodesensor.Namespace}, confCm)
	if err != nil && errors.IsNotFound(err) {
		// does not exist, create
		if err := r.Create(ctx, r.nodeSensorConfigmap(cmName, nodesensor, logger)); err != nil {
			logger.Error(err, "Failed to create new Configmap", "Configmap.Namespace", nodesensor.Namespace, "Configmap.Name", cmName)
			return nil, updated, err
		}

		logger.Info("Creating FalconNodeSensor Configmap")
		return nil, updated, nil
	} else if err != nil {
		logger.Error(err, "error getting Configmap")
		return nil, updated, err
	} else {
		err = r.Update(ctx, r.nodeSensorConfigmap(cmName, nodesensor, logger))
		if err != nil {
			logger.Error(err, "Failed to update Configmap", "Configmap.Namespace", nodesensor.Namespace, "Configmap.Name", cmName)
			return nil, updated, err
		}

		updated = true
	}

	return confCm, updated, nil
}

func (r *FalconNodeSensorReconciler) nodeSensorConfigmap(name string, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) *corev1.ConfigMap {
	cm, _ := node.DaemonsetConfigMap(name, nodesensor.Namespace, &nodesensor.Spec.Falcon)

	err := controllerutil.SetControllerReference(nodesensor, cm, r.Scheme)
	if err != nil {
		logger.Error(err, "unable to set controller reference")
	}
	return cm
}

func (r *FalconNodeSensorReconciler) nodeSensorDaemonset(name string, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) *appsv1.DaemonSet {
	ds := node.Daemonset(name, nodesensor)

	// NOTE: calling SetControllerReference, and setting owner references in
	// general, is important as it allows deleted objects to be garbage collected.
	err := controllerutil.SetControllerReference(nodesensor, ds, r.Scheme)
	if err != nil {
		logger.Error(err, "unable to set controller reference")
	}

	return ds
}

// If an update is needed, this will update the tolerations from the given DaemonSet
func updateDaemonSetTolerations(ds *appsv1.DaemonSet, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) bool {
	tolerations := &ds.Spec.Template.Spec.Tolerations
	origTolerations := nodesensor.Spec.Node.Tolerations
	tolerationsUpdate := !reflect.DeepEqual(*tolerations, origTolerations)
	if tolerationsUpdate {
		logger.Info("Updating FalconNodeSensor DaemonSet Tolerations")
		*tolerations = origTolerations
	}
	return tolerationsUpdate
}

// If an update is needed, this will update the containervolumes from the given DaemonSet
func updateDaemonSetContainerVolumes(ds *appsv1.DaemonSet, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) bool {
	origDS := node.Daemonset(ds.Name, nodesensor)
	containerVolumeMounts := &ds.Spec.Template.Spec.Containers[0].VolumeMounts
	containerVolumeMountsUpdates := !reflect.DeepEqual(*containerVolumeMounts, origDS.Spec.Template.Spec.Containers[0].VolumeMounts)
	if containerVolumeMountsUpdates {
		logger.Info("Updating FalconNodeSensor DaemonSet container volumeMounts")
		*containerVolumeMounts = origDS.Spec.Template.Spec.Containers[0].VolumeMounts
	}

	return containerVolumeMountsUpdates
}

// If an update is needed, this will update the volumes from the given DaemonSet
func updateDaemonSetVolumes(ds *appsv1.DaemonSet, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) bool {
	origDS := node.Daemonset(ds.Name, nodesensor)
	volumeMounts := &ds.Spec.Template.Spec.Volumes
	volumeMountsUpdates := !reflect.DeepEqual(*volumeMounts, origDS.Spec.Template.Spec.Volumes)
	if volumeMountsUpdates {
		logger.Info("Updating FalconNodeSensor DaemonSet volumeMounts")
		*volumeMounts = origDS.Spec.Template.Spec.Volumes
	}

	return volumeMountsUpdates
}

// If an update is needed, this will update the InitContainer image reference from the given DaemonSet
func updateDaemonSetImages(ds *appsv1.DaemonSet, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) bool {
	initImage := &ds.Spec.Template.Spec.InitContainers[0].Image
	origImg := common.GetFalconImage(nodesensor)
	imgUpdate := *initImage != origImg
	if imgUpdate {
		logger.Info("Updating FalconNodeSensor DaemonSet InitContainer image", "Original Image", origImg, "Current Image", initImage)
		*initImage = origImg
	}

	image := &ds.Spec.Template.Spec.Containers[0].Image
	imgUpdate = *image != origImg
	if imgUpdate {
		logger.Info("Updating FalconNodeSensor DaemonSet image", "Original Image", origImg, "Current Image", image)
		*image = origImg
	}

	return imgUpdate
}
