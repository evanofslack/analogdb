package com.evanofslack.analogdb.analytics.config

import org.springframework.beans.factory.annotation.Value
import org.springframework.context.annotation.Configuration
import org.springframework.kafka.annotation.EnableKafka

@Configuration
@EnableKafka
class KafkaConfig {
    @Value("\${kafka.topic}")
    lateinit var topic: String

    @Value("\${spring.kafka.consumer.bootstrap-servers}")
    lateinit var bootstrapServers: String

    @Value("\${spring.kafka.consumer.group-id}")
    lateinit var groupId: String
}

