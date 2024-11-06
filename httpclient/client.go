package httpclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/ipfans/components/v2/ctxkeys"
	"github.com/ipfans/components/v2/utils"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Config struct {
	UserAgent          string        `koanf:"UserAgent"`          // User agent, Default: HttpClient/1.0
	Timeout            time.Duration `koanf:"timeout"`            // Timeout, Default: 60 seconds
	RetryCount         int           `koanf:"RetryCount"`         // Retry count, Default: 3
	Debug              bool          `koanf:"debug"`              // Enable debug log, include request and response. Default: false
	DisableRetry       bool          `koanf:"DisableRetry"`       // Disable retry, Default: false
	LogAllResponse     bool          `koanf:"LogAllResponse"`     // Log all response body, Default only json/xml. It only works when debug is true. Default: false
	MaxRequestBodySize int64         `koanf:"MaxRequestBodySize"` // Max request body size, Default: 5 * 1024 (5KB)
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

// WithLogger sets the logger for the httpclient
func WithLogger(logger zerolog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

func betterPrintHeaders(headers http.Header) string {
	var builder strings.Builder
	for k, v := range headers {
		builder.WriteString(fmt.Sprintf("%s: ", k))
		builder.WriteString(strings.Join(v, ", "))
		builder.WriteString("\n")
	}
	return builder.String()
}

// New creates a new resty client
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
		client = client.OnBeforeRequest(func(client *resty.Client, request *resty.Request) error {
			logger := options.logger.Debug().Ctx(request.Context()).
				Str("method", request.Method).
				Str("url", request.URL).
				Any("headers", betterPrintHeaders(request.Header))
			if request.Body != nil {
				kind := reflect.Indirect(reflect.ValueOf(request.Body)).Type().Kind()
				switch kind {
				case reflect.String:
					logger = logger.Str("body", request.Body.(string))
				case reflect.Struct, reflect.Map, reflect.Slice:
					logger = logger.Interface("body", request.Body)
				default:
					switch v := request.Body.(type) {
					case io.Reader:
						logger = logger.Str("body", "***** REQUEST BODY IS AN IO.READER *****")
					default:
						logger = logger.Str("body", fmt.Sprintf("%v", v))
					}
				}
			}
			logger.Msg("httpclient request")
			return nil
		}).OnAfterResponse(func(client *resty.Client, response *resty.Response) error {
			logger := options.logger.Debug().Ctx(response.Request.Context()).
				Int("status_code", response.StatusCode()).
				Any("headers", betterPrintHeaders(response.Request.Header))
			if cfg.LogAllResponse {
				logger.Str("body", response.String())
			} else {
				ct := response.Header().Get("content-type")
				fileSize := response.Size()
				if fileSize == 0 {
					fileSize = response.RawResponse.ContentLength
				}
				if fileSize > cfg.MaxRequestBodySize {
					logger.Str("body", "***** RESPONSE TOO LARGE (size - "+fmt.Sprintf("%d", fileSize)+") *****")
				} else if resty.IsJSONType(ct) || resty.IsXMLType(ct) {
					logger.Str("body", response.String())
				}
			}
			logger.Msg("httpclient response")
			return nil
		}).OnError(func(req *resty.Request, err error) {
			options.logger.Error().Ctx(req.Context()).
				Err(err).
				Msg("http request error")
		})
	}
	if !cfg.DisableRetry {
		client = client.SetRetryCount(utils.DefaultValue(cfg.RetryCount, 3)).
			SetRetryMaxWaitTime(10 * time.Second).
			SetRetryWaitTime(500 * time.Millisecond)
	}
	return client
}

// NewRequest creates a new request with the given context and jwt
func NewRequest(ctx context.Context, client *resty.Client, jwt string) *resty.Request {
	req := client.R().SetContext(ctx)
	if jwt != "" {
		req = req.SetAuthScheme("Bearer").SetAuthToken(jwt)
	}
	return req
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

// WrapClient wraps the resty client to add request id to the request
func WrapClient(client *resty.Client) *http.Client {
	return &http.Client{
		Transport: &wrapperTransport{client: client},
	}
}
