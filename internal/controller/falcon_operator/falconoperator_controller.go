package falcon

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	arv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/version"
	"github.com/go-logr/logr"
)

// FalconOperatorReconciler reconciles a FalconOperator object
type FalconOperatorReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	OpenShift bool
}

//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconoperators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconoperators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconoperators/finalizers,verbs=update
//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconadmissions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconadmissions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconadmissions/finalizers,verbs=update
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
//+kubebuilder:rbac:groups="admissionregistration.k8s.io",resources=validatingwebhookconfigurations,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=create;get;list;update;watch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=create;get;list;update;watch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the FalconOperator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *FalconOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	FalconOperator := &falconv1alpha1.FalconOperator{}
	err := r.Get(ctx, req.NamespacedName, FalconOperator)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then, it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("FalconOperator resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}

		log.Error(err, "Failed to get FalconOperator resource")
		return ctrl.Result{}, err
	}

	log.Info("Before first set status")

	// Let's just set the status as Unknown when no status is available
	if len(FalconOperator.Status.Conditions) == 0 {
		err := r.StatusUpdate(ctx, req, log, FalconOperator, falconv1alpha1.ConditionPending,
			metav1.ConditionFalse,
			falconv1alpha1.ReasonReqNotMet,
			"FalconOperator progressing")
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	log.Info("After first set status")

	if FalconOperator.Status.Version != version.Get() {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			err := r.Get(ctx, req.NamespacedName, FalconOperator)
			if err != nil {
				return err
			}
			FalconOperator.Status.Version = version.Get()
			return r.Status().Update(ctx, FalconOperator)
		})
		if err != nil {
			log.Error(err, "Failed to update FalconOperator status for FalconOperator.Status.Version")
			return ctrl.Result{}, err
		}
	}

	// log.Info("Before reconcile namespace")
	// if err := r.reconcileNamespace(ctx, log, FalconOperator); err != nil {
	// 	return ctrl.Result{}, err
	// }
	// log.Info("After reconcile namespace")
	log.Info("Before reconcile admission")
	if err = r.reconcileFalconAdmission(ctx, log, FalconOperator); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Before last status update")
	err = r.StatusUpdate(ctx, req, log, FalconOperator, falconv1alpha1.ConditionSuccess,
		metav1.ConditionTrue,
		falconv1alpha1.ReasonInstallSucceeded,
		"FalconOperator installation completed")

	log.Info("After last status update")

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *FalconOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&falconv1alpha1.FalconOperator{}).
		Owns(&falconv1alpha1.FalconAdmission{}).
		Owns(&corev1.Namespace{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.ResourceQuota{}).
		Owns(&corev1.Secret{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&arv1.ValidatingWebhookConfiguration{}).
		Complete(r)
}

// func (r *FalconOperatorReconciler) reconcileNamespace(ctx context.Context, log logr.Logger, FalconOperator *falconv1alpha1.FalconOperator) error {
// 	namespace := assets.Namespace(FalconOperator.Spec.InstallNamespace)
// 	existingNamespace := &corev1.Namespace{}

// 	err := r.Client.Get(ctx, types.NamespacedName{Name: FalconOperator.Spec.InstallNamespace}, existingNamespace)
// 	if err != nil && apierrors.IsNotFound(err) {
// 		err = r.Client.Create(ctx, namespace)
// 		if err != nil {
// 			return err
// 		}

// 		return nil
// 	} else if err != nil {
// 		log.Error(err, "Failed to get FalconOperator Namespace")
// 		return err
// 	}

// 	return nil
// }

func (r *FalconOperatorReconciler) reconcileFalconAdmission(ctx context.Context, log logr.Logger, FalconOperator *falconv1alpha1.FalconOperator) error {
	updated := false
	existingFalconAdmission := &falconv1alpha1.FalconAdmission{}
	newFalconAdmission := &falconv1alpha1.FalconAdmission{}
	if FalconOperator.Spec.FalconAdmissionConfig != nil {
		newFalconAdmission = &falconv1alpha1.FalconAdmission{Spec: *FalconOperator.Spec.FalconAdmissionConfig}
	}

	log.Info("checking if falcon admission already exists")
	err := r.Client.Get(ctx, types.NamespacedName{Name: newFalconAdmission.Name}, existingFalconAdmission)
	// if err != nil && apierrors.IsNotFound(err) && *FalconOperator.Spec.DeployAdmissionController {
	// 	if err = ctrl.SetControllerReference(FalconOperator, newFalconAdmission, r.Scheme); err != nil {
	// 		return fmt.Errorf("unable to set controller reference for %s: %v", newFalconAdmission.ObjectMeta.Name, err)
	// 	}
	// 	return r.Create(ctx, log, FalconOperator, newFalconAdmission)
	// }

	if *FalconOperator.Spec.DeployAdmissionController {
		if err != nil && apierrors.IsNotFound(err) {
			if err = ctrl.SetControllerReference(FalconOperator, newFalconAdmission, r.Scheme); err != nil {
				return fmt.Errorf("unable to set controller reference for %s: %v", newFalconAdmission.ObjectMeta.Name, err)
			}
			return r.Create(ctx, log, FalconOperator, newFalconAdmission)
		}
	} else if err == nil {
		return r.Delete(ctx, log, FalconOperator, existingFalconAdmission)
	}

	if !reflect.DeepEqual(newFalconAdmission.Spec.FalconAPI, existingFalconAdmission.Spec.FalconAPI) {
		existingFalconAdmission.Spec.FalconAPI = newFalconAdmission.Spec.FalconAPI
		updated = true
	}

	if updated {
		if err := r.Update(ctx, log, FalconOperator, existingFalconAdmission); err != nil {
			return err
		}
	}

	return nil
}

func (r *FalconOperatorReconciler) StatusUpdate(ctx context.Context, req ctrl.Request, log logr.Logger, FalconOperator *falconv1alpha1.FalconOperator, condType string, status metav1.ConditionStatus, reason string, message string) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := r.Get(ctx, req.NamespacedName, FalconOperator)
		if err != nil {
			return err
		}

		meta.SetStatusCondition(&FalconOperator.Status.Conditions, metav1.Condition{
			Status:             status,
			Reason:             reason,
			Message:            message,
			Type:               condType,
			ObservedGeneration: FalconOperator.GetGeneration(),
		})

		return r.Status().Update(ctx, FalconOperator)
	})
	if err != nil {
		log.Error(err, "Failed to update FalconOperator status")
		return err
	}

	return nil
}

