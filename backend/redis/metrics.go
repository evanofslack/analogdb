package redis

import (
	"github.com/evanofslack/analogdb/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type cacheStats struct {
	hits   uint64
	misses uint64
	errors uint64
}

type cacheCollector struct {
	cache       *Cache
	cacheHits   *prometheus.Desc
	cacheMisses *prometheus.Desc
	cacheErrors *prometheus.Desc
}

func newCacheCollector(cache *Cache, instance string) *cacheCollector {

	fqNameHits := prometheus.BuildFQName(metrics.AnalogdbNamespace, metrics.CacheSubsystem, "cache_hits")
	fqNameMisses := prometheus.BuildFQName(metrics.AnalogdbNamespace, metrics.CacheSubsystem, "cache_hits")
	fqNameErrors := prometheus.BuildFQName(metrics.AnalogdbNamespace, metrics.CacheSubsystem, "cache_errors")
	constLabels := prometheus.Labels{"instance": instance}

	return &cacheCollector{
		cache:       cache,
		cacheHits:   prometheus.NewDesc(fqNameHits, "Number of cache hits", nil, constLabels),
		cacheMisses: prometheus.NewDesc(fqNameMisses, "Number of cache misses", nil, constLabels),
		cacheErrors: prometheus.NewDesc(fqNameErrors, "Number of cache errors", nil, constLabels),
	}
}

func (collector *cacheCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.cacheHits
	ch <- collector.cacheMisses
	ch <- collector.cacheErrors
}

func (collector *cacheCollector) Collect(ch chan<- prometheus.Metric) {
	stats := collector.cache.stats()
	ch <- prometheus.MustNewConstMetric(collector.cacheHits, prometheus.CounterValue, float64(stats.hits))
	ch <- prometheus.MustNewConstMetric(collector.cacheMisses, prometheus.CounterValue, float64(stats.misses))
	ch <- prometheus.MustNewConstMetric(collector.cacheErrors, prometheus.CounterValue, float64(stats.errors))
}
