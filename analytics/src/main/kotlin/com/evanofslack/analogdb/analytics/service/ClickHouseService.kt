package com.evanofslack.analogdb.analytics.service

import com.evanofslack.analogdb.analytics.Event
import com.evanofslack.analogdb.analytics.config.ClickHouseConfig
import java.sql.PreparedStatement
import java.sql.SQLException
import java.util.concurrent.ConcurrentLinkedQueue
import javax.sql.DataSource
import org.slf4j.LoggerFactory
import org.springframework.scheduling.annotation.Scheduled
import org.springframework.stereotype.Service

@Service
class ClickHouseService(
        private val dataSource: DataSource,
        private val config: ClickHouseConfig,
        private val metrics: MetricsService
) {
    private val log = LoggerFactory.getLogger(ClickHouseService::class.java)
    private val queue = ConcurrentLinkedQueue<Event>()

    init {
        createTableIfNotExists()
    }

    fun addEvent(event: Event) {
        queue.add(event)
        metrics.updateBatchSize(queue.size)

        if (queue.size >= config.batchSize) {
            flush()
        }
    }

    @Scheduled(fixedDelayString = "\${clickhouse.flushIntervalMs}")
    fun flush() {
        if (queue.isEmpty()) return

        val events = mutableListOf<Event>()
        while (events.size < config.batchSize && !queue.isEmpty()) {
            queue.poll()?.let { events.add(it) }
        }

        metrics.updateBatchSize(queue.size)
        if (events.isNotEmpty()) {
            insertBatch(events)
        }
    }

    private fun insertBatch(events: List<Event>) {
        if (events.isEmpty()) return

        val sql = buildInsertSql()
        val startTime = System.currentTimeMillis()

        try {
            dataSource.connection.use { conn ->
                conn.prepareStatement(sql).use { stmt ->
                    for (event in events) {
                        bindEventParams(stmt, event)
                        stmt.addBatch()
                    }
                    stmt.executeBatch()
                    log.info("Inserted ${events.size} events to ClickHouse")
                    metrics.recordClickhouseInsertTime(System.currentTimeMillis() - startTime)
                }
            }
        } catch (e: SQLException) {
            log.error("Failed to insert batch to ClickHouse: ${e.message}")
            throw e
        }
    }

    private fun bindEventParams(stmt: PreparedStatement, event: Event) {
        var idx = 1
        stmt.setString(idx++, event.requestId)
        stmt.setString(idx++, event.remoteIp)
        stmt.setString(idx++, event.url)
        stmt.setString(idx++, event.path)
        stmt.setString(idx++, event.protocol)
        stmt.setString(idx++, event.scheme)
        stmt.setString(idx++, event.method)
        stmt.setString(idx++, event.userAgent)
        stmt.setInt(idx++, event.responseCode)
        stmt.setString(idx++, event.hostname)
        stmt.setBoolean(idx++, event.authorized)
        stmt.setLong(idx++, event.startTime)
        stmt.setLong(idx++, event.endTime)
        stmt.setLong(idx++, event.requestTimeMs)
        stmt.setInt(idx++, event.bytesIn)
        stmt.setInt(idx, event.bytesOut)
    }

    private fun buildInsertSql(): String {
        return """
            INSERT INTO ${config.database}.${config.table} (
                request_id, remote_ip, url, path, protocol, scheme, method, 
                user_agent, response_code, hostname, authorized, 
                start_time, end_time, request_time_ms, bytes_in, bytes_out
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """.trimIndent()
    }

    private fun createTableIfNotExists() {
        val sql =
                """
            CREATE TABLE IF NOT EXISTS ${config.database}.${config.table} (
                request_id String,
                remote_ip String,
                url String,
                path String,
                protocol String,
                scheme String,
                method String,
                user_agent String,
                response_code Int32,
                hostname String,
                authorized Boolean,
                start_time Int64,
                end_time Int64,
                request_time_ms Int64,
                bytes_in Int32,
                bytes_out Int32,
                event_time DateTime DEFAULT now()
            ) ENGINE = MergeTree()
            PARTITION BY toYYYYMM(event_time)
            ORDER BY (event_time, request_id)
        """.trimIndent()

        try {
            dataSource.connection.use { conn ->
                conn.createStatement().use { stmt ->
                    stmt.execute(sql)
                    log.info("ClickHouse table ${config.table} created or already exists")
                }
            }
        } catch (e: SQLException) {
            log.error("Failed to create ClickHouse table: ${e.message}")
            throw e
        }
    }
}
