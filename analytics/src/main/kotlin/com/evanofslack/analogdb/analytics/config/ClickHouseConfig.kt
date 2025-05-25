package com.evanofslack.analogdb.analytics.config

import com.zaxxer.hikari.HikariConfig
import com.zaxxer.hikari.HikariDataSource
import java.net.URI
import javax.sql.DataSource
import org.springframework.beans.factory.annotation.Value
import org.springframework.context.annotation.Bean
import org.springframework.context.annotation.Configuration

@Configuration
class ClickHouseConfig {
    @Value("\${clickhouse.url}") lateinit var url: String

    @Value("\${clickhouse.table}") lateinit var table: String

    @Value("\${clickhouse.batchSize}") var batchSize: Int = 1000

    @Value("\${clickhouse.flushIntervalMs}") var flushIntervalMs: Long = 10000

    // Extract database from URL
    val database: String
        get() {
            val uri = URI(url.replace("jdbc:clickhouse://", "http://"))
            return uri.path.substring(1) // Remove leading '/'
        }

    @Bean
    fun dataSource(): DataSource {
        println("DEBUG: Attempting to connect to: $url")
        val config = HikariConfig()
        config.jdbcUrl = url
        config.driverClassName = "com.clickhouse.jdbc.ClickHouseDriver"
        config.connectionTestQuery = "SELECT 1"
        config.maximumPoolSize = 10

        return HikariDataSource(config)
    }
}