func (r *FalconOperatorReconciler) Create(ctx context.Context, log logr.Logger, FalconOperator *falconv1alpha1.FalconOperator, obj runtime.Object) error {
	switch t := obj.(type) {
	case client.Object:
		name := t.GetName()
		namespace := t.GetNamespace()
		gvk := t.GetObjectKind().GroupVersionKind()
		log.Info(fmt.Sprintf("Creating Falcon Admission object %s %s in namespace %s", gvk.Kind, name, namespace))
		err := r.Client.Create(ctx, t)
		if err != nil {
			if errors.IsAlreadyExists(err) {
				log.Info(fmt.Sprintf("Falcon Admission object %s %s already exists in namespace %s", gvk.Kind, name, namespace))
			} else {
				return fmt.Errorf("failed to create %s %s in namespace %s: %v", gvk.Kind, name, namespace, err)
			}
		}

		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			meta.SetStatusCondition(&FalconOperator.Status.Conditions, metav1.Condition{
				Type:    fmt.Sprintf("%sReady", strings.ToUpper(gvk.Kind[:1])+gvk.Kind[1:]),
				Status:  metav1.ConditionTrue,
				Reason:  "Created",
				Message: fmt.Sprintf("Successfully created %s %s in %s", gvk.Kind, name, namespace),
			})

			return r.Client.Status().Update(ctx, FalconOperator)
		})

		return err
	default:
		return fmt.Errorf("Unrecognized kube object type: %T", obj)
	}
}

func (r *FalconOperatorReconciler) Update(ctx context.Context, log logr.Logger, FalconOperator *falconv1alpha1.FalconOperator, obj runtime.Object) error {
	switch t := obj.(type) {
	case client.Object:
		name := t.GetName()
		namespace := t.GetNamespace()
		gvk := t.GetObjectKind().GroupVersionKind()
		log.Info(fmt.Sprintf("Updating %s %s in namespace %s", gvk.Kind, name, namespace))
		err := r.Client.Update(ctx, t)
		if err != nil {
			if errors.IsNotFound(err) {
				log.Info(fmt.Sprintf("%s %s does not exist in namespace %s", gvk.Kind, name, namespace))
			}
			return fmt.Errorf("cannot update object %s %s in namespace %s: %v", gvk.Kind, name, namespace, err)
		}

		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			meta.SetStatusCondition(&FalconOperator.Status.Conditions, metav1.Condition{
				Type:    fmt.Sprintf("%sReady", strings.ToUpper(gvk.Kind[:1])+gvk.Kind[1:]),
				Status:  metav1.ConditionTrue,
				Reason:  "Updated",
				Message: fmt.Sprintf("Successfully updated %s %s in %s", gvk.Kind, name, namespace),
			})

			return r.Client.Status().Update(ctx, FalconOperator)
		})

		return err
	default:
		return fmt.Errorf("unrecognized kube object type: %T", obj)
	}
}

func (r *FalconOperatorReconciler) Delete(ctx context.Context, log logr.Logger, FalconOperator *falconv1alpha1.FalconOperator, obj runtime.Object) error {
	switch t := obj.(type) {
	case client.Object:
		name := t.GetName()
		namespace := t.GetNamespace()
		gvk := t.GetObjectKind().GroupVersionKind()
		log.Info(fmt.Sprintf("Deleting Falcon Admission object %s %s in namespace %s", gvk.Kind, name, namespace))
		err := r.Client.Update(ctx, t)
		if err != nil {
			if errors.IsNotFound(err) {
				log.Info(fmt.Sprintf("Falcon Admission object %s %s does not exist in namespace %s", gvk.Kind, name, namespace))
			}
			return fmt.Errorf("cannot update object %s %s in namespace %s: %v", gvk.Kind, name, namespace, err)
		}

		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			meta.SetStatusCondition(&FalconOperator.Status.Conditions, metav1.Condition{
				Type:    fmt.Sprintf("%sReady", strings.ToUpper(gvk.Kind[:1])+gvk.Kind[1:]),
				Status:  metav1.ConditionTrue,
				Reason:  "Deleted",
				Message: fmt.Sprintf("Successfully deleted %s %s in %s", gvk.Kind, name, namespace),
			})

			return r.Client.Status().Update(ctx, FalconOperator)
		})

		return err
	default:
		return fmt.Errorf("unrecognized kube object type: %T", obj)
	}
}
