package com.evanofslack.analogdb.analytics.config

import com.evanofslack.analogdb.analytics.util.EventDeserializer
import org.springframework.context.annotation.Bean
import org.springframework.context.annotation.Configuration

@Configuration
class AppConfig {
    @Bean
    fun eventDeserializer(): EventDeserializer {
        return EventDeserializer()
    }
}

