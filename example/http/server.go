package httpserver

import (
	"context"
	"errors"
	"fmt"
	"github.com/PycMono/go-context-sdk/tracing"
	"io"
	"net"
	"net/http"
	"os"
)

type server struct {
	welcome  string
	forward  string
	ip       string
	hostname string
}

// NewServer creates a new server
func NewServer(forward, grpcForward, welcome string) (*server, error) {
	svr := &server{forward: forward, welcome: welcome}
	svr.hostname, _ = os.Hostname()
	svr.ip, _ = externalIP()

	return svr, nil
}

// SpanHandler echos back the request body as a response
func (s *server) SpanHandler(writer http.ResponseWriter, request *http.Request) {
	span := tracing.StartNewSpan("span-test-parent")
	defer span.Finish()

	nspan := span.StartChildSpan("span-test-op1")
	nspan.SetTag("my-custom-account", "user1")
	nspan.Finish()

	nspan = span.StartChildSpan("span-test-op2")
	nspan.Finish()

	ctx := span.WithContextSetOpentracing(request.Context())
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:5005/test-span", nil)
	tracing.Inject(req, span)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		httpError(writer, err)
		return
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	s.writeResult(ctx, writer, b)

	//resp, err := grequestsx.Get(s.forward+"/test-span", opts)
	//if err != nil {
	//	httpError(writer, err)
	//	return
	//}
	//s.writeResult(bizctx, writer, resp.Bytes())
	//return

	s.writeResult(ctx, writer, []byte("ok"))
}

func (s *server) Redirect(writer http.ResponseWriter, request *http.Request) {
	span, _ := tracing.Extract(request)
	nspan := span.StartChildSpan("span-test-op2")
	nspan.Finish()
	ctx := nspan.WithContextSetOpentracing(request.Context())
	s.writeResult(ctx, writer, []byte("ok"))
}

func (s *server) writeResult(ctx context.Context, writer http.ResponseWriter, data []byte) {
	span := tracing.SpanFromContext(ctx)

	writer.Write(data)
	writer.Write([]byte(fmt.Sprintf("----via %s, ip: %s-----\n", s.hostname, s.ip)))
	writer.Write([]byte(fmt.Sprintf("----headers: %v", span)))
	writer.Write([]byte("\n\n"))
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
