package controllers

import (
	"context"
	"reflect"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	"github.com/crowdstrike/falcon-operator/internal/controller/common/image"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/registry/pulltoken"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *FalconClusterGuardReconciler) injectFalconSecretData(ctx context.Context, fcg *falconv1alpha1.FalconClusterGuard) error {
	r.log.V(1).Info("Injecting Falcon secret data into Spec.FalconAPI - sensitive manifest values will be overwritten with values in k8s secret")
	return k8sutils.InjectFalconSecretData(ctx, r, fcg)
}

func (r *FalconClusterGuardReconciler) reconcileImagePullSecret(ctx context.Context, req ctrl.Request, fcg *falconv1alpha1.FalconClusterGuard) error {
	token, err := pulltoken.CrowdStrike(ctx, r.apiConfig)
	if err != nil {
		r.log.Error(err, "Failed to get CrowdStrike registry pull token")
		return err
	}

	secretData := map[string][]byte{corev1.DockerConfigJsonKey: common.CleanDecodedBase64(token)}
	desired := assets.Secret(common.FalconPullSecretName, fcg.Spec.InstallNamespace, "falcon-operator", secretData, corev1.SecretTypeDockerConfigJson)

	existing := &corev1.Secret{}
	err = common.GetNamespacedObject(ctx, r.Client, r.Reader,
		types.NamespacedName{Name: common.FalconPullSecretName, Namespace: fcg.Spec.InstallNamespace}, existing)
	if err != nil && apierrors.IsNotFound(err) {
		return k8sutils.Create(r.Client, r.RuntimeScheme, ctx, req, r.log, fcg, &fcg.Status, desired)
	} else if err != nil {
		r.log.Error(err, "Failed to get FalconClusterGuard registry pull secret")
		return err
	}

	if !reflect.DeepEqual(desired.Data, existing.Data) {
		existing.Data = desired.Data
		existing.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
		return k8sutils.Update(r.Client, ctx, req, r.log, fcg, &fcg.Status, existing)
	}
	return nil
}

// syncDiscoveredImage keeps the ImageReady condition and status.sensor in step with the discovered imageURI.
// The discovered tag is version locked in status, so a repository change (e.g. a new build stage) clears it to force
// rediscovery, and a newly discovered tag (e.g. after a version change) is persisted so the registry is not queried again.
// It returns true when status was updated; the caller must requeue, since the update refreshes fcg from the API server
// and drops data injected from the FalconSecret.
func (r *FalconClusterGuardReconciler) syncDiscoveredImage(ctx context.Context, fcg *falconv1alpha1.FalconClusterGuard, imageURI string) (bool, error) {
	imageReady := meta.FindStatusCondition(fcg.Status.Conditions, falconv1alpha1.ConditionImageReady)
	if imageReady == nil || imageReady.Message == imageURI {
		return false, nil
	}

	if image.Repository(imageReady.Message) != image.Repository(imageURI) {
		r.log.Info("Falcon Cluster Guard image repository changed; rediscovering image tag", "previous", imageReady.Message, "repository", image.Repository(imageURI))
		fcg.Status.Sensor = nil
		meta.RemoveStatusCondition(&fcg.Status.Conditions, falconv1alpha1.ConditionImageReady)
	} else {
		r.log.Info("Falcon Cluster Guard image changed", "previous", imageReady.Message, "image", imageURI)
		meta.SetStatusCondition(&fcg.Status.Conditions, metav1.Condition{
			Status:             metav1.ConditionTrue,
			Reason:             falconv1alpha1.ReasonDiscovered,
			Message:            imageURI,
			Type:               falconv1alpha1.ConditionImageReady,
			ObservedGeneration: fcg.GetGeneration(),
		})
	}

	if err := r.Status().Update(ctx, fcg); err != nil {
		r.log.Error(err, "Failed to update FalconClusterGuard image status")
		return false, err
	}
	return true, nil
}

// clearDiscoveredImage discards a previously discovered image once the image is no longer discovered (e.g. image or
// registry.type changed), so its tag is not reused if discovery is enabled again.
// It returns true when status was updated; the caller must requeue, as with syncDiscoveredImage.
func (r *FalconClusterGuardReconciler) clearDiscoveredImage(ctx context.Context, fcg *falconv1alpha1.FalconClusterGuard) (bool, error) {
	if meta.FindStatusCondition(fcg.Status.Conditions, falconv1alpha1.ConditionImageReady) == nil {
		return false, nil
	}

	r.log.Info("Falcon Cluster Guard image is no longer discovered from the CrowdStrike registry; clearing discovered image")
	fcg.Status.Sensor = nil
	meta.RemoveStatusCondition(&fcg.Status.Conditions, falconv1alpha1.ConditionImageReady)
	if err := r.Status().Update(ctx, fcg); err != nil {
		r.log.Error(err, "Failed to update FalconClusterGuard image status")
		return false, err
	}
	return true, nil
}

// setFinalizer adds (present=true) or removes (present=false) the FalconFinalizer on the CR.
// It patches only metadata.finalizers on a freshly fetched copy, so in-memory spec changes such as
// injected FalconSecret credentials are never written back to the CR.
func (r *FalconClusterGuardReconciler) setFinalizer(ctx context.Context, fcg *falconv1alpha1.FalconClusterGuard, present bool) error {
	if controllerutil.ContainsFinalizer(fcg, common.FalconFinalizer) == present {
		return nil
	}

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &falconv1alpha1.FalconClusterGuard{}
		if err := r.Get(ctx, client.ObjectKeyFromObject(fcg), latest); err != nil {
			return err
		}
		patchBase := latest.DeepCopy()
		var changed bool
		if present {
			changed = controllerutil.AddFinalizer(latest, common.FalconFinalizer)
		} else {
			changed = controllerutil.RemoveFinalizer(latest, common.FalconFinalizer)
		}
		if changed {
			if err := r.Patch(ctx, latest, client.MergeFromWithOptions(patchBase, client.MergeFromWithOptimisticLock{})); err != nil {
				return err
			}
		}
		fcg.SetFinalizers(latest.GetFinalizers())
		return nil
	})
	if err != nil {
		r.log.Error(err, "Failed to update FalconClusterGuard finalizers", "finalizer", common.FalconFinalizer, "present", present)
		return err
	}
	r.log.Info("Updated FalconClusterGuard finalizers", "finalizer", common.FalconFinalizer, "present", present)
	return nil
}
