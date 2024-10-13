package configuration

import (
	"github.com/go-viper/mapstructure/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Option func(*koanf.Koanf) error

func WithConfigFile(path string, parser koanf.Parser) Option {
	return func(conf *koanf.Koanf) error {
		return conf.Load(file.Provider(path), parser)
	}
}

func WithProvider(provider koanf.Provider, parser koanf.Parser) Option {
	return func(conf *koanf.Koanf) error {
		return conf.Load(provider, parser)
	}
}

func Load(config interface{}, opts ...Option) (err error) {
	conf := koanf.New(".")
	for _, opt := range opts {
		if err = opt(conf); err != nil {
			return err
		}
	}

	if err = conf.UnmarshalWithConf("", config, koanf.UnmarshalConf{
		Tag: "koanf",
		DecoderConfig: &mapstructure.DecoderConfig{
			DecodeHook:       mapstructure.ComposeDecodeHookFunc(mapstructure.StringToTimeDurationHookFunc()),
			Result:           config,
			WeaklyTypedInput: true,
			Squash:           true,
		},
	}); err != nil {
		return err
	}
	return nil
}
