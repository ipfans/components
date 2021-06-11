package redis

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
)

type Handler func(*redis.Client)

func NewRedisClient(handlers ...Handler) func(*viper.Viper) (*redis.Client, error) {
	return func(v *viper.Viper) (client *redis.Client, err error) {
		client = redis.NewClient(&redis.Options{
			Addr:         v.GetString("redis.addr"),
			Username:     v.GetString("redis.username"),
			Password:     v.GetString("redis.password"),
			DB:           v.GetInt("redis.db"),
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		})
		for _, handle := range handlers {
			handle(client)
		}
		if err = client.Ping(context.TODO()).Err(); err != nil {
			return
		}
		return
	}
}
