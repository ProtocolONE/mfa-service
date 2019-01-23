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
	"log"
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
	cfg := &Config{}

	if err := envconfig.Process("", cfg); err != nil {
		log.Fatalf("Config init failed with error: %s\n", err)
	}

	r := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})

	defer func() {
		err := r.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	var service micro.Service

	options := []micro.Option{
		micro.Name(mfa.ServiceName),
		micro.Version(mfa.Version),
		micro.WrapHandler(prometheusPlugin.NewHandlerWrapper((*proto.MfaService)(nil))),
	}

	if cfg.KubernetesHost == "" {
		service = micro.NewService(options...)
		log.Println("Initialize micro service")
	} else {
		service = k8s.NewService(options...)
		log.Println("Initialize k8s service")
	}

	service.Init()

	err := proto.RegisterMfaServiceHandler(service.Server(), &mfa.Service{Redis: r})
	if err != nil {
		log.Fatal(err)
	}

	initHealth(cfg)
	initMetrics()

	go func() {
		if err = http.ListenAndServe(fmt.Sprintf(":%d", cfg.MetricsPort), nil); err != nil {
			log.Fatal("Metrics listen failed")
		}
	}()

	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}

func initHealth(cfg *Config) {
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
		log.Fatal("Health check register failed")
	}

	log.Printf("Health check listening on :%d", cfg.MetricsPort)

	if err = h.Start(); err != nil {
		log.Fatal("Health check start failed")
	}

	http.HandleFunc("/health", handlers.NewJSONHandlerFunc(h, nil))
}

func initMetrics() {
	http.Handle("/metrics", promhttp.Handler())
}

func (c *customHealthCheck) Status() (interface{}, error) {
	return "ok", nil
}
