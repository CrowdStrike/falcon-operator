package falcon

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	"github.com/crowdstrike/falcon-operator/internal/controller/common/sensorversion"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/k8s_utils"
	"github.com/crowdstrike/falcon-operator/pkg/node"
	"github.com/crowdstrike/falcon-operator/version"
	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/proxy"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	clog "sigs.k8s.io/controller-runtime/pkg/log"
)

// FalconNodeSensorReconciler reconciles a FalconNodeSensor object
type FalconNodeSensorReconciler struct {
	client.Client
	Reader          client.Reader
	Log             logr.Logger
	Scheme          *runtime.Scheme
	reconcileObject func(client.Object)
	tracker         sensorversion.Tracker
}

// SetupWithManager sets up the controller with the Manager.
func (r *FalconNodeSensorReconciler) SetupWithManager(mgr ctrl.Manager, tracker sensorversion.Tracker) error {
	nodeSensorController, err := ctrl.NewControllerManagedBy(mgr).
		For(&falconv1alpha1.FalconNodeSensor{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&corev1.Secret{}).
		Build(r)
	if err != nil {
		return err
	}

	r.reconcileObject, err = k8sutils.NewReconcileTrigger(nodeSensorController)
	if err != nil {
		return err
	}

	r.tracker = tracker
	return nil
}

func (r *FalconNodeSensorReconciler) GetK8sClient() client.Client {
	return r.Client
}

func (r *FalconNodeSensorReconciler) GetK8sReader() client.Reader {
	return r.Reader
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
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterrolebindings,verbs=get;list;watch;create
//+kubebuilder:rbac:groups="security.openshift.io",resources=securitycontextconstraints,resourceNames=privileged,verbs=use
//+kubebuilder:rbac:groups="scheduling.k8s.io",resources=priorityclasses,verbs=get;list;watch;create;delete;update
//+kubebuilder:rbac:groups="",resources=pods;services;nodes;daemonsets;replicasets;deployments;jobs;ingresses;cronjobs;persistentvolumes,verbs=get;watch;list

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
			r.tracker.StopTracking(req.NamespacedName)

			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("FalconNodeSensor resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get FalconNodeSensor")
		return ctrl.Result{}, err
	}

	validate, err := k8sutils.CheckRunningPodLabels(r.Reader, ctx, nodesensor.Spec.InstallNamespace, common.CRLabels("daemonset", nodesensor.Name, common.FalconKernelSensor))
	if err != nil {
		return ctrl.Result{}, err
	}
	if !validate {
		err = r.conditionsUpdate(falconv1alpha1.ConditionFailed,
			metav1.ConditionFalse,
			falconv1alpha1.ReasonReqNotMet,
			"FalconNodeSensor must not be installed in a namespace with other workloads running. Please change the namespace in the CR configuration.",
			ctx, req.NamespacedName, nodesensor, logger)
		if err != nil {
			return ctrl.Result{}, err
		}
		logger.Error(nil, "FalconNodeSensor is attempting to install in a namespace with existing pods. Please update the CR configuration to a namespace that does not have workoads already running.")
		return ctrl.Result{}, nil
	}

	dsCondition := meta.FindStatusCondition(nodesensor.Status.Conditions, falconv1alpha1.ConditionSuccess)
	if dsCondition == nil {
		err = r.conditionsUpdate(falconv1alpha1.ConditionPending,
			metav1.ConditionFalse,
			falconv1alpha1.ReasonReqNotMet,
			"FalconNodeSensor progressing",
			ctx, req.NamespacedName, nodesensor, logger)
		if err != nil {
			return ctrl.Result{Requeue: true}, err
		}
	}

	if nodesensor.Status.Version != version.Get() {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			err := r.Get(ctx, req.NamespacedName, nodesensor)
			if err != nil {
				return err
			}

			nodesensor.Status.Version = version.Get()
			return r.Status().Update(ctx, nodesensor)
		})
		if err != nil {
			log.Error(err, "Failed to update FalconNodeSensor status for nodesensor.Status.Version")
			return ctrl.Result{}, err
		}
	}

	created, err := r.handleNamespace(ctx, nodesensor, logger)
	if err != nil {
		return ctrl.Result{}, err
	}
	if created {
		return ctrl.Result{Requeue: true}, nil
	}

	err = r.handlePriorityClass(ctx, nodesensor, logger)
	if err != nil {
		return ctrl.Result{}, err
	}

	serviceAccount := common.NodeServiceAccountName

	created, err = r.handlePermissions(ctx, nodesensor, logger)
	if err != nil {
		return ctrl.Result{}, err
	}
	if created {
		return ctrl.Result{Requeue: true}, nil
	}

	if nodesensor.Spec.Node.ServiceAccount.Annotations != nil {
		err = r.handleSAAnnotations(ctx, nodesensor, logger)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if shouldTrackSensorVersions(nodesensor) {
		apiConfig, apiConfigErr := nodesensor.Spec.FalconAPI.ApiConfigWithSecret(ctx, r.Reader, nodesensor.Spec.FalconSecret)
		if apiConfigErr != nil {
			return ctrl.Result{}, apiConfigErr
		}

		getSensorVersion := sensorversion.NewFalconCloudQuery(falcon.NodeSensor, apiConfig)
		r.tracker.Track(req.NamespacedName, getSensorVersion, r.reconcileObjectWithName, nodesensor.Spec.Node.Advanced.IsAutoUpdatingForced())
	} else {
		r.tracker.StopTracking(req.NamespacedName)
	}

	// Inject Falcon secrets before handling config map updates
	if nodesensor.Spec.FalconSecret.Enabled {
		if err = r.injectFalconSecretData(ctx, nodesensor, logger); err != nil {
			return ctrl.Result{}, err
		}
	}

	config, err := node.NewConfigCache(ctx, nodesensor)
	if err != nil {
		return ctrl.Result{}, err
	}

	sensorConf, updated, err := r.handleConfigMaps(ctx, config, nodesensor, logger)
	if err != nil {
		err = r.conditionsUpdate(falconv1alpha1.ConditionFailed,
			metav1.ConditionFalse,
			falconv1alpha1.ReasonInstallFailed,
			"FalconNodeSensor ConfigMap failed to be installed",
			ctx, req.NamespacedName, nodesensor, logger)
		if err != nil {
			return ctrl.Result{}, err
		}

		logger.Error(err, "error handling configmap")
		return ctrl.Result{}, nil
	}

	if sensorConf == nil {
		err = r.conditionsUpdate(falconv1alpha1.ConditionConfigMapReady,
			metav1.ConditionTrue,
			falconv1alpha1.ReasonInstallSucceeded,
			"FalconNodeSensor ConfigMap has been successfully created",
			ctx, req.NamespacedName, nodesensor, logger)
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

			ctx, req.NamespacedName, nodesensor, logger)
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

	err = common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: nodesensor.Name, Namespace: nodesensor.Spec.InstallNamespace}, daemonset)
	if err != nil && errors.IsNotFound(err) {
		ds := assets.Daemonset(nodesensor.Name, image, serviceAccount, nodesensor)

		err := controllerutil.SetControllerReference(nodesensor, ds, r.Scheme)
		if err != nil {
			logger.Error(err, "Unable to assign Controller Reference to the DaemonSet")
		}

		if len(proxy.ReadProxyVarsFromEnv()) > 0 {
			for i, container := range ds.Spec.Template.Spec.Containers {
				ds.Spec.Template.Spec.Containers[i].Env = append(container.Env, proxy.ReadProxyVarsFromEnv()...)
			}
		}

		_, err = r.updateDaemonSetTolerations(ctx, ds, nodesensor, logger)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.Create(ctx, ds)
		if err != nil {
			logger.Error(err, "Failed to create new DaemonSet")
			err = r.conditionsUpdate(falconv1alpha1.ConditionFailed,
				metav1.ConditionFalse,
				falconv1alpha1.ReasonInstallFailed,
				"FalconNodeSensor DaemonSet failed to be installed",
				ctx, req.NamespacedName, nodesensor, logger)
			logger.Error(err, "Failed to create new DaemonSet", "DaemonSet.Namespace", ds.Namespace, "DaemonSet.Name", ds.Name)
			return ctrl.Result{}, err
		}

		err = r.conditionsUpdate(falconv1alpha1.ConditionDaemonSetReady,
			metav1.ConditionTrue,
			falconv1alpha1.ReasonInstallSucceeded,
			"FalconNodeSensor DaemonSet has been successfully installed",
			ctx, req.NamespacedName, nodesensor, logger)
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
		imgUpdate := updateDaemonSetImages(dsUpdate, image, logger)
		affUpdate := updateDaemonSetAffinity(dsUpdate, nodesensor, logger)
		containerVolUpdate := updateDaemonSetContainerVolumes(dsUpdate, dsTarget, logger)
		volumeUpdates := updateDaemonSetVolumes(dsUpdate, dsTarget, logger)
		resources := updateDaemonSetResources(dsUpdate, dsTarget, logger)
		initResources := updateDaemonSetInitContainerResources(dsUpdate, dsTarget, logger)
		pc := updateDaemonSetPriorityClass(dsUpdate, dsTarget, logger)
		capabilities := updateDaemonSetCapabilities(dsUpdate, dsTarget, logger)
		initArgs := updateDaemonSetInitArgs(dsUpdate, dsTarget, logger)
		proxyUpdates := updateDaemonSetContainerProxy(dsUpdate, logger)
		tolsUpdate, err := r.updateDaemonSetTolerations(ctx, dsUpdate, nodesensor, logger)
		if err != nil {
			return ctrl.Result{}, err
		}

		// Update the daemonset and re-spin pods with changes
		if imgUpdate || tolsUpdate || affUpdate || containerVolUpdate || volumeUpdates || resources || pc || capabilities || initArgs || initResources || proxyUpdates || updated {
			err = r.Update(ctx, dsUpdate)
			if err != nil {
				err = r.conditionsUpdate(falconv1alpha1.ConditionDaemonSetReady,
					metav1.ConditionTrue,
					falconv1alpha1.ReasonUpdateFailed,
					"FalconNodeSensor DaemonSet update has failed",
					ctx, req.NamespacedName, nodesensor, logger)
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
				ctx, req.NamespacedName, nodesensor, logger)
			if err != nil {
				return ctrl.Result{}, err
			}
			logger.Info("FalconNodeSensor DaemonSet configuration changed. Pods have been restarted.")
		}
	}

	imgVer := common.ImageVersion(image)
	if nodesensor.Status.Sensor != imgVer {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			err := r.Get(ctx, req.NamespacedName, nodesensor)
			if err != nil {
				return err
			}

			nodesensor.Status.Sensor = imgVer
			return r.Status().Update(ctx, nodesensor)
		})
		if err != nil {
			log.Error(err, "Failed to update FalconNodeSensor status for nodesensor.Status.Sensor")
			return ctrl.Result{}, err
		}
	}

	err = r.conditionsUpdate(falconv1alpha1.ConditionSuccess,
		metav1.ConditionTrue,
		falconv1alpha1.ReasonInstallSucceeded,
		"FalconNodeSensor installation completed",
		ctx, req.NamespacedName, nodesensor, logger)
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
	err := common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: nodesensor.Spec.InstallNamespace}, &ns)
	if err != nil && errors.IsNotFound(err) {
		ns = corev1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: nodesensor.Spec.InstallNamespace,
			},
		}

		err = ctrl.SetControllerReference(nodesensor, &ns, r.Scheme)
		if err != nil {
			logger.Error(err, "Unable to assign Controller Reference to the Namespace")
		}

		err = r.Create(ctx, &ns)
		if err != nil && !errors.IsAlreadyExists(err) {
			logger.Error(err, "Failed to create new namespace", "Namespace.Name", nodesensor.Spec.InstallNamespace)
			return false, err
		}

		return true, nil
	} else if err != nil {
		logger.Error(err, "Failed to get FalconNodeSensor Namespace")
		return false, err
	}

	return false, nil
}

