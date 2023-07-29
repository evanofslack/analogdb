import { HttpInstrumentation } from "@opentelemetry/instrumentation-http";
import {
  CompositePropagator,
  W3CTraceContextPropagator,
} from "@opentelemetry/core";
import { OTLPTraceExporter } from "@opentelemetry/exporter-trace-otlp-grpc";
import { registerInstrumentations } from "@opentelemetry/instrumentation";
import {
  detectResourcesSync,
  envDetector,
  Resource,
} from "@opentelemetry/resources";
import { BatchSpanProcessor } from "@opentelemetry/sdk-trace-base";
import { NodeTracerProvider } from "@opentelemetry/sdk-trace-node";
import { SemanticResourceAttributes } from "@opentelemetry/semantic-conventions";
import { otelURL, serviceName } from "./constants.js";

export function register() {
  registerInstrumentations({
    instrumentations: [new HttpInstrumentation()],
  });

  const resource = Resource.default()
    .merge(
      new Resource({
        [SemanticResourceAttributes.SERVICE_NAME]: serviceName,
      })
    )
    .merge(detectResourcesSync({ detectors: [envDetector] }));
  const provider = new NodeTracerProvider({ resource });

  const exporter = new OTLPTraceExporter({
    url: otelURL,
  });

  console.log("exporting OTEL traces to endpoint: " + otelURL);

  let processor = new BatchSpanProcessor(exporter);

  const propagator = new CompositePropagator({
    propagators: [new W3CTraceContextPropagator()],
  });

  provider.addSpanProcessor(processor);

  provider.register({ propagator });

  process.on("SIGTERM", () => {
    provider.shutdown();
  });
}

register();
