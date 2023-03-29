package falcon

import (
	"context"
	"reflect"
	"strings"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	common_assets "github.com/crowdstrike/falcon-operator/pkg/assets"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/k8s_utils"
	"github.com/crowdstrike/falcon-operator/pkg/node"
	"github.com/crowdstrike/falcon-operator/pkg/node/assets"
	"github.com/crowdstrike/falcon-operator/version"
	"github.com/go-logr/logr"
	securityv1 "github.com/openshift/api/security/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
		Owns(&corev1.Secret{}).
		Complete(r)
}

// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;delete;deletecollection

//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconnodesensors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconnodesensors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconnodesensors/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;create;update
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
//+kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=get;list;watch;create;use
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterroles,verbs=get;list;watch;create
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterrolebindings,verbs=get;list;watch;create

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

	dsCondition := meta.FindStatusCondition(nodesensor.Status.Conditions, falconv1alpha1.ConditionSuccess)
	if dsCondition == nil {
		err = r.conditionsUpdate(falconv1alpha1.ConditionPending,
			metav1.ConditionFalse,
			falconv1alpha1.ReasonReqNotMet,
			"FalconNodeSensor progressing",
			ctx, nodesensor, logger)
		if err != nil {
			return ctrl.Result{Requeue: true}, err
		}
	}

	created, err := r.handleNamespace(ctx, nodesensor, logger)
	if err != nil {
		return ctrl.Result{}, err
	}
	if created {
		return ctrl.Result{Requeue: true}, nil
	}

	serviceAccount, err := r.isServiceAccountRequired(ctx, logger)
	if err != nil {
		return ctrl.Result{}, err
	}
	if serviceAccount != "" {
		created, err = r.handlePermissions(ctx, nodesensor, logger)
		if err != nil {
			return ctrl.Result{}, err
		}
		if created {
			return ctrl.Result{Requeue: true}, nil
		}
	}

	if nodesensor.Spec.Node.ServiceAccount.Annotations != nil {
		err = r.handleSAAnnotations(ctx, nodesensor, logger)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	config, err := node.NewConfigCache(ctx, logger, nodesensor)
	if err != nil {
		return ctrl.Result{}, err
	}

	sensorConf, updated, err := r.handleConfigMaps(ctx, config, nodesensor, logger)
	if err != nil {
		err = r.conditionsUpdate(falconv1alpha1.ConditionFailed,
			metav1.ConditionFalse,
			falconv1alpha1.ReasonInstallFailed,
			"FalconNodeSensor ConfigMap failed to be installed",
			ctx, nodesensor, logger)
		if err != nil {
			return ctrl.Result{}, err
		}

		logger.Error(err, "error handling configmap")
		return ctrl.Result{}, err
	}
	if sensorConf == nil {
		err = r.conditionsUpdate(falconv1alpha1.ConditionConfigMapReady,
			metav1.ConditionTrue,
			falconv1alpha1.ReasonInstallSucceeded,
			"FalconNodeSensor ConfigMap has been successfully created",
			ctx, nodesensor, logger)
		if err != nil {
			return ctrl.Result{}, err
		}

		// this just got created, so re-queue.
		logger.Info("Configmap was just created. Re-queuing")
		return ctrl.Result{Requeue: true}, nil
	}
	if updated {
		err = r.conditionsUpdate(falconv1alpha1.ConditionConfigMapReady,
			metav1.ConditionTrue,
			falconv1alpha1.ReasonUpdateSucceeded,
			"FalconNodeSensor ConfigMap has been successfully updated",

			ctx, nodesensor, logger)
		if err != nil {
			return ctrl.Result{}, err
		}

		logger.Info("Configmap was updated")
	}

	err = r.handleCrowdStrikeSecrets(ctx, config, nodesensor, logger)
	if err != nil {
		return ctrl.Result{}, err
	}

	image, err := config.GetImageURI(ctx, logger)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Check if the daemonset already exists, if not create a new one
	daemonset := &appsv1.DaemonSet{}

	err = r.Get(ctx, types.NamespacedName{Name: nodesensor.Name, Namespace: nodesensor.TargetNs()}, daemonset)
	if err != nil && errors.IsNotFound(err) {
		// Define a new daemonset
		ds := r.nodeSensorDaemonset(nodesensor.Name, image, serviceAccount, nodesensor, logger)

		err = r.Create(ctx, ds)
		if err != nil {
			err = r.conditionsUpdate(falconv1alpha1.ConditionFailed,
				metav1.ConditionFalse,
				falconv1alpha1.ReasonInstallFailed,
				"FalconNodeSensor DaemonSet failed to be installed",
				ctx, nodesensor, logger)
			logger.Error(err, "Failed to create new DaemonSet", "DaemonSet.Namespace", ds.Namespace, "DaemonSet.Name", ds.Name)
			return ctrl.Result{}, err
		}

		err = r.conditionsUpdate(falconv1alpha1.ConditionDaemonSetReady,
			metav1.ConditionTrue,
			falconv1alpha1.ReasonInstallSucceeded,
			"FalconNodeSensor DaemonSet has been successfully installed",
			ctx, nodesensor, logger)
		if err != nil {
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
		dsTarget := assets.Daemonset(dsUpdate.Name, image, serviceAccount, nodesensor)

		// Objects to check for updates to re-spin pods
		imgUpdate := updateDaemonSetImages(dsUpdate, image, nodesensor, logger)
		tolsUpdate := updateDaemonSetTolerations(dsUpdate, nodesensor, logger)
		containerVolUpdate := updateDaemonSetContainerVolumes(dsUpdate, dsTarget, logger)
		volumeUpdates := updateDaemonSetVolumes(dsUpdate, dsTarget, logger)

		// Update the daemonset and re-spin pods with changes
		if imgUpdate || tolsUpdate || containerVolUpdate || volumeUpdates || updated {
			err = r.Update(ctx, dsUpdate)
			if err != nil {
				err = r.conditionsUpdate(falconv1alpha1.ConditionDaemonSetReady,
					metav1.ConditionTrue,
					falconv1alpha1.ReasonUpdateFailed,
					"FalconNodeSensor DaemonSet update has failed",
					ctx, nodesensor, logger)
				logger.Error(err, "Failed to update DaemonSet", "DaemonSet.Namespace", dsUpdate.Namespace, "DaemonSet.Name", dsUpdate.Name)
				return ctrl.Result{}, err
			}

			err := k8s_utils.RestartDaemonSet(ctx, r.Client, dsUpdate)
			if err != nil {
				logger.Error(err, "Failed to restart pods after DaemonSet configuration changed.")
				return ctrl.Result{}, err
			}

			err = r.conditionsUpdate(falconv1alpha1.ConditionDaemonSetReady,
				metav1.ConditionTrue,
				falconv1alpha1.ReasonUpdateSucceeded,
				"FalconNodeSensor DaemonSet has been successfully updated",
				ctx, nodesensor, logger)
			if err != nil {
				return ctrl.Result{}, err
			}
			logger.Info("FalconNodeSensor DaemonSet configuration changed. Pods have been restarted.")
		}
	}

	if nodesensor.Status.Sensor == nil {
		nodesensor.Status.Sensor = &strings.Split(image, ":")[1]
		err = r.Status().Update(ctx, nodesensor)
		if err != nil {
			log.Error(err, "Failed to update FalconNodeSensor status for nodesensor.Status.Sensor")
			return ctrl.Result{}, err
		}
	}

	if nodesensor.Status.Version == "" {
		nodesensor.Status.Version = version.Get()
		err = r.Status().Update(ctx, nodesensor)
		if err != nil {
			log.Error(err, "Failed to update FalconNodeSensor status for nodesensor.Status.Version")
			return ctrl.Result{}, err
		}
	}

	err = r.conditionsUpdate(falconv1alpha1.ConditionSuccess,
		metav1.ConditionTrue,
		falconv1alpha1.ReasonInstallSucceeded,
		"FalconNodeSensor installation completed",
		ctx, nodesensor, logger)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	// Check if the FalconNodeSensor instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isDSMarkedToBeDeleted := nodesensor.GetDeletionTimestamp() != nil
	if isDSMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(nodesensor, common.FalconFinalizer) {
			logger.Info("Successfully finalized daemonset")
			// Allows the cleanup to be disabled by disableCleanup option
			if !*nodesensor.Spec.Node.NodeCleanup {
				// Run finalization logic for common.FalconFinalizer. If the
				// finalization logic fails, don't remove the finalizer so
				// that we can retry during the next reconciliation.
				if err := r.finalizeDaemonset(ctx, image, serviceAccount, nodesensor, logger); err != nil {
					return ctrl.Result{}, err
				}
			} else {
				logger.Info("Skipping cleanup because it is disabled", "disableCleanup", *nodesensor.Spec.Node.NodeCleanup)
			}

			// Remove common.FalconFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(nodesensor, common.FalconFinalizer)
			err := r.Update(ctx, nodesensor)
			if err != nil {
				return ctrl.Result{}, err
			}
			log.Info("Removing finalizer")

		}
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(nodesensor, common.FalconFinalizer) {
		controllerutil.AddFinalizer(nodesensor, common.FalconFinalizer)
		err = r.Update(ctx, nodesensor)
		if err != nil {
			logger.Error(err, "Unable to update finalizer")
			return ctrl.Result{}, err
		}
		log.Info("Adding finalizer")

	}

	return ctrl.Result{}, nil
}

// handleNamespace creates and updates the namespace
func (r *FalconNodeSensorReconciler) handleNamespace(ctx context.Context, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) (bool, error) {
	ns := corev1.Namespace{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: nodesensor.TargetNs()}, &ns)
	if err == nil || (err != nil && !errors.IsNotFound(err)) {
		return false, err
	}

	ns = corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: nodesensor.TargetNs(),
		},
	}
	err = ctrl.SetControllerReference(nodesensor, &ns, r.Scheme)
	if err != nil {
		logger.Error(err, "Unable to assign Controller Reference to the Namespace")
	}
	err = r.Client.Create(ctx, &ns)
	if err != nil && !errors.IsAlreadyExists(err) {
		logger.Error(err, "Failed to create new namespace", "Namespace.Name", nodesensor.TargetNs())
		return false, err
	}
	return true, nil
}

