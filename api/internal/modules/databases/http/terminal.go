package http

import (
	"encoding/json"
	"net/http"
	"os/exec"

	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"nhooyr.io/websocket"

	"aether/internal/modules/databases/domain"
)

func (h *Handler) DbTerminal(c *gin.Context) {
	dbID, err := uuid.Parse(c.Param("dbID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	containerID, err := h.databases.ContainerID(c.Request.Context(), dbID, orgID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active container"})
		return
	}
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusInternalError, "closed")

	shell := c.Query("shell")
	switch shell {
	case "bash", "zsh", "fish", "ash":
	default:
		shell = "sh"
	}
	cmd := exec.Command("podman", "exec", "-it", containerID, shell)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		conn.Close(websocket.StatusInternalError, "terminal unavailable")
		return
	}
	defer func() {
		_ = ptmx.Close()
		_ = cmd.Wait()
	}()

	ctx := c.Request.Context()
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, rerr := ptmx.Read(buf)
			if n > 0 {
				if werr := conn.Write(ctx, websocket.MessageBinary, buf[:n]); werr != nil {
					return
				}
			}
			if rerr != nil {
				return
			}
		}
	}()

	for {
		typ, data, rerr := conn.Read(ctx)
		if rerr != nil {
			return
		}
		if typ == websocket.MessageText {
			var ctrl struct {
				Type string `json:"type"`
				Cols uint16 `json:"cols"`
				Rows uint16 `json:"rows"`
			}
			if json.Unmarshal(data, &ctrl) == nil && ctrl.Type == "resize" && ctrl.Cols > 0 && ctrl.Rows > 0 {
				_ = pty.Setsize(ptmx, &pty.Winsize{Cols: ctrl.Cols, Rows: ctrl.Rows})
			}
			continue
		}
		_, _ = ptmx.Write(data)
	}
}
