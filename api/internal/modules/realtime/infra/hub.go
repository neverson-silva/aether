package infra

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"nhooyr.io/websocket"

	"aether/internal/modules/realtime/domain"
)

const (
	defaultMaxSend       = 256
	defaultHeartbeat     = 15 * time.Second
	defaultReadTimeout   = 45 * time.Second
	defaultMaxSubs       = 32
	defaultReplayLimit   = 100
	defaultMaxConnPerOrg = 64
)

type HubOptions struct {
	SubscribeOrg func(ctx context.Context, orgID uuid.UUID, handler func(domain.Event)) (func(), error)
	Replay       func(ctx context.Context, orgID uuid.UUID, afterSeq int64, limit int) ([]domain.Event, error)
	Authorize    func(ctx context.Context, scope string, orgID uuid.UUID) error
}

type Hub struct {
	opts    HubOptions
	mu      sync.Mutex
	clients map[*Client]struct{}
	orgs    map[string]*orgState
}

type orgState struct {
	clients int
	unsub   func()
}

func NewHub(opts HubOptions) *Hub {
	return &Hub{opts: opts, clients: map[*Client]struct{}{}, orgs: map[string]*orgState{}}
}

func (h *Hub) Add(conn *websocket.Conn, orgID, userID uuid.UUID) *Client {
	c := &Client{
		hub:    h,
		conn:   conn,
		orgID:  orgID,
		userID: userID,
		send:   make(chan []byte, defaultMaxSend),
		scopes: map[string]bool{},
	}
	h.mu.Lock()
	if len(h.clients) >= defaultMaxConnPerOrg {
		h.mu.Unlock()
		return nil
	}
	h.clients[c] = struct{}{}
	h.ensureOrg(orgID)
	h.mu.Unlock()
	return c
}

func (h *Hub) ensureOrg(orgID uuid.UUID) {
	key := orgID.String()
	st, ok := h.orgs[key]
	if !ok {
		st = &orgState{}
		h.orgs[key] = st
		unsub, err := h.opts.SubscribeOrg(context.Background(), orgID, func(ev domain.Event) {
			h.fanout(orgID, ev)
		})
		if err != nil {
			return
		}
		st.unsub = unsub
	}
	st.clients++
}

func (h *Hub) remove(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c)
	key := c.orgID.String()
	if st, ok := h.orgs[key]; ok {
		st.clients--
		if st.clients <= 0 {
			if st.unsub != nil {
				st.unsub()
			}
			delete(h.orgs, key)
		}
	}
}

func (h *Hub) fanout(orgID uuid.UUID, ev domain.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		if c.orgID == orgID && c.wants(ev) {
			c.enqueue(ev, false)
		}
	}
}

func (c *Client) wants(ev domain.Event) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.scopes["org"] {
		return true
	}
	if ev.AppID != "" && c.scopes["app:"+ev.AppID] {
		return true
	}
	if ev.ResourceType != "" && ev.ResourceID != "" && c.scopes[ev.ResourceType+":"+ev.ResourceID] {
		return true
	}
	return false
}

func (c *Client) enqueue(ev domain.Event, replay bool) {
	data, err := marshalEvent(ev, replay)
	if err != nil {
		return
	}
	c.mu.Lock()
	drops := c.dropped
	c.dropped = 0
	c.mu.Unlock()
	if drops > 0 {
		select {
		case c.send <- droppedFrame(drops):
		default:
			c.mu.Lock()
			c.dropped += drops
			c.mu.Unlock()
			return
		}
	}
	select {
	case c.send <- data:
	default:
		c.mu.Lock()
		c.dropped++
		c.mu.Unlock()
	}
}

type wsInbound struct {
	Op   string   `json:"op"`
	Subs []string `json:"subs"`
	Seq  int64    `json:"seq"`
}

func (h *Hub) Run(c *Client, ctx context.Context) {
	defer func() {
		h.remove(c)
		_ = c.conn.Close(websocket.StatusNormalClosure, "closed")
	}()
	go c.writer(ctx)
	for {
		readCtx, cancel := context.WithTimeout(ctx, defaultReadTimeout)
		var msg wsInbound
		err := readJSON(readCtx, c.conn, &msg)
		cancel()
		if err != nil {
			return
		}
		switch msg.Op {
		case "subscribe":
			h.handleSubscribe(c, ctx, msg)
		case "unsubscribe":
			h.handleUnsubscribe(c, msg)
		case "ping":
			c.writeNow("pong", nil)
		}
	}
}

func (h *Hub) handleSubscribe(c *Client, ctx context.Context, msg wsInbound) {
	if len(msg.Subs) == 0 {
		c.writeError("bad_request", "empty subs")
		return
	}
	accepted := make([]string, 0, len(msg.Subs))
	for _, scope := range msg.Subs {
		if len(c.scopes) >= defaultMaxSubs {
			c.writeError("too_many_subs", "subscription limit reached")
			break
		}
		if err := h.opts.Authorize(ctx, scope, c.orgID); err != nil {
			c.writeError("forbidden", "scope not allowed: "+scope)
			continue
		}
		c.mu.Lock()
		c.scopes[scope] = true
		c.mu.Unlock()
		accepted = append(accepted, scope)
	}
	if msg.Seq >= 0 && len(accepted) > 0 {
		if replay, err := h.opts.Replay(ctx, c.orgID, msg.Seq, defaultReplayLimit); err == nil {
			for _, ev := range replay {
				if c.wants(ev) {
					c.enqueue(ev, true)
				}
			}
		}
	}
	if len(accepted) > 0 {
		c.writeSubscribed(accepted)
	}
}

func (h *Hub) handleUnsubscribe(c *Client, msg wsInbound) {
	removed := make([]string, 0, len(msg.Subs))
	c.mu.Lock()
	for _, scope := range msg.Subs {
		if c.scopes[scope] {
			delete(c.scopes, scope)
			removed = append(removed, scope)
		}
	}
	c.mu.Unlock()
	if len(removed) > 0 {
		c.writeUnsubscribed(removed)
	}
}

type Client struct {
	hub     *Hub
	conn    *websocket.Conn
	orgID   uuid.UUID
	userID  uuid.UUID
	send    chan []byte
	dropped int
	mu      sync.Mutex
	scopes  map[string]bool
}

func (c *Client) writer(ctx context.Context) {
	ticker := time.NewTicker(defaultHeartbeat)
	defer ticker.Stop()
	for {
		select {
		case data := <-c.send:
			if err := writeRaw(ctx, c.conn, data); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.Ping(ctx); err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *Client) writeNow(op string, ev *domain.Event) {
	data, _ := marshalReply(op, "", ev, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = writeRaw(ctx, c.conn, data)
}

func (c *Client) writeSubscribed(subs []string) {
	data, _ := marshalSubs("subscribed", subs)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = writeRaw(ctx, c.conn, data)
}

func (c *Client) writeUnsubscribed(subs []string) {
	data, _ := marshalSubs("unsubscribed", subs)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = writeRaw(ctx, c.conn, data)
}

func (c *Client) writeError(code, message string) {
	data, _ := marshalError(code, message)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = writeRaw(ctx, c.conn, data)
}
