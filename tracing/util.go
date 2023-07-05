package tracing

import "github.com/google/uuid"

// NewRequestID 生成一个新的requestID
func NewRequestID() string {
	u := uuid.New().String()
	return u[0:14] + "9" + u[15:]
}
