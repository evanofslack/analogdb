import { OTLPTraceExporter } from "@opentelemetry/exporter-trace-otlp-http";
import { resourceFromAttributes } from "@opentelemetry/resources";
import { NodeSDK } from "@opentelemetry/sdk-node";
import { SimpleSpanProcessor } from "@opentelemetry/sdk-trace-node";
import { ATTR_SERVICE_NAME } from "@opentelemetry/semantic-conventions";

const resource = resourceFromAttributes({
  [ATTR_SERVICE_NAME]: "api-service",
});

const spanProcessor = new SimpleSpanProcessor(
  new OTLPTraceExporter({
    url:
      process.env.OTEL_EXPORTER_OTLP_TRACES_ENDPOINT ||
      "http://otel-collector:4318",
  })
);

const sdk = new NodeSDK({
  resource: resource,
  spanProcessor: spanProcessor,
});

console.log("OpenTelemetry initializing...");
console.log(`Service: ${process.env.OTEL_SERVICE_NAME || "nextjs-app"}`);
console.log(
  `Endpoint: ${
    process.env.OTEL_EXPORTER_OTLP_TRACES_ENDPOINT ||
    "http://localhost:4318/v1/traces"
  }`
);

sdk.start();

console.log("OpenTelemetry started successfully");

process.on("SIGTERM", () => {
  sdk
    .shutdown()
    .then(() => console.log("OpenTelemetry terminated"))
    .catch((error) => console.log("Error terminating OpenTelemetry", error))
    .finally(() => process.exit(0));
});
