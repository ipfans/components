//go:build tools

package components

import (
	_ "go.uber.org/mock/gomock"
	_ "go.uber.org/mock/mockgen"
)

//go:generate go run -mod=mod go.uber.org/mock/mockgen@latest -destination=./mocks/redis/mock_redis.go -package=redis github.com/redis/go-redis/v9 Cmdable
//go:generate go run -mod=mod go.uber.org/mock/mockgen@latest -destination=./mocks/lifecycle/mock_lifecycle.go -package=lifecycle github.com/ipfans/components/v2/lifecycle Lifecycle
