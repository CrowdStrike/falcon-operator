package falcon

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"time"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	"github.com/crowdstrike/falcon-operator/pkg/aws"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/registry/pulltoken"
	"github.com/crowdstrike/falcon-operator/version"
	"github.com/go-logr/logr"
	imagev1 "github.com/openshift/api/image/v1"
	"github.com/operator-framework/operator-lib/proxy"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
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
)

// FalconImageAnalyzerReconciler reconciles a FalconImageAnalyzer object
type FalconImageAnalyzerReconciler struct {
	client.Client
	Reader client.Reader
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *FalconImageAnalyzerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&falconv1alpha1.FalconImageAnalyzer{}).
		Owns(&corev1.Namespace{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Complete(r)
}

func (r *FalconImageAnalyzerReconciler) GetK8sClient() client.Client {
	return r.Client
}

func (r *FalconImageAnalyzerReconciler) GetK8sReader() client.Reader {
	return r.Reader
}

//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconimageanalyzers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconimageanalyzers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconimageanalyzers/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update
//+kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="security.openshift.io",resources=securitycontextconstraints,resourceNames=privileged,verbs=use
//+kubebuilder:rbac:groups="image.openshift.io",resources=imagestreams,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=create;get;list;update;watch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=create;get;list;update;watch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the FalconImageAnalyzer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *FalconImageAnalyzerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	updated := false
	log := log.FromContext(ctx)

	// Fetch the FalconImageAnalyzer instance
	falconImageAnalyzer := &falconv1alpha1.FalconImageAnalyzer{}
	err := r.Get(ctx, req.NamespacedName, falconImageAnalyzer)
	if err != nil {
		if errors.IsNotFound(err) {
			// If the custom resource is not found then, it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("FalconImageAnalyzer resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}

		log.Error(err, "Failed to get FalconImageAnalyzer resource")
		return ctrl.Result{}, err
	}

	validate, err := k8sutils.CheckRunningPodLabels(r.Reader, ctx, falconImageAnalyzer.Spec.InstallNamespace, common.CRLabels("deployment", falconImageAnalyzer.Name, common.FalconImageAnalyzer))
	if err != nil {
		return ctrl.Result{}, err
	}
	if !validate {
		err = k8sutils.ConditionsUpdate(r.Client, ctx, req, log, falconImageAnalyzer, &falconImageAnalyzer.Status, metav1.Condition{
			Status:             metav1.ConditionFalse,
			Reason:             falconv1alpha1.ReasonReqNotMet,
			Type:               falconv1alpha1.ConditionFailed,
			Message:            "falconImageAnalyzer must not be installed in a namespace with other workloads running. Please change the namespace in the CR configuration.",
			ObservedGeneration: falconImageAnalyzer.GetGeneration(),
		})
		if err != nil {
			return ctrl.Result{}, err
		}
		log.Error(nil, "falconImageAnalyzer is attempting to install in a namespace with existing pods. Please update the CR configuration to a namespace that does not have workoads already running.")
		return ctrl.Result{}, err
	}

	// Let's just set the status as Unknown when no status is available
	if len(falconImageAnalyzer.Status.Conditions) == 0 {
		meta.SetStatusCondition(&falconImageAnalyzer.Status.Conditions, metav1.Condition{Type: falconv1alpha1.ConditionPending, Status: metav1.ConditionUnknown, Reason: "Reconciling", Message: "Starting reconciliation"})
		if err = r.Status().Update(ctx, falconImageAnalyzer); err != nil {
			log.Error(err, "Failed to update FalconImageAnalyzer status")
			return ctrl.Result{}, err
		}

		// Let's re-fetch the Custom Resource after update the status
		// so that we have the latest state of the resource on the cluster and we will avoid
		// raise the issue "the object has been modified, please apply
		// your changes to the latest version and try again" which would re-trigger the reconciliation
		// if we try to update it again in the following operations
		if err := r.Get(ctx, req.NamespacedName, falconImageAnalyzer); err != nil {
			log.Error(err, "Failed to re-fetch FalconImageAnalyzer")
			return ctrl.Result{}, err
		}
	}

	if falconImageAnalyzer.Status.Version != version.Get() {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			err = r.Get(ctx, req.NamespacedName, falconImageAnalyzer)
			if err != nil {
				log.Error(err, "Failed to re-fetch FalconImageAnalyzer for status update")
				return err
			}

			falconImageAnalyzer.Status.Version = version.Get()
			return r.Status().Update(ctx, falconImageAnalyzer)
		})
		if err != nil {
			log.Error(err, "Failed to update FalconImageAnalyzer status for falconImageAnalyzer.Status.Version")
			return ctrl.Result{}, err
		}

	}

	if err := r.reconcileNamespace(ctx, req, log, falconImageAnalyzer); err != nil {
		return ctrl.Result{}, err
	}

	// Image being set will override other image based settings
	if falconImageAnalyzer.Spec.Image != "" {
		if _, err := r.setImageTag(ctx, falconImageAnalyzer); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to set Falcon Image Analyzer version: %v", err)
		}
	} else if os.Getenv("RELATED_IMAGE_IMAGE_ANALYZER") != "" && falconImageAnalyzer.Spec.FalconAPI == nil {
		if _, err := r.setImageTag(ctx, falconImageAnalyzer); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to set Falcon Image Analyzer version: %v", err)
		}
	} else {
		switch falconImageAnalyzer.Spec.Registry.Type {
		case falconv1alpha1.RegistryTypeECR:
			if _, err := aws.UpsertECRRepo(ctx, "falcon-image-analyzer"); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to reconcile ECR repository: %v", err)
			}
		case falconv1alpha1.RegistryTypeOpenshift:
			stream, err := r.reconcileImageStream(ctx, req, log, falconImageAnalyzer)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to reconcile Image Stream")
			}
			if stream == nil {
				return ctrl.Result{}, nil
			}
		}

		if r.imageMirroringEnabled(falconImageAnalyzer) {
			if err := r.PushImage(ctx, log, falconImageAnalyzer); err != nil {
				return ctrl.Result{}, fmt.Errorf("cannot refresh Falcon Image  image: %v", err)
			}
		} else {
			updated, err = r.verifyCrowdStrike(ctx, log, falconImageAnalyzer)
			if updated {
				return ctrl.Result{}, nil
			}
			if err != nil {
				log.Error(err, "Failed to verify CrowdStrike Image  Image Registry access")
				time.Sleep(time.Second * 5)
				return ctrl.Result{RequeueAfter: 5 * time.Second}, err
			}

			if err := r.reconcileRegistrySecret(ctx, req, log, falconImageAnalyzer); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	if err := r.reconcileServiceAccount(ctx, req, log, falconImageAnalyzer); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileClusterRoleBinding(ctx, req, log, falconImageAnalyzer); err != nil {
		return ctrl.Result{}, err
	}

	if falconImageAnalyzer.Spec.FalconSecret.Enabled {
		if err = r.injectFalconSecretData(ctx, falconImageAnalyzer, log); err != nil {
			return ctrl.Result{}, err
		}
	}

	configUpdated, err := r.reconcileConfigMap(ctx, req, log, falconImageAnalyzer)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileImageAnalyzerDeployment(ctx, req, log, falconImageAnalyzer)
	if err != nil {
		return ctrl.Result{}, err
	}

	if configUpdated {
		err = r.imageAnalyzerDeploymentUpdate(ctx, req, log, falconImageAnalyzer)
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	if err := k8sutils.ConditionsUpdate(r.Client, ctx, req, log, falconImageAnalyzer, &falconImageAnalyzer.Status, metav1.Condition{
		Status:             metav1.ConditionTrue,
		Reason:             falconv1alpha1.ReasonInstallSucceeded,
		Type:               falconv1alpha1.ConditionSuccess,
		Message:            "FalconImageAnalyzer installation completed",
		ObservedGeneration: falconImageAnalyzer.GetGeneration(),
	}); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update FalconImageAnalyzer installation completion condition: %v", err)
	}

	return ctrl.Result{}, nil
}

