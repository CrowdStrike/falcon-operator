package falcon

import (
	"context"
	"fmt"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/assets/container"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/registry/pulltoken"
	"github.com/go-logr/logr"
	arv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	clog "sigs.k8s.io/controller-runtime/pkg/log"
)

// FalconContainerReconciler reconciles a FalconContainer object
type FalconContainerReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	RestConfig *rest.Config
}

// SetupWithManager sets up the controller with the Manager.
func (r *FalconContainerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&falconv1alpha1.FalconContainer{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Owns(&arv1.MutatingWebhookConfiguration{}).
		Complete(r)
}

//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconcontainers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconcontainers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconcontainers/finalizers,verbs=update

// +kubebuilder:rbac:groups=image.openshift.io,resources=imagestreams,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="batch",resources=jobs,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;delete
// +kubebuilder:rbac:groups="",resources=pods/log,verbs=get

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="admissionregistration.k8s.io",resources=mutatingwebhookconfigurations,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterroles,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterrolebindings,verbs=get;list;watch;create;update;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the FalconContainer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *FalconContainerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := clog.FromContext(ctx)
	falconContainer := &falconv1alpha1.FalconContainer{}
	err := r.Get(ctx, req.NamespacedName, falconContainer)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Cannot get the Falcon Container custom resource")
		return ctrl.Result{}, err
	}

	log.Info("reconcile is", "NamespacedName", req.NamespacedName)
	req.Namespace = "falcon-operator-system"
	falconContainer.Namespace = "falcon-operator-system"
	log.Info("hardcoded namespace is now", "NamespacedName", req.NamespacedName)

	// Check if the daemonset already exists, if not create a new one
	if (falconContainer.Spec.FalconContainerSensorConfig.EnablePullSecrets || falconContainer.Spec.Registry.Type != "") && len(falconContainer.Spec.FalconContainerSensorConfig.Namespaces) > 0 {
		for _, ns := range falconContainer.Spec.FalconContainerSensorConfig.Namespaces {
			log.Info("Creating a new Docker secret for namespace", "Namespace", ns)
			dockerNSSecret, dockerNSSecretUpdated, err := r.handleContainerDockerSecrets(ctx, ns, falconContainer, log)
			if err != nil {
				log.Error(err, "error handling Docker Secret for namespace", ns)
				return ctrl.Result{}, err
			}
			if dockerNSSecret == nil {
				// this just got created, so re-queue.
				log.Info("Namespaced Docker Secret was just created. Re-queuing", "Namespace", ns)
				//return ctrl.Result{Requeue: true}, nil
			}
			if dockerNSSecretUpdated {
				log.Info("Namespaced Docker Secret was updated", "Namespace", ns)
			}
		}
	}

	if falconContainer.Spec.FalconContainerSensorConfig.EnablePullSecrets || falconContainer.Spec.Registry.Type != "" {
		falconContainer.Spec.FalconContainerSensorConfig.EnablePullSecrets = true
		dockerSecret, dockerSecretUpdated, err := r.handleContainerDockerSecrets(ctx, "", falconContainer, log)
		if err != nil {
			log.Error(err, "error handling Docker Secret for Injector")
			return ctrl.Result{}, err
		}
		if dockerSecret == nil {
			// this just got created, so re-queue.
			//log.Info("Docker Secret for Injector was just created. Re-queuing")
			//return ctrl.Result{Requeue: true}, nil
			log.Info("Docker Secret for Injector was just created.")
		}
		if dockerSecretUpdated {
			log.Info("Docker Secret for Injector was updated")
		}
	}

	// Check if the daemonset already exists, if not create a new one
	tlsSecret, dockerTLSUpdated, err := r.handleContainerTLSSecret(ctx, falconContainer, log)
	if err != nil {
		log.Error(err, "error handling Docker TLS Secret")
		return ctrl.Result{}, err
	}
	if tlsSecret == nil {
		// this just got created, so re-queue.
		//log.Info("Docker TLS Secret was just created. Re-queuing")
		//return ctrl.Result{Requeue: true}, nil
		log.Info("Docker TLS Secret was just created.")
	}
	if dockerTLSUpdated {
		log.Info("Docker TLS Secret was updated")
	}

	// Check if the daemonset already exists, if not create a new one
	sensorConf, confUpdated, err := r.handleConfigMaps(ctx, falconContainer, log)
	if err != nil {
		log.Error(err, "error handling configmap")
		return ctrl.Result{}, err
	}
	if sensorConf == nil {
		// this just got created, so re-queue.
		//log.Info("Configmap was just created. Re-queuing")
		//return ctrl.Result{Requeue: true}, nil
		log.Info("Configmap was just created.")
	}
	if confUpdated {
		log.Info("Configmap was updated")
	}

	// Check if the daemonset already exists, if not create a new one
	service, serviceUpdated, err := r.handleContainerService(ctx, falconContainer, log)
	if err != nil {
		log.Error(err, "error handling Service")
		return ctrl.Result{}, err
	}
	if service == nil {
		// this just got created, so re-queue.
		//log.Info("Service was just created. Re-queuing")
		//return ctrl.Result{Requeue: true}, nil
		log.Info("Service was just created.")
	}
	if serviceUpdated {
		log.Info("Service was updated")
	}

	// Check if the daemonset already exists, if not create a new one
	dep, depUpdated, err := r.handleContainerDeployment(ctx, falconContainer, log)
	if err != nil {
		log.Error(err, "error handling Deployment")
		return ctrl.Result{}, err
	}
	if dep == nil {
		// this just got created, so re-queue.
		log.Info("Deployment was just created. Re-queuing")
		return ctrl.Result{Requeue: true}, nil
	}
	if depUpdated {
		log.Info("Deployment was updated")
	}

	webhook, webhookUpdated, err := r.handleContainerWebhook(ctx, falconContainer, dep.Spec, log)
	if err != nil {
		log.Error(err, "error handling MutatingWebhook")
		return ctrl.Result{}, err
	}
	if webhook == nil {
		// this just got created, so re-queue.
		//log.Info("MutatingWebhook was just created. Re-queuing")
		//return ctrl.Result{Requeue: true}, nil
		log.Info("MutatingWebhook was just created.")
	}
	if webhookUpdated {
		log.Info("MutatingWebhook was updated")
	}

	return ctrl.Result{}, nil
}