// handleConfigMaps creates and updates the node sensor configmap
func (r *FalconNodeSensorReconciler) handleConfigMaps(ctx context.Context, config *node.ConfigCache, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) (*corev1.ConfigMap, bool, error) {
	var updated bool
	cmName := nodesensor.Name + "-config"
	confCm := &corev1.ConfigMap{}

	err := r.Get(ctx, types.NamespacedName{Name: cmName, Namespace: nodesensor.TargetNs()}, confCm)
	if err != nil && errors.IsNotFound(err) {
		// does not exist, create
		configmap, err := r.nodeSensorConfigmap(cmName, config, nodesensor)
		if err != nil {
			logger.Error(err, "Failed to format new Configmap", "Configmap.Namespace", nodesensor.TargetNs(), "Configmap.Name", cmName)
			return nil, updated, err
		}
		if err := r.Create(ctx, configmap); err != nil {
			if errors.IsAlreadyExists(err) {
				// We have got NotFound error during the Get(), but then we have got AlreadyExists error from Create(). Client cache is invalid.
				err = r.Update(ctx, configmap)
				if err != nil {
					logger.Error(err, "Failed to update Configmap", "Configmap.Namespace", nodesensor.TargetNs(), "Configmap.Name", cmName)
				}
				return configmap, updated, nil
			} else {
				logger.Error(err, "Failed to create new Configmap", "Configmap.Namespace", nodesensor.TargetNs(), "Configmap.Name", cmName)
				return nil, updated, err

			}
		}

		logger.Info("Creating FalconNodeSensor Configmap")
		return nil, updated, nil
	} else if err != nil {
		logger.Error(err, "error getting Configmap")
		return nil, updated, err
	}

	configmap, err := r.nodeSensorConfigmap(cmName, config, nodesensor)
	if err != nil {
		logger.Error(err, "Failed to format existing Configmap", "Configmap.Namespace", nodesensor.TargetNs(), "Configmap.Name", cmName)
		return nil, updated, err
	}
	if !reflect.DeepEqual(confCm.Data, configmap.Data) {
		err = r.Update(ctx, configmap)
		if err != nil {
			logger.Error(err, "Failed to update Configmap", "Configmap.Namespace", nodesensor.TargetNs(), "Configmap.Name", cmName)
			return nil, updated, err
		}

		updated = true
	}

	return confCm, updated, nil
}

