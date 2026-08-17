package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	deploydomain "aether/internal/deployments/domain"
)

func (h *Handler) AppStart(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, deploydomain.ErrValidation)
		return
	}
	state, err := h.appOps.Start(c.Request.Context(), appID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "started", "state": state})
}

func (h *Handler) AppStop(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, deploydomain.ErrValidation)
		return
	}
	state, err := h.appOps.Stop(c.Request.Context(), appID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "stopped", "state": state})
}

func (h *Handler) AppRestart(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, deploydomain.ErrValidation)
		return
	}
	state, err := h.appOps.Restart(c.Request.Context(), appID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "restarted", "state": state})
}

func (h *Handler) AppRebuild(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, deploydomain.ErrValidation)
		return
	}
	dep, err := h.appOps.Rebuild(c.Request.Context(), appID, orgID(c), triggeredBy(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusAccepted, deploymentDTO(dep))
}

func (h *Handler) AppStates(c *gin.Context) {
	states, err := h.appOps.States(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, states)
}

func (h *Handler) AppStatesStream(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()

	states, err := h.appOps.States(c.Request.Context(), orgID(c))
	if err != nil {
		return
	}
	for appID, state := range states {
		writeStateEvent(c, appID, state)
	}
	heartbeat := time.NewTicker(5 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-heartbeat.C:
			states, err := h.appOps.States(c.Request.Context(), orgID(c))
			if err != nil {
				return
			}
			for appID, state := range states {
				writeStateEvent(c, appID, state)
			}
		}
	}
}

func (h *Handler) AppTimeline(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, deploydomain.ErrValidation)
		return
	}
	entries, err := h.appOps.Timeline(c.Request.Context(), appID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(entries))
	for _, entry := range entries {
		ts := entry.CreatedAt
		out = append(out, gin.H{
			"id": entry.ID, "aggregate_type": "deployment", "aggregate_id": entry.ID,
			"sequence": entry.Number, "type": "deploy." + entry.Status,
			"payload": gin.H{"trigger": entry.Trigger, "triggered_by": entry.TriggeredBy, "number": entry.Number},
			"ts":      ts,
		})
	}
	c.JSON(http.StatusOK, out)
}

func writeStateEvent(c *gin.Context, appID uuid.UUID, state string) {
	data, _ := json.Marshal(map[string]string{"app_id": appID.String(), "state": state})
	_, _ = fmt.Fprintf(c.Writer, "event: state\ndata: %s\n\n", data)
	c.Writer.Flush()
}