// handlePriorityClass creates and updates the priority class
func (r *FalconNodeSensorReconciler) handlePriorityClass(ctx context.Context, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) error {
	existingPC := &schedulingv1.PriorityClass{}
	pcName := nodesensor.Spec.Node.PriorityClass.Name
	update := false

	if pcName == "" && nodesensor.Spec.Node.GKE.Enabled == nil && nodesensor.Spec.Node.PriorityClass.Deploy == nil {
		return nil
	} else if pcName != "" && nodesensor.Spec.Node.PriorityClass.Deploy == nil &&
		(nodesensor.Spec.Node.GKE.Enabled != nil && *nodesensor.Spec.Node.GKE.Enabled) {
		//logger.Info("Skipping PriorityClass creation on GKE AutoPilot because an existing priority class name was provided")
		return nil
	} else if pcName != "" && (nodesensor.Spec.Node.PriorityClass.Deploy == nil || !*nodesensor.Spec.Node.PriorityClass.Deploy) {
		//logger.Info("Skipping PriorityClass creation because an existing priority class name was provided")
		return nil
	}

	if pcName == "" {
		pcName = nodesensor.Name + "-priorityclass"
		nodesensor.Spec.Node.PriorityClass.Name = pcName
	}

	pc := assets.PriorityClass(pcName, nodesensor.Spec.Node.PriorityClass.Value)

	err := common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: pcName, Namespace: nodesensor.Spec.InstallNamespace}, existingPC)
	if err != nil && errors.IsNotFound(err) {
		err = ctrl.SetControllerReference(nodesensor, pc, r.Scheme)
		if err != nil {
			logger.Error(err, "Unable to assign Controller Reference to the PriorityClass")
		}

		err = r.Create(ctx, pc)
		if err != nil {
			logger.Error(err, "Failed to create PriorityClass", "PriorityClass.Name", pcName)
			return err
		}
		logger.Info("Creating FalconNodeSensor PriorityClass")

		return nil
	} else if err != nil {
		logger.Error(err, "Failed to get FalconNodeSensor PriorityClass")
		return err
	}

	if nodesensor.Spec.Node.PriorityClass.Value != nil && existingPC.Value != *nodesensor.Spec.Node.PriorityClass.Value {
		update = true
	}

	if nodesensor.Spec.Node.PriorityClass.Name != "" && existingPC.Name != nodesensor.Spec.Node.PriorityClass.Name {
		update = true
	}

	if update {
		err = r.Delete(ctx, existingPC)
		if err != nil {
			return err
		}

		err = ctrl.SetControllerReference(nodesensor, pc, r.Scheme)
		if err != nil {
			logger.Error(err, "Unable to assign Controller Reference to the PriorityClass")
		}

		err = r.Create(ctx, pc)
		if err != nil {
			return err
		}
		logger.Info("Updating FalconNodeSensor PriorityClass")
	}

	return nil
}

