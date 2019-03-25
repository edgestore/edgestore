package server

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

// NotFoundHandler is a helper function that calls Server.Abort.
func NotFoundHandler(c *gin.Context) {
	Abort(c, http.StatusNotFound, http.StatusText(http.StatusNotFound))
}

// LoggerHandler returns a gin.HandlerFunc (middleware) that logs requests using logrus.
//
// Requests with errors are logged using logrus.Error().
// Requests without errors are logged using logrus.Info().
//
// It receives:
//   1. A time package format string (e.g. time.RFC3339).
//   2. A boolean stating whether to use UTC time zone or local.
func LoggerHandler(logger logrus.FieldLogger, timeFormat string, utc bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		// some evil middlewares modify this values
		path := c.Request.URL.Path
		c.Next()

		end := time.Now()
		latency := end.Sub(start)
		if utc {
			end = end.UTC()
		}

		entry := logger.WithFields(logrus.Fields{
			"status":       c.Writer.Status(),
			"method":       c.Request.Method,
			"uri":          c.Request.RequestURI,
			"path":         path,
			"content_type": c.ContentType(),
			"remote-addr":  c.ClientIP(),
			"user-agent":   c.Request.UserAgent(),
			"x-request-id": c.GetHeader("X-Request-Id"),
			"latency":      latency,
			"time":         end.Format(timeFormat),
		})

		if len(c.Errors) > 0 {
			// Append error field if this is an erroneous request.
			entry.Error(c.Errors.String())
		} else {
			entry.Info()
		}
	}
}

// CORSHandler returns a gin.HandlerFunc (middleware) to enable CORSHandler support to all origins.
func CORSHandler() gin.HandlerFunc {
	config := cors.DefaultConfig()
	allowHeaders := []string{
		"Accept",
		"Authorization",
		"Content-Type",
		"Keep-Alive",
		"Origin",
		"User-Agent",
		"X-Requested-With",
	}
	config.AllowHeaders = append(config.AllowHeaders, allowHeaders...)
	config.AllowAllOrigins = true
	config.AllowCredentials = true
	return cors.New(config)
}

// RequestIDHandler injects a special header X-Request-Id to response headers
// that could be used to track incoming requests for monitoring/debugging
// purposes.
func RequestIDHandler() gin.HandlerFunc {

	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-Id")
		if reqID == "" {
			gen := uuid.Must(uuid.NewV4())
			reqID = gen.String()
		}

		c.Writer.Header().Set("X-Request-Id", reqID)
		c.Next()
	}
}

// NoCacheHandler is a middleware func for setting the Cache-Control to no-cache.
func NoCacheHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.Next()
	}
}
