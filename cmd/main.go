/*
Copyright 2021 CrowdStrike
*/

package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"strings"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	"k8s.io/client-go/discovery"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	imagev1 "github.com/openshift/api/image/v1"
	arv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	admissioncontroller "github.com/crowdstrike/falcon-operator/internal/controller/admission"
	"github.com/crowdstrike/falcon-operator/internal/controller/common/sensorversion"
	containercontroller "github.com/crowdstrike/falcon-operator/internal/controller/falcon_container"
	falcondeployment "github.com/crowdstrike/falcon-operator/internal/controller/falcon_deployment"
	imageanalyzercontroller "github.com/crowdstrike/falcon-operator/internal/controller/falcon_image_analyzer"
	nodecontroller "github.com/crowdstrike/falcon-operator/internal/controller/falcon_node"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/version"
	// +kubebuilder:scaffold:imports
)

const defaultSensorAutoUpdateInterval = time.Hour * 24
const defaultLeaseDuration = time.Second * 30
const defaultRenewDeadline = time.Second * 20

var (
	scheme            = runtime.NewScheme()
	setupLog          = ctrl.Log.WithName("setup")
	environment       = "Kubernetes"
	requiredCacheObjs = map[client.Object]cache.ByObject{
		&falconv1alpha1.FalconAdmission{}:  {},
		&falconv1alpha1.FalconNodeSensor{}: {},
		&falconv1alpha1.FalconContainer{}:  {},
		&falconv1alpha1.FalconDeployment{}: {},
		&schedulingv1.PriorityClass{}: {
			Label: labels.SelectorFromSet(labels.Set{common.FalconComponentKey: common.FalconKernelSensor}),
		},
		&corev1.Service{}: {
			Label: labels.SelectorFromSet(labels.Set{common.FalconProviderKey: common.FalconProviderValue}),
		},
		&corev1.ResourceQuota{}: {
			Label: labels.SelectorFromSet(labels.Set{common.FalconComponentKey: common.FalconAdmissionController}),
		},
		&appsv1.Deployment{}: {
			Label: labels.SelectorFromSet(labels.Set{common.FalconProviderKey: common.FalconProviderValue}),
		},
		&corev1.ConfigMap{}: {
			Label: labels.SelectorFromSet(labels.Set{common.FalconProviderKey: common.FalconProviderValue}),
		},
		&appsv1.DaemonSet{}: {
			Label: labels.SelectorFromSet(labels.Set{common.FalconComponentKey: common.FalconKernelSensor}),
		},
		&arv1.MutatingWebhookConfiguration{}: {
			Label: labels.SelectorFromSet(labels.Set{common.FalconComponentKey: common.FalconSidecarSensor}),
		},
		&arv1.ValidatingWebhookConfiguration{}: {
			Label: labels.SelectorFromSet(labels.Set{common.FalconComponentKey: common.FalconAdmissionController}),
		},
		&corev1.Namespace{}: {
			Label: labels.SelectorFromSet(labels.Set{common.FalconInstanceNameKey: "namespace"}),
		},
		&corev1.Secret{}: {
			Label: labels.SelectorFromSet(labels.Set{common.FalconInstanceNameKey: "secret"}),
		},
		&rbacv1.ClusterRoleBinding{}: {
			Label: labels.SelectorFromSet(labels.Set{common.FalconInstanceNameKey: "clusterrolebinding"}),
		},
		&corev1.ServiceAccount{}: {
			Label: labels.SelectorFromSet(labels.Set{common.FalconInstanceNameKey: "serviceaccount"}),
		},
	}
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(imagev1.AddToScheme(scheme))

	utilruntime.Must(falconv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var profileAddr string
	var enableProfiling bool
	var enableHTTP2 bool
	var secureMetrics bool
	var tlsOpts []func(*tls.Config)
	var ver bool
	var err error
	var sensorAutoUpdateInterval time.Duration
	var leaseDuration time.Duration
	var renewDeadline time.Duration

	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&profileAddr, "profile-bind-address", "localhost:8082", "The address the profiling endpoint binds to.")
	flag.BoolVar(&enableProfiling, "profile", false, "Enable profiling.")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	flag.BoolVar(&secureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&ver, "version", false, "Print version")
	flag.DurationVar(&sensorAutoUpdateInterval, "sensor-auto-update-interval", defaultSensorAutoUpdateInterval, "The rate at which the Falcon API is queried for new sensor versions")
	flag.DurationVar(&leaseDuration, "lease-duration", defaultLeaseDuration, "The duration that non-leader candidates will wait to force acquire leadership.")
	flag.DurationVar(&renewDeadline, "renew-deadline", defaultRenewDeadline, "the duration that the acting controlplane will retry refreshing leadership before giving up.")

	// Openshift does not support persisting command line arguments when deploying the operator.
	// The ARGS env var must be used instead if operator deployment options are updated.
	if env := os.Getenv("ARGS"); env != "" {
		os.Args = append(os.Args, strings.Split(env, " ")...)
		setupLog.Info(fmt.Sprintf("configuring deployment args from env with the following options: %s", env))
	}

	opts := zap.Options{
		Development: true,
	}

	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	if ver {
		fmt.Printf("%s version: %q, go version: %q\n", os.Args[0], version.Get(), version.GoVersion)
		os.Exit(0)
	}

	dc, err := discovery.NewDiscoveryClientForConfig(ctrl.GetConfigOrDie())
	if err != nil {
		setupLog.Error(err, "failed to create discovery client")
		os.Exit(1)
	}

	openShift := isOpenShift(dc)

	if openShift {
		environment = "OpenShift"

		setupLog.Info(fmt.Sprintf("openshift api is available. cluster is running %s", environment))

		if !strings.Contains(version.Get(), "certified") {
			setupLog.V(1).Info("WARNING: this operator is not certified for OpenShift. Please install and use the certified operator for proper OpenShift support.")
		}

		requiredCacheObjs[&imagev1.ImageStream{}] = cache.ByObject{
			Label: labels.SelectorFromSet(labels.Set{common.FalconProviderKey: common.FalconProviderValue}),
		}
	} else {
		setupLog.Info(fmt.Sprintf("openshift api is not available. cluster is running %s", environment))
	}

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		TLSOpts:       tlsOpts,
	}

	if secureMetrics {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/metrics/filters#WithAuthenticationAndAuthorization
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization

		// TODO(user): If CertDir, CertName, and KeyName are not specified, controller-runtime will automatically
		// generate self-signed certificates for the metrics server. While convenient for development and testing,
		// this setup is not recommended for production.

		// TODO(user): If cert-manager is enabled in config/default/kustomization.yaml,
		// you can uncomment the following lines to use the certificate managed by cert-manager.
		// metricsServerOptions.CertDir = "/tmp/k8s-metrics-server/metrics-certs"
		// metricsServerOptions.CertName = "tls.crt"
		// metricsServerOptions.KeyName = "tls.key"
	}

	options := ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "falcon-operator-lock",
		LeaseDuration:          &leaseDuration,
		RenewDeadline:          &renewDeadline,
		Cache: cache.Options{
			ByObject: requiredCacheObjs,
		},
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	certManager, err := isCertManagerInstalled(dc)
	if err != nil {
		setupLog.Error(err, "could not determine if cert-manager is installed")
		os.Exit(1)
	}
	if certManager {
		setupLog.Info("cert-manager installation found")
	} else {
		setupLog.Info("cert-manager installation not found")
	}

	ctx := ctrl.SetupSignalHandler()
	tracker := sensorversion.NewTracker(ctx, sensorAutoUpdateInterval)

	if err = (&containercontroller.FalconContainerReconciler{
		Client:     mgr.GetClient(),
		Reader:     mgr.GetAPIReader(),
		Scheme:     mgr.GetScheme(),
		RestConfig: mgr.GetConfig(),
	}).SetupWithManager(mgr, tracker); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "FalconContainer")
		os.Exit(1)
	}
	if err = (&nodecontroller.FalconNodeSensorReconciler{
		Client: mgr.GetClient(),
		Reader: mgr.GetAPIReader(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr, tracker); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "FalconNodeSensor")
		os.Exit(1)
	}
	if err = (&admissioncontroller.FalconAdmissionReconciler{
		Client:    mgr.GetClient(),
		Reader:    mgr.GetAPIReader(),
		Scheme:    mgr.GetScheme(),
		OpenShift: openShift,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "FalconAdmission")
		os.Exit(1)
	}
	if err = (&imageanalyzercontroller.FalconImageAnalyzerReconciler{
		Client: mgr.GetClient(),
		Reader: mgr.GetAPIReader(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "FalconImageAnalyzer")
		os.Exit(1)
	}
	if err = (&falcondeployment.FalconDeploymentReconciler{
		Client: mgr.GetClient(),
		Reader: mgr.GetAPIReader(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "FalconDeployment")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	if enableProfiling {
		setupLog.Info("Establishing profile endpoint.")
		go func() {
			pprofMux := http.NewServeMux()
			pprofMux.HandleFunc("/debug/pprof/", pprof.Index)
			pprofMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			pprofMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
			pprofMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
			pprofMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
			srv := &http.Server{Addr: profileAddr, ReadHeaderTimeout: time.Second * 10, ReadTimeout: time.Second * 60, WriteTimeout: time.Second * 10, Handler: pprofMux}
			err := srv.ListenAndServe()
			if err != nil {
				setupLog.Error(err, "unable to establish profile endpoint")
			}
		}()
	}

	go tracker.StartTracking()

	setupLog.Info("starting manager", "version", version.Get(), "go version", version.GoVersion)
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func isOpenShift(client discovery.DiscoveryInterface) bool {
	_, err := client.ServerResourcesForGroupVersion("image.openshift.io/v1")
	return err == nil
}

func isCertManagerInstalled(client discovery.DiscoveryInterface) (bool, error) {
	return discovery.IsResourceEnabled(client, certv1.SchemeGroupVersion.WithResource("issuers"))
}
