package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/ipfans/components/v2/ctxkeys"
	"github.com/ipfans/components/v2/lifecycle"
	"github.com/ipfans/components/v2/utils"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Config struct {
	Address    string `koanf:"address"`
	Production bool   `koanf:"production"`
}

func CORSMiddleware(config ...cors.Config) gin.HandlerFunc {
	if len(config) > 0 {
		return cors.New(config[0])
	}
	return cors.New(cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           24 * time.Hour,
	})
}

func LoggerMiddleware(getUid func(c *gin.Context) string, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		method := c.Request.Method
		path := c.Request.URL.Path
		ip := c.ClientIP()
		uid := getUid(c)
		request_id := c.GetString("X-Request-ID")
		if request_id == "" {
			request_id = utils.NewUUID()
		}

		ctx = logger.With().Str("uid", uid).Str("method", method).Str("path", path).Str("ip", ip).Str("request_id", request_id).Logger().WithContext(ctx)
		ctx = ctxkeys.SetRequestID(ctx, request_id)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
		c.Header("X-Request-ID", request_id)
	}
}

func New(lc lifecycle.Lifecycle, cfg Config, handlers ...gin.HandlerFunc) *gin.Engine {
	var srv *http.Server
	if cfg.Production {
		gin.SetMode(gin.ReleaseMode)
	}
	if len(handlers) > 0 {
		router := gin.New()
		router.Use(handlers...)

		srv = &http.Server{
			Addr:    cfg.Address,
			Handler: router,
		}
		lc.Append(lifecycle.Hook{
			OnStart: func(ctx context.Context) error {
				go func() {
					if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
						log.Fatal().Err(err).Msg("Start http server failed")
					}
				}()
				return nil
			},
			OnStop: func(_ context.Context) error {
				return srv.Close()
			},
		})
		return router
	}
	router := gin.Default()

	srv = &http.Server{
		Addr:    cfg.Address,
		Handler: router,
	}
	lc.Append(lifecycle.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Fatal().Err(err).Msg("Start http server failed")
				}
			}()
			return nil
		},
		OnStop: func(_ context.Context) error {
			return srv.Close()
		},
	})
	return router
}
