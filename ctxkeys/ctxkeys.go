package ctxkeys

import "context"

type key string

const (
	requestID key = "request_id"
)

func RequestID(ctx context.Context) string {
	val := ctx.Value(requestID)
	if val == nil {
		return ""
	}
	return val.(string)
}

func SetRequestID(ctx context.Context, reqID string) context.Context {
	return context.WithValue(ctx, requestID, reqID)
}