// handleConfigMaps creates and updates the node sensor configmap
func (r *FalconContainerReconciler) handleConfigMaps(ctx context.Context, falconContainer *falconv1alpha1.FalconContainer, log logr.Logger) (*corev1.ConfigMap, bool, error) {
	var updated bool
	cmName := falconContainer.Name + "-config"
	confCm := &corev1.ConfigMap{}

	err := r.Get(ctx, types.NamespacedName{Name: cmName, Namespace: falconContainer.Namespace}, confCm)

	if falconContainer.Spec.FalconAPI.CID == nil && falconContainer.Spec.FalconContainerSensor.CID == "" {
		cid, err := common.FalconCID(ctx, falconContainer.Spec.FalconAPI.CID, falconContainer.Spec.FalconAPI.ApiConfig())
		if err != nil {
			log.Error(err, "Failed to get Falcon CID", "CID", cid)
		}

		if confCm.Data["FALCONCTL_OPT_CID"] != cid || confCm.Data["FALCONCTL_OPT_CID"] == "" {
			log.Info("Setting the Falcon CID", "CID", cid)
			falconContainer.Spec.FalconAPI.CID = &cid
			falconContainer.Spec.FalconContainerSensor.CID = cid
		} else {
			log.Info("Skipping setting Falcon CID as it is already set")
			falconContainer.Spec.FalconContainerSensor.CID = confCm.Data["FALCONCTL_OPT_CID"]
		}
	}

	if falconContainer.Spec.Registry.Type != "" && falconContainer.Spec.FalconContainerSensorConfig.Image == "" {
		image, tag, err := common.ImageInfo(ctx, falconContainer.Spec.Version, falconContainer.Spec.FalconAPI.ApiConfig())
		if err != nil {
			log.Error(err, "Failed to get registry information", "Image", image, "Tag", tag)
		}

		newImage := fmt.Sprintf("%s:%s", image, tag)
		if confCm.Data["FALCON_IMAGE"] != newImage || confCm.Data["FALCON_IMAGE"] == "" {
			log.Info("Setting Container Image", "Image", image, "Tag", tag)
			falconContainer.Spec.FalconContainerSensorConfig.Image = newImage
		} else {
			log.Info("Skipping setting Falcon Image as it is already set")
			falconContainer.Spec.FalconContainerSensorConfig.Image = confCm.Data["FALCON_IMAGE"]
		}
	}

	if err != nil && errors.IsNotFound(err) {
		// does not exist, create
		configmap := r.containerConfigMap(ctx, cmName, falconContainer, log)
		if err := r.Create(ctx, configmap); err != nil {
			log.Error(err, "Failed to create new Configmap", "Configmap.Namespace", falconContainer.Namespace, "Configmap.Name", cmName)
			return nil, updated, err
		}

		return nil, updated, nil
	} else if err != nil {
		log.Error(err, "error getting Configmap")
		return nil, updated, err
	} else {
		configmap := r.containerConfigMap(ctx, cmName, falconContainer, log)
		err = r.Update(ctx, configmap)
		if err != nil {
			log.Error(err, "Failed to update Configmap", "Configmap.Namespace", falconContainer.Namespace, "Configmap.Name", cmName)
			return nil, updated, err
		}
		updated = true
	}

	return confCm, updated, nil
}

