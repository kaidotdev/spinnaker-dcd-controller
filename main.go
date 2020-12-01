package main

import (
	"flag"
	"net/http"
	"os"
	"spinnaker-dcd-controller/controllers"

	"github.com/sirupsen/logrus"

	applicationV1 "spinnaker-dcd-controller/api/v1"

	"github.com/spinnaker/roer/spinnaker"
	"github.com/spinnaker/spin/cmd/gateclient"
	"github.com/spinnaker/spin/cmd/output"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = applicationV1.AddToScheme(scheme)
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var spinnakerEndpoint string
	var verbose bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager.")
	flag.StringVar(&spinnakerEndpoint, "spinnaker-endpoint", "http://spin-gate.spinnaker.svc.cluster.local:8084", "The endpoint of Spinnaker Gate.")
	flag.BoolVar(&verbose, "verbose", false, "Make the operation more talkative.")
	flag.Parse()

	ctrl.SetLogger(zap.Logger(true))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "spinnaker-dcd-controller",
		Port:               9443,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if err := (&controllers.ApplicationReconciler{
		Client:          mgr.GetClient(),
		Log:             ctrl.Log.WithName("controllers").WithName("Application"),
		Scheme:          mgr.GetScheme(),
		Recorder:        mgr.GetEventRecorderFor("spinnaker-dcd-controller"),
		SpinnakerClient: spinnaker.New(spinnakerEndpoint, &http.Client{}),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Application")
		os.Exit(1)
	}
	if err := (&controllers.PipelineTemplateReconciler{
		Client:          mgr.GetClient(),
		Log:             ctrl.Log.WithName("controllers").WithName("PipelineTemplate"),
		Scheme:          mgr.GetScheme(),
		Recorder:        mgr.GetEventRecorderFor("spinnaker-dcd-controller"),
		SpinnakerClient: spinnaker.New(spinnakerEndpoint, &http.Client{}),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PipelineTemplate")
		os.Exit(1)
	}
	if err := (&controllers.PipelineReconciler{
		Client:          mgr.GetClient(),
		Log:             ctrl.Log.WithName("controllers").WithName("Pipeline"),
		Scheme:          mgr.GetScheme(),
		Recorder:        mgr.GetEventRecorderFor("spinnaker-dcd-controller"),
		SpinnakerClient: spinnaker.New(spinnakerEndpoint, &http.Client{}),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Pipeline")
		os.Exit(1)
	}
	if err := (&controllers.CanaryConfigReconciler{
		Client:          mgr.GetClient(),
		Log:             ctrl.Log.WithName("controllers").WithName("CanaryConfig"),
		Scheme:          mgr.GetScheme(),
		Recorder:        mgr.GetEventRecorderFor("spinnaker-dcd-controller"),
		SpinnakerClient: buildSpinGateClient(spinnakerEndpoint),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CanaryConfig")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func buildSpinGateClient(spinnakerEndpoint string) gateclient.GatewayClient {
	oft, _ := output.ParseOutputFormat("json")
	ui := output.NewUI(true, true, oft, os.Stdout, os.Stderr)
	gateClient, _ := gateclient.NewGateClient(ui, spinnakerEndpoint, "", "", false)

	return *gateClient
}
