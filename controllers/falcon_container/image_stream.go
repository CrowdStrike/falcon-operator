package falcon

import (
	"context"
	"fmt"
	"reflect"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	types "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	imagev1 "github.com/openshift/api/image/v1"
)

const (
	imageStreamName = "falcon-sidecar-container"
)

func (r *FalconContainerReconciler) reconcileImageStream(ctx context.Context, log logr.Logger, falconContainer *falconv1alpha1.FalconContainer) (*imagev1.ImageStream, error) {
	imageStream := assets.ImageStream(imageStreamName, r.imageNamespace(falconContainer))
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