func (r *FalconContainerReconciler) containerConfigMap(ctx context.Context, name string, falconContainer *falconv1alpha1.FalconContainer, log logr.Logger) *corev1.ConfigMap {
	conf := container.ContainerConfigMap(name, falconContainer)

	// NOTE: calling SetControllerReference, and setting owner references in
	// general, is important as it allows deleted objects to be garbage collected.
	err := controllerutil.SetControllerReference(falconContainer, conf, r.Scheme)
	if err != nil {
		log.Error(err, "unable to set controller reference")
	}

	return conf
}

func (r *FalconContainerReconciler) handleContainerDockerSecrets(ctx context.Context, ns string, falconContainer *falconv1alpha1.FalconContainer, log logr.Logger) (*corev1.Secret, bool, error) {
	var updated bool
	dockerSecret := &corev1.Secret{}
	secrets := &corev1.SecretList{}
	secNS := falconContainer.Namespace
	secName := falconContainer.Name

	if len(ns) > 0 {
		secNS = ns
		secName = common.FalconPullSecret
	}

	err := r.Get(ctx, types.NamespacedName{Name: falconContainer.Name, Namespace: secNS}, dockerSecret)
	if err != nil {
		log.Error(err, "Failed to get Secrets List", "Secrets", secrets)
	}

	pulltoken, err := pulltoken.CrowdStrike(ctx, falconContainer.Spec.FalconAPI.ApiConfig())
	if err != nil {
		log.Error(err, "Failed to get pull token")
	} else {
		log.Info("Received the pull token")
	}

	if falconContainer.Spec.FalconContainerSensorConfig.ContainerRegistryPullSecret != "" {
		pulltoken = []byte(falconContainer.Spec.FalconContainerSensorConfig.ContainerRegistryPullSecret)
	}

	cleanDecodedToken := common.CleanDecodedBase64(pulltoken)

	err = r.Get(ctx, types.NamespacedName{Name: secName, Namespace: secNS}, dockerSecret)
	if err != nil && errors.IsNotFound(err) {

		// Define a new daemonset
		dockSec := r.containerDockerSecret(cleanDecodedToken, secName, secNS, falconContainer, log)

		err = r.Create(ctx, dockSec)
		if err != nil {
			log.Error(err, "Failed to create new Docker Secret", "Secret.Namespace", dockSec.Namespace, "Secret.Name", dockSec.Name)
			return nil, updated, err
		}

		log.Info("Created a new Docker Secret", "Secret.Namespace", dockSec.Namespace, "Secret.Name", dockSec.Name)
		// Daemonset created successfully - return and requeue
		return nil, updated, nil

	} else if err != nil {
		log.Error(err, "error getting Docker Secret")
		return nil, updated, err
	} else {
		/*
			dockSec := r.containerDockerSecret(cleanDecodedToken, secName, secNS, falconContainer, log)
			err = r.Update(ctx, dockSec)
			if err != nil {
				log.Error(err, "Failed to update Docker Secret", "Secret.Namespace", dockSec.Namespace, "Secret.Name", dockSec.Name)
				return nil, updated, err
			}*/

		updated = false
	}

	return dockerSecret, updated, nil
}

func (r *FalconContainerReconciler) containerDockerSecret(dockerRegistry []byte, name string, ns string, falconContainer *falconv1alpha1.FalconContainer, log logr.Logger) *corev1.Secret {
	secret := container.ContainerDockerSecrets(name, ns, dockerRegistry, falconContainer)

	// NOTE: calling SetControllerReference, and setting owner references in
	// general, is important as it allows deleted objects to be garbage collected.
	if ns == falconContainer.Namespace {
		err := controllerutil.SetControllerReference(falconContainer, secret, r.Scheme)
		if err != nil {
			log.Error(err, "unable to set controller reference")
		}
	}

	return secret
}

