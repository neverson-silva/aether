package core

import (
	"context"
	"fmt"
	"time"

	"aether/internal/domain"
)

type autopilotState struct {
	lastAction map[string]time.Time
}

func (c *Core) StartAutopilot(ctx context.Context) {
	if c.autopilot == nil {
		c.autopilot = &autopilotState{lastAction: map[string]time.Time{}}
	}
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		c.autopilotLoop(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.autopilotLoop(ctx)
			}
		}
	}()
}

func (c *Core) autopilotLoop(ctx context.Context) {
	policies, err := c.Store.ListPolicies()
	if err != nil {
		return
	}
	for _, p := range policies {
		if !p.Enabled {
			continue
		}
		c.evaluatePolicy(ctx, &p)
	}
}

func (c *Core) evaluatePolicy(ctx context.Context, p *domain.AppPolicy) {
	app, err := c.Store.GetApp(p.AppID)
	if err != nil {
		return
	}
	deploys, err := c.Store.ListDeployments(app.ID, 1)
	if err != nil || len(deploys) == 0 || deploys[0].ContainerID == "" {
		return
	}
	cid := deploys[0].ContainerID
	st, err := c.Driver.Stats(ctx, cid)
	if err != nil || st.MemBytes == 0 {
		return
	}
	if c.autopilot == nil {
		return
	}
	last, ok := c.autopilot.lastAction[p.AppID]
	if ok && time.Since(last) < time.Duration(p.CooldownMin)*time.Minute {
		return
	}
	memLimitMB := int64(st.MemLimit / (1 << 20))
	if memLimitMB == 0 {
		memLimitMB = app.Resources.MemMB
	}
	usagePct := float64(st.MemBytes) / float64(st.MemLimit) * 100
	if st.MemLimit == 0 {
		usagePct = 0
	}
	if usagePct > float64(p.ScaleUpPct) && memLimitMB < p.MemMaxMB {
		newMem := memLimitMB * 2
		if newMem > p.MemMaxMB {
			newMem = p.MemMaxMB
		}
		if err := c.Driver.UpdateResources(ctx, cid, newMem, fmt.Sprintf("%g", p.CPUMax)); err == nil {
			c.autopilot.lastAction[p.AppID] = time.Now()
			detail := fmt.Sprintf("mem %dMiB -> %dMiB (uso %d%%)", memLimitMB, newMem, int(usagePct))
			c.Store.AddAutopilotEvent(app.ID, "scale_up", detail)
			c.NotifyOrg(app.OrgID, "Autopilot scale-up: "+app.Name, detail)
		}
		return
	}
	if usagePct < float64(p.ScaleDownPct) && memLimitMB > p.MemMinMB {
		newMem := memLimitMB / 2
		if newMem < p.MemMinMB {
			newMem = p.MemMinMB
		}
		if err := c.Driver.UpdateResources(ctx, cid, newMem, fmt.Sprintf("%g", p.CPUMin)); err == nil {
			c.autopilot.lastAction[p.AppID] = time.Now()
			detail := fmt.Sprintf("mem %dMiB -> %dMiB (uso %d%%)", memLimitMB, newMem, int(usagePct))
			c.Store.AddAutopilotEvent(app.ID, "scale_down", detail)
			c.NotifyOrg(app.OrgID, "Autopilot scale-down: "+app.Name, detail)
		}
	}
}

func (c *Core) SavePolicy(p *domain.AppPolicy) error {
	p.UpdatedAt = time.Now().UTC()
	return c.Store.SavePolicy(p)
}

func (c *Core) PolicyEvents(appID string) ([]domain.AutopilotEvent, error) {
	return c.Store.ListAutopilotEvents(appID, 20)
}
