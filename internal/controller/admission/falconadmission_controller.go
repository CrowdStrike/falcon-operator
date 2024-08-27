package controllers

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
	"github.com/crowdstrike/falcon-operator/pkg/tls"
	"github.com/crowdstrike/falcon-operator/version"
	"github.com/go-logr/logr"
	imagev1 "github.com/openshift/api/image/v1"
	"github.com/operator-framework/operator-lib/proxy"
	arv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// FalconAdmissionReconciler reconciles a FalconAdmission object
type FalconAdmissionReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	OpenShift bool
}

// SetupWithManager sets up the controller with the Manager.
func (r *FalconAdmissionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&falconv1alpha1.FalconAdmission{}).
		Owns(&corev1.Namespace{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.ResourceQuota{}).
		Owns(&corev1.Secret{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&arv1.ValidatingWebhookConfiguration{}).
		Complete(r)
}

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
// the FalconAdmission object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *FalconAdmissionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	updated := false
	log := log.FromContext(ctx)

	// Fetch the FalconAdmission instance
	falconAdmission := &falconv1alpha1.FalconAdmission{}
	err := r.Get(ctx, req.NamespacedName, falconAdmission)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then, it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("FalconAdmission resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}

		log.Error(err, "Failed to get FalconAdmission resource")
		return ctrl.Result{}, err
	}

	validate, err := k8sutils.CheckRunningPodLabels(r.Client, ctx, falconAdmission.Spec.InstallNamespace, common.CRLabels("deployment", falconAdmission.Name, common.FalconAdmissionController))
	if err != nil {
		return ctrl.Result{}, err
	}
	if !validate {
		err = k8sutils.ConditionsUpdate(r.Client, ctx, req, log, falconAdmission, &falconAdmission.Status, metav1.Condition{
			Status:             metav1.ConditionFalse,
			Reason:             falconv1alpha1.ReasonReqNotMet,
			Type:               falconv1alpha1.ConditionFailed,
			Message:            "FalconAdmission must not be installed in a namespace with other workloads running. Please change the namespace in the CR configuration.",
			ObservedGeneration: falconAdmission.GetGeneration(),
		})
		if err != nil {
			return ctrl.Result{}, err
		}
		log.Error(nil, "FalconAdmission is attempting to install in a namespace with existing pods. Please update the CR configuration to a namespace that does not have workoads already running.")
		return ctrl.Result{}, nil
	}

	// Let's just set the status as Unknown when no status is available
	if len(falconAdmission.Status.Conditions) == 0 {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			meta.SetStatusCondition(&falconAdmission.Status.Conditions, metav1.Condition{Type: falconv1alpha1.ConditionPending, Status: metav1.ConditionUnknown, Reason: "Reconciling", Message: "Starting reconciliation"})
			return r.Status().Update(ctx, falconAdmission)
		})
		if err != nil {
			log.Error(err, "Failed to update FalconAdmission status")
			return ctrl.Result{}, err
		}
	}

	if falconAdmission.Status.Version != version.Get() {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			err := r.Get(ctx, req.NamespacedName, falconAdmission)
			if err != nil {
				return err
			}
			falconAdmission.Status.Version = version.Get()
			return r.Status().Update(ctx, falconAdmission)
		})
		if err != nil {
			log.Error(err, "Failed to update FalconAdmission status for falconAdmission.Status.Version")
			return ctrl.Result{}, err
		}
	}

	if err := r.reconcileNamespace(ctx, req, log, falconAdmission); err != nil {
		return ctrl.Result{}, err
	}

	// Image being set will override other image based settings
	if falconAdmission.Spec.Image != "" {
		if _, err := r.setImageTag(ctx, falconAdmission); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to set Falcon Admission Image version: %v", err)
		}
	} else if os.Getenv("RELATED_IMAGE_ADMISSION_CONTROLLER") != "" && falconAdmission.Spec.FalconAPI == nil {
		if _, err := r.setImageTag(ctx, falconAdmission); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to set Falcon Admission Image version: %v", err)
		}
	} else {
		switch falconAdmission.Spec.Registry.Type {
		case falconv1alpha1.RegistryTypeECR:
			if _, err := aws.UpsertECRRepo(ctx, "falcon-kac"); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to reconcile ECR repository: %v", err)
			}
		case falconv1alpha1.RegistryTypeOpenshift:
			stream, err := r.reconcileImageStream(ctx, req, log, falconAdmission)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to reconcile Image Stream")
			}
			if stream == nil {
				return ctrl.Result{}, nil
			}
		}

		// Create a CA Bundle ConfigMap if CACertificate attribute is set; overridden by the presence of a CACertificateConfigMap value
		if falconAdmission.Spec.Registry.TLS.CACertificateConfigMap == "" && falconAdmission.Spec.Registry.TLS.CACertificate != "" {
			if _, err := r.reconcileRegistryCABundleConfigMap(ctx, req, log, falconAdmission); err != nil {
				return ctrl.Result{}, err
			}
		}

		if r.imageMirroringEnabled(falconAdmission) {
			if err := r.PushImage(ctx, log, falconAdmission); err != nil {
				return ctrl.Result{}, fmt.Errorf("cannot refresh Falcon Admission image: %v", err)
			}
		} else {
			updated, err = r.verifyCrowdStrike(ctx, log, falconAdmission)
			if updated {
				return ctrl.Result{}, nil
			}
			if err != nil {
				log.Error(err, "Failed to verify CrowdStrike Admission Image Registry access")
				time.Sleep(time.Second * 5)
				return ctrl.Result{RequeueAfter: 5 * time.Second}, err
			}

			if err := r.reconcileRegistrySecret(ctx, req, log, falconAdmission); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	if err := r.reconcileServiceAccount(ctx, req, log, falconAdmission); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileClusterRoleBinding(ctx, req, log, falconAdmission); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileRole(ctx, req, log, falconAdmission); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileRoleBinding(ctx, req, log, falconAdmission); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileResourceQuota(ctx, req, log, falconAdmission); err != nil {
		return ctrl.Result{}, err
	}

	admissionTLSSecret, err := r.reconcileTLSSecret(ctx, req, log, falconAdmission)
	if err != nil {
		return ctrl.Result{}, err
	}

	configUpdated, err := r.reconcileConfigMap(ctx, req, log, falconAdmission)
	if err != nil {
		return ctrl.Result{}, err
	}

	serviceUpdated, err := r.reconcileService(ctx, req, log, falconAdmission)
	if err != nil {
		return ctrl.Result{}, err
	}

	webhookUpdated, err := r.reconcileAdmissionValidatingWebHook(ctx, req, log, falconAdmission, admissionTLSSecret.Data["ca.crt"])
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileAdmissionDeployment(ctx, req, log, falconAdmission)
	if err != nil {
		return ctrl.Result{}, err
	}

	pod, err := k8sutils.GetReadyPod(r.Client, ctx, falconAdmission.Spec.InstallNamespace, map[string]string{common.FalconComponentKey: common.FalconAdmissionController})
	if err != nil && err != k8sutils.ErrNoWebhookServicePodReady {
		log.Error(err, "Failed to find Ready admission controller pod")
		return ctrl.Result{}, err
	}
	if pod.Name == "" {
		log.Info("Looking for a Ready admission controller pod", "namespace", falconAdmission.Spec.InstallNamespace)
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	if configUpdated || serviceUpdated || webhookUpdated {
		err = r.admissionDeploymentUpdate(ctx, req, log, falconAdmission)
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	if err := k8sutils.ConditionsUpdate(r.Client, ctx, req, log, falconAdmission, &falconAdmission.Status, metav1.Condition{
		Status:             metav1.ConditionTrue,
		Reason:             falconv1alpha1.ReasonInstallSucceeded,
		Type:               falconv1alpha1.ConditionSuccess,
		Message:            "FalconAdmission installation completed",
		ObservedGeneration: falconAdmission.GetGeneration(),
	}); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update FalconAdmission installation completion condition: %v", err)
	}

	return ctrl.Result{}, nil
}

func (r *FalconAdmissionReconciler) reconcileResourceQuota(ctx context.Context, req ctrl.Request, log logr.Logger, falconAdmission *falconv1alpha1.FalconAdmission) error {
	existingRQ := &corev1.ResourceQuota{}
	defaultPodLimit := "5"

	if falconAdmission.Spec.ResQuota.PodLimit != "" {
		defaultPodLimit = falconAdmission.Spec.ResQuota.PodLimit
	}

	rq := assets.ResourceQuota(falconAdmission.Name, falconAdmission.Spec.InstallNamespace, common.FalconAdmissionController, defaultPodLimit)

	err := r.Get(ctx, types.NamespacedName{Name: falconAdmission.Name, Namespace: falconAdmission.Spec.InstallNamespace}, existingRQ)
	if err != nil && apierrors.IsNotFound(err) {
		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconAdmission, &falconAdmission.Status, rq)
		if err != nil {
			return err
		}

		return nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconAdmission ResourceQuota")
		return err
	}

	podLimit := resource.MustParse(defaultPodLimit)
	if existingRQ.Spec.Hard["pods"] != podLimit {
		err = k8sutils.Update(r.Client, ctx, req, log, falconAdmission, &falconAdmission.Status, rq)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *FalconAdmissionReconciler) reconcileTLSSecret(ctx context.Context, req ctrl.Request, log logr.Logger, falconAdmission *falconv1alpha1.FalconAdmission) (*corev1.Secret, error) {
	existingTLSSecret := &corev1.Secret{}
	name := falconAdmission.Name + "-tls"

	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: falconAdmission.Spec.InstallNamespace}, existingTLSSecret)
	if err != nil && apierrors.IsNotFound(err) {
		validity := 3650
		if falconAdmission.Spec.AdmissionConfig.TLS.Validity != nil {
			validity = *falconAdmission.Spec.AdmissionConfig.TLS.Validity
		}

		certInfo := tls.CertInfo{
			CommonName: fmt.Sprintf("%s.%s.svc", falconAdmission.Name, falconAdmission.Spec.InstallNamespace),
			DNSNames: []string{fmt.Sprintf("%s.%s.svc", falconAdmission.Name, falconAdmission.Spec.InstallNamespace), fmt.Sprintf("%s.%s.svc.cluster.local", falconAdmission.Name, falconAdmission.Spec.InstallNamespace),
				fmt.Sprintf("%s.%s", falconAdmission.Name, falconAdmission.Spec.InstallNamespace), falconAdmission.Name},
		}

		c, k, b, err := tls.CertSetup(falconAdmission.Spec.InstallNamespace, validity, certInfo)
		if err != nil {
			log.Error(err, "Failed to generate FalconAdmission PKI")
			return &corev1.Secret{}, err
		}

		secretData := map[string][]byte{
			"tls.crt": c,
			"tls.key": k,
			"ca.crt":  b,
		}

		admissionTLSSecret := assets.Secret(name, falconAdmission.Spec.InstallNamespace, common.FalconAdmissionController, secretData, corev1.SecretTypeTLS)
		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconAdmission, &falconAdmission.Status, admissionTLSSecret)
		if err != nil {
			return &corev1.Secret{}, err
		}
		return admissionTLSSecret, nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconAdmission TLS Secret")
		return &corev1.Secret{}, err
	}

	return existingTLSSecret, nil
}