// handleConfigMaps creates and updates the node sensor configmap
func (r *FalconNodeSensorReconciler) handleConfigMaps(ctx context.Context, config *node.ConfigCache, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) (*corev1.ConfigMap, bool, error) {
	var updated bool
	cmName := assets.DaemonsetConfigMapName(nodesensor)

	confCm := &corev1.ConfigMap{}
	configmap := assets.SensorConfigMap(cmName, nodesensor.Spec.InstallNamespace, common.FalconKernelSensor, config.SensorEnvVars())

	err := common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: cmName, Namespace: nodesensor.Spec.InstallNamespace}, confCm)
	if err != nil && errors.IsNotFound(err) {
		// does not exist, create
		err = controllerutil.SetControllerReference(nodesensor, configmap, r.Scheme)
		if err != nil {
			logger.Error(err, "Failed to format new Configmap", "Configmap.Namespace", nodesensor.Spec.InstallNamespace, "Configmap.Name", cmName)
			return nil, updated, err
		}

		if err := r.Create(ctx, configmap); err != nil {
			if errors.IsAlreadyExists(err) {
				// We have got NotFound error during the Get(), but then we have got AlreadyExists error from Create(). Client cache is invalid.
				err = r.Update(ctx, configmap)
				if err != nil {
					logger.Error(err, "Failed to update Configmap", "Configmap.Namespace", nodesensor.Spec.InstallNamespace, "Configmap.Name", cmName)
				}
				return configmap, updated, nil
			} else {
				logger.Error(err, "Failed to create new Configmap", "Configmap.Namespace", nodesensor.Spec.InstallNamespace, "Configmap.Name", cmName)
				return nil, updated, err

			}
		}

		logger.Info("Creating FalconNodeSensor Configmap")
		return nil, updated, nil
	} else if err != nil {
		logger.Error(err, "error getting Configmap")
		return nil, updated, err
	}

	if !reflect.DeepEqual(confCm.Data, configmap.Data) {
		err = r.Update(ctx, configmap)
		if err != nil {
			logger.Error(err, "Failed to update Configmap", "Configmap.Namespace", nodesensor.Spec.InstallNamespace, "Configmap.Name", cmName)
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

	err := common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: common.FalconPullSecretName, Namespace: nodesensor.Spec.InstallNamespace}, &secret)
	if err == nil || !errors.IsNotFound(err) {
		return err
	}

	pulltoken, err := config.GetPullToken(ctx)
	if err != nil {
		return err
	}

	secretData := map[string][]byte{corev1.DockerConfigJsonKey: common.CleanDecodedBase64(pulltoken)}
	secret = *assets.Secret(common.FalconPullSecretName, nodesensor.Spec.InstallNamespace, common.FalconKernelSensor, secretData, corev1.SecretTypeDockerConfigJson)
	err = ctrl.SetControllerReference(nodesensor, &secret, r.Scheme)
	if err != nil {
		logger.Error(err, "Unable to assign Controller Reference to the Pull Secret")
	}
	err = r.Client.Create(ctx, &secret)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			logger.Error(err, "Failed to create new Pull Secret", "Secret.Namespace", nodesensor.Spec.InstallNamespace, "Secret.Name", common.FalconPullSecretName)
			return err
		}
	} else {
		logger.Info("Created a new Pull Secret", "Secret.Namespace", nodesensor.Spec.InstallNamespace, "Secret.Name", common.FalconPullSecretName)
	}
	return nil
}