// handleCrowdStrikeSecrets creates and updates the image pull secrets for the nodesensor
func (r *FalconNodeSensorReconciler) handleCrowdStrikeSecrets(ctx context.Context, config *node.ConfigCache, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) error {
	if !config.UsingCrowdStrikeRegistry() {
		return nil
	}
	secret := corev1.Secret{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: common.FalconPullSecretName, Namespace: nodesensor.TargetNs()}, &secret)
	if err == nil || !errors.IsNotFound(err) {
		return err
	}

	pulltoken, err := config.GetPullToken(ctx)
	if err != nil {
		return err
	}

	secret = common_assets.PullSecret(nodesensor.TargetNs(), pulltoken)
	err = ctrl.SetControllerReference(nodesensor, &secret, r.Scheme)
	if err != nil {
		logger.Error(err, "Unable to assign Controller Reference to the Pull Secret")
	}
	err = r.Client.Create(ctx, &secret)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			logger.Error(err, "Failed to create new Pull Secret", "Secret.Namespace", nodesensor.TargetNs(), "Secret.Name", common.FalconPullSecretName)
			return err
		}
	} else {
		logger.Info("Created a new Pull Secret", "Secret.Namespace", nodesensor.TargetNs(), "Secret.Name", common.FalconPullSecretName)
	}
	return nil
}

