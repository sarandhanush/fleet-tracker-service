package handlers

import (
	"context"
	"net/http"
	"time"

	"fleet-tracker-service/internal/auth"
	"fleet-tracker-service/internal/service"

	"github.com/gin-gonic/gin"
)

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func LoginHandler(a *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r loginReq
		if err := c.BindJSON(&r); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		if r.Username != "demo-user" || r.Password != "demo-pass" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		tok, err := a.GenerateToken("demo-user", 24*time.Hour)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate token"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": tok})
	}
}

func IngestHandler(svc *service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload service.IngestPayload
		if err := c.BindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}
		if err := svc.Ingest(c.Request.Context(), payload); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func StatusHandler(svc *service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		vid := c.Query("vehicle_id")
		if vid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "vehicle_id required"})
			return
		}
		st, err := svc.GetStatus(c.Request.Context(), vid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, st)
	}
}

func TripsHandler(svc *service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		vid := c.Query("vehicle_id")
		if vid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "vehicle_id required"})
			return
		}
		trips, err := svc.GetTripsLast24h(c.Request.Context(), vid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, trips)
	}
}

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