func updateDaemonSetContainerProxy(ds *appsv1.DaemonSet, logger logr.Logger) bool {
	updated := false
	if len(proxy.ReadProxyVarsFromEnv()) > 0 {
		for i, container := range ds.Spec.Template.Spec.Containers {
			newContainerEnv := common.AppendUniqueEnvVars(container.Env, proxy.ReadProxyVarsFromEnv())
			updatedContainerEnv := common.UpdateEnvVars(container.Env, proxy.ReadProxyVarsFromEnv())
			if !equality.Semantic.DeepEqual(ds.Spec.Template.Spec.Containers[i].Env, newContainerEnv) {
				ds.Spec.Template.Spec.Containers[i].Env = newContainerEnv
				updated = true
			}
			if !equality.Semantic.DeepEqual(ds.Spec.Template.Spec.Containers[i].Env, updatedContainerEnv) {
				ds.Spec.Template.Spec.Containers[i].Env = updatedContainerEnv
				updated = true
			}
			if updated {
				logger.Info("Updating FalconNodeSensor DaemonSet Proxy Settings")
			}
		}
	}

	return updated
}

// If an update is needed, this will update the tolerations from the given DaemonSet
func (r *FalconNodeSensorReconciler) updateDaemonSetTolerations(ctx context.Context, ds *appsv1.DaemonSet, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) (bool, error) {
	tolerations := &ds.Spec.Template.Spec.Tolerations
	origTolerations := nodesensor.Spec.Node.Tolerations
	tolerationsUpdate := !equality.Semantic.DeepEqual(*tolerations, *origTolerations)
	if tolerationsUpdate {
		logger.Info("Updating FalconNodeSensor DaemonSet Tolerations")
		mergedTolerations := k8s_utils.MergeTolerations(*tolerations, *origTolerations)
		*tolerations = mergedTolerations
		nodesensor.Spec.Node.Tolerations = &mergedTolerations

		if err := r.Update(ctx, nodesensor); err != nil {
			logger.Error(err, "Failed to update FalconNodeSensor Tolerations")
			return false, err
		}
	}
	return tolerationsUpdate, nil
}

