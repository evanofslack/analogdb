package com.evanofslack.analogdb.analytics

import org.springframework.boot.autoconfigure.SpringBootApplication
import org.springframework.boot.runApplication
import org.springframework.scheduling.annotation.EnableScheduling

@SpringBootApplication
@EnableScheduling
class AnalyticsApplication

fun main(args: Array<String>) {
    runApplication<AnalyticsApplication>(*args)
}
