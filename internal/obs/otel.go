package obs

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type OtelMetrics struct {
	meter metric.Meter
	mp    *sdkmetric.MeterProvider

	cacheHits   metric.Int64Counter
	cacheMisses metric.Int64Counter
	cacheErrors metric.Int64Counter
	subscribers metric.Int64Gauge
	queueDepth  metric.Int64Gauge
	events      metric.Int64Counter
}

func NewOtelMetrics(ctx context.Context, endpoint string) (*OtelMetrics, error) {
	exp, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithEndpointURL(endpoint))
	if err != nil {
		return nil, err
	}
	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceName("aether")))
	if err != nil {
		return nil, err
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(15*time.Second))),
		sdkmetric.WithResource(res),
	)
	meter := mp.Meter("aether.runtime")
	m := &OtelMetrics{meter: meter, mp: mp}
	if m.cacheHits, err = meter.Int64Counter("aether.cache.hits"); err != nil {
		return nil, err
	}
	if m.cacheMisses, err = meter.Int64Counter("aether.cache.misses"); err != nil {
		return nil, err
	}
	if m.cacheErrors, err = meter.Int64Counter("aether.cache.errors"); err != nil {
		return nil, err
	}
	if m.subscribers, err = meter.Int64Gauge("aether.pubsub.subscribers"); err != nil {
		return nil, err
	}
	if m.queueDepth, err = meter.Int64Gauge("aether.queue.depth"); err != nil {
		return nil, err
	}
	if m.events, err = meter.Int64Counter("aether.events.published"); err != nil {
		return nil, err
	}
	otel.SetMeterProvider(mp)
	return m, nil
}

func (m *OtelMetrics) Record(ctx context.Context, hits, misses, errors int64, subscribers map[string]int, queueDepth int64, events int64) {
	m.cacheHits.Add(ctx, hits)
	m.cacheMisses.Add(ctx, misses)
	m.cacheErrors.Add(ctx, errors)
	totalSubs := int64(0)
	for _, n := range subscribers {
		totalSubs += int64(n)
	}
	m.subscribers.Record(ctx, totalSubs)
	m.queueDepth.Record(ctx, queueDepth)
	m.events.Add(ctx, events)
}

func (m *OtelMetrics) Shutdown(ctx context.Context) {
	if err := m.mp.Shutdown(ctx); err != nil {
		log.Printf("[otel] shutdown: %v", err)
	}
}
