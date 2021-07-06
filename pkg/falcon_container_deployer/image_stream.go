package falcon_container_deployer

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"

	imagev1 "github.com/openshift/api/image/v1"
)

const (
	IMAGE_STREAM_NAME = "falcon-container"
)

func (d *FalconContainerDeployer) UpsertImageStream() (stream *imagev1.ImageStream, err error) {
	stream, err = d.GetImageStream()
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, d.CreateImageStream()
		} else if meta.IsNoMatchError(err) {
			return nil, fmt.Errorf("Image Stream Kind is not available on the cluster: %v", err)
		}
	}
	return stream, err
}

func (d *FalconContainerDeployer) GetImageStream() (*imagev1.ImageStream, error) {
	var stream imagev1.ImageStream
	err := d.Client.Get(d.Ctx, types.NamespacedName{Name: IMAGE_STREAM_NAME, Namespace: d.Namespace()}, &stream)
	return &stream, err
}

func (d *FalconContainerDeployer) CreateImageStream() error {
	imageStream := &imagev1.ImageStream{
		TypeMeta:   metav1.TypeMeta{APIVersion: imagev1.SchemeGroupVersion.String(), Kind: "ImageStream"},
		ObjectMeta: metav1.ObjectMeta{Name: IMAGE_STREAM_NAME, Namespace: d.Namespace()},
		Spec:       imagev1.ImageStreamSpec{},
	}
	err := d.Client.Create(d.Ctx, imageStream)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			d.Log.Error(err, "Failed to create new ImageStream", "ImageStream.Namespace", imageStream.Namespace, "ImageStream.Name", imageStream.Name)
			return err
		}
	} else {
		d.Log.Info("Created a new ImageStream", "ImageStream.Namespace", d.Namespace(), "ImageStream.Name", imageStream.Name)
	}
	return nil
}