func (r *FalconImageAnalyzerReconciler) reconcileImageAnalyzerDeployment(ctx context.Context, req ctrl.Request, log logr.Logger, falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer) error {
	imageUri, err := r.imageUri(ctx, falconImageAnalyzer)
	if err != nil {
		return fmt.Errorf("unable to determine falcon container image URI: %v", err)
	}

	existingDeployment := &appsv1.Deployment{}
	dep := assets.ImageAnalyzerDeployment(falconImageAnalyzer.Name, falconImageAnalyzer.Spec.InstallNamespace, common.FalconImageAnalyzer, imageUri, falconImageAnalyzer)
	updated := false

	if len(proxy.ReadProxyVarsFromEnv()) > 0 {
		for i, container := range dep.Spec.Template.Spec.Containers {
			dep.Spec.Template.Spec.Containers[i].Env = append(container.Env, proxy.ReadProxyVarsFromEnv()...)
		}
	}

	err = common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: falconImageAnalyzer.Name, Namespace: falconImageAnalyzer.Spec.InstallNamespace}, existingDeployment)
	if err != nil && apierrors.IsNotFound(err) {
		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconImageAnalyzer, &falconImageAnalyzer.Status, dep)
		if err != nil {
			return err
		}

		return nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconImageAnalyzer Deployment")
		return err
	}

	if len(proxy.ReadProxyVarsFromEnv()) > 0 {
		for i, container := range existingDeployment.Spec.Template.Spec.Containers {
			newContainerEnv := common.AppendUniqueEnvVars(container.Env, proxy.ReadProxyVarsFromEnv())
			updatedContainerEnv := common.UpdateEnvVars(container.Env, proxy.ReadProxyVarsFromEnv())
			if !equality.Semantic.DeepEqual(existingDeployment.Spec.Template.Spec.Containers[i].Env, newContainerEnv) {
				existingDeployment.Spec.Template.Spec.Containers[i].Env = newContainerEnv
				updated = true
			}
			if !equality.Semantic.DeepEqual(existingDeployment.Spec.Template.Spec.Containers[i].Env, updatedContainerEnv) {
				existingDeployment.Spec.Template.Spec.Containers[i].Env = updatedContainerEnv
				updated = true
			}
			if updated {
				log.Info("Updating FalconNodeSensor Deployment Proxy Settings")
			}
		}
	}

	if !reflect.DeepEqual(dep.Spec.Template.Spec.Containers[0].Image, existingDeployment.Spec.Template.Spec.Containers[0].Image) {
		existingDeployment.Spec.Template.Spec.Containers[0].Image = dep.Spec.Template.Spec.Containers[0].Image
		updated = true
	}

	if !reflect.DeepEqual(dep.Spec.Template.Spec.Containers[0].ImagePullPolicy, existingDeployment.Spec.Template.Spec.Containers[0].ImagePullPolicy) {
		existingDeployment.Spec.Template.Spec.Containers[0].ImagePullPolicy = dep.Spec.Template.Spec.Containers[0].ImagePullPolicy
		updated = true
	}

	if !reflect.DeepEqual(dep.Spec.Template.Spec.ImagePullSecrets, existingDeployment.Spec.Template.Spec.ImagePullSecrets) {
		existingDeployment.Spec.Template.Spec.ImagePullSecrets = dep.Spec.Template.Spec.ImagePullSecrets
		updated = true
	}

	if !reflect.DeepEqual(dep.Spec.Template.Spec.Containers[0].Ports, existingDeployment.Spec.Template.Spec.Containers[0].Ports) {
		existingDeployment.Spec.Template.Spec.Containers[0].Ports = dep.Spec.Template.Spec.Containers[0].Ports
		updated = true
	}

	if !reflect.DeepEqual(existingDeployment.Spec.Strategy.RollingUpdate, dep.Spec.Strategy.RollingUpdate) {
		existingDeployment.Spec.Strategy.RollingUpdate = dep.Spec.Strategy.RollingUpdate
		updated = true
	}

	if !reflect.DeepEqual(existingDeployment.Spec.Template.Spec.Affinity.NodeAffinity, dep.Spec.Template.Spec.Affinity.NodeAffinity) {
		existingDeployment.Spec.Template.Spec.Affinity.NodeAffinity = dep.Spec.Template.Spec.Affinity.NodeAffinity
		updated = true
	}

	if updated {
		if err := k8sutils.Update(r.Client, ctx, req, log, falconImageAnalyzer, &falconImageAnalyzer.Status, existingDeployment); err != nil {
			return err
		}
	}

	return nil
}