func (r *FalconNodeSensorReconciler) nodeSensorConfigmap(name string, config *node.ConfigCache, nodesensor *falconv1alpha1.FalconNodeSensor) (*corev1.ConfigMap, error) {
	cm := assets.DaemonsetConfigMap(name, nodesensor.TargetNs(), config)

	err := controllerutil.SetControllerReference(nodesensor, cm, r.Scheme)
	if err != nil {
		return nil, err
	}
	return cm, nil
}

func (r *FalconNodeSensorReconciler) nodeSensorDaemonset(name, image, serviceAccount string, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) *appsv1.DaemonSet {
	ds := assets.Daemonset(name, image, serviceAccount, nodesensor)

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
func updateDaemonSetContainerVolumes(ds, origDS *appsv1.DaemonSet, logger logr.Logger) bool {
	containerVolumeMounts := &ds.Spec.Template.Spec.Containers[0].VolumeMounts
	containerVolumeMountsUpdates := !reflect.DeepEqual(*containerVolumeMounts, origDS.Spec.Template.Spec.Containers[0].VolumeMounts)
	if containerVolumeMountsUpdates {
		logger.Info("Updating FalconNodeSensor DaemonSet container volumeMounts")
		*containerVolumeMounts = origDS.Spec.Template.Spec.Containers[0].VolumeMounts
	}

	return containerVolumeMountsUpdates
}

// If an update is needed, this will update the volumes from the given DaemonSet
func updateDaemonSetVolumes(ds, origDS *appsv1.DaemonSet, logger logr.Logger) bool {
	volumeMounts := &ds.Spec.Template.Spec.Volumes
	volumeMountsUpdates := !reflect.DeepEqual(*volumeMounts, origDS.Spec.Template.Spec.Volumes)
	if volumeMountsUpdates {
		logger.Info("Updating FalconNodeSensor DaemonSet volumeMounts")
		*volumeMounts = origDS.Spec.Template.Spec.Volumes
	}

	return volumeMountsUpdates
}

// If an update is needed, this will update the InitContainer image reference from the given DaemonSet
func updateDaemonSetImages(ds *appsv1.DaemonSet, origImg string, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) bool {
	initImage := &ds.Spec.Template.Spec.InitContainers[0].Image
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

// handlePermissions creates and updates the service account, role and role binding
func (r *FalconNodeSensorReconciler) handlePermissions(ctx context.Context, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) (bool, error) {
	created, err := r.handleScc(ctx, nodesensor, logger)
	if created || err != nil {
		return created, err
	}
	created, err = r.handleServiceAccount(ctx, nodesensor, logger)
	if created || err != nil {
		return created, err
	}
	created, err = r.handleClusterRole(ctx, nodesensor, logger)
	if created || err != nil {
		return created, err
	}
	return r.handleClusterRoleBinding(ctx, nodesensor, logger)
}

func (r *FalconNodeSensorReconciler) isServiceAccountRequired(ctx context.Context, logger logr.Logger) (string, error) {
	scc := securityv1.SecurityContextConstraints{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: common.NodeSccName}, &scc)
	if err == nil || errors.IsNotFound(err) {
		return common.NodeServiceAccountName, nil
	}
	if meta.IsNoMatchError(err) {
		return "", nil
	}
	return common.NodeServiceAccountName, err
}

// handleScc creates and update SCC
func (r *FalconNodeSensorReconciler) handleScc(ctx context.Context, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) (bool, error) {
	scc := securityv1.SecurityContextConstraints{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: common.NodeSccName}, &scc)
	if err == nil || (err != nil && !errors.IsNotFound(err)) {
		return false, err
	}
	scc = securityv1.SecurityContextConstraints{
		TypeMeta: metav1.TypeMeta{
			APIVersion: securityv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: common.NodeClusterRoleBindingName,
		},
		AllowPrivilegedContainer: true,
		RunAsUser:                securityv1.RunAsUserStrategyOptions{Type: securityv1.RunAsUserStrategyRunAsAny},
		SELinuxContext:           securityv1.SELinuxContextStrategyOptions{Type: securityv1.SELinuxStrategyRunAsAny},
		FSGroup:                  securityv1.FSGroupStrategyOptions{Type: securityv1.FSGroupStrategyRunAsAny},
		SupplementalGroups:       securityv1.SupplementalGroupsStrategyOptions{Type: securityv1.SupplementalGroupsStrategyRunAsAny},
		AllowHostDirVolumePlugin: true,
		AllowHostIPC:             true,
		AllowHostNetwork:         true,
		AllowHostPID:             true,
		AllowHostPorts:           true,
		ReadOnlyRootFilesystem:   false,
		RequiredDropCapabilities: []corev1.Capability{},
		DefaultAddCapabilities:   []corev1.Capability{},
		AllowedCapabilities:      []corev1.Capability{},
		Groups:                   []string{},
		Volumes: []securityv1.FSType{
			securityv1.FSTypeConfigMap,
			securityv1.FSTypeDownwardAPI,
			securityv1.FSTypeEmptyDir,
			securityv1.FSTypePersistentVolumeClaim,
			securityv1.FSProjected,
			securityv1.FSTypeSecret,
		},
	}
	err = ctrl.SetControllerReference(nodesensor, &scc, r.Scheme)
	if err != nil {
		logger.Error(err, "Unable to assign Controller Reference to the ClusterRoleBinding")
	}
	logger.Info("Creating FalconNodeSensor ClusterRoleBinding")
	err = r.Client.Create(ctx, &scc)
	if err != nil && !errors.IsAlreadyExists(err) {
		logger.Error(err, "Failed to create new ClusterRoleBinding", "ClusteRoleBinding.Name", common.NodeClusterRoleBindingName)
		return false, err
	}
	return true, nil
}

