"use client";

import { EventMap, pageView, track } from "@lib/analytics";
import { usePathname } from "next/navigation";
import { useReportWebVitals } from "next/web-vitals";
import { useEffect } from "react";

type VitalName = EventMap["web_vital"]["metric"];
type WebVital = Parameters<Parameters<typeof useReportWebVitals>[0]>[0];

const metrics: string[] = ["LCP", "INP", "CLS", "FCP", "TTFB"];
const sampled = Math.random() < 0.25;

function reportVital(metric: WebVital) {
  if (!sampled || !metrics.includes(metric.name)) return;
  track("web_vital", {
    metric: metric.name as VitalName,
    value: metric.value,
    rating: metric.rating,
  });
}

export default function Analytics() {
  const pathname = usePathname();

  useEffect(() => {
    pageView(pathname);
  }, [pathname]);

  useReportWebVitals(reportVital);

  return null;
}
