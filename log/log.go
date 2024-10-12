package log

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/arthurkiller/rollingwriter"
	"github.com/ipfans/components/utils"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/bridges/otelzerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

type OpenTelemetry struct {
	Enabled     bool   `koanf:"enabled"`      // Optional, Default: false
	PackageName string `koanf:"package_name"` // Optional, Default: empty. It will be used to filter the logs by package name.
	Endpoint    string `koanf:"endpoint"`     // Optional, Default: empty. It will be required if OpenTelemetry is enabled.
	AuthMethod  string `koanf:"auth_method"`  // Optional, Default: empty. One of "Bearer", "Basic", "Header", "".
	AuthToken   string `koanf:"auth_token"`   // Optional, Default: empty.
	AuthHeader  string `koanf:"auth_header"`  // Optional, Default: empty. It will be used when AuthMethod is "Header".
}

type Config struct {
	OpenTelemetry OpenTelemetry `koanf:"opentelemetry" mapstructure:",squash"`
	NoGlobal      bool          `koanf:"no_global"` // do not replace `github.com/rs/zerolog/log.Logger` or register global logger.
	NoColor       bool          `koanf:"no_color"`  // Disable color, it will be ignored if OpenTelemetry is enabled.
	Output        string        `koanf:"output"`    // Output writer, Default: stdout. It can be "stdout", "stderr", or a file path.
	Level         string        `koanf:"level"`     // Default level is zerolog.DebugLevel. Default: "info". It can be "debug", "info", "warn", "error", "fatal", "panic".
}

type option struct {
	w          io.Writer
	provider   otellog.LoggerProvider
	loggerFunc func(logger zerolog.Logger) zerolog.Logger
	otelAttrs  []attribute.KeyValue
}

type Handler func(opt option)

// WithWriter to force use the given writer.
func WithWriter(w io.Writer) Handler {
	return func(opt option) {
		opt.w = w
	}
}

// WithProvider to force use the given provider.
func WithProvider(provider otellog.LoggerProvider) Handler {
	return func(opt option) {
		opt.provider = provider
	}
}

// ExtendLocalLogger to extend the local logger instance.
func ExtendLocalLogger(fn func(logger zerolog.Logger) zerolog.Logger) Handler {
	return func(opt option) {
		opt.loggerFunc = fn
	}
}

func WithOtelAttributes(attrs ...attribute.KeyValue) Handler {
	return func(opt option) {
		opt.otelAttrs = attrs
	}
}

// New returns a new zerolog.Logger instance. If a provider is provided, it will be used to create the logger. Notice: use context to close the provider and writer.
func New(ctx context.Context, handlers ...Handler) func(conf Config) (zerolog.Logger, error) {
	return func(conf Config) (logger zerolog.Logger, err error) {
		opt := option{}
		for _, handler := range handlers {
			handler(opt)
		}

		var writer rollingwriter.RollingWriter
		if opt.w == nil {
			switch utils.DefaultValue(strings.ToLower(conf.Output), "stdout") {
			case "stdout":
				opt.w = os.Stdout
			case "stderr":
				opt.w = os.Stderr
			default:
				writer, err = rollingwriter.NewWriter(rollingwriter.WithLock(), rollingwriter.WithLogPath(conf.Output), rollingwriter.WithoutRollingPolicy())
				if err != nil {
					return
				}
				opt.w = writer
			}
		}

		level, err := zerolog.ParseLevel(utils.DefaultValue(conf.Level, "info"))
		if err != nil {
			return
		}
		logger = zerolog.New(opt.w).With().Logger().Level(level)

		// Init the opentelemetry otelhttplog provider
		var logExporter *otlploghttp.Exporter
		if conf.OpenTelemetry.Enabled && opt.provider == nil {
			if conf.OpenTelemetry.Endpoint != "" {
				err = errors.New("otel endpoint is required")
				return
			}
			headers := map[string]string{"VL-Stream-Fields": "telemetry.sdk.language,severity,service.name"}
			switch conf.OpenTelemetry.AuthMethod {
			case "Bearer":
				headers["Authorization"] = "Bearer " + conf.OpenTelemetry.AuthToken
			case "Basic":
				headers["Authorization"] = "Basic " + conf.OpenTelemetry.AuthToken
			case "Header":
				headers[conf.OpenTelemetry.AuthHeader] = conf.OpenTelemetry.AuthToken
			default:
			}

			logExporter, err = otlploghttp.New(context.TODO(),
				otlploghttp.WithEndpointURL(conf.OpenTelemetry.Endpoint),
				otlploghttp.WithHeaders(headers),
			)
			if err != nil {
				err = fmt.Errorf("failed to create otel exporter: %w", err)
				return
			}
			attrs := []attribute.KeyValue{
				semconv.ServiceName(conf.OpenTelemetry.PackageName),
			}
			attrs = append(attrs, opt.otelAttrs...)
			res := resource.NewWithAttributes(
				semconv.SchemaURL,
				attrs...,
			)
			logProvider := sdklog.NewLoggerProvider(
				sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
				sdklog.WithResource(res),
			)
			opt.provider = logProvider
		}

		go func() {
			<-ctx.Done()
			if writer != nil {
				writer.Close()
			}
			if logExporter != nil {
				logExporter.Shutdown(context.TODO())
			}
		}()

		if conf.OpenTelemetry.Enabled {
			// Use the first provider and ignore the rest opentelemetry configuration.
			hook := otelzerolog.NewHook(conf.OpenTelemetry.PackageName, otelzerolog.WithLoggerProvider(opt.provider))
			logger = logger.Hook(hook)
			if !conf.NoGlobal {
				log.Logger = logger
			}
			return
		}

		// Use local logger instance if opentelemetry is not enabled
		logger = zerolog.New(opt.w).With().Logger().Level(level)
		if opt.loggerFunc != nil {
			logger = opt.loggerFunc(logger)
		}
		log.Logger = logger
		return
	}
}