// If an update is needed, this will update the affinity from the given DaemonSet
func updateDaemonSetAffinity(ds *appsv1.DaemonSet, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) bool {
	nodeAffinity := ds.Spec.Template.Spec.Affinity
	origNodeAffinity := corev1.Affinity{NodeAffinity: &nodesensor.Spec.Node.NodeAffinity}
	affinityUpdate := !equality.Semantic.DeepEqual(nodeAffinity.NodeAffinity, origNodeAffinity.NodeAffinity)
	if affinityUpdate {
		logger.Info("Updating FalconNodeSensor DaemonSet NodeAffinity")
		*nodeAffinity = origNodeAffinity
	}
	return affinityUpdate
}

// If an update is needed, this will update the containervolumes from the given DaemonSet
func updateDaemonSetContainerVolumes(ds, origDS *appsv1.DaemonSet, logger logr.Logger) bool {
	containerVolumeMounts := &ds.Spec.Template.Spec.Containers[0].VolumeMounts
	containerVolumeMountsUpdates := !equality.Semantic.DeepEqual(*containerVolumeMounts, origDS.Spec.Template.Spec.Containers[0].VolumeMounts)
	if containerVolumeMountsUpdates {
		logger.Info("Updating FalconNodeSensor DaemonSet Container volumeMounts")
		*containerVolumeMounts = origDS.Spec.Template.Spec.Containers[0].VolumeMounts
	}

	containerVolumeMounts = &ds.Spec.Template.Spec.InitContainers[0].VolumeMounts
	containerVolumeMountsUpdates = !equality.Semantic.DeepEqual(*containerVolumeMounts, origDS.Spec.Template.Spec.InitContainers[0].VolumeMounts)
	if containerVolumeMountsUpdates {
		logger.Info("Updating FalconNodeSensor DaemonSet InitContainer volumeMounts")
		*containerVolumeMounts = origDS.Spec.Template.Spec.InitContainers[0].VolumeMounts
	}

	return containerVolumeMountsUpdates
}

// If an update is needed, this will update the volumes from the given DaemonSet
func updateDaemonSetVolumes(ds, origDS *appsv1.DaemonSet, logger logr.Logger) bool {
	volumeMounts := &ds.Spec.Template.Spec.Volumes
	volumeMountsUpdates := !equality.Semantic.DeepEqual(*volumeMounts, origDS.Spec.Template.Spec.Volumes)
	if volumeMountsUpdates {
		logger.Info("Updating FalconNodeSensor DaemonSet volumes")
		*volumeMounts = origDS.Spec.Template.Spec.Volumes
	}

	return volumeMountsUpdates
}

