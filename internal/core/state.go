package core

import (
	"context"
	"time"
)

func (c *Core) appStateKey(appID string) string {
	return "app:state:" + appID
}

func (c *Core) PublishAppState(appID, state string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.RT.State.Set(ctx, c.appStateKey(appID), []byte(state), 0); err != nil {
		return
	}
}

func (c *Core) CurrentAppState(appID string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	v, ok, err := c.RT.State.Get(ctx, c.appStateKey(appID))
	if err != nil || !ok {
		return "", false
	}
	return string(v), true
}

func (c *Core) reconcileAppStates(ctx context.Context) {
	apps, err := c.Store.ListAllApps()
	if err != nil {
		return
	}
	for _, a := range apps {
		state := c.AppState(a.ID)
		c.PublishAppState(a.ID, state)
	}
}

func (c *Core) WatchAppStates(pattern string, h func(ctx context.Context, appID string, state string)) (func(), error) {
	ctx, cancel := context.WithCancel(context.Background())
	stop, err := c.RT.State.Changes(ctx, pattern, func(ctx context.Context, key string, value []byte) {
		if len(key) <= len("app:state:") {
			return
		}
		h(ctx, key[len("app:state:"):], string(value))
	})
	if err != nil {
		cancel()
		return nil, err
	}
	return func() {
		stop()
		cancel()
	}, nil
}
