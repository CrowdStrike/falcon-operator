package falcon

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"dario.cat/mergo"
	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/version"
	"github.com/go-logr/logr"
)

// FalconDeploymentReconciler reconciles a FalconDeployment object
type FalconDeploymentReconciler struct {
	client.Client
	Reader    client.Reader
	Scheme    *runtime.Scheme
	OpenShift bool
}

//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falcondeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falcondeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falcondeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=resourcequotas,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update
//+kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=replicationcontrollers,verbs=get;list;watch
//+kubebuilder:rbac:groups="apps",resources=daemonsets,verbs=get;list;watch
//+kubebuilder:rbac:groups="apps",resources=replicasets,verbs=get;list;watch
//+kubebuilder:rbac:groups="apps",resources=statefulsets,verbs=get;list;watch
//+kubebuilder:rbac:groups="batch",resources=cronjobs;jobs,verbs=get;list;watch
//+kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="coordination.k8s.io",resources=leases,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="image.openshift.io",resources=imagestreams,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=create;get;list;update;watch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=create;get;list;update;watch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the FalconDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *FalconDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("reconciling FalconDeployment")

	falconDeployment := &falconv1alpha1.FalconDeployment{}
	err := r.Get(ctx, req.NamespacedName, falconDeployment)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then, it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("FalconDeployment resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}

		log.Error(err, "Failed to get FalconDeployment resource")
		return ctrl.Result{}, err
	}

	// Let's just set the status as Unknown when no status is available
	if len(falconDeployment.Status.Conditions) == 0 {
		err := r.statusUpdate(ctx, req, log, falconDeployment, falconv1alpha1.ConditionPending,
			metav1.ConditionFalse,
			falconv1alpha1.ReasonReqNotMet,
			"FalconDeployment progressing",
			nil)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if falconDeployment.Status.Version != version.Get() {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			err := r.Get(ctx, req.NamespacedName, falconDeployment)
			if err != nil {
				return err
			}
			falconDeployment.Status.Version = version.Get()
			return r.Status().Update(ctx, falconDeployment)
		})
		if err != nil {
			log.Error(err, "Failed to update FalconDeployment status for FalconDeployment.Status.Version")
			return ctrl.Result{}, err
		}
	}

	if falconDeployment.Spec.FalconAPI != nil {
		cloud, err := falconDeployment.Spec.FalconAPI.FalconCloudWithSecret(ctx, r.Reader, falconDeployment.Spec.FalconSecret)
		if err != nil {
			log.Error(err, "Failed to get Cloud Region")
			return ctrl.Result{}, err
		}

		falconDeployment.Spec.FalconAPI.CloudRegion = cloud.String()
	}

	if err = r.reconcileAdmissionController(ctx, log, falconDeployment); err != nil {
		return ctrl.Result{}, err
	}

	if err = r.reconcileNodeSensor(ctx, log, falconDeployment); err != nil {
		if strings.Contains(err.Error(), "deletion to complete") {
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	if err = r.reconcileContainerSensor(ctx, log, falconDeployment); err != nil {
		return ctrl.Result{}, err
	}

	if err = r.reconcileImageAnalyzer(ctx, log, falconDeployment); err != nil {
		return ctrl.Result{}, err
	}

	if err = r.reconcileClusterGuard(ctx, log, falconDeployment); err != nil {
		if strings.Contains(err.Error(), "waiting for namespace") {
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	// Set or clear migration condition based on legacy component usage
	nodeSensorEnabled := falconDeployment.Spec.DeployNodeSensor != nil && *falconDeployment.Spec.DeployNodeSensor
	admissionEnabled := falconDeployment.Spec.DeployAdmissionController != nil && *falconDeployment.Spec.DeployAdmissionController
	var migrationCondition *metav1.Condition
	if nodeSensorEnabled || admissionEnabled {
		var parts []string
		if nodeSensorEnabled {
			parts = append(parts, "deployNodeSensor")
		}
		if admissionEnabled {
			parts = append(parts, "deployAdmissionController")
		}
		migrationCondition = &metav1.Condition{
			Type:    "MigrationRequired",
			Status:  metav1.ConditionTrue,
			Reason:  "LegacyComponentsEnabled",
			Message: strings.Join(parts, " and ") + " are deprecated and will be removed in a future release. Migrate to FalconClusterGuard (spec.deployClusterGuard: true).",
		}
	}

	err = r.statusUpdate(ctx, req, log, falconDeployment, falconv1alpha1.ConditionSuccess,
		metav1.ConditionTrue,
		falconv1alpha1.ReasonInstallSucceeded,
		"FalconDeployment installation completed",
		migrationCondition)

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *FalconDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&falconv1alpha1.FalconDeployment{}).
		Owns(&falconv1alpha1.FalconAdmission{}).
		Owns(&falconv1alpha1.FalconClusterGuard{}).
		Owns(&falconv1alpha1.FalconContainer{}).
		Owns(&falconv1alpha1.FalconImageAnalyzer{}).
		Owns(&falconv1alpha1.FalconNodeSensor{}).
		Complete(r)
}

func (r *FalconDeploymentReconciler) reconcileAdmissionController(ctx context.Context, log logr.Logger, falconDeployment *falconv1alpha1.FalconDeployment) error {
	var admissionList falconv1alpha1.FalconAdmissionList
	existingFalconAdmission := &falconv1alpha1.FalconAdmission{}
	updated := false

	if err := r.Client.List(ctx, &admissionList); err != nil {
		return fmt.Errorf("unable to get FalconAdmissionList: %s", err)
	}

	if len(admissionList.Items) != 0 {
		existingFalconAdmission.ObjectMeta = metav1.ObjectMeta{
			Name:      "falcon-kac",
			Namespace: admissionList.Items[0].GetNamespace(),
		}
	}

	if *falconDeployment.Spec.DeployAdmissionController {
		newFalconAdmission := &falconv1alpha1.FalconAdmission{}
		newFalconAdmission.Spec.FalconAPI = falconDeployment.Spec.FalconAPI
		newFalconAdmission.Spec.Registry = falconDeployment.Spec.Registry
		newFalconAdmission.Spec.FalconSecret = falconDeployment.Spec.FalconSecret
		newFalconAdmission.ObjectMeta = metav1.ObjectMeta{
			Name:      "falcon-kac",
			Namespace: falconDeployment.Spec.FalconAdmission.InstallNamespace,
		}

		if err := mergo.Merge(&newFalconAdmission.Spec, falconDeployment.Spec.FalconAdmission, mergo.WithOverride); err != nil {
			return fmt.Errorf("unable to merge specs for FalconAdmission: %v", err)
		}

		if len(admissionList.Items) == 0 {
			if err := ctrl.SetControllerReference(falconDeployment, newFalconAdmission, r.Scheme); err != nil {
				return fmt.Errorf("unable to set controller reference for %s: %v", newFalconAdmission.Name, err)
			}
			return r.create(ctx, log, falconDeployment, newFalconAdmission)
		}

		err := r.Client.Get(ctx, types.NamespacedName{Name: existingFalconAdmission.Name, Namespace: existingFalconAdmission.Namespace}, existingFalconAdmission)

		if err != nil {
			log.Error(err, "Failed to get FalconAdmission resource")
			return err
		}

		if !equality.Semantic.DeepEqual(newFalconAdmission.Spec, existingFalconAdmission.Spec) {
			existingFalconAdmission.Spec = newFalconAdmission.Spec
			updated = true
		}

		if updated {
			if err := r.update(ctx, log, falconDeployment, existingFalconAdmission); err != nil {
				return err
			}
		}
	} else if len(admissionList.Items) != 0 {
		err := r.Client.Get(ctx, types.NamespacedName{Name: existingFalconAdmission.Name, Namespace: existingFalconAdmission.Namespace}, existingFalconAdmission)
		if err != nil {
			log.Error(err, "Failed to get FalconAdmission resource")
			return err
		}
		return r.delete(ctx, log, falconDeployment, existingFalconAdmission)
	}

	return nil
}

func (r *FalconDeploymentReconciler) reconcileNodeSensor(ctx context.Context, log logr.Logger, falconDeployment *falconv1alpha1.FalconDeployment) error {
	var nodeSensorList falconv1alpha1.FalconNodeSensorList
	existingNodeSensor := &falconv1alpha1.FalconNodeSensor{}
	updated := false

	if err := r.Client.List(ctx, &nodeSensorList); err != nil {
		return fmt.Errorf("unable to get FalconNodeSensorList: %s", err)
	}

	if len(nodeSensorList.Items) != 0 {
		existingNodeSensor.ObjectMeta = metav1.ObjectMeta{
			Name:      "falcon-node-sensor",
			Namespace: nodeSensorList.Items[0].GetNamespace(),
		}
	}

	if *falconDeployment.Spec.DeployNodeSensor {
		newNodeSensor := &falconv1alpha1.FalconNodeSensor{}
		newNodeSensor.Spec.FalconAPI = falconDeployment.Spec.FalconAPI
		newNodeSensor.Spec.FalconSecret = falconDeployment.Spec.FalconSecret
		newNodeSensor.ObjectMeta = metav1.ObjectMeta{
			Name:      "falcon-node-sensor",
			Namespace: falconDeployment.Spec.FalconNodeSensor.InstallNamespace,
		}

		if err := mergo.Merge(&newNodeSensor.Spec, falconDeployment.Spec.FalconNodeSensor, mergo.WithOverride); err != nil {
			return fmt.Errorf("unable to merge specs for FalconNodeSensor: %v", err)
		}

		if len(nodeSensorList.Items) == 0 {
			if err := ctrl.SetControllerReference(falconDeployment, newNodeSensor, r.Scheme); err != nil {
				return fmt.Errorf("unable to set controller reference for %s: %v", newNodeSensor.Name, err)
			}
			return r.create(ctx, log, falconDeployment, newNodeSensor)
		}

		err := r.Client.Get(ctx, types.NamespacedName{Name: existingNodeSensor.Name, Namespace: existingNodeSensor.Namespace}, existingNodeSensor)

		if err != nil {
			log.Error(err, "Failed to get FalconNodeSensor resource")
			return err
		}

		if !equality.Semantic.DeepEqual(newNodeSensor.Spec, existingNodeSensor.Spec) {
			existingNodeSensor.Spec = newNodeSensor.Spec
			updated = true
		}

		if updated {
			if err := r.update(ctx, log, falconDeployment, existingNodeSensor); err != nil {
				return err
			}
		}
	} else if len(nodeSensorList.Items) != 0 {
		err := r.Client.Get(ctx, types.NamespacedName{Name: existingNodeSensor.Name, Namespace: existingNodeSensor.Namespace}, existingNodeSensor)
		if err != nil {
			log.Error(err, "Failed to get FalconNodeSensor resource")
			return err
		}
		return r.delete(ctx, log, falconDeployment, existingNodeSensor)
	}

	return nil
}

func (r *FalconDeploymentReconciler) reconcileImageAnalyzer(ctx context.Context, log logr.Logger, falconDeployment *falconv1alpha1.FalconDeployment) error {
	var imageAnalyzerList falconv1alpha1.FalconImageAnalyzerList
	existingImageAnalyzer := &falconv1alpha1.FalconImageAnalyzer{}
	updated := false

	if err := r.Client.List(ctx, &imageAnalyzerList); err != nil {
		return fmt.Errorf("unable to get FalconImageAnalyzerList: %s", err)
	}

	if len(imageAnalyzerList.Items) != 0 {
		existingImageAnalyzer.ObjectMeta = metav1.ObjectMeta{
			Name:      "falcon-image-analyzer",
			Namespace: imageAnalyzerList.Items[0].GetNamespace(),
		}
	}

	if *falconDeployment.Spec.DeployImageAnalyzer {
		newImageAnalyzer := &falconv1alpha1.FalconImageAnalyzer{}
		newImageAnalyzer.Spec.FalconAPI = falconDeployment.Spec.FalconAPI
		newImageAnalyzer.Spec.Registry = falconDeployment.Spec.Registry
		newImageAnalyzer.Spec.FalconSecret = falconDeployment.Spec.FalconSecret
		newImageAnalyzer.ObjectMeta = metav1.ObjectMeta{
			Name:      "falcon-image-analyzer",
			Namespace: falconDeployment.Spec.FalconNodeSensor.InstallNamespace,
		}
		if err := mergo.Merge(&newImageAnalyzer.Spec, falconDeployment.Spec.FalconImageAnalyzer, mergo.WithOverride); err != nil {
			return fmt.Errorf("unable to merge specs for FalconImageAnalyzer: %v", err)
		}

		if len(imageAnalyzerList.Items) == 0 {
			if err := ctrl.SetControllerReference(falconDeployment, newImageAnalyzer, r.Scheme); err != nil {
				return fmt.Errorf("unable to set controller reference for %s: %v", newImageAnalyzer.Name, err)
			}
			return r.create(ctx, log, falconDeployment, newImageAnalyzer)
		}

		err := r.Client.Get(ctx, types.NamespacedName{Name: existingImageAnalyzer.Name, Namespace: existingImageAnalyzer.Namespace}, existingImageAnalyzer)

		if err != nil {
			log.Error(err, "Failed to get FalconImageAnalyzer resource")
			return err
		}

		if !equality.Semantic.DeepEqual(newImageAnalyzer.Spec, existingImageAnalyzer.Spec) {
			existingImageAnalyzer.Spec = newImageAnalyzer.Spec
			updated = true
		}

		if updated {
			if err := r.update(ctx, log, falconDeployment, existingImageAnalyzer); err != nil {
				return err
			}
		}
	} else if len(imageAnalyzerList.Items) != 0 {
		err := r.Client.Get(ctx, types.NamespacedName{Name: existingImageAnalyzer.Name, Namespace: existingImageAnalyzer.Namespace}, existingImageAnalyzer)
		if err != nil {
			log.Error(err, "Failed to get FalconImageAnalyzer resource")
			return err
		}
		return r.delete(ctx, log, falconDeployment, existingImageAnalyzer)
	}

	return nil
}

func (r *FalconDeploymentReconciler) reconcileContainerSensor(ctx context.Context, log logr.Logger, falconDeployment *falconv1alpha1.FalconDeployment) error {
	var containerSensorList falconv1alpha1.FalconContainerList
	existingContainerSensor := &falconv1alpha1.FalconContainer{}
	updated := false

	if err := r.Client.List(ctx, &containerSensorList); err != nil {
		return fmt.Errorf("unable to get FalconContainerList: %s", err)
	}
	if len(containerSensorList.Items) != 0 {
		existingContainerSensor.ObjectMeta = metav1.ObjectMeta{
			Name:      "falcon-container-sensor",
			Namespace: containerSensorList.Items[0].GetNamespace(),
		}
	}

	if *falconDeployment.Spec.DeployContainerSensor {
		newContainerSensor := &falconv1alpha1.FalconContainer{}
		newContainerSensor.Spec.FalconAPI = falconDeployment.Spec.FalconAPI
		newContainerSensor.Spec.Registry = falconDeployment.Spec.Registry
		newContainerSensor.Spec.FalconSecret = falconDeployment.Spec.FalconSecret
		newContainerSensor.ObjectMeta = metav1.ObjectMeta{
			Name:      "falcon-container-sensor",
			Namespace: falconDeployment.Spec.FalconContainerSensor.InstallNamespace,
		}

		if err := mergo.Merge(&newContainerSensor.Spec, falconDeployment.Spec.FalconContainerSensor, mergo.WithOverride); err != nil {
			return fmt.Errorf("unable to merge specs for FalconContainerSensor: %v", err)
		}

		if len(containerSensorList.Items) == 0 {
			if err := ctrl.SetControllerReference(falconDeployment, newContainerSensor, r.Scheme); err != nil {
				return fmt.Errorf("unable to set controller reference for %s: %v", newContainerSensor.Name, err)
			}
			return r.create(ctx, log, falconDeployment, newContainerSensor)
		}

		err := r.Client.Get(ctx, types.NamespacedName{Name: existingContainerSensor.Name, Namespace: existingContainerSensor.Namespace}, existingContainerSensor)

		if err != nil {
			log.Error(err, "Failed to get FalconContainerSensor resource")
			return err
		}

		if !equality.Semantic.DeepEqual(newContainerSensor.Spec, existingContainerSensor.Spec) {
			existingContainerSensor.Spec = newContainerSensor.Spec
			updated = true
		}

		if updated {
			if err := r.update(ctx, log, falconDeployment, existingContainerSensor); err != nil {
				return err
			}
		}
	} else if len(containerSensorList.Items) != 0 {
		err := r.Client.Get(ctx, types.NamespacedName{Name: existingContainerSensor.Name, Namespace: existingContainerSensor.Namespace}, existingContainerSensor)
		if err != nil {
			log.Error(err, "Failed to get FalconContainerSensor resource")
			return err
		}
		return r.delete(ctx, log, falconDeployment, existingContainerSensor)
	}

	return nil
}

func (r *FalconDeploymentReconciler) reconcileClusterGuard(ctx context.Context, log logr.Logger, falconDeployment *falconv1alpha1.FalconDeployment) error {
	var clusterGuardList falconv1alpha1.FalconClusterGuardList
	existingClusterGuard := &falconv1alpha1.FalconClusterGuard{}
	updated := false

	if err := r.Client.List(ctx, &clusterGuardList); err != nil {
		return fmt.Errorf("unable to get FalconClusterGuardList: %s", err)
	}

	if len(clusterGuardList.Items) != 0 {
		existingClusterGuard.ObjectMeta = metav1.ObjectMeta{
			Name:      "falcon-clusterguard",
			Namespace: clusterGuardList.Items[0].GetNamespace(),
		}
	}

	if *falconDeployment.Spec.DeployClusterGuard {
		installNamespace := falconDeployment.Spec.FalconClusterGuard.InstallNamespace
		if installNamespace == "" {
			installNamespace = "falcon-system"
		}
		ns := &corev1.Namespace{}
		if err := r.Reader.Get(ctx, types.NamespacedName{Name: installNamespace}, ns); err == nil && ns.DeletionTimestamp != nil {
			return fmt.Errorf("waiting for namespace %s to finish deleting before deploying FalconClusterGuard", installNamespace)
		} else if err != nil && !apierrors.IsNotFound(err) {
			return err
		}

		newClusterGuard := &falconv1alpha1.FalconClusterGuard{}
		newClusterGuard.Spec.FalconAPI = falconDeployment.Spec.FalconAPI
		newClusterGuard.Spec.FalconSecret = falconDeployment.Spec.FalconSecret
		newClusterGuard.ObjectMeta = metav1.ObjectMeta{
			Name:      "falcon-clusterguard",
			Namespace: falconDeployment.Spec.FalconClusterGuard.InstallNamespace,
		}

		if err := mergo.Merge(&newClusterGuard.Spec, falconDeployment.Spec.FalconClusterGuard, mergo.WithOverride); err != nil {
			return fmt.Errorf("unable to merge specs for FalconClusterGuard: %v", err)
		}

		if len(clusterGuardList.Items) == 0 {
			if err := ctrl.SetControllerReference(falconDeployment, newClusterGuard, r.Scheme); err != nil {
				return fmt.Errorf("unable to set controller reference for %s: %v", newClusterGuard.Name, err)
			}
			return r.create(ctx, log, falconDeployment, newClusterGuard)
		}

		err := r.Client.Get(ctx, types.NamespacedName{Name: existingClusterGuard.Name, Namespace: existingClusterGuard.Namespace}, existingClusterGuard)
		if err != nil {
			log.Error(err, "Failed to get FalconClusterGuard resource")
			return err
		}

		if !reflect.DeepEqual(newClusterGuard.Spec, existingClusterGuard.Spec) {
			existingClusterGuard.Spec = newClusterGuard.Spec
			updated = true
		}

		if updated {
			if err := r.update(ctx, log, falconDeployment, existingClusterGuard); err != nil {
				return err
			}
		}
	} else if len(clusterGuardList.Items) != 0 {
		err := r.Client.Get(ctx, types.NamespacedName{Name: existingClusterGuard.Name, Namespace: existingClusterGuard.Namespace}, existingClusterGuard)
		if err != nil {
			log.Error(err, "Failed to get FalconClusterGuard resource")
			return err
		}
		return r.delete(ctx, log, falconDeployment, existingClusterGuard)
	}

	return nil
}

func (r *FalconDeploymentReconciler) statusUpdate(ctx context.Context, req ctrl.Request, log logr.Logger, falconDeployment *falconv1alpha1.FalconDeployment, condType string, status metav1.ConditionStatus, reason string, message string, migration *metav1.Condition) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := r.Get(ctx, req.NamespacedName, falconDeployment)
		if err != nil {
			return err
		}

		if migration != nil {
			migration.ObservedGeneration = falconDeployment.GetGeneration()
			meta.SetStatusCondition(&falconDeployment.Status.Conditions, *migration)
		} else {
			meta.RemoveStatusCondition(&falconDeployment.Status.Conditions, "MigrationRequired")
		}

		meta.SetStatusCondition(&falconDeployment.Status.Conditions, metav1.Condition{
			Status:             status,
			Reason:             reason,
			Message:            message,
			Type:               condType,
			ObservedGeneration: falconDeployment.GetGeneration(),
		})

		return r.Status().Update(ctx, falconDeployment)
	})
	if err != nil {
		log.Error(err, "Failed to update FalconDeployment status")
		return err
	}

	return nil
}

