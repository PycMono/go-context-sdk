package httpserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	"github.com/PycMono/go-context-sdk/bizctx"
	"github.com/PycMono/go-context-sdk/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type server struct {
	welcome  string
	forward  string
	ip       string
	hostname string
	client   *http.Client
}

// NewServer creates a new server
func NewServer(forward, grpcForward, welcome string) (*server, error) {
	svr := &server{
		forward: forward,
		welcome: welcome,
		client: &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}
	svr.hostname, _ = os.Hostname()
	svr.ip, _ = externalIP()

	return svr, nil
}

// SpanHandler echos back the request body as a response
func (s *server) SpanHandler(writer http.ResponseWriter, request *http.Request) {
	ctx := tracing.Extract(request.Context(), request.Header)

	// 从 context 中恢复 parent span 并创建子 span
	ctx, span := tracing.StartSpan(ctx, "span-test-parent", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	_, nspan := tracing.StartSpan(ctx, "span-test-op1")
	nspan.SetAttributes(attribute.String("my-custom-account", "user1"))
	nspan.AddEvent("custom-log", trace.WithAttributes(
		attribute.String("event", "custom-log"),
		attribute.String("account", "user1"),
	))
	nspan.End()

	_, nspan = tracing.StartSpan(ctx, "span-test-op2")
	nspan.End()

	// 在内存中设置业务上下文（bizctx 的 HTTP 传输由 go-gin-sdk 或业务方自行实现）
	ctx = bizctx.WithKV(ctx, bizctx.UserID("user1"), bizctx.TenantID("tenant1"))

	// 发起下游 HTTP 调用（tracing 跨服务传播保留，bizctx HTTP 传播需由上层框架处理）
	target := s.forward
	if target == "" {
		target = "http://localhost:5005/test-span"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		httpError(writer, err)
		return
	}
	resp, err := s.client.Do(req)
	if err != nil {
		httpError(writer, err)
		return
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		httpError(writer, err)
		return
	}
	s.writeResult(ctx, writer, b)

	s.writeResult(ctx, writer, []byte("ok"))
}

func (s *server) Redirect(writer http.ResponseWriter, request *http.Request) {
	ctx := tracing.Extract(request.Context(), request.Header)
	ctx, nspan := tracing.StartSpan(ctx, "span-test-op2", trace.WithSpanKind(trace.SpanKindServer))
	defer nspan.End()

	s.writeResult(ctx, writer, []byte("ok"))
}

func (s *server) writeResult(ctx context.Context, writer http.ResponseWriter, data []byte) {
	span := trace.SpanFromContext(ctx)

	writer.Write(data)
	writer.Write([]byte(fmt.Sprintf("----via %s, ip: %s-----\n", s.hostname, s.ip)))
	writer.Write([]byte(fmt.Sprintf("----headers: %v\n", span)))
	writer.Write([]byte(fmt.Sprintf("----userid: %s, tenantid: %s\n",
		bizctx.GetUserID(ctx), bizctx.GetTenantID(ctx))))
	writer.Write([]byte("\n"))
}

func httpError(writer http.ResponseWriter, err error) {
	writer.WriteHeader(http.StatusInternalServerError)
	writer.Write([]byte(fmt.Sprintf("Error: %v", err)))
}

func externalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}
