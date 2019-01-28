package main

import (
	"fmt"
	"github.com/InVisionApp/go-health"
	"github.com/InVisionApp/go-health/handlers"
	prometheusPlugin "github.com/ProtocolONE/go-micro-plugins/wrapper/monitoring/prometheus"
	"github.com/ProtocolONE/mfa-service/pkg"
	"github.com/ProtocolONE/mfa-service/pkg/proto"
	"github.com/go-redis/redis"
	"github.com/kelseyhightower/envconfig"
	"github.com/micro/go-micro"
	k8s "github.com/micro/kubernetes/go/micro"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type Config struct {
	RedisAddr      string `envconfig:"REDIS_ADDR" required:"true"`
	KubernetesHost string `envconfig:"KUBERNETES_SERVICE_HOST" required:"false"`
	MetricsPort    int    `envconfig:"METRICS_PORT" required:"false" default:"8081"`
}

type customHealthCheck struct{}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any

	cfg := &Config{}

	if err := envconfig.Process("", cfg); err != nil {
		logger.Fatal("Config init failed with error", zap.Error(err))
	}

	r := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})

	defer func() {
		err := r.Close()
		if err != nil {
			logger.Error("redis shutdown with error", zap.Error(err))
		}
	}()

	var service micro.Service

	options := []micro.Option{
		micro.Name(mfa.ServiceName),
		micro.Version(mfa.Version),
		micro.WrapHandler(prometheusPlugin.NewHandlerWrapper()),
	}

	if cfg.KubernetesHost == "" {
		service = micro.NewService(options...)
		logger.Info("Initialize micro service")
	} else {
		service = k8s.NewService(options...)
		logger.Info("Initialize k8s service")
	}

	service.Init()

	err := proto.RegisterMfaServiceHandler(service.Server(), mfa.NewService(r, logger))
	if err != nil {
		logger.Fatal("Register MfaServiceHandler failed with error", zap.Error(err))
	}

	initHealth(cfg, logger)
	initMetrics()

	go func() {
		if err = http.ListenAndServe(fmt.Sprintf(":%d", cfg.MetricsPort), nil); err != nil {
			logger.Fatal("Metrics listen failed")
		}
	}()

	if err := service.Run(); err != nil {
		logger.Fatal("service run failed with error", zap.Error(err))
	}
}

func initHealth(cfg *Config, logger *zap.Logger) {
	h := health.New()
	err := h.AddChecks([]*health.Config{
		{
			Name:     "health-check",
			Checker:  &customHealthCheck{},
			Interval: time.Duration(1) * time.Second,
			Fatal:    true,
		},
	})

	if err != nil {
		logger.Fatal("Health check register failed with error", zap.Error(err))
	}

	logger.Info("Health check listening on port", zap.Int("port", cfg.MetricsPort))

	if err = h.Start(); err != nil {
		logger.Fatal("Health check start failed with error", zap.Error(err))
	}

	http.HandleFunc("/health", handlers.NewJSONHandlerFunc(h, nil))
}

func initMetrics() {
	http.Handle("/metrics", promhttp.Handler())
}

func (c *customHealthCheck) Status() (interface{}, error) {
	return "ok", nil
}