func (r *FalconContainerReconciler) handleContainerTLSSecret(ctx context.Context, falconContainer *falconv1alpha1.FalconContainer, log logr.Logger) (*corev1.Secret, bool, error) {
	var updated bool
	secretTLSCerts := &corev1.Secret{}
	secretName := falconContainer.Name + "-tls"
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: falconContainer.Namespace}, secretTLSCerts)
	if err != nil && errors.IsNotFound(err) {
		// Define a new daemonset
		secretCerts := r.containerTLSSecret(secretName, falconContainer.Namespace, falconContainer, log)

		err = r.Create(ctx, secretCerts)
		if err != nil {
			log.Error(err, "Failed to create new TLS Secret", "TLS.Namespace", secretCerts.Namespace, "TLS.Name", secretCerts.Name)
			return nil, updated, nil
		}

		log.Info("Created a new TLS Secret", "TLS.Namespace", secretCerts.Namespace, "TLS.Name", secretCerts.Name)
		// Daemonset created successfully - return and requeue
		return nil, updated, nil

	} else if err != nil {
		log.Error(err, "error getting TLS Secret")
		return nil, updated, err
	} else {
		/*secretCerts := r.containerTLSSecret(falconContainer.Name, falconContainer.Namespace, falconContainer, log)
		err = r.Update(ctx, secretCerts)
		if err != nil {
			log.Error(err, "Failed to update TLS Secret", "TLS.Namespace", secretCerts.Namespace, "TLS.Name", secretCerts.Name)
			return nil, updated, err
		}*/

		updated = false
	}

	return secretTLSCerts, updated, nil
}

func (r *FalconContainerReconciler) containerTLSSecret(name string, ns string, falconContainer *falconv1alpha1.FalconContainer, log logr.Logger) *corev1.Secret {
	secret := container.ContainerTLSSecret(name, ns, falconContainer)

	// NOTE: calling SetControllerReference, and setting owner references in
	// general, is important as it allows deleted objects to be garbage collected.
	err := controllerutil.SetControllerReference(falconContainer, secret, r.Scheme)
	if err != nil {
		log.Error(err, "unable to set controller reference")
	}

	return secret
}

func (r *FalconContainerReconciler) handleContainerDeployment(ctx context.Context, falconContainer *falconv1alpha1.FalconContainer, log logr.Logger) (*appsv1.Deployment, bool, error) {
	var updated bool
	deployment := &appsv1.Deployment{}

	err := r.Get(ctx, types.NamespacedName{Name: falconContainer.Name, Namespace: falconContainer.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		// Define a new daemonset
		dep := r.containerDeployment(falconContainer.Name, falconContainer.Namespace, falconContainer, log)

		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return nil, updated, nil
		}

		log.Info("Created a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		// Daemonset created successfully - return and requeue
		return nil, updated, nil

	} else if err != nil {
		log.Error(err, "error getting Deployment")
		return nil, updated, err
	} else {
		/*
			dep := r.containerDeployment(falconContainer.Name, falconContainer.Namespace, falconContainer, log)
			err = r.Update(ctx, dep)
			if err != nil {
				log.Error(err, "Failed to update Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
				return nil, updated, err
			}
		*/
		updated = false
	}

	return deployment, updated, nil
}

func (r *FalconContainerReconciler) containerDeployment(name string, ns string, falconContainer *falconv1alpha1.FalconContainer, log logr.Logger) *appsv1.Deployment {
	dep := container.ContainerDeployment(name, ns, falconContainer)

	// NOTE: calling SetControllerReference, and setting owner references in
	// general, is important as it allows deleted objects to be garbage collected.
	err := controllerutil.SetControllerReference(falconContainer, dep, r.Scheme)
	if err != nil {
		log.Error(err, "unable to set controller reference")
	}

	return dep
}

func (r *FalconContainerReconciler) handleContainerService(ctx context.Context, falconContainer *falconv1alpha1.FalconContainer, log logr.Logger) (*corev1.Service, bool, error) {
	var updated bool
	service := &corev1.Service{}
	serviceName := falconContainer.Name

	err := r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: falconContainer.Namespace}, service)
	if err != nil && errors.IsNotFound(err) {
		// Define a new daemonset
		ser := r.containerService(serviceName, falconContainer.Namespace, falconContainer, log)

		err = r.Create(ctx, ser)
		if err != nil {
			log.Error(err, "Failed to create new Service", "Service.Namespace", ser.Namespace, "Service.Name", ser.Name)
			return nil, updated, nil
		}

		log.Info("Created a new Service", "Service.Namespace", ser.Namespace, "Service.Name", ser.Name)
		// Daemonset created successfully - return and requeue
		return nil, updated, nil

	} else if err != nil {
		log.Error(err, "error getting TLS Secret")
		return nil, updated, err
	} else {
		/*ser := r.containerService(serviceName, falconContainer.Namespace, falconContainer, log)
		err = r.Update(ctx, ser)
		if err != nil {
			log.Error(err, "Failed to update Service", "Service.Namespace", ser.Namespace, "Service.Name", ser.Name)
			return nil, updated, err
		}*/

		updated = false
	}

	return service, updated, nil
}

