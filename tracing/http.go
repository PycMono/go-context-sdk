package tracing

import (
	"net/http"
	"strings"
)

// Inject 将Span信息注入到 `http.Request` 中
func Inject(r *http.Request, span *Span) error {
	if r == nil {
		return ErrNullRequest
	}

	if span == nil {
		return nil
	}
	for k, v := range span.SpanContext {
		r.Header.Set(k, v)
	}
	return nil
}

// Extract 从 `http.Request` 提取Span信息
func Extract(r *http.Request, extHeaders ...string) (*Span, error) {
	if r == nil {
		return nil, ErrNullRequest
	}

	// 获取请求头中的span信息
	if span := SpanFromContext(r.Context()); !span.Empty() {
		return span, nil
	}

	span := extract(r.Header, extHeaders...)
	span.Set(HeaderRequestPath, r.URL.Path)
	return span, nil
}

func extract(headers map[string][]string, extHeaders ...string) *Span {
	out := &Span{SpanContext: make(SpanContext)}

	// 定义允许的 tracing header 白名单
	allowed := map[string]bool{
		"x-b3-traceid":      true,
		"x-b3-parentspanid": true,
		"x-b3-spanid":       true,
		"x-b3-sampled":      true,
		HeaderRequestID:     true,
		HeaderAuthAccountID: true,
		HeaderRequestPath:   true,
	}
	for _, h := range extHeaders {
		allowed[strings.ToLower(h)] = true
	}

	for rawKey, rawValue := range headers {
		var value string
		if len(rawValue) > 0 {
			value = rawValue[0]
		}
		key := strings.ToLower(rawKey)
		if !allowed[key] {
			continue
		}
		out.Set(key, value)
	}
	return out
}
