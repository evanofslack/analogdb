package events

import (
	"github.com/evanofslack/analogdb/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type eventStats struct {
	enqueued      prometheus.Counter
	dropped       prometheus.Counter
	writeErrors   prometheus.Counter
	queueDepth    prometheus.GaugeFunc
	writeDuration prometheus.Histogram
}

func newEventStats(depth func() float64) *eventStats {
	return &eventStats{
		enqueued: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.EventsSubsystem,
			Name:      "enqueued_total",
			Help:      "Number of events added to the queue",
		}),
		dropped: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.EventsSubsystem,
			Name:      "dropped_total",
			Help:      "Number of events dropped because the queue was full or closed",
		}),
		writeErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.EventsSubsystem,
			Name:      "write_errors_total",
			Help:      "Number of events that failed to write to kafka",
		}),
		queueDepth: prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.EventsSubsystem,
			Name:      "queue_depth",
			Help:      "Number of events waiting in the queue",
		}, depth),
		writeDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: metrics.AnalogdbNamespace,
			Subsystem: metrics.EventsSubsystem,
			Name:      "write_duration_seconds",
			Help:      "Duration of batch writes to kafka",
		}),
	}
}

func (stats *eventStats) register(registerer prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		stats.enqueued,
		stats.dropped,
		stats.writeErrors,
		stats.queueDepth,
		stats.writeDuration,
	}
	for _, c := range collectors {
		if err := registerer.Register(c); err != nil {
			return err
		}
	}
	return nil
}
