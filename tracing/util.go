package tracing

import "github.com/google/uuid"

// NewRequestID 生成一个新的 requestID（标准 UUID v4）
func NewRequestID() string {
	return uuid.New().String()
}
