package com.evanofslack.analogdb.analytics.config

import io.micrometer.core.instrument.MeterRegistry
import org.springframework.context.annotation.Bean
import org.springframework.context.annotation.Configuration
import io.micrometer.core.aop.TimedAspect

@Configuration
class MetricsConfig {
    @Bean
    fun timedAspect(registry: MeterRegistry): TimedAspect {
        return TimedAspect(registry)
    }
}

