package infra

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"nhooyr.io/websocket"

	"aether/internal/realtime/domain"
)

func noopSubscribeOrg(ctx context.Context, orgID uuid.UUID, handler func(domain.Event)) (func(), error) {
	return func() {}, nil
}

func noopReplay(ctx context.Context, orgID uuid.UUID, afterSeq int64, limit int) ([]domain.Event, error) {
	return nil, nil
}

func noopAuthorize(ctx context.Context, scope string, orgID uuid.UUID) error {
	return nil
}

func newTestHub() *Hub {
	return NewHub(HubOptions{
		SubscribeOrg: noopSubscribeOrg,
		Replay:       noopReplay,
		Authorize:    noopAuthorize,
	})
}

func addClient(h *Hub, orgID uuid.UUID, scopes ...string) *Client {
	c := &Client{
		hub:    h,
		conn:   &websocket.Conn{},
		orgID:  orgID,
		send:   make(chan []byte, defaultMaxSend),
		scopes: map[string]bool{},
	}
	for _, s := range scopes {
		c.scopes[s] = true
	}
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
	return c
}

func readEvent(t *testing.T, c *Client, timeout time.Duration) (domain.Event, bool) {
	t.Helper()
	select {
	case data := <-c.send:
		var out wsOutbound
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if out.Op != "event" || out.Ev == nil {
			return domain.Event{}, false
		}
		return *out.Ev, true
	case <-time.After(timeout):
		return domain.Event{}, false
	}
}

func TestHubFanoutScoped(t *testing.T) {
	h := newTestHub()
	org := uuid.New()
	appA := uuid.New()
	appB := uuid.New()

	all := addClient(h, org, "org")
	scopedA := addClient(h, org, "app:"+appA.String())
	scopedB := addClient(h, org, "app:"+appB.String())

	ev := domain.Event{
		ID: "e1", Type: "deploy.started", AppID: appA.String(),
		ResourceType: "deployment", ResourceID: uuid.New().String(), Seq: 1,
	}
	h.fanout(org, ev)

	if got, ok := readEvent(t, all, 500*time.Millisecond); !ok || got.ID != "e1" {
		t.Fatalf("cliente org não recebeu: %+v", got)
	}
	if got, ok := readEvent(t, scopedA, 500*time.Millisecond); !ok || got.ID != "e1" {
		t.Fatalf("cliente app A não recebeu: %+v", got)
	}
	select {
	case data := <-scopedB.send:
		t.Fatalf("cliente app B recebeu evento errado: %s", string(data))
	case <-time.After(150 * time.Millisecond):
	}

	other := uuid.New()
	h.fanout(other, ev)
	if _, ok := readEvent(t, all, 250*time.Millisecond); ok {
		t.Fatalf("evento de outra org vazou para cliente")
	}
}

func TestClientWants(t *testing.T) {
	appA := uuid.New()
	evApp := domain.Event{AppID: appA.String(), ResourceType: "deployment", ResourceID: "dep1"}
	evRes := domain.Event{ResourceType: "database", ResourceID: "db1"}

	cases := []struct {
		name   string
		scopes []string
		ev     domain.Event
		want   bool
	}{
		{"org pega tudo", []string{"org"}, evApp, true},
		{"app coincide", []string{"app:" + appA.String()}, evApp, true},
		{"app diferente", []string{"app:zzz"}, evApp, false},
		{"recurso coincide", []string{"database:db1"}, evRes, true},
		{"recurso diferente", []string{"database:db9"}, evRes, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Client{scopes: map[string]bool{}}
			for _, s := range tc.scopes {
				c.scopes[s] = true
			}
			if got := c.wants(tc.ev); got != tc.want {
				t.Fatalf("wants(%+v) = %v, esperado %v", tc.ev, got, tc.want)
			}
		})
	}
}

func TestEnqueueBackpressure(t *testing.T) {
	c := &Client{send: make(chan []byte, 4), scopes: map[string]bool{}}
	for i := 0; i < 4; i++ {
		c.enqueue(domain.Event{ID: "x", Type: "t"}, false)
	}
	c.enqueue(domain.Event{ID: "drop", Type: "t"}, false)
	if c.dropped != 1 {
		t.Fatalf("dropped esperado 1, got %d", c.dropped)
	}
	// drena a fila para liberar espaço
	for i := 0; i < 4; i++ {
		<-c.send
	}
	c.enqueue(domain.Event{ID: "y", Type: "t"}, false)
	// o primeiro frame deve ser o "dropped"
	select {
	case data := <-c.send:
		var out wsOutbound
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if out.Op != "dropped" || out.N != 1 {
			t.Fatalf("esperado frame dropped, got %+v", out)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("frame dropped não enviado")
	}
}