// handleRoleBinding creates and updates RoleBinding
func (r *FalconNodeSensorReconciler) handleClusterRoleBinding(ctx context.Context, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) (bool, error) {
	binding := rbacv1.ClusterRoleBinding{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: common.NodeClusterRoleBindingName}, &binding)
	if err == nil || (err != nil && !errors.IsNotFound(err)) {
		return false, err
	}
	binding = rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: common.NodeClusterRoleBindingName,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     common.NodeClusterRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      common.NodeServiceAccountName,
				Namespace: nodesensor.TargetNs(),
			},
		},
	}
	err = ctrl.SetControllerReference(nodesensor, &binding, r.Scheme)
	if err != nil {
		logger.Error(err, "Unable to assign Controller Reference to the ClusterRoleBinding")
	}
	logger.Info("Creating FalconNodeSensor ClusterRoleBinding")
	err = r.Client.Create(ctx, &binding)
	if err != nil && !errors.IsAlreadyExists(err) {
		logger.Error(err, "Failed to create new ClusterRoleBinding", "ClusteRoleBinding.Name", common.NodeClusterRoleBindingName)
		return false, err
	}
	return true, nil

}

// handleClusterRole creates and updates the ClusterRole
func (r *FalconNodeSensorReconciler) handleClusterRole(ctx context.Context, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) (bool, error) {
	role := rbacv1.ClusterRole{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: common.NodeClusterRoleName}, &role)
	if err == nil || (err != nil && !errors.IsNotFound(err)) {
		return false, err
	}
	role = rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nodesensor.TargetNs(),
			Name:      common.NodeClusterRoleName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:         []string{"use"},
				Resources:     []string{"securitycontextconstraints"},
				ResourceNames: []string{common.NodeSccName},
				APIGroups:     []string{"security.openshift.io"},
			},
		},
	}
	err = ctrl.SetControllerReference(nodesensor, &role, r.Scheme)
	if err != nil {
		logger.Error(err, "Unable to assign Controller Reference to the Role")
	}
	logger.Info("Creating FalconNodeSensor ClusterRole")
	err = r.Client.Create(ctx, &role)
	if err != nil && !errors.IsAlreadyExists(err) {
		logger.Error(err, "Failed to create new ClusterRole", "ClusterRole.Name", common.NodeClusterRoleName)
		return false, err
	}
	return true, nil
}

