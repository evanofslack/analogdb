import { apiHeaders } from "@lib/client";
import { rateLimitKey, resolveClientIp } from "@lib/clientIp";
import { baseURL } from "@lib/constants";
import { consume } from "@lib/rateLimit";
import { after } from "next/server";

const maxBytes = 32 * 1024;
const minute = 60 * 1000;
const contentTypes = ["text/plain", "application/json"];

let droppedLoggedAt = 0;

function done() {
  return new Response(null, {
    status: 204,
    headers: { "Cache-Control": "no-store" },
  });
}

function logDrop(reason: string, ip: string) {
  if (Date.now() - droppedLoggedAt < minute) return;
  droppedLoggedAt = Date.now();
  console.warn("dropping ui events", { reason, ip });
}

async function forward(body: string, ip: string, userAgent: string) {
  const headers: Record<string, string> = {
    ...apiHeaders,
    "Content-Type": "application/json",
    "X-Analogdb-Visitor-UA": userAgent,
  };
  if (ip !== "unknown") headers["X-Analogdb-Visitor-IP"] = ip;
  try {
    const response = await fetch(`${baseURL}/events`, {
      method: "POST",
      headers,
      body,
      signal: AbortSignal.timeout(5000),
    });
    if (!response.ok) {
      console.warn("forwarding ui events failed", { status: response.status });
    }
  } catch (err) {
    console.warn("forwarding ui events failed", err);
  }
}

export async function POST(request: Request) {
  if (request.headers.get("sec-fetch-site") !== "same-origin") return done();

  const contentType = request.headers.get("content-type") ?? "";
  if (!contentTypes.some((type) => contentType.startsWith(type))) {
    return done();
  }
  if (Number(request.headers.get("content-length")) > maxBytes) return done();

  const ip = resolveClientIp(request.headers.get("x-forwarded-for"));
  if (!consume("events", rateLimitKey(ip)).ok) {
    logDrop("rate limited", ip);
    return done();
  }

  let body: string;
  try {
    body = await request.text();
  } catch {
    return done();
  }
  if (!body || body.length > maxBytes) return done();

  const userAgent = request.headers.get("user-agent") ?? "";
  after(() => forward(body, ip, userAgent));
  return done();
}
