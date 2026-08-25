// Package router wires the HTTP layer together: it configures middleware
// (logging, recovery, CORS) and registers the API routes served by the
// handler package.
package router

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/handler"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/store"
)

// New builds a fully configured Gin engine backed by the given service.
func New(svc *store.Service) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// Permissive CORS suitable for local development. Any loopback origin
	// (any port) is allowed so Vite picking a different free port (5174,
	// ...) doesn't break POSTs — browsers send Origin on POST even when
	// same-origin through the dev proxy. Tighten for production.
	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return strings.HasPrefix(origin, "http://localhost:") ||
				strings.HasPrefix(origin, "http://127.0.0.1:")
		},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	h := handler.New(svc)

	// Liveness/readiness probe.
	r.GET("/healthz", h.Health)

	api := r.Group("/api/v1")
	{
		api.GET("/cluster/status", h.ClusterStatus)
		api.POST("/cluster/exec", h.ExecCommand)
		api.GET("/files", h.ReadFile)
		api.POST("/files", h.WriteFile)
		api.GET("/files/complete", h.Complete)
		api.GET("/questions", h.ListQuestions)
		api.GET("/questions/:id", h.GetQuestion)

		api.POST("/sessions", h.StartSession)
		api.GET("/sessions/:id", h.GetSession)
		api.POST("/sessions/:id/answers", h.SubmitAnswer)
		api.POST("/sessions/:id/end", h.EndSession)
		api.POST("/sessions/:id/cleanup", h.CleanupSession)
	}

	return r
}
