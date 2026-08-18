package infra

import (
	"context"
	"database/sql"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"aether/internal/monitoring/domain"
	gen "aether/internal/infrastructure/pg/gen"
)

type Store struct {
	q  gen.Querier
	db *sql.DB
}

func NewStore(pool *pgxpool.Pool) *Store {
	db := stdlib.OpenDBFromPool(pool)
	return &Store{q: gen.New(db), db: db}
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) InsertSample(ctx context.Context, p domain.HistoryPoint, ts time.Time) error {
	return s.q.InsertMonitoringSample(ctx, gen.InsertMonitoringSampleParams{
		Ts: ts, HostCpu: p.HostCPU, HostMem: p.HostMem,
		AetherCpu: p.AetherCPU, AetherMem: int64(p.AetherMem), AetherMemPct: p.AetherMemPct,
		UserCpu: p.UserCPU, UserMem: int64(p.UserMem), UserMemPct: p.UserMemPct,
		NetRx: p.NetRx, NetTx: p.NetTx,
	})
}

func (s *Store) InsertResourceSample(ctx context.Context, r domain.ResourcePoint, id, name, owner string, ts time.Time) error {
	return s.q.InsertMonitoringResourceSample(ctx, gen.InsertMonitoringResourceSampleParams{
		Ts: ts, ResourceID: id, Name: name, Owner: owner,
		Cpu: r.CPU, Mem: int64(r.Mem), NetRx: r.NetRx, NetTx: r.NetTx,
	})
}

func (s *Store) ListSamples(ctx context.Context, from, to time.Time) ([]domain.HistoryPoint, error) {
	rows, err := s.q.ListMonitoringSamples(ctx, gen.ListMonitoringSamplesParams{Ts: from, Ts_2: to})
	if err != nil {
		return nil, err
	}
	out := make([]domain.HistoryPoint, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.HistoryPoint{
			TS: r.Ts.Unix(), HostCPU: r.HostCpu, HostMem: r.HostMem,
			AetherCPU: r.AetherCpu, AetherMem: uint64(r.AetherMem), AetherMemPct: r.AetherMemPct,
			UserCPU: r.UserCpu, UserMem: uint64(r.UserMem), UserMemPct: r.UserMemPct,
			NetRx: r.NetRx, NetTx: r.NetTx,
		})
	}
	return out, nil
}

func (s *Store) ListResourceSamples(ctx context.Context, id string, from, to time.Time) ([]domain.ResourcePoint, error) {
	rows, err := s.q.ListMonitoringResourceSamples(ctx, gen.ListMonitoringResourceSamplesParams{
		ResourceID: id, Ts: from, Ts_2: to,
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.ResourcePoint, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.ResourcePoint{TS: r.Ts.Unix(), CPU: r.Cpu, Mem: uint64(r.Mem), NetRx: r.NetRx, NetTx: r.NetTx})
	}
	return out, nil
}

func (s *Store) Purge(ctx context.Context, before time.Time) error {
	if err := s.q.PurgeMonitoringSamples(ctx, before); err != nil {
		return err
	}
	return s.q.PurgeMonitoringResourceSamples(ctx, before)
}
