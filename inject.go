package goctx

import "net/http"

// HttpHeaderInject http 请求头注入
func HttpHeaderInject(r *http.Request, in BizCtxValue) error {
	if r == nil {
		return ErrNullRequest
	}
	if in == nil {
		return nil
	}

	for k, v := range in {
		r.Header.Set(k, v)
	}
	return nil
}
