package falcon

import (
	"context"
	"fmt"
	"reflect"

	"github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	imagev1 "github.com/openshift/api/image/v1"
)

const (
	imageStreamName = "falcon-container"
)

func (r *FalconContainerReconciler) reconcileImageStream(ctx context.Context, log logr.Logger, falconContainer *v1alpha1.FalconContainer) (*imagev1.ImageStream, error) {
	imageStream := r.newImageStream(falconContainer)
	existingImageStream := &imagev1.ImageStream{}

	err := r.Client.Get(ctx, types.NamespacedName{Name: imageStreamName, Namespace: r.imageNamespace(falconContainer)}, existingImageStream)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = ctrl.SetControllerReference(falconContainer, imageStream, r.Scheme); err != nil {
				return &imagev1.ImageStream{}, fmt.Errorf("unable to set controller reference on image stream %s: %v", imageStreamName, err)
			}

			return imageStream, r.Create(ctx, log, falconContainer, imageStream)
		}

		return &imagev1.ImageStream{}, fmt.Errorf("unable to query existing image stream %s: %v", imageStreamName, err)
	}

	if reflect.DeepEqual(imageStream.Spec, existingImageStream.Spec) {
		return existingImageStream, nil
	}

	existingImageStream.Spec = imageStream.Spec

	return existingImageStream, r.Update(ctx, log, falconContainer, existingImageStream)
}

func (r *FalconContainerReconciler) newImageStream(falconContainer *v1alpha1.FalconContainer) *imagev1.ImageStream {
	return &imagev1.ImageStream{
		TypeMeta:   metav1.TypeMeta{APIVersion: imagev1.SchemeGroupVersion.String(), Kind: "ImageStream"},
		ObjectMeta: metav1.ObjectMeta{Name: imageStreamName, Namespace: r.imageNamespace(falconContainer)},
		Spec:       imagev1.ImageStreamSpec{},
	}
}
