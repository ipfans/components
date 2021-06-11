package config

import (
	"github.com/spf13/viper"
)

type Handler func(v *viper.Viper)

func NewConfigure(handlers ...Handler) func() (*viper.Viper, error) {
	return func() (v *viper.Viper, err error) {
		v = viper.GetViper()
		v.SetConfigFile("config.yml")
		v.AddConfigPath(".")
		v.AutomaticEnv()
		for _, handler := range handlers {
			handler(v)
		}
		err = v.ReadInConfig()
		return
	}
}
