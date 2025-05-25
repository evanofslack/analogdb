package com.evanofslack.analogdb.analytics.service

import io.micrometer.core.instrument.Counter
import io.micrometer.core.instrument.MeterRegistry
import io.micrometer.core.instrument.Timer
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger
import org.springframework.stereotype.Service

@Service
class MetricsService(private val registry: MeterRegistry) {
    private val eventsReceived =
            Counter.builder("analytics.events.received")
                    .description("Number of events received from Kafka")
                    .register(registry)

    private val eventsProcessed =
            Counter.builder("analytics.events.processed")
                    .description("Number of events successfully processed")
                    .register(registry)

    private val eventsFailed =
            Counter.builder("analytics.events.failed")
                    .description("Number of events that failed processing")
                    .register(registry)

    private val clickhouseInsertTimer =
            Timer.builder("analytics.clickhouse.insert.time")
                    .description("Time taken to insert events into ClickHouse")
                    .register(registry)

    // Use AtomicInteger to track the batch size
    private val batchSizeValue = AtomicInteger(0)

    // Register the gauge with the registry
    init {
        registry.gauge("analytics.batch.size", batchSizeValue)
    }

    fun incrementEventsReceived() = eventsReceived.increment()

    fun incrementEventsProcessed() = eventsProcessed.increment()

    fun incrementEventsFailed() = eventsFailed.increment()

    fun recordClickhouseInsertTime(timeMs: Long) =
            clickhouseInsertTimer.record(timeMs, TimeUnit.MILLISECONDS)

    fun updateBatchSize(size: Int) {
        batchSizeValue.set(size)
    }
}

