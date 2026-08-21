package tracing

import (
	"os"
	"sync/atomic"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const (
	instrumentationScopeName    = "github.com/PycMono/go-context-sdk"
	instrumentationScopeVersion = "v1.2.0"
)

type tracerProviderHolder struct {
	provider trace.TracerProvider
}

var activeTracerProvider atomic.Pointer[tracerProviderHolder]

func init() {
	activeTracerProvider.Store(&tracerProviderHolder{provider: noop.NewTracerProvider()})
}

// Init sets the OTel TracerProvider used by this package. A nil provider is ignored.
func Init(provider trace.TracerProvider) {
	if provider == nil {
		return
	}
	activeTracerProvider.Store(&tracerProviderHolder{provider: provider})
}

func tracer() trace.Tracer {
	provider := activeTracerProvider.Load().provider
	return provider.Tracer(
		instrumentationScopeName,
		trace.WithInstrumentationVersion(instrumentationScopeVersion),
	)
}

// ServiceName returns the configured service name or the local hostname.
func ServiceName() string {
	if serviceName := os.Getenv("OTEL_SERVICE_NAME"); serviceName != "" {
		return serviceName
	}
	if serviceName := os.Getenv("SERVICE_NAME"); serviceName != "" {
		return serviceName
	}
	serviceName, _ := os.Hostname()
	return serviceName
}
