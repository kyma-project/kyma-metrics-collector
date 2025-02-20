package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/kyma-project/kyma-metrics-collector/env"
	"github.com/kyma-project/kyma-metrics-collector/options"
	"github.com/kyma-project/kyma-metrics-collector/pkg/collector/edp"
	"github.com/kyma-project/kyma-metrics-collector/pkg/config"
	"github.com/kyma-project/kyma-metrics-collector/pkg/keb"
	log "github.com/kyma-project/kyma-metrics-collector/pkg/logger"
	kmcmetrics "github.com/kyma-project/kyma-metrics-collector/pkg/metrics"
	kmcotel "github.com/kyma-project/kyma-metrics-collector/pkg/otel"
	kmcprocess "github.com/kyma-project/kyma-metrics-collector/pkg/process"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource/node"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource/pvc"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource/redis"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource/vsc"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime/kubeconfigprovider"
	"github.com/kyma-project/kyma-metrics-collector/pkg/service"
)

const (
	metricsPath            = "/metrics"
	healthzPath            = "/healthz"
	edpCredentialsFile     = "/edp-credentials/token"
	kubeconfigProviderName = "kubeconfig"
)

func main() {
	opts := options.ParseArgs()
	logger := log.NewLogger(opts.LogLevel)
	logger.Infof("Starting application with options: %v", opts.String())

	logger.Info("Setting up OTel SDK")

	otelShutdown, err := kmcotel.SetupSDK(context.Background())
	if err != nil {
		logger.With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).Fatal("Set up OTel SDK")
	}

	defer func() {
		if err := otelShutdown(context.Background()); err != nil {
			logger.Errorf("Failed to shutdown OTel SDK: %v", err)
		}
	}()

	kmcmetrics.RegisterTLSCacheMetrics()

	cfg := new(env.Config)
	if err := envconfig.Process("", cfg); err != nil {
		logger.With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).Fatal("Load env config")
	}

	// Load public cloud specs
	publicCloudSpecs, err := config.LoadPublicCloudSpecs(cfg)
	if err != nil {
		logger.With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).Fatal("Load public cloud spec")
	}

	logger.Debugf("public cloud spec: %v", publicCloudSpecs)

	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		logger.With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).Fatal("Load InCluster Config")
	}

	secretCacheClient, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		logger.With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).Fatal("Setup secrets client")
	}

	// Create a client for KEB communication
	kebConfig := new(keb.Config)
	if err := envconfig.Process("", kebConfig); err != nil {
		logger.With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).Fatal("Load KEB config")
	}

	kebClient := keb.NewClient(kebConfig, logger)
	logger.Debugf("keb config: %v", kebConfig)

	// Creating EDP client
	edpConfig := new(edp.Config)
	if err := envconfig.Process("", edpConfig); err != nil {
		logger.With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).Fatal("Load EDP config")
	}

	// read the token from the mounted secret
	token, err := getEDPToken()
	if err != nil {
		logger.With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).Fatal("Load EDP token")
	}

	edpConfig.Token = token

	edpClient := edp.NewClient(edpConfig, logger)

	nodeScanner := node.NewScanner(publicCloudSpecs)
	pvcScanner := pvc.NewScanner()
	redisScanner := redis.NewScanner(publicCloudSpecs)
	vscScanner := vsc.NewScanner()
	edpCollector := edp.NewCollector(
		edpClient,
		logger,
		nodeScanner,
		pvcScanner,
		redisScanner,
		vscScanner,
	)

	kubeconfigProvider := kubeconfigprovider.New(secretCacheClient.CoreV1(), logger, opts.KubeconfigCacheTTL, kubeconfigProviderName)

	kmcProcess, err := kmcprocess.New(
		kebClient,
		edpClient,
		edpCollector,
		kubeconfigProvider,
		publicCloudSpecs,
		opts.ScrapeInterval,
		opts.WorkerPoolSize,
		logger,
	)
	if err != nil {
		logger.With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).Fatal("Create KMC process")
	}

	// Start execution
	go kmcProcess.Start()

	// add debug service.
	if opts.DebugPort > 0 {
		enableDebugging(opts.DebugPort, logger)
	}

	router := mux.NewRouter()
	router.Path(healthzPath).HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})
	router.Path(metricsPath).Handler(promhttp.Handler())

	kmcSvr := service.Server{
		Addr:   fmt.Sprintf(":%d", opts.ListenAddr),
		Logger: logger,
		Router: router,
	}

	// Start a server to cater to the metrics and healthz endpoints
	kmcSvr.Start()
}

func enableDebugging(debugPort int, log *zap.SugaredLogger) {
	debugRouter := mux.NewRouter()
	// for security reason we always listen on localhost
	debugSvc := service.Server{
		Addr:   fmt.Sprintf("127.0.0.1:%d", debugPort),
		Logger: log,
		Router: debugRouter,
	}

	debugRouter.HandleFunc("/debug/pprof/", pprof.Index)
	debugRouter.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	debugRouter.HandleFunc("/debug/pprof/profile", pprof.Profile)
	debugRouter.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	debugRouter.HandleFunc("/debug/pprof/trace", pprof.Trace)
	debugRouter.Handle("/debug/pprof/block", pprof.Handler("block"))
	debugRouter.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	debugRouter.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	debugRouter.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))

	go func() {
		debugSvc.Start()
	}()
}

// getEDPToken read the EDP token from the mounted secret file.
func getEDPToken() (string, error) {
	token, err := os.ReadFile(edpCredentialsFile)
	if err != nil {
		return "", err
	}

	trimmedToken := strings.TrimSuffix(string(token), "\n")

	return trimmedToken, nil
}
