package http

import (
	"errors"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"aether/internal/modules/traefikfs/application"
)

type Handler struct {
	filesystem *application.Filesystem
}

func New(filesystem *application.Filesystem) *Handler {
	return &Handler{filesystem: filesystem}
}

type fileRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (h *Handler) List(c *gin.Context) {
	entries, err := h.filesystem.List()
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": entries})
}

func (h *Handler) Status(c *gin.Context) {
	status, err := h.filesystem.Status(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h *Handler) Read(c *gin.Context) {
	file, err := h.filesystem.Read(c.Query("path"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, file)
}

func (h *Handler) Write(c *gin.Context) {
	var request fileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, application.ErrInvalidConfig)
		return
	}
	if err := h.filesystem.Write(request.Path, request.Content); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "saved", "path": request.Path})
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.filesystem.Delete(c.Query("path")); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) Restart(c *gin.Context) {
	if err := h.filesystem.Restart(c.Request.Context()); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "restarted"})
}

func writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, application.ErrInvalidPath), errors.Is(err, application.ErrInvalidConfig), errors.Is(err, application.ErrFileTooLarge):
		status = http.StatusBadRequest
	case errors.Is(err, application.ErrProtectedFile), errors.Is(err, application.ErrFileNotEditable):
		status = http.StatusForbidden
	case errors.Is(err, application.ErrTraefikNotRunning):
		status = http.StatusServiceUnavailable
	case errors.Is(err, os.ErrNotExist):
		status = http.StatusNotFound
	}
	c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
}