func (r *FalconDeploymentReconciler) create(ctx context.Context, log logr.Logger, falconDeployment *falconv1alpha1.FalconDeployment, obj runtime.Object) error {
	switch t := obj.(type) {
	case client.Object:
		name := t.GetName()
		namespace := t.GetNamespace()
		gvk, err := apiutil.GVKForObject(t, r.Scheme)
		if err != nil {
			panic(err)
		}
		// gvk := t.GetObjectKind().GroupVersionKind()
		log.Info(fmt.Sprintf("Creating %s %s in namespace %s", gvk.Kind, name, namespace))
		err = r.Client.Create(ctx, t)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				log.Info(fmt.Sprintf("Falcon %s %s already exists in namespace %s", gvk.Kind, name, namespace))
			} else {
				return fmt.Errorf("failed to create %s %s in namespace %s %v", gvk.Kind, name, namespace, err)
			}
		}

		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err := r.Get(ctx, client.ObjectKeyFromObject(falconDeployment), falconDeployment); err != nil {
				return err
			}
			meta.SetStatusCondition(&falconDeployment.Status.Conditions, metav1.Condition{
				Type:    fmt.Sprintf("%sReady", strings.ToUpper(gvk.Kind[:1])+gvk.Kind[1:]),
				Status:  metav1.ConditionTrue,
				Reason:  "Created",
				Message: fmt.Sprintf("Successfully created %s %s in %s", gvk.Kind, name, namespace),
			})

			return r.Client.Status().Update(ctx, falconDeployment)
		})

		return err
	default:
		return fmt.Errorf("unrecognized kube object type: %T", obj)
	}
}

