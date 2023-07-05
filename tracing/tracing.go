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
	var stop = make(chan os.Signal, 2)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGABRT, syscall.SIGALRM, syscall.SIGBUS, syscall.SIGCHLD, syscall.SIGCONT, syscall.SIGEMT, syscall.SIGFPE, syscall.SIGHUP, syscall.SIGILL, syscall.SIGINFO, syscall.SIGINT, syscall.SIGIO, syscall.SIGIOT, syscall.SIGKILL, syscall.SIGPIPE, syscall.SIGPROF, syscall.SIGQUIT, syscall.SIGSEGV, syscall.SIGSTOP, syscall.SIGSYS, syscall.SIGTERM, syscall.SIGTRAP, syscall.SIGTSTP, syscall.SIGTTIN, syscall.SIGTTOU, syscall.SIGURG, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGVTALRM, syscall.SIGWINCH, syscall.SIGXCPU, syscall.SIGXFSZ)

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
		close(stop)
		err = closer.Close()
		if err != nil {
			log.Println("tracer close:", err)
		}
	}()

	opentracing.SetGlobalTracer(tracer)

	select {
	case st := <-stop:
		{
			log.Println("tracer stop:", st)
			return
		}
	}
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