// If an update is needed, this will update the InitContainer image reference from the given DaemonSet
func updateDaemonSetImages(ds *appsv1.DaemonSet, origImg string, logger logr.Logger) bool {
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

// If an update is needed, this will update the resources from the given DaemonSet
func updateDaemonSetResources(ds, origDS *appsv1.DaemonSet, logger logr.Logger) bool {
	resources := &ds.Spec.Template.Spec.Containers[0].Resources
	resourcesUpdates := !equality.Semantic.DeepEqual(*resources, origDS.Spec.Template.Spec.Containers[0].Resources)
	if resourcesUpdates {
		logger.Info("Updating FalconNodeSensor DaemonSet resources")
		*resources = origDS.Spec.Template.Spec.Containers[0].Resources

	}

	return resourcesUpdates
}

func updateDaemonSetInitContainerResources(ds, origDS *appsv1.DaemonSet, logger logr.Logger) bool {
	resources := &ds.Spec.Template.Spec.InitContainers[0].Resources
	resourcesUpdates := !equality.Semantic.DeepEqual(*resources, origDS.Spec.Template.Spec.InitContainers[0].Resources)
	if resourcesUpdates {
		logger.Info("Updating FalconNodeSensor DaemonSet InitContainer resources")
		*resources = origDS.Spec.Template.Spec.InitContainers[0].Resources
	}

	return resourcesUpdates
}

// If an update is needed, this will update the priority class from the given DaemonSet
func updateDaemonSetPriorityClass(ds, origDS *appsv1.DaemonSet, logger logr.Logger) bool {
	priorityClass := &ds.Spec.Template.Spec.PriorityClassName
	priorityClassUpdates := *priorityClass != origDS.Spec.Template.Spec.PriorityClassName
	if priorityClassUpdates {
		logger.Info("Updating FalconNodeSensor DaemonSet priority class")
		*priorityClass = origDS.Spec.Template.Spec.PriorityClassName
	}

	return priorityClassUpdates
}

// If an update is needed, this will update the capabilities from the given DaemonSet
func updateDaemonSetCapabilities(ds, origDS *appsv1.DaemonSet, logger logr.Logger) bool {
	capabilities := &ds.Spec.Template.Spec.Containers[0].SecurityContext.Capabilities
	capabilitiesUpdates := !equality.Semantic.DeepEqual(*capabilities, origDS.Spec.Template.Spec.Containers[0].SecurityContext.Capabilities)
	if capabilitiesUpdates {
		logger.Info("Updating FalconNodeSensor DaemonSet Container capabilities")
		*capabilities = origDS.Spec.Template.Spec.Containers[0].SecurityContext.Capabilities
	}

	capabilities = &ds.Spec.Template.Spec.InitContainers[0].SecurityContext.Capabilities
	capabilitiesUpdates = !equality.Semantic.DeepEqual(*capabilities, origDS.Spec.Template.Spec.InitContainers[0].SecurityContext.Capabilities)
	if capabilitiesUpdates {
		logger.Info("Updating FalconNodeSensor DaemonSet InitContainer capabilities")
		*capabilities = origDS.Spec.Template.Spec.InitContainers[0].SecurityContext.Capabilities
	}

	return capabilitiesUpdates
}

// If an update is needed, this will update the init args from the given DaemonSet
func updateDaemonSetInitArgs(ds, origDS *appsv1.DaemonSet, logger logr.Logger) bool {
	initArgs := &ds.Spec.Template.Spec.InitContainers[0].Args
	initArgsUpdates := !equality.Semantic.DeepEqual(*initArgs, origDS.Spec.Template.Spec.InitContainers[0].Args)
	if initArgsUpdates {
		logger.Info("Updating FalconNodeSensor DaemonSet init args")
		*initArgs = origDS.Spec.Template.Spec.InitContainers[0].Args
	}

	return initArgsUpdates
}

// handlePermissions creates and updates the service account, role and role binding
func (r *FalconNodeSensorReconciler) handlePermissions(ctx context.Context, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) (bool, error) {
	created, err := r.handleServiceAccount(ctx, nodesensor, logger)
	if created || err != nil {
		return created, err
	}

	return r.handleClusterRoleBinding(ctx, nodesensor, logger)
}

// handleRoleBinding creates and updates RoleBinding
func (r *FalconNodeSensorReconciler) handleClusterRoleBinding(ctx context.Context, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) (bool, error) {
	binding := rbacv1.ClusterRoleBinding{}

	err := common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: common.NodeClusterRoleBindingName}, &binding)
	if err != nil && errors.IsNotFound(err) {
		binding = rbacv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacv1.SchemeGroupVersion.String(),
				Kind:       "ClusterRoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   common.NodeClusterRoleBindingName,
				Labels: common.CRLabels("clusterrolebinding", common.NodeClusterRoleBindingName, common.FalconKernelSensor),
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "falcon-operator-node-sensor-role",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      common.NodeServiceAccountName,
					Namespace: nodesensor.Spec.InstallNamespace,
				},
			},
		}

		err = ctrl.SetControllerReference(nodesensor, &binding, r.Scheme)
		if err != nil {
			logger.Error(err, "Unable to assign Controller Reference to the ClusterRoleBinding")
		}

		logger.Info("Creating FalconNodeSensor ClusterRoleBinding")
		err = r.Create(ctx, &binding)
		if err != nil && !errors.IsAlreadyExists(err) {
			logger.Error(err, "Failed to create new ClusterRoleBinding", "ClusteRoleBinding.Name", common.NodeClusterRoleBindingName)
			return false, err
		}

		return true, nil
	} else if err != nil {
		logger.Error(err, "Failed to get FalconNodeSensor ClusterRoleBinding")
		return false, err
	}

	return false, nil
}