func (r *FalconDeploymentReconciler) update(ctx context.Context, log logr.Logger, falconDeployment *falconv1alpha1.FalconDeployment, obj runtime.Object) error {
	switch t := obj.(type) {
	case client.Object:
		name := t.GetName()
		namespace := t.GetNamespace()
		gvk := t.GetObjectKind().GroupVersionKind()
		log.Info(fmt.Sprintf("Updating %s %s in namespace %s", gvk.Kind, name, namespace))
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			// Re-fetch to get the latest ResourceVersion; keep the spec already set on t.
			current := t.DeepCopyObject().(client.Object)
			if fetchErr := r.Client.Get(ctx, client.ObjectKeyFromObject(t), current); fetchErr != nil {
				return fetchErr
			}
			t.SetResourceVersion(current.GetResourceVersion())
			return r.Client.Update(ctx, t)
		})
		if err != nil {
			if apierrors.IsNotFound(err) {
				log.Info(fmt.Sprintf("%s %s does not exist in namespace %s", gvk.Kind, name, namespace))
			}
			return fmt.Errorf("cannot update object %s %s in namespace %s: %v", gvk.Kind, name, namespace, err)
		}

		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err := r.Get(ctx, client.ObjectKeyFromObject(falconDeployment), falconDeployment); err != nil {
				return err
			}
			meta.SetStatusCondition(&falconDeployment.Status.Conditions, metav1.Condition{
				Type:    fmt.Sprintf("%sReady", strings.ToUpper(gvk.Kind[:1])+gvk.Kind[1:]),
				Status:  metav1.ConditionTrue,
				Reason:  "Updated",
				Message: fmt.Sprintf("Successfully updated %s %s in %s", gvk.Kind, name, namespace),
			})

			return r.Client.Status().Update(ctx, falconDeployment)
		})

		return err
	default:
		return fmt.Errorf("unrecognized kube object type: %T", obj)
	}
}

