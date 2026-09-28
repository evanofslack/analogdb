package redis

import (
	"sync/atomic"

	"github.com/evanofslack/analogdb/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/extra/redisprometheus/v9"
)

const (
	errorKindRedis  = "redis"
	errorKindDecode = "decode"
)

type cacheStats struct {
	hits         uint64
	misses       uint64
	redisErrors  uint64
	decodeErrors uint64
}

func newCacheStats() *cacheStats {
	stats := &cacheStats{}
	return stats
}

func (stats *cacheStats) incHits() {
	atomic.AddUint64(&stats.hits, 1)
}

func (stats *cacheStats) getHits() uint64 {
	return atomic.LoadUint64(&stats.hits)
}

func (stats *cacheStats) incMisses() {
	atomic.AddUint64(&stats.misses, 1)
}

func (stats *cacheStats) getMisses() uint64 {
	return atomic.LoadUint64(&stats.misses)
}

func (stats *cacheStats) incErrors(kind string) {
	if kind == errorKindDecode {
		atomic.AddUint64(&stats.decodeErrors, 1)
		return
	}
	atomic.AddUint64(&stats.redisErrors, 1)
}

func (stats *cacheStats) getErrors(kind string) uint64 {
	if kind == errorKindDecode {
		return atomic.LoadUint64(&stats.decodeErrors)
	}
	return atomic.LoadUint64(&stats.redisErrors)
}

type cacheCollector struct {
	caches      []*Cache
	cacheHits   *prometheus.Desc
	cacheMisses *prometheus.Desc
	cacheErrors *prometheus.Desc
}

func newCacheCollector() *cacheCollector {
	fqNameHits := prometheus.BuildFQName(metrics.AnalogdbNamespace, metrics.CacheSubsystem, "hits")
	fqNameMisses := prometheus.BuildFQName(metrics.AnalogdbNamespace, metrics.CacheSubsystem, "misses")
	fqNameErrors := prometheus.BuildFQName(metrics.AnalogdbNamespace, metrics.CacheSubsystem, "errors")
	variableLabels := []string{"instance"}

	return &cacheCollector{
		cacheHits:   prometheus.NewDesc(fqNameHits, "Number of cache hits", variableLabels, nil),
		cacheMisses: prometheus.NewDesc(fqNameMisses, "Number of cache misses", variableLabels, nil),
		cacheErrors: prometheus.NewDesc(fqNameErrors, "Number of cache errors", []string{"instance", "kind"}, nil),
	}
}

func (collector *cacheCollector) registerCache(cache *Cache) {
	collector.caches = append(collector.caches, cache)
}

func (collector *cacheCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.cacheHits
	ch <- collector.cacheMisses
	ch <- collector.cacheErrors
}

func (collector *cacheCollector) Collect(ch chan<- prometheus.Metric) {
	// get stats for each cache we have registered
	for _, cache := range collector.caches {

		hits := float64(cache.stats.getHits())
		misses := float64(cache.stats.getMisses())
		instance := cache.instance

		ch <- prometheus.MustNewConstMetric(collector.cacheHits, prometheus.CounterValue, hits, instance)
		ch <- prometheus.MustNewConstMetric(collector.cacheMisses, prometheus.CounterValue, misses, instance)
		for _, kind := range []string{errorKindRedis, errorKindDecode} {
			errors := float64(cache.stats.getErrors(kind))
			ch <- prometheus.MustNewConstMetric(collector.cacheErrors, prometheus.CounterValue, errors, instance, kind)
		}
	}
}

type genStats struct {
	invalidations *prometheus.CounterVec
	incrErrors    prometheus.Counter
}

func newGenStats() *genStats {
	return &genStats{
		invalidations: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.CacheSubsystem,
			Name:      "invalidations_total",
			Help:      "Number of cache invalidations after writes",
		}, []string{"entity"}),
		incrErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.CacheSubsystem,
			Name:      "gen_incr_errors_total",
			Help:      "Number of failed cache generation increments",
		}),
	}
}

func (stats *genStats) register(registerer prometheus.Registerer) {
	registerer.MustRegister(stats.invalidations, stats.incrErrors)
}

func newRedisCollector(client redisprometheus.StatGetter) *redisprometheus.Collector {
	collector := redisprometheus.NewCollector(metrics.AnalogdbNamespace, metrics.RedisSubsystem, client)
	return collector
}
