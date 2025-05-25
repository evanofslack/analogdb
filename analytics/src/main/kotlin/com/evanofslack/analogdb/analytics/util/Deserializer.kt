package com.evanofslack.analogdb.analytics.util

import com.evanofslack.analogdb.analytics.Event
import com.google.protobuf.util.JsonFormat
import org.slf4j.LoggerFactory

class EventDeserializer {
    private val logger = LoggerFactory.getLogger(EventDeserializer::class.java)
    private val jsonParser = JsonFormat.parser()

    fun deserialize(jsonString: String): Event? {
        return try {
            val builder = Event.newBuilder()
            jsonParser.merge(jsonString, builder)
            builder.build()
        } catch (e: Exception) {
            logger.error("Failed to deserialize event: ${e.message}")
            null
        }
    }
}