// handleServiceAccount creates and updates the service account and grants necessary permissions to it
func (r *FalconNodeSensorReconciler) handleServiceAccount(ctx context.Context, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) (bool, error) {
	sa := corev1.ServiceAccount{}

	err := common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: common.NodeServiceAccountName, Namespace: nodesensor.Spec.InstallNamespace}, &sa)
	if err != nil && errors.IsNotFound(err) {
		sa = corev1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "ServiceAccount",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: nodesensor.Spec.InstallNamespace,
				Name:      common.NodeServiceAccountName,
				Labels:    common.CRLabels("serviceaccount", common.NodeServiceAccountName, common.FalconKernelSensor),
			},
		}

		err = ctrl.SetControllerReference(nodesensor, &sa, r.Scheme)
		if err != nil {
			logger.Error(err, "Unable to assign Controller Reference to the ServiceAccount")
		}

		logger.Info("Creating FalconNodeSensor ServiceAccount")
		err = r.Create(ctx, &sa)
		if err != nil && !errors.IsAlreadyExists(err) {
			logger.Error(err, "Failed to create new ServiceAccount", "Namespace.Name", nodesensor.Spec.InstallNamespace, "ServiceAccount.Name", common.NodeServiceAccountName)
			return false, err
		}

		return true, nil
	} else if err != nil {
		logger.Error(err, "Failed to get FalconNodeSensor ServiceAccount")
		return false, err
	}

	return false, nil
}

// handleServiceAccount creates and updates the service account and grants necessary permissions to it
func (r *FalconNodeSensorReconciler) handleSAAnnotations(ctx context.Context, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) error {
	sa := corev1.ServiceAccount{}
	saAnnotations := nodesensor.Spec.Node.ServiceAccount.Annotations

	err := common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: common.NodeServiceAccountName, Namespace: nodesensor.Spec.InstallNamespace}, &sa)
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
		logger.Error(err, "Failed to update ServiceAccount Annotations", "ServiceAccount.Namespace", nodesensor.Spec.InstallNamespace, "Annotations", saAnnotations)
		return err
	}
	logger.Info("Updating FalconNodeSensor ServiceAccount Annotations", "Annotations", saAnnotations)

	return nil
}

