package falcon

import (
	"context"
	"fmt"
	"reflect"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/assets/node"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/k8s_utils"
	"github.com/crowdstrike/falcon-operator/pkg/registry/pulltoken"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	err = r.handleCrowdStrikeSecrets(ctx, nodesensor, logger)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Check if the daemonset already exists, if not create a new one
	daemonset := &appsv1.DaemonSet{}

	err = r.Get(ctx, types.NamespacedName{Name: nodesensor.Name, Namespace: nodesensor.Namespace}, daemonset)
	if err != nil && errors.IsNotFound(err) {
		// Define a new daemonset
		ds, err := r.nodeSensorDaemonset(ctx, nodesensor.Name, nodesensor, logger)
		if err != nil {
			return ctrl.Result{}, err
		}

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
		imgUpdate, err := updateDaemonSetImages(ctx, dsUpdate, nodesensor, logger)
		if err != nil {
			return ctrl.Result{}, err
		}
		tolsUpdate := updateDaemonSetTolerations(dsUpdate, nodesensor, logger)
		containerVolUpdate, err := updateDaemonSetContainerVolumes(ctx, dsUpdate, nodesensor, logger)
		if err != nil {
			return ctrl.Result{}, err
		}
		volumeUpdates, err := updateDaemonSetVolumes(ctx, dsUpdate, nodesensor, logger)
		if err != nil {
			return ctrl.Result{}, err
		}

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
		configmap, err := r.nodeSensorConfigmap(cmName, nodesensor)
		if err != nil {
			logger.Error(err, "Failed to format new Configmap", "Configmap.Namespace", nodesensor.Namespace, "Configmap.Name", cmName)
			return nil, updated, err
		}
		if err := r.Create(ctx, configmap); err != nil {
			logger.Error(err, "Failed to create new Configmap", "Configmap.Namespace", nodesensor.Namespace, "Configmap.Name", cmName)
			return nil, updated, err
		}

		logger.Info("Creating FalconNodeSensor Configmap")
		return nil, updated, nil
	} else if err != nil {
		logger.Error(err, "error getting Configmap")
		return nil, updated, err
	}

	configmap, err := r.nodeSensorConfigmap(cmName, nodesensor)
	if err != nil {
		logger.Error(err, "Failed to format existing Configmap", "Configmap.Namespace", nodesensor.Namespace, "Configmap.Name", cmName)
		return nil, updated, err
	}
	if !reflect.DeepEqual(confCm.Data, configmap.Data) {
		err = r.Update(ctx, configmap)
		if err != nil {
			logger.Error(err, "Failed to update Configmap", "Configmap.Namespace", nodesensor.Namespace, "Configmap.Name", cmName)
			return nil, updated, err
		}

		updated = true
	}

	return confCm, updated, nil
}

const (
	SECRET_NAME        = "crowdstrike-falcon-pull-secret"
	SECRET_LABEL_VALUE = "crowdstrike"
)

// handleCrowdStrikeSecrets creates and updates the image pull secrets for the nodesensor
func (r *FalconNodeSensorReconciler) handleCrowdStrikeSecrets(ctx context.Context, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) error {
	if nodesensor.Spec.Node.Image != "" {
		return nil
	}
	if nodesensor.Spec.FalconAPI == nil {
		return fmt.Errorf("Missing falcon_api configuration")
	}

	secret := corev1.Secret{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: SECRET_NAME, Namespace: nodesensor.Namespace}, &secret)
	if err == nil || !errors.IsNotFound(err) {
		return err
	}
	pulltoken, err := pulltoken.CrowdStrike(ctx, nodesensor.Spec.FalconAPI.ApiConfig())
	if err != nil {
		return err
	}

	secret = corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      SECRET_NAME,
			Namespace: nodesensor.Namespace,
			Labels: map[string]string{
				common.FalconProviderKey: SECRET_LABEL_VALUE,
			},
		},
		Data: map[string][]byte{
			".dockerconfigjson": pulltoken,
		},
		Type: corev1.SecretTypeDockerConfigJson,
	}
	err = ctrl.SetControllerReference(nodesensor, &secret, r.Scheme)
	if err != nil {
		logger.Error(err, "Unable to assign Controller Reference to the Pull Secret")
	}
	err = r.Client.Create(ctx, &secret)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			logger.Error(err, "Failed to create new Pull Secret", "Secret.Namespace", nodesensor.Namespace, "Secret.Name", SECRET_NAME)
			return err
		}
	} else {
		logger.Info("Created a new Pull Secret", "Secret.Namespace", nodesensor.Namespace, "Secret.Name", SECRET_NAME)
	}
	return nil
}

func (r *FalconNodeSensorReconciler) nodeSensorConfigmap(name string, nodesensor *falconv1alpha1.FalconNodeSensor) (*corev1.ConfigMap, error) {
	cm := node.DaemonsetConfigMap(name, nodesensor.Namespace, &nodesensor.Spec.Falcon)

	err := controllerutil.SetControllerReference(nodesensor, cm, r.Scheme)
	if err != nil {
		return nil, err
	}
	return cm, nil
}

func (r *FalconNodeSensorReconciler) nodeSensorDaemonset(ctx context.Context, name string, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) (*appsv1.DaemonSet, error) {
	ds, err := node.Daemonset(ctx, name, nodesensor)
	if err != nil {
		return nil, err
	}

	// NOTE: calling SetControllerReference, and setting owner references in
	// general, is important as it allows deleted objects to be garbage collected.
	err = controllerutil.SetControllerReference(nodesensor, ds, r.Scheme)
	if err != nil {
		logger.Error(err, "unable to set controller reference")
	}

	return ds, nil
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
func updateDaemonSetContainerVolumes(ctx context.Context, ds *appsv1.DaemonSet, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) (bool, error) {
	origDS, err := node.Daemonset(ctx, ds.Name, nodesensor)
	if err != nil {
		return true, err
	}

	containerVolumeMounts := &ds.Spec.Template.Spec.Containers[0].VolumeMounts
	containerVolumeMountsUpdates := !reflect.DeepEqual(*containerVolumeMounts, origDS.Spec.Template.Spec.Containers[0].VolumeMounts)
	if containerVolumeMountsUpdates {
		logger.Info("Updating FalconNodeSensor DaemonSet container volumeMounts")
		*containerVolumeMounts = origDS.Spec.Template.Spec.Containers[0].VolumeMounts
	}

	return containerVolumeMountsUpdates, nil
}

// If an update is needed, this will update the volumes from the given DaemonSet
func updateDaemonSetVolumes(ctx context.Context, ds *appsv1.DaemonSet, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) (bool, error) {
	origDS, err := node.Daemonset(ctx, ds.Name, nodesensor)
	if err != nil {
		return true, err
	}
	volumeMounts := &ds.Spec.Template.Spec.Volumes
	volumeMountsUpdates := !reflect.DeepEqual(*volumeMounts, origDS.Spec.Template.Spec.Volumes)
	if volumeMountsUpdates {
		logger.Info("Updating FalconNodeSensor DaemonSet volumeMounts")
		*volumeMounts = origDS.Spec.Template.Spec.Volumes
	}

	return volumeMountsUpdates, nil
}

// If an update is needed, this will update the InitContainer image reference from the given DaemonSet
func updateDaemonSetImages(ctx context.Context, ds *appsv1.DaemonSet, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) (bool, error) {
	initImage := &ds.Spec.Template.Spec.InitContainers[0].Image
	origImg, err := common.GetFalconImage(ctx, nodesensor)
	if err != nil {
		return false, err
	}
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

	return imgUpdate, nil
}