func (r *FalconAdmissionReconciler) reconcileService(ctx context.Context, req ctrl.Request, log logr.Logger, falconAdmission *falconv1alpha1.FalconAdmission) (bool, error) {
	existingService := &corev1.Service{}
	selector := map[string]string{common.FalconComponentKey: common.FalconAdmissionController}
	port := int32(443)

	if falconAdmission.Spec.AdmissionConfig.Port != nil {
		port = *falconAdmission.Spec.AdmissionConfig.Port
	}

	service := assets.Service(falconAdmission.Name, falconAdmission.Spec.InstallNamespace, common.FalconAdmissionController, selector, common.FalconAdmissionServiceHTTPSName, port)

	err := r.Get(ctx, types.NamespacedName{Name: falconAdmission.Name, Namespace: falconAdmission.Spec.InstallNamespace}, existingService)
	if err != nil && apierrors.IsNotFound(err) {
		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconAdmission, &falconAdmission.Status, service)
		if err != nil {
			return false, err
		}

		return false, nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconAdmission Service")
		return false, err
	}

	if !reflect.DeepEqual(service.Spec.Ports, existingService.Spec.Ports) {
		existingService.Spec.Ports = service.Spec.Ports
		if err := k8sutils.Update(r.Client, ctx, req, log, falconAdmission, &falconAdmission.Status, existingService); err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

func (r *FalconAdmissionReconciler) reconcileAdmissionValidatingWebHook(ctx context.Context, req ctrl.Request, log logr.Logger, falconAdmission *falconv1alpha1.FalconAdmission, cabundle []byte) (bool, error) {
	existingWebhook := &arv1.ValidatingWebhookConfiguration{}
	disabledNamespaces := append(common.DefaultDisabledNamespaces, falconAdmission.Spec.AdmissionConfig.DisabledNamespaces.Namespaces...)
	const webhookName = "validating.admission.falcon.crowdstrike.com"
	failPolicy := arv1.Ignore
	port := int32(443)

	if r.OpenShift {
		ocpNS, err := k8sutils.GetOpenShiftNamespaceNamesSort(ctx, r.Client)
		if err != nil {
			return false, err
		}
		disabledNamespaces = append(disabledNamespaces, ocpNS...)
	}

	falconNS, err := k8sutils.GetRunningFalconNS(r.Client, ctx)
	if err != nil {
		return false, err
	}

	disabledNamespaces = append(disabledNamespaces, falconNS...)

	if falconAdmission.Spec.AdmissionConfig.FailurePolicy != "" {
		failPolicy = falconAdmission.Spec.AdmissionConfig.FailurePolicy
	}

	if falconAdmission.Spec.AdmissionConfig.Port != nil {
		port = *falconAdmission.Spec.AdmissionConfig.Port
	}

	webhook := assets.ValidatingWebhook(falconAdmission.Name, falconAdmission.Spec.InstallNamespace, webhookName, cabundle, port, failPolicy, disabledNamespaces)
	updated := false

	err = r.Get(ctx, types.NamespacedName{Name: webhookName}, existingWebhook)
	if err != nil && apierrors.IsNotFound(err) {
		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconAdmission, &falconAdmission.Status, webhook)
		if err != nil {
			return false, err
		}

		return false, nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconAdmission Validating Webhook")
		return false, err
	}

	if !reflect.DeepEqual(webhook.Webhooks[0].FailurePolicy, existingWebhook.Webhooks[0].FailurePolicy) {
		updated = true
	}

	if !reflect.DeepEqual(webhook.Webhooks[0].ClientConfig, existingWebhook.Webhooks[0].ClientConfig) {
		updated = true
	}

	if !reflect.DeepEqual(webhook.Webhooks[0].NamespaceSelector, existingWebhook.Webhooks[0].NamespaceSelector) {
		updated = true
	}

	if updated {
		existingWebhook.Webhooks = webhook.Webhooks
		if err := k8sutils.Update(r.Client, ctx, req, log, falconAdmission, &falconAdmission.Status, existingWebhook); err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

func (r *FalconAdmissionReconciler) reconcileAdmissionDeployment(ctx context.Context, req ctrl.Request, log logr.Logger, falconAdmission *falconv1alpha1.FalconAdmission) error {
	imageUri, err := r.imageUri(ctx, falconAdmission)
	if err != nil {
		return fmt.Errorf("unable to determine falcon container image URI: %v", err)
	}

	if falconAdmission.Spec.AdmissionConfig.ContainerPort == nil {
		port := int32(4443)
		falconAdmission.Spec.AdmissionConfig.ContainerPort = &port
	}

	existingDeployment := &appsv1.Deployment{}
	dep := assets.AdmissionDeployment(falconAdmission.Name, falconAdmission.Spec.InstallNamespace, common.FalconAdmissionController, imageUri, falconAdmission, log)
	updated := false

	if len(proxy.ReadProxyVarsFromEnv()) > 0 {
		for i, container := range dep.Spec.Template.Spec.Containers {
			dep.Spec.Template.Spec.Containers[i].Env = append(container.Env, proxy.ReadProxyVarsFromEnv()...)
		}
	}

	err = r.Get(ctx, types.NamespacedName{Name: falconAdmission.Name, Namespace: falconAdmission.Spec.InstallNamespace}, existingDeployment)
	if err != nil && apierrors.IsNotFound(err) {
		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconAdmission, &falconAdmission.Status, dep)
		if err != nil {
			return err
		}

		return nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconAdmission Deployment")
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

	if !reflect.DeepEqual(dep.Spec.Replicas, existingDeployment.Spec.Replicas) {
		existingDeployment.Spec.Replicas = dep.Spec.Replicas
		updated = true
	}

	if !reflect.DeepEqual(dep.Spec.Template.Spec.TopologySpreadConstraints, existingDeployment.Spec.Template.Spec.TopologySpreadConstraints) {
		existingDeployment.Spec.Template.Spec.TopologySpreadConstraints = dep.Spec.Template.Spec.TopologySpreadConstraints
		updated = true
	}

	for i, containers := range dep.Spec.Template.Spec.Containers {
		if !reflect.DeepEqual(containers.Resources, existingDeployment.Spec.Template.Spec.Containers[i].Resources) {
			existingDeployment.Spec.Template.Spec.Containers[i].Resources = containers.Resources
			updated = true
		}

		if !reflect.DeepEqual(containers.LivenessProbe.ProbeHandler.HTTPGet.Port, existingDeployment.Spec.Template.Spec.Containers[i].LivenessProbe.ProbeHandler.HTTPGet.Port) {
			existingDeployment.Spec.Template.Spec.Containers[i].LivenessProbe.ProbeHandler.HTTPGet.Port = containers.LivenessProbe.ProbeHandler.HTTPGet.Port
			updated = true
		}

		if !reflect.DeepEqual(containers.StartupProbe.ProbeHandler.HTTPGet.Port, existingDeployment.Spec.Template.Spec.Containers[i].StartupProbe.ProbeHandler.HTTPGet.Port) {
			existingDeployment.Spec.Template.Spec.Containers[i].StartupProbe.ProbeHandler.HTTPGet.Port = containers.StartupProbe.ProbeHandler.HTTPGet.Port
			updated = true
		}
	}

	// TODO(hirendra): Add reconciliation logic for following fields in admissionConfig.
	// admissionConfig:
	//   snapshotsEnabled: false
	//   snapshotsInterval: 10h
	//   watcherEnabled: false

	if updated {
		if err := k8sutils.Update(r.Client, ctx, req, log, falconAdmission, &falconAdmission.Status, existingDeployment); err != nil {
			return err
		}
	}

	return nil
}

func (r *FalconAdmissionReconciler) reconcileRegistrySecret(ctx context.Context, req ctrl.Request, log logr.Logger, falconAdmission *falconv1alpha1.FalconAdmission) error {
	pulltoken, err := pulltoken.CrowdStrike(ctx, r.falconApiConfig(ctx, falconAdmission))
	if err != nil {
		return fmt.Errorf("unable to get registry pull token: %v", err)
	}

	secretData := map[string][]byte{corev1.DockerConfigJsonKey: common.CleanDecodedBase64(pulltoken)}
	secret := assets.Secret(common.FalconPullSecretName, falconAdmission.Spec.InstallNamespace, "falcon-operator", secretData, corev1.SecretTypeDockerConfigJson)
	existingSecret := &corev1.Secret{}

	err = r.Get(ctx, types.NamespacedName{Name: common.FalconPullSecretName, Namespace: falconAdmission.Spec.InstallNamespace}, existingSecret)
	if err != nil && apierrors.IsNotFound(err) {
		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconAdmission, &falconAdmission.Status, secret)
		if err != nil {
			return err
		}

		return nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconAdmission Registry Pull Secret")
		return err
	}

	if !reflect.DeepEqual(secret.Data, existingSecret.Data) {
		err = k8sutils.Update(r.Client, ctx, req, log, falconAdmission, &falconAdmission.Status, existingSecret)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *FalconAdmissionReconciler) reconcileImageStream(ctx context.Context, req ctrl.Request, log logr.Logger, falconAdmission *falconv1alpha1.FalconAdmission) (*imagev1.ImageStream, error) {
	const imageStreamName = "falcon-admission-controller"
	namespace := r.imageNamespace(falconAdmission)
	imageStream := assets.ImageStream(imageStreamName, namespace, common.FalconAdmissionController)
	existingImageStream := &imagev1.ImageStream{}

	err := r.Get(ctx, types.NamespacedName{Name: imageStreamName, Namespace: namespace}, existingImageStream)
	if err != nil && apierrors.IsNotFound(err) {
		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconAdmission, &falconAdmission.Status, imageStream)
		if err != nil {
			return imageStream, err
		}

		return imageStream, nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconAdmission ImageStream")
		return existingImageStream, err
	}

	if !reflect.DeepEqual(imageStream.Spec, existingImageStream.Spec) {
		existingImageStream.Spec = imageStream.Spec
		err = k8sutils.Update(r.Client, ctx, req, log, falconAdmission, &falconAdmission.Status, existingImageStream)
		if err != nil {
			return existingImageStream, err
		}
	}

	return existingImageStream, nil
}

func (r *FalconAdmissionReconciler) reconcileNamespace(ctx context.Context, req ctrl.Request, log logr.Logger, falconAdmission *falconv1alpha1.FalconAdmission) error {
	namespace := assets.Namespace(falconAdmission.Spec.InstallNamespace)
	existingNamespace := &corev1.Namespace{}

	err := r.Get(ctx, types.NamespacedName{Name: falconAdmission.Spec.InstallNamespace}, existingNamespace)
	if err != nil && apierrors.IsNotFound(err) {
		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconAdmission, &falconAdmission.Status, namespace)
		if err != nil {
			return err
		}

		return nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconAdmission Namespace")
		return err
	}

	return nil
}

func (r *FalconAdmissionReconciler) admissionDeploymentUpdate(ctx context.Context, req ctrl.Request, log logr.Logger, falconAdmission *falconv1alpha1.FalconAdmission) error {
	existingDeployment := &appsv1.Deployment{}
	configVersion := "falcon.config.version"
	err := r.Get(ctx, types.NamespacedName{Name: falconAdmission.Name, Namespace: falconAdmission.Spec.InstallNamespace}, existingDeployment)
	if err != nil && apierrors.IsNotFound(err) {
		return err
	} else if err != nil {
		log.Error(err, "Failed to get FalconAdmission Deployment")
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

	log.Info("Rolling FalconAdmission Deployment due to non-deployment configuration change")
	if err := k8sutils.Update(r.Client, ctx, req, log, falconAdmission, &falconAdmission.Status, existingDeployment); err != nil {
		return err
	}

	return nil
}