// handleServiceAccount creates and updates the service account and grants necessary permissions to it
func (r *FalconNodeSensorReconciler) handleServiceAccount(ctx context.Context, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) (bool, error) {
	sa := corev1.ServiceAccount{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: common.NodeServiceAccountName, Namespace: nodesensor.TargetNs()}, &sa)
	if err == nil || (err != nil && !errors.IsNotFound(err)) {
		return false, err
	}
	sa = corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nodesensor.TargetNs(),
			Name:      common.NodeServiceAccountName,
		},
	}
	err = ctrl.SetControllerReference(nodesensor, &sa, r.Scheme)
	if err != nil {
		logger.Error(err, "Unable to assign Controller Reference to the ServiceAccount")
	}
	logger.Info("Creating FalconNodeSensor ServiceAccount")
	err = r.Client.Create(ctx, &sa)
	if err != nil && !errors.IsAlreadyExists(err) {
		logger.Error(err, "Failed to create new ServiceAccount", "Namespace.Name", nodesensor.TargetNs())
		return false, err
	}
	return true, nil
}

// handleServiceAccount creates and updates the service account and grants necessary permissions to it
func (r *FalconNodeSensorReconciler) handleSAAnnotations(ctx context.Context, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) error {
	sa := corev1.ServiceAccount{}
	saAnnotations := nodesensor.Spec.Node.ServiceAccount.Annotations

	err := r.Get(ctx, types.NamespacedName{Name: common.NodeServiceAccountName, Namespace: nodesensor.TargetNs()}, &sa)
	if err != nil && errors.IsNotFound(err) {
		logger.Error(err, "Could not get FalconNodeSensor ServiceAccount")
		return err
	}

	// If there are no existing annotations, go ahead and create a map
	if sa.Annotations == nil {
		sa.Annotations = make(map[string]string)
	}

	// Add the CR configured annotations to the service account
	for key, value := range saAnnotations {
		sa.Annotations[key] = value
	}

	err = r.Update(ctx, &sa)
	if err != nil {
		logger.Error(err, "Failed to update ServiceAccount Annotations", "ServiceAccount.Namespace", nodesensor.TargetNs(), "Annotations", saAnnotations)
		return err
	}
	logger.Info("Updating FalconNodeSensor ServiceAccount Annotations", "Annotations", saAnnotations)

	return nil
}

// statusUpdate updates the FalconNodeSensor CR conditions
func (r *FalconNodeSensorReconciler) conditionsUpdate(condType string, status metav1.ConditionStatus, reason string, message string, ctx context.Context, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) error {
	if !meta.IsStatusConditionPresentAndEqual(nodesensor.Status.Conditions, condType, status) {
		meta.SetStatusCondition(&nodesensor.Status.Conditions, metav1.Condition{
			Status:             status,
			Reason:             reason,
			Message:            message,
			Type:               condType,
			ObservedGeneration: nodesensor.GetGeneration(),
		})

		if err := r.Status().Update(ctx, nodesensor); err != nil {
			logger.Error(err, "Failed to update FalconNodeSensor status", "Failed to update the Condition at Reasoning", reason)
			return err
		}
	}

	return nil
}

