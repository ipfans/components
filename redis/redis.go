package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	Addr        string `koanf:"addr"`         // Required, Example: 127.0.0.1:6379
	DisablePing bool   `koanf:"disable_ping"` // Optional, Default: false
	Username    string `koanf:"username"`     // Optional, If empty, no username is used
	Password    string `koanf:"password"`     // Optional, If empty, no password is used
	DB          int    `koanf:"db"`           // Optional, Default: 0
}

func New(conf Config) (client *redis.Client, err error) {
	client = redis.NewClient(&redis.Options{
		Addr:         conf.Addr,
		Username:     conf.Username,
		Password:     conf.Password,
		DB:           conf.DB,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
	if !conf.DisablePing {
		if err = client.Ping(context.TODO()).Err(); err != nil {
			return
		}
	}
	return
}
