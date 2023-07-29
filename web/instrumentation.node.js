import { NodeSDK } from "@opentelemetry/sdk-node";
import { OTLPTraceExporter } from "@opentelemetry/exporter-trace-otlp-proto";
import { Resource } from "@opentelemetry/resources";
import { SemanticResourceAttributes } from "@opentelemetry/semantic-conventions";
import { SimpleSpanProcessor } from "@opentelemetry/sdk-trace-node";
import { otelURL, serviceName } from "./constants.js";

const exporterOptions = {
  url: otelURL,
};

const traceExporter = new OTLPTraceExporter(exporterOptions);

const sdk = new NodeSDK({
  resource: new Resource({
    [SemanticResourceAttributes.SERVICE_NAME]: serviceName,
  }),
  spanProcessor: new SimpleSpanProcessor(traceExporter),
});

sdk.start();