// finalizeDaemonset deletes the Daemonset running the Falcon Sensor and then runs a Daemonset to cleanup the /opt/CrowdStrike directory
func (r *FalconNodeSensorReconciler) finalizeDaemonset(ctx context.Context, image string, serviceAccount string, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) error {
	dsCleanupName := nodesensor.Name + "-cleanup"
	daemonset := &appsv1.DaemonSet{}
	pods := corev1.PodList{}
	dsList := &appsv1.DaemonSetList{}
	var nodeCount int32 = 0

	// Get a list of DS and return the DS within the correct NS
	if err := r.List(ctx, dsList, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{common.FalconComponentKey: common.FalconKernelSensor}),
		Namespace:     nodesensor.TargetNs(),
	}); err != nil {
		return err
	}

	// Delete the Daemonset containing the sensor
	if err := r.Delete(ctx,
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: nodesensor.Name, Namespace: nodesensor.TargetNs(),
			},
		}); err != nil && !errors.IsNotFound(err) {
		logger.Error(err, "Failed to cleanup Falcon sensor DaemonSet pods")
		return err
	}

	// Check if the cleanup DS is created. If not, create it.
	err := r.Get(ctx, types.NamespacedName{Name: dsCleanupName, Namespace: nodesensor.TargetNs()}, daemonset)
	if err != nil && errors.IsNotFound(err) {
		// Define a new DS for cleanup
		ds := assets.RemoveNodeDirDaemonset(dsCleanupName, image, serviceAccount, nodesensor)

		// Create the cleanup DS
		err = r.Create(ctx, ds)
		if err != nil {
			logger.Error(err, "Failed to delete node directory with cleanup DaemonSet", "Path", common.FalconHostInstallDir)
			return err
		}

		// Start inifite loop to check that all pods have either completed or are running in the DS
		for {
			// List all pods with the "cleanup" label in the appropriate NS
			if err := r.List(ctx, &pods, &client.ListOptions{
				LabelSelector: labels.SelectorFromSet(labels.Set{common.FalconInstanceNameKey: "cleanup"}),
				Namespace:     nodesensor.TargetNs(),
			}); err != nil {
				return err
			}
			// Reset completedCount each loop, to ensure we don't count the same node(s) multiple times
			var completedCount int32 = 0
			// Reset the nodeCount to the desired number of pods to be scheduled for cleanup each loop, in case the cluster has scaled down
			for _, dSet := range dsList.Items {
				nodeCount = dSet.Status.DesiredNumberScheduled
				logger.Info("Setting DaemonSet node count", "Number of nodes", nodeCount)
			}

			// When the pods have a status of completed or running, increment the count.
			// The reason running is an acceptable value is because the pods should be running the sleep command and have already cleaned up /opt/CrowdStrike
			for _, pod := range pods.Items {
				if pod.Status.Phase == "Completed" || pod.Status.Phase == "Running" || pod.Status.Phase == "CrashLoopBackOff" {
					completedCount++
				}
			}

			// Break out of the infinite loop for cleanup when the completed or running DS count reaches the desired node count
			if completedCount == nodeCount {
				logger.Info("Clean up pods should be done. Continuing deleting.")
				break
			} else {
				logger.Info("Waiting for cleanup pods to complete. Retrying....", "Number of nodes still processing task", nodeCount-completedCount)
			}
			err = r.Get(ctx, types.NamespacedName{Name: dsCleanupName, Namespace: nodesensor.TargetNs()}, daemonset)
			if err != nil && errors.IsNotFound(err) {
				logger.Info("Clean-up daemonset has been removed")
				break
			}
		}

		// The cleanup DS should be completed so delete the cleanup DS
		if err := r.Delete(ctx,
			&appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: dsCleanupName, Namespace: nodesensor.TargetNs(),
				},
			}); err != nil && !errors.IsNotFound(err) {
			logger.Error(err, "Failed to cleanup Falcon sensor DaemonSet pods")
			return err
		}

		// If we have gotten here, the cleanup should be successful
		logger.Info("Successfully deleted node directory", "Path", common.FalconDataDir)
	} else if err != nil {
		logger.Error(err, "error getting the cleanup DaemonSet")
		return err
	}

	logger.Info("Successfully finalized daemonset")
	return nil
}