func (r *FalconContainerReconciler) containerService(name string, ns string, falconContainer *falconv1alpha1.FalconContainer, log logr.Logger) *corev1.Service {
	service := container.ContainerService(name, ns, falconContainer)

	// NOTE: calling SetControllerReference, and setting owner references in
	// general, is important as it allows deleted objects to be garbage collected.
	err := controllerutil.SetControllerReference(falconContainer, service, r.Scheme)
	if err != nil {
		log.Error(err, "unable to set controller reference")
	}

	return service
}

func (r *FalconContainerReconciler) handleContainerWebhook(ctx context.Context, falconContainer *falconv1alpha1.FalconContainer, depSpec appsv1.DeploymentSpec, log logr.Logger) (*arv1.MutatingWebhookConfiguration, bool, error) {
	var updated bool
	webhook := &arv1.MutatingWebhookConfiguration{}
	replicaCount := falconContainer.Spec.FalconContainerSensorConfig.Replicas

	if depSpec.Replicas != nil {
		replicaCount = *depSpec.Replicas
	}

	err := r.Get(ctx, types.NamespacedName{Name: falconContainer.Name, Namespace: falconContainer.Namespace}, webhook)
	if err != nil && errors.IsNotFound(err) {
		// Define a new daemonset
		if replicaCount > 0 {
			wh := r.containerWebhook(falconContainer.Name, falconContainer.Namespace, falconContainer, log)

			err = r.Create(ctx, wh)
			if err != nil {
				log.Error(err, "Failed to create new MutatingWebhookConfiguration", "MutatingWebhookConfiguration.Namespace", wh.Namespace, "MutatingWebhookConfiguration.Name", wh.Name)
				return nil, updated, nil
			}

			log.Info("Created a new MutatingWebhookConfiguration", "MutatingWebhookConfiguration.Namespace", wh.Namespace, "MutatingWebhookConfiguration.Name", wh.Name)
			// Daemonset created successfully - return and requeue
			return nil, updated, nil
		}
	} else if err != nil {
		log.Error(err, "error getting MutatingWebhookConfiguration")
		return nil, updated, err

	} else {
		if replicaCount > 0 {
			/*
				wh := r.containerWebhook(falconContainer.Name, falconContainer.Namespace, falconContainer, log)
				err = r.Update(ctx, wh)
				if err != nil {
					log.Error(err, "Failed to update MutatingWebhookConfiguration", "MutatingWebhookConfiguration.Namespace", wh.Namespace, "MutatingWebhookConfiguration.Name", wh.Name)
					return nil, updated, err
				}
			*/

			updated = false
		} else {
			wh := r.containerWebhook(falconContainer.Name, falconContainer.Namespace, falconContainer, log)
			err := r.Delete(ctx, wh)
			if err != nil {
				log.Error(err, "Failed to delete MutatingWebhookConfiguration", "MutatingWebhookConfiguration.Namespace", wh.Namespace, "MutatingWebhookConfiguration.Name", wh.Name)
				return nil, updated, err
			}

			log.Info("Number of deployement replicas set to 0. Deleting the MutatingWebhookConfiguration to allow cluster access and the ability to re-deploy the sensor", "MutatingWebhookConfiguration.Namespace", wh.Namespace, "MutatingWebhookConfiguration.Name", wh.Name)
		}
	}

	return webhook, updated, nil
}

func (r *FalconContainerReconciler) containerWebhook(name string, ns string, falconContainer *falconv1alpha1.FalconContainer, log logr.Logger) *arv1.MutatingWebhookConfiguration {
	wh := container.ContainerMutatingWebhook(name, ns, falconContainer)

	// NOTE: calling SetControllerReference, and setting owner references in
	// general, is important as it allows deleted objects to be garbage collected.
	err := controllerutil.SetControllerReference(falconContainer, wh, r.Scheme)
	if err != nil {
		log.Error(err, "unable to set controller reference")
	}

	return wh
}