func (r *FalconImageAnalyzerReconciler) reconcileRegistrySecret(ctx context.Context, req ctrl.Request, log logr.Logger, falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer) error {
	falconApiConfig, err := r.falconApiConfig(ctx, falconImageAnalyzer)
	if err != nil {
		return err
	}

	pulltoken, err := pulltoken.CrowdStrike(ctx, falconApiConfig)
	if err != nil {
		return fmt.Errorf("unable to get registry pull token: %v", err)
	}

	secretData := map[string][]byte{corev1.DockerConfigJsonKey: common.CleanDecodedBase64(pulltoken)}
	secret := assets.Secret(common.FalconPullSecretName, falconImageAnalyzer.Spec.InstallNamespace, "falcon-operator", secretData, corev1.SecretTypeDockerConfigJson)
	existingSecret := &corev1.Secret{}

	err = common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: common.FalconPullSecretName, Namespace: falconImageAnalyzer.Spec.InstallNamespace}, existingSecret)
	if err != nil && apierrors.IsNotFound(err) {
		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconImageAnalyzer, &falconImageAnalyzer.Status, secret)
		if err != nil {
			return err
		}

		return nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconImageAnalyzer Registry Pull Secret")
		return err
	}

	if !reflect.DeepEqual(secret.Data, existingSecret.Data) {
		err = k8sutils.Update(r.Client, ctx, req, log, falconImageAnalyzer, &falconImageAnalyzer.Status, existingSecret)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *FalconImageAnalyzerReconciler) reconcileImageStream(ctx context.Context, req ctrl.Request, log logr.Logger, falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer) (*imagev1.ImageStream, error) {
	const imageStreamName = "falcon-image-analyzer"
	namespace := r.imageNamespace(falconImageAnalyzer)
	imageStream := assets.ImageStream(imageStreamName, namespace, common.FalconImageAnalyzer)
	existingImageStream := &imagev1.ImageStream{}

	err := common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: imageStreamName, Namespace: namespace}, existingImageStream)
	if err != nil && apierrors.IsNotFound(err) {
		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconImageAnalyzer, &falconImageAnalyzer.Status, imageStream)
		if err != nil {
			return imageStream, err
		}

		return imageStream, nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconImageAnalyzer ImageStream")
		return existingImageStream, err
	}

	if !reflect.DeepEqual(imageStream.Spec, existingImageStream.Spec) {
		existingImageStream.Spec = imageStream.Spec
		err = k8sutils.Update(r.Client, ctx, req, log, falconImageAnalyzer, &falconImageAnalyzer.Status, existingImageStream)
		if err != nil {
			return existingImageStream, err
		}
	}

	return existingImageStream, nil
}

