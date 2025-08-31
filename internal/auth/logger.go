package auth

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger is a middleware that logs incoming requests
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		dur := time.Since(start)
		status := c.Writer.Status()
		ctx := context.Background()
		_ = ctx
		println(time.Now().Format(time.RFC3339), c.Request.Method, c.Request.URL.Path, "status=", status, "duration=", dur.String())
	}
}
