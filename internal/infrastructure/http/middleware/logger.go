package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/LaFabric-LinkTIC/FlowEngine/pkg/logger"
)

// Logger middleware logs HTTP requests.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		statusCode := c.Writer.Status()
		method := c.Request.Method
		clientIP := c.ClientIP()
		requestID := c.GetString("request_id")

		if query != "" {
			path = path + "?" + query
		}

		// Structured logging
		logAttrs := []any{
			"status", statusCode,
			"method", method,
			"path", path,
			"ip", clientIP,
			"latency", latency.String(),
			"latency_ns", latency.Nanoseconds(),
			"request_id", requestID,
		}

		if len(c.Errors) > 0 {
			logAttrs = append(logAttrs, "errors", c.Errors.String())
			logger.Error("HTTP Request Failed", logAttrs...)
		} else {
			if statusCode >= 500 {
				logger.Error("HTTP Request Error", logAttrs...)
			} else if statusCode >= 400 {
				logger.Warn("HTTP Request Warning", logAttrs...)
			} else {
				logger.Info("HTTP Request", logAttrs...)
			}
		}
	}
}
