package redis

import (
	"sync/atomic"

	"github.com/evanofslack/analogdb/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type cacheStats struct {
	hits   uint64
	misses uint64
	errors uint64
}

func newCacheStats() *cacheStats {
	stats := &cacheStats{
		hits:   0,
		misses: 0,
		errors: 0,
	}
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

func (stats *cacheStats) incErrors() {
	atomic.AddUint64(&stats.errors, 1)	
}

func (stats *cacheStats) getErrors() uint64 {
	return atomic.LoadUint64(&stats.errors)
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
		cacheErrors: prometheus.NewDesc(fqNameErrors, "Number of cache errors", variableLabels, nil),
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
		errors := float64(cache.stats.getErrors())
		instance := cache.instance

		ch <- prometheus.MustNewConstMetric(collector.cacheHits, prometheus.CounterValue, hits, instance)
		ch <- prometheus.MustNewConstMetric(collector.cacheMisses, prometheus.CounterValue, misses, instance)
		ch <- prometheus.MustNewConstMetric(collector.cacheErrors, prometheus.CounterValue, errors, instance)
	}
}
