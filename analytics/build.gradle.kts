import com.google.protobuf.gradle.*
import org.jetbrains.kotlin.gradle.tasks.KotlinCompile

plugins {
    id("org.springframework.boot") version "3.2.1"
    id("io.spring.dependency-management") version "1.1.4"
    id("com.google.protobuf") version "0.9.4"
    kotlin("jvm") version "1.9.21"
    kotlin("plugin.spring") version "1.9.21"
}

group = "com.evanofslack"

version = "0.1.0"

java.sourceCompatibility = JavaVersion.VERSION_17

repositories { mavenCentral() }

// Configure source sets to include proto files from parent directory
sourceSets { main { proto { srcDir("proto") } } }

dependencies {
    implementation("org.springframework.boot:spring-boot-starter")
    implementation("org.springframework.boot:spring-boot-starter-web")
    implementation("org.springframework.kafka:spring-kafka")
    implementation("org.jetbrains.kotlin:kotlin-reflect")
    implementation("org.jetbrains.kotlin:kotlin-stdlib")
    implementation("com.clickhouse:clickhouse-jdbc:0.4.6")
    implementation("com.zaxxer:HikariCP:5.0.1")
    implementation("io.micrometer:micrometer-registry-prometheus")
    implementation("org.springframework.boot:spring-boot-starter-actuator")

    // Updated protobuf and grpc versions for compatibility
    implementation("com.google.protobuf:protobuf-java:3.24.4")
    implementation("com.google.protobuf:protobuf-java-util:3.24.4")
    implementation("io.grpc:grpc-stub:1.58.0")
    implementation("io.grpc:grpc-protobuf:1.58.0")

    // Required for Java 9+ compatibility
    if (JavaVersion.current().isJava9Compatible()) {
        implementation("javax.annotation:javax.annotation-api:1.3.2")
    }

    implementation("com.fasterxml.jackson.module:jackson-module-kotlin")
    implementation("net.logstash.logback:logstash-logback-encoder:7.4")

    testImplementation("org.springframework.boot:spring-boot-starter-test")
    testImplementation("org.springframework.kafka:spring-kafka-test")
}


protobuf {
    protoc { artifact = "com.google.protobuf:protoc:3.24.4" }

    plugins { create("grpc") { artifact = "io.grpc:protoc-gen-grpc-java:1.58.0" } }

    generateProtoTasks {
        // Use "main" source set, not the directory path
        ofSourceSet("main").forEach { it.plugins { create("grpc") } }
    }
}

tasks.withType<KotlinCompile> {
    kotlinOptions {
        freeCompilerArgs += "-Xjsr305=strict"
        jvmTarget = "17"
    }
}

tasks.withType<Test> { useJUnitPlatform() }
