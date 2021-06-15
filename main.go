/*
Copyright 2021 Pelotech.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/fluxcd/pkg/runtime/events"
	"github.com/fluxcd/pkg/runtime/metrics"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	crtlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	konfigurationv1 "github.com/pelotech/jsonnet-controller/api/v1"
	"github.com/pelotech/jsonnet-controller/controllers"
	"github.com/pelotech/jsonnet-controller/pkg/gencert"
	//+kubebuilder:scaffold:imports
)

var (
	controllerName = "kubecfg-controller"

	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(konfigurationv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme

	utilruntime.Must(sourcev1.AddToScheme(scheme))
}

func main() {
	var (
		tlsCertDir           string
		webPort              int
		eventsAddr           string
		metricsAddr          string
		enableLeaderElection bool
		probeAddr            string
		watchAllNamespaces   bool
		reconcileOpts        controllers.ReconcilerOptions
	)

	flag.IntVar(&webPort, "web-bind-port", 9443, "The port to bind the web server to.")
	flag.StringVar(&tlsCertDir, "tls-cert-dir", "", "The path to certificates and keys to use for the webserver. A self-signed certificate will be generated if not provided.")
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&eventsAddr, "events-addr", "", "The address for an external events receiver.")
	flag.BoolVar(&watchAllNamespaces, "watch-all-namespaces", true,
		"Watch for Konfigurations in all namespaces, if set to false it will only watch the runtime namespace.")

	// Reconcile options
	flag.IntVar(&reconcileOpts.HTTPRetryMax, "http-retry-max", 5, "Maximum number of times to retry fetching a source artifact")
	flag.IntVar(&reconcileOpts.MaxConcurrentReconciles, "max-concurrent-reconciles", 3, "Number of reconcilations to allow to run at a time")
	flag.DurationVar(&reconcileOpts.DependencyRequeueInterval, "dependency-requeue-interval", 30*time.Second, "The interval at which failing dependencies are reevaluated.")
	flag.StringVar(&reconcileOpts.JsonnetCacheDirectory, "jsonnet-cache", "/cache", "The directory to cache jsonnet assets")
	flag.DurationVar(&reconcileOpts.DryRunRequestTimeout, "dry-run-timeout", 10*time.Second, "The timeout for dry-run requests")
	// Zap options
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	// Setup logging
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	if tlsCertDir == "" {
		var err error
		setupLog.Info("Generating self-signed certificates for the webhook server")
		tlsCertDir, err = gencert.GenerateCert()
		if err != nil {
			setupLog.Error(err, "unable to generate a self-signed certificate")
			os.Exit(1)
		}
	}

	var eventRecorder *events.Recorder
	if eventsAddr != "" {
		if er, err := events.NewRecorder(eventsAddr, controllerName); err != nil {
			setupLog.Error(err, "unable to create event recorder")
			os.Exit(1)
		} else {
			eventRecorder = er
		}
	}

	metricsRecorder := metrics.NewRecorder()
	crtlmetrics.Registry.MustRegister(metricsRecorder.Collectors()...)

	watchNamespace := ""
	if !watchAllNamespaces {
		watchNamespace = os.Getenv("POD_NAMESPACE")
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   webPort,
		CertDir:                tlsCertDir,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "54bd3b09.kubecfg.io",
		Namespace:              watchNamespace,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	konfigurationController := &controllers.KonfigurationReconciler{
		Client:                mgr.GetClient(),
		Scheme:                mgr.GetScheme(),
		EventRecorder:         mgr.GetEventRecorderFor(controllerName),
		ExternalEventRecorder: eventRecorder,
		MetricsRecorder:       metricsRecorder,
		StatusPoller:          polling.NewStatusPoller(mgr.GetClient(), mgr.GetRESTMapper()),
	}

	mgr.GetWebhookServer().Register("/dry-run", konfigurationController.DryRunFunc())

	if err = konfigurationController.SetupWithManager(setupLog, mgr, &reconcileOpts); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Konfiguration")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