// statusUpdate updates the FalconNodeSensor CR conditions
func (r *FalconNodeSensorReconciler) conditionsUpdate(condType string, status metav1.ConditionStatus, reason string, message string, ctx context.Context, nsType types.NamespacedName, nodesensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) error {
	if !meta.IsStatusConditionPresentAndEqual(nodesensor.Status.Conditions, condType, status) {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			err := r.Get(ctx, nsType, nodesensor)
			if err != nil {
				return err
			}

			meta.SetStatusCondition(&nodesensor.Status.Conditions, metav1.Condition{
				Status:             status,
				Reason:             reason,
				Message:            message,
				Type:               condType,
				ObservedGeneration: nodesensor.GetGeneration(),
			})

			return r.Status().Update(ctx, nodesensor)
		})
		if err != nil {
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
	listOptions := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{common.FalconComponentKey: common.FalconKernelSensor}),
		Namespace:     nodesensor.Spec.InstallNamespace,
	}

	if err := r.List(ctx, dsList, listOptions); err != nil {
		if err = r.Reader.List(ctx, dsList, listOptions); err != nil {
			return err
		}
	}

	// Delete the Daemonset containing the sensor
	if err := r.Delete(ctx,
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: nodesensor.Name, Namespace: nodesensor.Spec.InstallNamespace,
			},
		}); err != nil && !errors.IsNotFound(err) {
		logger.Error(err, "Failed to cleanup Falcon sensor DaemonSet pods")
		return err
	}

	// Check if the cleanup DS is created. If not, create it.
	err := common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: dsCleanupName, Namespace: nodesensor.Spec.InstallNamespace}, daemonset)
	if err != nil && errors.IsNotFound(err) {
		// Define a new DS for cleanup
		ds := assets.RemoveNodeDirDaemonset(dsCleanupName, image, serviceAccount, nodesensor)

		// Create the cleanup DS
		err = r.Create(ctx, ds)
		if err != nil {
			logger.Error(err, "Failed to delete node directory with cleanup DaemonSet", "Path", common.FalconHostInstallDir)
			return err
		}

		var lastCompletedCount int32
		var lastNodeCount int32
		var crashloopingPodNodes []string
		// Start inifite loop to check that all pods have either completed or are running in the DS
		for {
			// List all pods with the "cleanup" label in the appropriate NS
			cleanupListOptions := &client.ListOptions{
				LabelSelector: labels.SelectorFromSet(labels.Set{common.FalconInstanceNameKey: "cleanup"}),
				Namespace:     nodesensor.Spec.InstallNamespace,
			}
			if err := r.List(ctx, &pods, cleanupListOptions); err != nil {
				if err = r.Reader.List(ctx, &pods, cleanupListOptions); err != nil {
					return err
				}
			}

			// Reset completedCount each loop, to ensure we don't count the same node(s) multiple times
			var completedCount int32 = 0
			// Reset the nodeCount to the desired number of pods to be scheduled for cleanup each loop, in case the cluster has scaled down
			for _, dSet := range dsList.Items {
				nodeCount = dSet.Status.DesiredNumberScheduled
				if lastNodeCount != nodeCount {
					logger.Info("Setting DaemonSet node count", "Number of nodes", nodeCount)
				}
				lastNodeCount = nodeCount
			}

			// When the pods have a status of completed or running, increment the count.
			// The reason running is an acceptable value is because the pods should be running the sleep command and have already cleaned up /opt/CrowdStrike
			for _, pod := range pods.Items {
				switch pod.Status.Phase {
				case "Running", "Succeeded":
					completedCount++
				case "Pending":
					if k8sutils.IsInitPodCrashLooping(&pod) {
						if !slices.Contains(crashloopingPodNodes, pod.Spec.NodeName) {
							logger.Info(fmt.Sprintf("/opt/CrowdStrike may have not been removed on node %s due to the cleanup pod crashlooping. See the troubleshooting section of the node sensor documentation for more information.", pod.Spec.NodeName))
							_ = append(crashloopingPodNodes, pod.Spec.NodeName)
						}
						completedCount++
					}
				}
			}

			// Break out of the infinite loop for cleanup when the completed or running DS count reaches the desired node count
			if completedCount == nodeCount {
				logger.Info("Clean up pods should be done. Continuing deleting.")
				break
			} else if completedCount < nodeCount && completedCount > 0 {
				if completedCount != lastCompletedCount {
					logger.Info("Waiting for cleanup pods to complete. Retrying....", "Number of pods still processing task", completedCount)
				}
				lastCompletedCount = completedCount
			}

			err = common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: dsCleanupName, Namespace: nodesensor.Spec.InstallNamespace}, daemonset)
			if err != nil && errors.IsNotFound(err) {
				logger.Info("Clean-up daemonset has been removed")
				break
			}
		}

		// The cleanup DS should be completed so delete the cleanup DS
		if err := r.Delete(ctx,
			&appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: dsCleanupName, Namespace: nodesensor.Spec.InstallNamespace,
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

func (r *FalconNodeSensorReconciler) reconcileObjectWithName(ctx context.Context, name types.NamespacedName) error {
	obj := &falconv1alpha1.FalconNodeSensor{}
	err := r.Get(ctx, name, obj)
	if err != nil {
		return err
	}

	clog.FromContext(ctx).Info("reconciling FalconNodeSensor object", "namespace", obj.Namespace, "name", obj.Name)
	r.reconcileObject(obj)
	return nil
}

func shouldTrackSensorVersions(obj *falconv1alpha1.FalconNodeSensor) bool {
	return obj.Spec.FalconAPI != nil && obj.Spec.Node.Advanced.IsAutoUpdating()
}

func (r *FalconNodeSensorReconciler) injectFalconSecretData(ctx context.Context, nodeSensor *falconv1alpha1.FalconNodeSensor, logger logr.Logger) error {
	logger.Info("injecting Falcon secret data into Spec.Falcon and Spec.FalconAPI - sensitive manifest values will be overwritten with values in k8s secret")

	return k8sutils.InjectFalconSecretData(ctx, r, nodeSensor)
}
