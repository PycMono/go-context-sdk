package main

import (
	"fmt"
	httpserver "github.com/PycMono/go-context-sdk/example/http"
	"golang.org/x/sync/errgroup"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var (
	g errgroup.Group
)

func main() {
	httpSvr, err := httpserver.NewServer(
		"",
		"",
		"")
	if err != nil {
	}

	http.HandleFunc("/span", httpSvr.SpanHandler)
	http.HandleFunc("/test-span", httpSvr.Redirect)
	//http.Handle("/", withTrace(http.HandlerFunc(httpSvr.EchoHandler)))

	g.Go(func() error {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
		for {
			si := <-c
			switch si {
			case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
				return shutdown()
			case syscall.SIGHUP:
			default:
				return nil
			}
		}
	})

	err = http.ListenAndServe(fmt.Sprintf(":%d", 5005), nil)
	if err != nil {
		panic(err)
	}
}

// shutdown 关闭服务
func shutdown() error {
	os.Exit(0)
	return nil
}
