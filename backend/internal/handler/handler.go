package handler

// Package handler contains the HTTP layer: it binds the service to Gin
// routes, parses requests, and renders JSON responses.

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/store"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/store/dto"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/store/memory"
)

// Handler wraps the application service and exposes HTTP methods.
type Handler struct {
	svc *store.Service
}

// New creates a Handler with the given service.
func New(svc *store.Service) *Handler {
	return &Handler{svc: svc}
}

// Health reports service liveness.
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ListQuestions returns all questions (without hints/solutions).
func (h *Handler) ListQuestions(c *gin.Context) {
	qs, err := h.svc.ListQuestions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if qs == nil {
		qs = []dto.QuestionSummary{}
	}
	c.JSON(http.StatusOK, gin.H{"questions": qs})
}

// GetQuestion returns a single question by ID (full detail).
func (h *Handler) GetQuestion(c *gin.Context) {
	q, err := h.svc.GetQuestion(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "question not found"})
		return
	}
	c.JSON(http.StatusOK, q)
}

// StartSession creates a new simulation session and provisions its
// cluster environment.
func (h *Handler) StartSession(c *gin.Context) {
	var req dto.StartSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	sess, err := h.svc.StartSession(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, sess)
}

// GetSession returns a session by ID.
func (h *Handler) GetSession(c *gin.Context) {
	sess, err := h.svc.GetSession(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	c.JSON(http.StatusOK, sess)
}

// SubmitAnswer grades an answer for a question within a session.
func (h *Handler) SubmitAnswer(c *gin.Context) {
	var req dto.SubmitAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if req.QuestionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "questionId is required"})
		return
	}
	res, err := h.svc.SubmitAnswer(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, memory.ErrNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// EndSession finalizes a session and returns results.
func (h *Handler) EndSession(c *gin.Context) {
	res, err := h.svc.EndSession(c.Param("id"))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, memory.ErrNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// ClusterStatus reports connectivity to the underlying cluster.
func (h *Handler) ClusterStatus(c *gin.Context) {
	connected, detail := h.svc.ClusterStatus(c.Request.Context())
	status := http.StatusOK
	if !connected {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{"connected": connected, "detail": detail})
}

// ExecCommand runs a command from the exam terminal against the cluster.
func (h *Handler) ExecCommand(c *gin.Context) {
	var req struct {
		Command string `json:"command"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Command == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "command is required"})
		return
	}
	res := h.svc.Exec(c.Request.Context(), req.Command)
	c.JSON(http.StatusOK, res)
}

// ReadFile loads a file from the exam sandbox for the built-in editors.
func (h *Handler) ReadFile(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}
	content, err := h.svc.ReadFile(path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"path": path, "content": content})
}

// WriteFile stores a file edited with the built-in vi/nano emulation.
func (h *Handler) WriteFile(c *gin.Context) {
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}
	if err := h.svc.WriteFile(req.Path, req.Content); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "path": req.Path})
}

// Complete returns Tab-completion candidates for an exam terminal line.
func (h *Handler) Complete(c *gin.Context) {
	line := c.Query("line")
	matches := h.svc.CompleteLine(line)
	if matches == nil {
		matches = []string{}
	}
	c.JSON(http.StatusOK, gin.H{"matches": matches})
}

// CleanupSession resets the cluster state created for/by a session.
func (h *Handler) CleanupSession(c *gin.Context) {
	logs, err := h.svc.CleanupSession(c.Request.Context(), c.Param("id"))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, memory.ErrNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}
