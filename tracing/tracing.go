package tracing

import (
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	cfg "github.com/uber/jaeger-client-go/config"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var (
	// AppName app name
	AppName string
)

func init() {
	var stop = make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT)

	// 启动tracing
	go start(stop)
}

func start(stop chan os.Signal) {
	conf, err := cfg.FromEnv()
	if err != nil {
		log.Println("Jaeger read configuration from Env failed:", err)
		return
	}
	if conf.Disabled {
		log.Println("Jaeger disabled")
		return
	}
	if len(conf.Reporter.CollectorEndpoint) > 0 {
		log.Println("Jaeger collectorEndpoint:", conf.Reporter.CollectorEndpoint)
	} else {
		log.Println("Jaeger localAgentHostPort:", conf.Reporter.LocalAgentHostPort)
	}

	if len(conf.ServiceName) == 0 {
		conf.ServiceName = ServiceName()
	}
	AppName = conf.ServiceName

	if len(conf.Sampler.Type) == 0 {
		conf.Sampler.Type = "const"
		conf.Sampler.Param = 1
	}
	conf.Gen128Bit = true
	logger := cfg.Logger(jaeger.StdLogger)
	tracer, closer, err := conf.NewTracer(logger)
	if err != nil {
		log.Println("Jaeger new tracer failed:", err)
		return
	}

	defer func() {
		log.Println("结束tracing...")
		err = closer.Close()
		if err != nil {
			log.Println("tracer close:", err)
		}
	}()

	opentracing.SetGlobalTracer(tracer)

	st := <-stop
	log.Println("tracer stop:", st)
}

// ServiceName 服务名称
func ServiceName() string {
	if svcName := os.Getenv("SERVICE_NAME"); len(svcName) > 0 {
		return svcName
	}
	if svcName := os.Getenv("JAEGER_SERVICE_NAME"); len(svcName) > 0 {
		if idx := strings.Index(svcName, "."); idx >= 0 {
			return svcName[:idx]
		}
		return svcName
	}

	appName, _ := os.Hostname()
	return appName
}
