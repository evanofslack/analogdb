package com.evanofslack.analogdb.analytics.service

import com.evanofslack.analogdb.analytics.config.KafkaConfig
import com.evanofslack.analogdb.analytics.util.EventDeserializer
import org.slf4j.LoggerFactory
import org.springframework.kafka.annotation.KafkaListener
import org.springframework.stereotype.Service

@Service
class KafkaConsumerService(
    private val clickHouseService: ClickHouseService,
    private val metrics: MetricsService,
    private val deserializer: EventDeserializer,
    private val kafkaConfig: KafkaConfig
) {
    private val log = LoggerFactory.getLogger(KafkaConsumerService::class.java)

    @KafkaListener(topics = ["#{kafkaConfig.topic}"], groupId = "#{kafkaConfig.groupId}")
    fun consume(message: String) {
        metrics.incrementEventsReceived()
        log.debug("Received message: $message")
        
        try {
            val event = deserializer.deserialize(message)
            if (event != null) {
                clickHouseService.addEvent(event)
                metrics.incrementEventsProcessed()
            } else {
                metrics.incrementEventsFailed()
            }
        } catch (e: Exception) {
            log.error("Error processing message: ${e.message}")
            metrics.incrementEventsFailed()
        }
    }
}

