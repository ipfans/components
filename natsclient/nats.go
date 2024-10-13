package natsclient

import (
	"github.com/ipfans/components/utils"
	"github.com/nats-io/nats.go"
)

type Config struct {
	URL      string   `koanf:"url"`      // Default: nats://localhost:4222
	Services []string `koanf:"services"` // Default: []
}

type Option func(*nats.Options)

func New(opts ...Option) func(conf Config) (*nats.Conn, error) {
	return func(conf Config) (conn *nats.Conn, err error) {
		opt := nats.GetDefaultOptions()
		opt.Url = utils.DefaultValue(conf.URL, "nats://127.0.0.1:4222")

		conn, err = opt.Connect()
		return
	}
}
