package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"aether/internal/domain"
	"aether/internal/druntime/pubsub"
	"aether/internal/security"
)

func (s *Server) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	orgID := claimsFrom(r).OrgID
	before := r.URL.Query().Get("before")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	notifs, err := s.core.Store.ListNotifications(orgID, before, limit)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, notifs)
}

func (s *Server) handleUnreadCount(w http.ResponseWriter, r *http.Request) {
	count, err := s.core.Store.UnreadNotificationCount(claimsFrom(r).OrgID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]int{"count": count})
}

func (s *Server) handleMarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	if err := s.core.Store.MarkNotificationRead(claimsFrom(r).OrgID, r.PathValue("id")); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "read"})
}

func (s *Server) handleMarkAllRead(w http.ResponseWriter, r *http.Request) {
	if err := s.core.Store.MarkAllNotificationsRead(claimsFrom(r).OrgID); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "read"})
}

func (s *Server) handleEventStream(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		token = r.Header.Get("Authorization")
		token = trimBearer(token)
	}
	if token == "" {
		if c, err := r.Cookie("aether_token"); err == nil {
			token = c.Value
		}
	}
	claims, err := s.core.VerifyToken(token)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming não suportado", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ctx := r.Context()
	activeOrg := r.URL.Query().Get("org")
	if activeOrg == "" {
		activeOrg = r.Header.Get("X-Aether-Org")
	}
	if activeOrg == "" {
		activeOrg = claims.OrgID
	}
	msgs := make(chan []byte, 256)
	sub, err := s.core.RT.PubSub.Subscribe(ctx, "notify:org:"+activeOrg, func(_ context.Context, m pubsub.Message) {
		select {
		case msgs <- m.Data:
		default:
		}
	}, pubsub.WithBuffer(256))
	if err != nil {
		http.Error(w, "stream indisponível", http.StatusInternalServerError)
		return
	}
	defer sub.Unsubscribe()

	notifs, _ := s.core.Store.ListNotifications(activeOrg, "", 20)
	for _, n := range notifs {
		fmt.Fprintf(w, "event: history\ndata: %s\n\n", notifJSON(n))
	}
	flusher.Flush()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-msgs:
			var n domain.Notification
			if err := json.Unmarshal(data, &n); err != nil {
				continue
			}
			fmt.Fprintf(w, "event: notification\ndata: %s\n\n", notifJSON(n))
			flusher.Flush()
		case <-heartbeat.C:
			fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func trimBearer(token string) string {
	if len(token) > 7 && token[:7] == "Bearer " {
		return token[7:]
	}
	return token
}

func notifJSON(n domain.Notification) string {
	var payload map[string]any
	_ = json.Unmarshal([]byte(n.Payload), &payload)
	if payload == nil {
		payload = map[string]any{}
	}
	out := map[string]any{
		"id":      n.ID,
		"type":    n.Type,
		"message": n.Message,
		"read":    n.Read,
		"ts":      n.CreatedAt.Format(time.RFC3339),
		"payload": payload,
	}
	b, _ := json.Marshal(out)
	return string(b)
}

var _ = security.Claims{}
