"use client";

import { pageView, track } from "@lib/analytics";
import { usePathname } from "next/navigation";
import { useReportWebVitals } from "next/web-vitals";
import { useEffect } from "react";

const metrics = ["LCP", "INP", "CLS", "FCP", "TTFB"];
const sampled = Math.random() < 0.25;

function reportVital(metric) {
  if (!sampled || !metrics.includes(metric.name)) return;
  track("web_vital", {
    metric: metric.name,
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
