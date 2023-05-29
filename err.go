package goctx

import "errors"

var (
	// ErrNullRequest 错误请求
	ErrNullRequest = errors.New("request is nil")
)
