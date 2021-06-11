package nats

import (
	"time"

	"github.com/nats-io/nats.go"
	"github.com/spf13/viper"
)

type Handler func(*nats.Conn)

type Config struct {
	Opts     []nats.Option
	Handlers []Handler
}

func NewNats(conf ...Config) func(*viper.Viper) (*nats.Conn, error) {
	return func(v *viper.Viper) (conn *nats.Conn, err error) {
		var c Config
		if len(conf) > 0 {
			c = conf[0]
		}
		opts := []nats.Option{
			nats.MaxReconnects(5),
			nats.ReconnectWait(time.Second / 10),
			nats.Timeout(3 * time.Second),
		}
		if v.GetString("nats.name") != "" {
			opts = append(opts, nats.Name(v.GetString("nats.name")))
		}
		opts = append(opts, c.Opts...)
		if conn, err = nats.Connect(v.GetString("nats.server"), opts...); err != nil {
			return
		}
		for _, handle := range c.Handlers {
			handle(conn)
		}
		return
	}
}
