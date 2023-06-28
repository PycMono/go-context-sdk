package tracing

import (
	"context"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"io"
	"os"
	"testing"
)

func Test1(t *testing.T) {
	tracer, closer := initJaeger("hello-world")
	defer closer.Close()

	span := tracer.StartSpan("mk-clue-svc")
	span.SetTag("word", "hello。。。")
	span.LogFields(
		log.String("event", "string-format"),
		log.String("value", "dddd"),
	)
	span.LogKV("event", "println")
	span.Finish()
}

func Test2(t *testing.T) {
	helloTo := os.Args[1]
	tracer, closer := initJaeger("hello-world")
	defer closer.Close()

	span := tracer.StartSpan("say-hello")
	span.SetTag("hello-to", helloTo)
	defer span.Finish()

	helloStr := formatString(span, helloTo)
	printHello(span, helloStr)
}

func formatString(rootSpan opentracing.Span, helloTo string) string {
	span := rootSpan.Tracer().StartSpan(
		"formatString",
		opentracing.ChildOf(rootSpan.Context()),
	)
	defer span.Finish()

	helloStr := fmt.Sprintf("Hello, %s!", helloTo)
	span.LogFields(
		log.String("event", "string-format"),
		log.String("value", helloStr),
	)

	return helloStr
}

func printHello(rootSpan opentracing.Span, helloStr string) {
	span := rootSpan.Tracer().StartSpan(
		"printHello",
		opentracing.ChildOf(rootSpan.Context()),
	)
	defer span.Finish()

	println(helloStr)
	span.LogKV("event", "println")
}

func Test3(t *testing.T) {
	helloTo := os.Args[1]
	tracer, closer := initJaeger("hello-world")
	defer closer.Close()

	opentracing.SetGlobalTracer(tracer)
	span := tracer.StartSpan("say-hello")
	span.SetTag("hello-to", helloTo)
	defer span.Finish()

	ext.HTTPUrl.Set(span, "www.baidu.com")
	ext.HTTPMethod.Set(span, "GET")

	ctx := context.Background()
	ctx = opentracing.ContextWithSpan(ctx, span)

	helloStr := formatString1(ctx, helloTo)
	printHello1(ctx, helloStr)

	span = opentracing.SpanFromContext(ctx)

	span.Tracer().Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		nil,
	)

	spanCtx, _ := span.Context().(jaeger.SpanContext)
	fmt.Println("---------------------")
	fmt.Println(spanCtx.TraceID())
}

func formatString1(ctx context.Context, helloTo string) string {
	span, _ := opentracing.StartSpanFromContext(ctx, "formatString")
	defer span.Finish()

	helloStr := fmt.Sprintf("Hello, %s!", helloTo)
	span.LogFields(
		log.String("event", "string-format"),
		log.String("value", helloStr),
	)

	return helloStr
}

func printHello1(ctx context.Context, helloStr string) {
	span, _ := opentracing.StartSpanFromContext(ctx, "printHello")
	defer span.Finish()

	println(helloStr)
	span.LogKV("event", "println")
}

// initJaeger returns an instance of Jaeger Tracer that samples 100% of traces and logs all spans to stdout.
func initJaeger(service string) (opentracing.Tracer, io.Closer) {
	cfg := &config.Configuration{
		ServiceName: service,
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans: true,
		},
	}
	tracer, closer, err := cfg.NewTracer(config.Logger(jaeger.StdLogger))
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	return tracer, closer
}