func (r *FalconDeploymentReconciler) delete(ctx context.Context, log logr.Logger, falconDeployment *falconv1alpha1.FalconDeployment, obj runtime.Object) error {
	switch t := obj.(type) {
	case client.Object:
		name := t.GetName()
		namespace := t.GetNamespace()
		gvk := t.GetObjectKind().GroupVersionKind()
		log.Info(fmt.Sprintf("Deleting %s %s in namespace %s", gvk.Kind, name, namespace))
		err := r.Client.Delete(ctx, t)
		if err != nil {
			if apierrors.IsNotFound(err) {
				log.Info(fmt.Sprintf("%s object %s does not exist in namespace %s", gvk.Kind, name, namespace))
			}
			return fmt.Errorf("cannot delete object %s %s in namespace %s: %v", gvk.Kind, name, namespace, err)
		}

		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err := r.Get(ctx, client.ObjectKeyFromObject(falconDeployment), falconDeployment); err != nil {
				return err
			}
			meta.SetStatusCondition(&falconDeployment.Status.Conditions, metav1.Condition{
				Type:    fmt.Sprintf("%sReady", strings.ToUpper(gvk.Kind[:1])+gvk.Kind[1:]),
				Status:  metav1.ConditionTrue,
				Reason:  "Deleted",
				Message: fmt.Sprintf("Successfully deleted %s %s in %s", gvk.Kind, name, namespace),
			})

			return r.Client.Status().Update(ctx, falconDeployment)
		})

		return err
	default:
		return fmt.Errorf("unrecognized kube object type: %T", obj)
	}
}