func (r *FalconImageAnalyzerReconciler) reconcileNamespace(ctx context.Context, req ctrl.Request, log logr.Logger, falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer) error {
	namespace := assets.Namespace(falconImageAnalyzer.Spec.InstallNamespace)
	namespace.ObjectMeta.Labels = common.CRLabels("namespace", falconImageAnalyzer.Spec.InstallNamespace, common.FalconImageAnalyzer)
	existingNamespace := &corev1.Namespace{}

	err := common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: falconImageAnalyzer.Spec.InstallNamespace}, existingNamespace)
	if err != nil && apierrors.IsNotFound(err) {
		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconImageAnalyzer, &falconImageAnalyzer.Status, namespace)
		if err != nil {
			return err
		}

		return nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconImageAnalyzer Namespace")
		return err
	}

	return nil
}

func (r *FalconImageAnalyzerReconciler) imageAnalyzerDeploymentUpdate(ctx context.Context, req ctrl.Request, log logr.Logger, falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer) error {
	existingDeployment := &appsv1.Deployment{}
	configVersion := "falcon.config.version"
	err := common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: falconImageAnalyzer.Name, Namespace: falconImageAnalyzer.Spec.InstallNamespace}, existingDeployment)
	if err != nil && apierrors.IsNotFound(err) {
		return err
	} else if err != nil {
		log.Error(err, "Failed to get FalconImageAnalyzer Deployment")
		return err
	}

	_, ok := existingDeployment.Spec.Template.Annotations[configVersion]
	if ok {
		i, err := strconv.Atoi(existingDeployment.Spec.Template.Annotations[configVersion])
		if err != nil {
			return err
		}

		existingDeployment.Spec.Template.Annotations[configVersion] = strconv.Itoa(i + 1)
	} else {
		existingDeployment.Spec.Template.Annotations[configVersion] = "1"
	}

	log.Info("Rolling FalconImageAnalyzer Deployment due to non-deployment configuration change")
	if err := k8sutils.Update(r.Client, ctx, req, log, falconImageAnalyzer, &falconImageAnalyzer.Status, existingDeployment); err != nil {
		return err
	}

	return nil
}

func (r *FalconImageAnalyzerReconciler) injectFalconSecretData(
	ctx context.Context,
	falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer,
	logger logr.Logger,
) error {
	logger.Info("injecting Falcon secret data into Spec.Falcon and Spec.FalconAPI - sensitive manifest values will be overwritten with values in k8s secret")

	return k8sutils.InjectFalconSecretData(ctx, r, falconImageAnalyzer)
}
