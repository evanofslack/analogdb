FROM gradle:8.5-jdk17 AS build

WORKDIR /app

COPY gradle/ gradle/
COPY gradlew gradlew.bat ./
COPY settings.gradle.kts build.gradle.kts ./
COPY gradle.properties* ./

RUN gradle dependencies --no-daemon

COPY proto/ proto/

RUN echo "=== Proto files copied ===" && \
    find proto/ -name "*.proto" -type f || echo "No .proto files found"

COPY src/ src/

RUN gradle build --no-daemon -x test && \
    gradle bootJar --no-daemon

FROM gcr.io/distroless/java17-debian12:nonroot

WORKDIR /app

COPY --from=build /app/build/libs/*.jar app.jar

USER nonroot:nonroot

EXPOSE 8080
EXPOSE 8081

ENTRYPOINT ["java", \
    "-XX:+UseContainerSupport", \
    "-XX:MaxRAMPercentage=75.0", \
    "-XX:+UseG1GC", \
    "-XX:+UseStringDeduplication", \
    "-Djava.security.egd=file:/dev/./urandom", \
    "-jar", \
    "/app/app.jar"]
