/*
Copyright 2021 CrowdStrike
*/

package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"strings"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	imagev1 "github.com/openshift/api/image/v1"
	arv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	admissioncontroller "github.com/crowdstrike/falcon-operator/internal/controller/admission"
	containercontroller "github.com/crowdstrike/falcon-operator/internal/controller/falcon_container"
	nodecontroller "github.com/crowdstrike/falcon-operator/internal/controller/falcon_node"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/version"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
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
	var ver bool
	var err error

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&profileAddr, "profile-bind-address", "localhost:8082", "The address the profiling endpoint binds to.")
	flag.BoolVar(&enableProfiling, "profile", false, "Enable profiling.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&ver, "version", false, "Print version")

	if env := os.Getenv("ARGS"); env != "" {
		os.Args = append(os.Args, strings.Split(env, " ")...)
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

	version.Print()

	options := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "70435a7a.crowdstrike.com",

		NewCache: cache.BuilderWithOptions(cache.Options{
			SelectorsByObject: cache.SelectorsByObject{
				&falconv1alpha1.FalconAdmission{}:  {},
				&falconv1alpha1.FalconNodeSensor{}: {},
				&falconv1alpha1.FalconContainer{}:  {},
				&corev1.Namespace{}:                {},
				&corev1.Secret{}:                   {},
				&rbacv1.ClusterRoleBinding{}:       {},
				&corev1.ServiceAccount{}:           {},
				&schedulingv1.PriorityClass{}: {
					Label: labels.SelectorFromSet(labels.Set{common.FalconComponentKey: common.FalconKernelSensor}),
				},
				&imagev1.ImageStream{}: {
					Label: labels.SelectorFromSet(labels.Set{common.FalconProviderKey: common.FalconProviderValue}),
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
			},
		},
		),
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&containercontroller.FalconContainerReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		RestConfig: mgr.GetConfig(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "FalconContainer")
		os.Exit(1)
	}
	if err = (&nodecontroller.FalconNodeSensorReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "FalconNodeSensor")
		os.Exit(1)
	}
	if err = (&admissioncontroller.FalconAdmissionReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "FalconAdmission")
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

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
