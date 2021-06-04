package falcon

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	imagev1 "github.com/openshift/api/image/v1"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_container_deployer"
)

const (
	IMAGE_STREAM_NAME = "falcon-container"
)

func (r *FalconConfigReconciler) phasePendingReconcile(ctx context.Context, instance *falconv1alpha1.FalconConfig, logger logr.Logger) (ctrl.Result, error) {
	logger.Info("Phase: Pending")
	d := falcon_container_deployer.FalconContainerDeployer{
		Ctx:      ctx,
		Client:   r.Client,
		Log:      logger,
		Instance: instance,
	}

	_, err := r.imageStream(ctx, instance.ObjectMeta.Namespace)
	if err != nil && errors.IsNotFound(err) {
		imageStream := &imagev1.ImageStream{
			TypeMeta:   metav1.TypeMeta{APIVersion: imagev1.SchemeGroupVersion.String(), Kind: "ImageStream"},
			ObjectMeta: metav1.ObjectMeta{Name: IMAGE_STREAM_NAME, Namespace: instance.ObjectMeta.Namespace},
			Spec:       imagev1.ImageStreamSpec{},
		}
		err = r.Client.Create(ctx, imageStream)
		if err != nil {
			if !errors.IsAlreadyExists(err) {
				logger.Error(err, "Failed to create new ImageStream", "ImageStream.Namespace", imageStream.Namespace, "ImageStream.Name", imageStream.Name)
				return ctrl.Result{}, err
			}
		}
		logger.Info("Created a new ImageStream", "ImageStream.Namespace", imageStream.Namespace, "ImageStream.Name", imageStream.Name)
		// It takes few moment for the ImageStream to be ready
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil

	} else if err != nil {
		return d.Error("Failed to get ImageStream", err)
	}

	instance.Status.ErrorMessage = ""
	instance.Status.Phase = falconv1alpha1.PhaseBuilding

	err = r.Client.Status().Update(ctx, instance)
	return ctrl.Result{}, err
}

func (r *FalconConfigReconciler) imageStream(ctx context.Context, namespace string) (stream imagev1.ImageStream, err error) {
	err = r.Client.Get(ctx, types.NamespacedName{Name: IMAGE_STREAM_NAME, Namespace: namespace}, &stream)
	return
}
