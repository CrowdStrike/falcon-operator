/*
Copyright 2021 CrowdStrike
*/

package falcon

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	types "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	imagev1 "github.com/openshift/api/image/v1"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_container"
	"github.com/crowdstrike/gofalcon/pkg/falcon_util"
)

// FalconConfigReconciler reconciles a FalconConfig object
type FalconConfigReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups=image.openshift.io,resources=imagestreams,verbs=get;list;create;update;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the FalconConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *FalconConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("falconconfig", req.NamespacedName)
	logger.Info("Reconciling FalconConfig")

	// your logic here
	falconConfig := &falconv1alpha1.FalconConfig{}
	err := r.Client.Get(ctx, req.NamespacedName, falconConfig)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "Cannot get the Falcon Config")
		return ctrl.Result{}, err
	}

	imageStream := imagev1.ImageStream{}
	err = r.Get(ctx, types.NamespacedName{Name: "falcon-container", Namespace: req.NamespacedName.Namespace}, &imageStream)
	if err != nil && errors.IsNotFound(err) {
		imageStream := &imagev1.ImageStream{
			TypeMeta:   metav1.TypeMeta{APIVersion: imagev1.SchemeGroupVersion.String(), Kind: "ImageStream"},
			ObjectMeta: metav1.ObjectMeta{Name: "falcon-container", Namespace: req.NamespacedName.Namespace},
			Spec:       imagev1.ImageStreamSpec{},
		}
		logger.Info("Creating a new ImageStream", "ImageStream.Namespace", imageStream.Namespace, "ImageStream.Name", imageStream.Name)
		err = r.Create(ctx, imageStream)
		if err != nil {
			logger.Error(err, "Failed to create new ImageStream", "ImageStream.Namespace", imageStream.Namespace, "ImageStream.Name", imageStream.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil

	} else if err != nil {
		logger.Error(err, "Failed to get ImageStream")
		return ctrl.Result{}, err
	}

	refreshImage, err := r.reconcileContainerImage(falconConfig)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("Error when reconciling Falcon Container Image: %w", err)
	}
	if refreshImage {
		err = r.refreshContainerImage(falconConfig)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("Error when reconciling Falcon Container Image: %w", err)
		}
		// TODO: write status
	}

	json, err := falcon_util.PrettyJson(falconConfig)
	if err != nil {
		logger.Error(err, "error")
	} else {
		_ = json
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FalconConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&falconv1alpha1.FalconConfig{}).
		Complete(r)
}

func (r *FalconConfigReconciler) refreshContainerImage(falconConfig *falconv1alpha1.FalconConfig) error {
	image := falcon_container.NewImageRefresher(context.Background(), r.Log, falconConfig.Spec.FalconAPI.ApiConfig())
	return image.Refresh(falconConfig.Spec.WorkloadProtectionSpec.LinuxContainerSpec.Registry)
}

func (r *FalconConfigReconciler) reconcileContainerImage(falconConfig *falconv1alpha1.FalconConfig) (bool, error) {
	if falconConfig.Status.WorkloadProtectionStatus == nil {
		return true, nil
	}
	return false, nil
}
