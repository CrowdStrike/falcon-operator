package image_analyzer

import (
	"context"
	"reflect"

	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/operator-framework/operator-lib/proxy"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

func (ia *ImageAnalyzer) deployment() *appsv1.Deployment {
	return assets.ImageAnalyzerDeploymentFromConfig(
		pkgcommon.FCGImageAnalyzerDeploymentName,
		ia.cfg.InstallNamespace,
		pkgcommon.FCGImageAnalyzerComponentName,
		ia.cfg.Image,
		ia.cfg.ImagePullPolicy,
		ia.cfg.ImagePullSecrets,
		pkgcommon.FCGImageAnalyzerServiceAccountName,
		pkgcommon.FCGImageAnalyzerConfigMapName,
		pkgcommon.FCGImageAnalyzerTLSSecretName,
		ia.cfg.ImageAnalyzerSpec,
	)
}

func (ia *ImageAnalyzer) reconcileDeployment(ctx context.Context) error {
	log := ia.r.GetLog()
	dep := ia.deployment()

	// Inject operator proxy env vars into the desired spec containers before create/update.
	if len(proxy.ReadProxyVarsFromEnv()) > 0 {
		for i, container := range dep.Spec.Template.Spec.Containers {
			dep.Spec.Template.Spec.Containers[i].Env = append(container.Env, proxy.ReadProxyVarsFromEnv()...)
		}
	}

	key := types.NamespacedName{Name: pkgcommon.FCGImageAnalyzerDeploymentName, Namespace: ia.cfg.InstallNamespace}
	existing := &appsv1.Deployment{}
	found, err := k8sutils.GetOrCreate(ctx, ia.r, ia.cfg.Request, ia.cfg.Owner, ia.cfg.Status, dep, existing, key,
		"Failed to get FCG Image Analyzer Deployment")
	if !found || err != nil {
		return err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := pkgcommon.GetNamespacedObject(ctx, ia.r, ia.r.GetK8sReader(), key, existing); err != nil {
			return err
		}

		updated := false
		container := dep.Spec.Template.Spec.Containers[0]
		existingContainer := &existing.Spec.Template.Spec.Containers[0]

		if !reflect.DeepEqual(container.Image, existingContainer.Image) {
			log.V(1).Info("Updating FCG Image Analyzer Deployment: container image changed",
				"old", existingContainer.Image, "new", container.Image)
			existingContainer.Image = container.Image
			updated = true
		}

		if !reflect.DeepEqual(container.ImagePullPolicy, existingContainer.ImagePullPolicy) {
			log.V(1).Info("Updating FCG Image Analyzer Deployment: container ImagePullPolicy changed",
				"old", existingContainer.ImagePullPolicy, "new", container.ImagePullPolicy)
			existingContainer.ImagePullPolicy = container.ImagePullPolicy
			updated = true
		}

		if !reflect.DeepEqual(container.Ports, existingContainer.Ports) {
			log.V(1).Info("Updating FCG Image Analyzer Deployment: container ports changed",
				"old", existingContainer.Ports, "new", container.Ports)
			existingContainer.Ports = container.Ports
			updated = true
		}

		if !equality.Semantic.DeepEqual(container.Resources, existingContainer.Resources) {
			log.V(1).Info("Updating FCG Image Analyzer Deployment: container resources changed",
				"old", existingContainer.Resources, "new", container.Resources)
			existingContainer.Resources = container.Resources
			updated = true
		}

		// Merge existing proxy env vars from the cluster into the spec env before comparing,
		// to avoid stripping proxy vars that were injected by the operator environment.
		mergedEnv := pkgcommon.MergeEnvVars(container.Env, existingContainer.Env, pkgcommon.ProxyEnvNamesWithLowerCase())
		if !equality.Semantic.DeepEqual(mergedEnv, existingContainer.Env) {
			log.V(1).Info("Updating FCG Image Analyzer Deployment: container env changed",
				"old", existingContainer.Env, "new", mergedEnv)
			existingContainer.Env = mergedEnv
			updated = true
		}

		// Reconcile proxy env vars: append any new proxy vars from the operator environment,
		// and update the values of any existing proxy vars that have changed.
		if len(proxy.ReadProxyVarsFromEnv()) > 0 {
			oldEnv := existingContainer.Env
			envAfterAppend := pkgcommon.AppendUniqueEnvVars(existingContainer.Env, proxy.ReadProxyVarsFromEnv())
			finalEnv := pkgcommon.UpdateEnvVars(envAfterAppend, proxy.ReadProxyVarsFromEnv())
			if !equality.Semantic.DeepEqual(oldEnv, finalEnv) {
				log.V(1).Info("Updating FCG Image Analyzer Deployment: proxy env vars changed",
					"old", oldEnv, "new", finalEnv)
				existingContainer.Env = finalEnv
				updated = true
			}
		}

		if !reflect.DeepEqual(dep.Spec.Template.Spec.ImagePullSecrets, existing.Spec.Template.Spec.ImagePullSecrets) {
			log.V(1).Info("Updating FCG Image Analyzer Deployment: ImagePullSecrets changed",
				"old", existing.Spec.Template.Spec.ImagePullSecrets,
				"new", dep.Spec.Template.Spec.ImagePullSecrets)
			existing.Spec.Template.Spec.ImagePullSecrets = dep.Spec.Template.Spec.ImagePullSecrets
			updated = true
		}

		if !equality.Semantic.DeepEqual(existing.Spec.Strategy.RollingUpdate, dep.Spec.Strategy.RollingUpdate) {
			log.V(1).Info("Updating FCG Image Analyzer Deployment: RollingUpdate strategy changed",
				"old", existing.Spec.Strategy.RollingUpdate,
				"new", dep.Spec.Strategy.RollingUpdate)
			existing.Spec.Strategy.RollingUpdate = dep.Spec.Strategy.RollingUpdate
			updated = true
		}

		if dep.Spec.Template.Spec.Affinity != nil {
			if existing.Spec.Template.Spec.Affinity == nil {
				existing.Spec.Template.Spec.Affinity = &corev1.Affinity{}
			}
			if !reflect.DeepEqual(dep.Spec.Template.Spec.Affinity.NodeAffinity, existing.Spec.Template.Spec.Affinity.NodeAffinity) {
				log.V(1).Info("Updating FCG Image Analyzer Deployment: NodeAffinity changed",
					"old", existing.Spec.Template.Spec.Affinity.NodeAffinity,
					"new", dep.Spec.Template.Spec.Affinity.NodeAffinity)
				existing.Spec.Template.Spec.Affinity.NodeAffinity = dep.Spec.Template.Spec.Affinity.NodeAffinity
				updated = true
			}
		}

		// Preserve tolerations added outside the operator (e.g. by platform admission controllers).
		mergedTolerations := dep.Spec.Template.Spec.Tolerations
		for _, existingTol := range existing.Spec.Template.Spec.Tolerations {
			found := false
			for _, specTol := range dep.Spec.Template.Spec.Tolerations {
				if existingTol.Key == specTol.Key && existingTol.Effect == specTol.Effect {
					found = true
					break
				}
			}
			if !found {
				mergedTolerations = append(mergedTolerations, existingTol)
			}
		}
		if !equality.Semantic.DeepEqual(existing.Spec.Template.Spec.Tolerations, mergedTolerations) {
			log.V(1).Info("Updating FCG Image Analyzer Deployment: Tolerations changed",
				"old", existing.Spec.Template.Spec.Tolerations,
				"new", mergedTolerations)
			existing.Spec.Template.Spec.Tolerations = mergedTolerations
			updated = true
		}

		if updated {
			existing.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("Deployment"))
			return k8sutils.Update(ia.r, ctx, ia.cfg.Request, log, ia.cfg.Owner, ia.cfg.Status, existing)
		}
		return nil
	})
	if err != nil {
		log.Error(err, "Failed to update FCG Image Analyzer Deployment after retries")
		return err
	}
	return nil
}
