package infra

import (
	"context"
	"encoding/json"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"

	"aether/internal/realtime/domain"
)

type wsOutbound struct {
	Op      string          `json:"op"`
	Subs    []string        `json:"subs,omitempty"`
	Code    string          `json:"code,omitempty"`
	Message string          `json:"message,omitempty"`
	Ev      *domain.Event   `json:"ev,omitempty"`
	N       int64           `json:"n,omitempty"`
	Replay  bool            `json:"replay,omitempty"`
}

func marshalEvent(ev domain.Event, replay bool) ([]byte, error) {
	return json.Marshal(wsOutbound{Op: "event", Ev: &ev, Replay: replay})
}

func droppedFrame(n int) []byte {
	data, _ := json.Marshal(wsOutbound{Op: "dropped", N: int64(n)})
	return data
}

func marshalReply(op string, message string, ev *domain.Event, n int64) ([]byte, error) {
	return json.Marshal(wsOutbound{Op: op, Message: message, Ev: ev, N: n})
}

func marshalSubs(op string, subs []string) ([]byte, error) {
	return json.Marshal(wsOutbound{Op: op, Subs: subs})
}

func marshalError(code, message string) ([]byte, error) {
	return json.Marshal(wsOutbound{Op: "error", Code: code, Message: message})
}

func readJSON(ctx context.Context, conn *websocket.Conn, v interface{}) error {
	return wsjson.Read(ctx, conn, v)
}

func writeRaw(ctx context.Context, conn *websocket.Conn, data []byte) error {
	return conn.Write(ctx, websocket.MessageText, data)
}