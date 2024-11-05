package httpclient

import (
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/ipfans/components/v2/ctxkeys"
	"github.com/ipfans/components/v2/utils"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Config struct {
	Debug     bool          `koanf:"debug"`
	UserAgent string        `koanf:"user_agent"`
	Timeout   time.Duration `koanf:"timeout"`
}

type Option func(*options)

type options struct {
	logger zerolog.Logger
}

func defaultOptions() *options {
	return &options{
		logger: log.Logger,
	}
}

func WithLogger(logger zerolog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

func New(cfg Config, opts ...Option) *resty.Client {
	options := defaultOptions()
	for _, o := range opts {
		o(options)
	}
	client := resty.New().
		SetHeader("User-Agent", utils.DefaultValue(cfg.UserAgent, "HttpClient/1.0")).
		SetTimeout(utils.DefaultValue(cfg.Timeout, 60*time.Second)).
		OnBeforeRequest(func(client *resty.Client, request *resty.Request) error {
			ctx := request.Context()
			reqID := ctxkeys.RequestID(ctx)
			if reqID == "" {
				reqID = utils.NewUUID()
			}
			request.SetHeader("X-Request-ID", reqID)
			return nil
		}).
		OnAfterResponse(func(client *resty.Client, response *resty.Response) error {
			return nil
		})
	if cfg.Debug {
		client.OnRequestLog(func(req *resty.RequestLog) error {
			options.logger.Debug().
				Str("request_id", req.Header.Get("X-Request-ID")).
				Any("headers", req.Header).
				Str("body", req.Body).
				Msg("http request")
			return nil
		})
		client.OnResponseLog(func(resp *resty.ResponseLog) error {
			options.logger.Debug().
				Str("request_id", resp.Header.Get("X-Request-ID")).
				Any("headers", resp.Header).
				Str("body", resp.Body).
				Msg("http response")
			return nil
		})
		client.OnError(func(req *resty.Request, err error) {
			options.logger.Error().
				Str("request_id", req.Header.Get("X-Request-ID")).
				Err(err).
				Msg("http request error")
		})
	}
	return client
}

type wrapperTransport struct {
	client *resty.Client
}

func (t *wrapperTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	reqID := ctxkeys.RequestID(ctx)
	if reqID == "" {
		reqID = utils.NewUUID()
	}
	resp, err := t.client.R().
		SetBody(req.Body).
		SetHeaderMultiValues(req.Header).
		SetHeader("X-Request-ID", reqID).
		SetDoNotParseResponse(true).
		Execute(req.Method, req.URL.String())
	if err != nil {
		return nil, err
	}
	return resp.RawResponse, nil
}

func WrapClient(client *resty.Client) *http.Client {
	return &http.Client{
		Transport: &wrapperTransport{client: client},
	}
}
