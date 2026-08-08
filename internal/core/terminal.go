package core

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	"aether/internal/druntime/pubsub"
	"aether/internal/runtime"
)

const (
	terminalHostTTL = 20 * time.Minute
	terminalIdleTTL = 30 * time.Minute
)

type terminalHost struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func (c *Core) TerminalChannels(appID string) (in, out, ctl string) {
	return "terminal:" + appID + ":in", "terminal:" + appID + ":out", "terminal:" + appID + ":ctl"
}

func (c *Core) StartTerminalHost(appID, containerID, shell string) {
	c.termMu.Lock()
	if _, ok := c.terminals[appID]; ok {
		c.termMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.terminals[appID] = &terminalHost{cancel: cancel, done: make(chan struct{})}
	c.termMu.Unlock()

	done := c.terminals[appID].done
	go func() {
		defer close(done)
		c.runTerminalHost(ctx, appID, containerID, shell)
		c.termMu.Lock()
		if h, ok := c.terminals[appID]; ok {
			h.cancel()
			delete(c.terminals, appID)
		}
		c.termMu.Unlock()
	}()
}

func (c *Core) runTerminalHost(ctx context.Context, appID, containerID, shell string) {
	lockCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	lock, ok, err := c.RT.Locks.Acquire(lockCtx, "lock:terminal:"+appID, terminalHostTTL)
	cancel()
	if err != nil || !ok {
		return
	}
	defer func() {
		rctx, rcancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer rcancel()
		_ = c.RT.Locks.Release(rctx, lock)
	}()

	stream, err := c.Driver.ExecStream(ctx, containerID, runtime.ExecRequest{Command: []string{shell}, TTY: true})
	if err != nil {
		return
	}
	defer stream.Close()
	_ = stream.Resize(120, 30)

	var idle atomic.Int64
	idle.Store(time.Now().Unix())
	touch := func() { idle.Store(time.Now().Unix()) }

	renewCtx, renewStop := context.WithCancel(ctx)
	defer renewStop()
	go func() {
		t := time.NewTicker(terminalHostTTL / 3)
		defer t.Stop()
		for {
			select {
			case <-renewCtx.Done():
				return
			case <-t.C:
				rctx, rcancel := context.WithTimeout(context.Background(), 5*time.Second)
				_ = c.RT.Locks.Renew(rctx, lock, terminalHostTTL)
				rcancel()
			}
		}
	}()
	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for {
			select {
			case <-renewCtx.Done():
				return
			case <-t.C:
				if time.Since(time.Unix(idle.Load(), 0)) > terminalIdleTTL {
					stream.Close()
					return
				}
			}
		}
	}()

	in, out, ctl := c.TerminalChannels(appID)
	stopIn, _ := c.RT.PubSub.Subscribe(ctx, in, func(_ context.Context, m pubsub.Message) {
		touch()
		_, _ = stream.Write(m.Data)
	}, pubsub.WithBuffer(256))
	defer stopIn.Unsubscribe()
	stopCtl, _ := c.RT.PubSub.Subscribe(ctx, ctl, func(_ context.Context, m pubsub.Message) {
		var r struct {
			Type string `json:"type"`
			Cols uint16 `json:"cols"`
			Rows uint16 `json:"rows"`
		}
		if json.Unmarshal(m.Data, &r) == nil && r.Type == "resize" && r.Cols > 0 && r.Rows > 0 {
			_ = stream.Resize(r.Cols, r.Rows)
		}
	})
	defer stopCtl.Unsubscribe()

	buf := make([]byte, 4096)
	for {
		n, rerr := stream.Stdout().Read(buf)
		if n > 0 {
			touch()
			pctx, pcancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = c.RT.PubSub.Publish(pctx, out, buf[:n])
			pcancel()
		}
		if rerr != nil {
			return
		}
	}
}

func (c *Core) StopTerminalHosts() {
	c.termMu.Lock()
	hosts := make([]*terminalHost, 0, len(c.terminals))
	for _, h := range c.terminals {
		hosts = append(hosts, h)
	}
	c.termMu.Unlock()
	for _, h := range hosts {
		h.cancel()
	}
	for _, h := range hosts {
		select {
		case <-h.done:
		case <-time.After(3 * time.Second):
		}
	}
}
