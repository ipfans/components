package log

import (
	"io"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Handler func(zerolog.Logger) zerolog.Logger

type Config struct {
	NoGlobal bool          // replace `github.com/rs/zerolog/log.Logger` or not.
	Writer   io.Writer     // Output writer
	NoColor  bool          // Disable color
	Level    zerolog.Level // Default level is zerolog.DebugLevel
	Handlers []Handler     // Handlers to handle zerolog.Logger
}

// NewConsoleLogger returns console writer logger instance.
func NewConsoleLogger(conf ...Config) func() (zerolog.Logger, error) {
	return func() (logger zerolog.Logger, err error) {
		var c Config
		c.Level = zerolog.InfoLevel
		if len(conf) > 0 {
			c = conf[0]
		}
		logger = log.Output(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
			w.Out = c.Writer
			if c.Writer == nil {
				w.Out = os.Stdout
			}
			w.NoColor = c.NoColor
		})).Level(c.Level)
		for _, handle := range c.Handlers {
			logger = handle(logger)
		}
		if !c.NoGlobal {
			log.Logger = logger
		}
		return
	}
}

// NewJSONLogger returns json logger instance.
func NewJSONLogger(conf ...Config) func() (zerolog.Logger, error) {
	return func() (logger zerolog.Logger, err error) {
		var c Config
		c.Level = zerolog.InfoLevel
		if len(conf) > 0 {
			c = conf[0]
		}
		if c.Writer == nil {
			c.Writer = os.Stdout
		}
		logger = zerolog.New(c.Writer).With().Timestamp().Logger()
		logger = logger.Level(c.Level)
		for _, handle := range c.Handlers {
			logger = handle(logger)
		}
		if !c.NoGlobal {
			log.Logger = logger
		}
		return
	}
}
