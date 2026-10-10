import "server-only";
import { rateLimitKey, resolveClientIp } from "@lib/clientIp";
import { Limited } from "@lib/rateLimited";
import { headers } from "next/headers";

export type Bucket =
  | "browse"
  | "search"
  | "imageSearch"
  | "download"
  | "login"
  | "report"
  | "events";

export type RateLimitResult = { ok: boolean; retryAfter: number };

const minute = 60 * 1000;
const maxEntries = 10000;

const limits: Record<Bucket, { limit: number; windowMs: number }> = {
  browse: { limit: 120, windowMs: minute },
  search: { limit: 30, windowMs: minute },
  imageSearch: { limit: 10, windowMs: minute },
  download: { limit: 20, windowMs: minute },
  login: { limit: 5, windowMs: 15 * minute },
  report: { limit: 5, windowMs: 15 * minute },
  events: { limit: 60, windowMs: minute },
};

const windows = new Map<
  Bucket,
  Map<string, { count: number; resetAt: number }>
>();
let unknownLoggedAt = 0;

function bucketWindows(bucket: Bucket) {
  let entries = windows.get(bucket);
  if (!entries) {
    entries = new Map();
    windows.set(bucket, entries);
  }
  return entries;
}

export function consume(bucket: Bucket, key: string): RateLimitResult {
  const { limit, windowMs } = limits[bucket];
  const entries = bucketWindows(bucket);
  const now = Date.now();
  if (entries.size > maxEntries) {
    entries.forEach((entry, entryKey) => {
      if (entry.resetAt <= now) entries.delete(entryKey);
    });
  }
  const entry = entries.get(key);
  if (!entry || entry.resetAt <= now) {
    entries.set(key, { count: 1, resetAt: now + windowMs });
    return { ok: true, retryAfter: 0 };
  }
  entry.count += 1;
  if (entry.count <= limit) return { ok: true, retryAfter: 0 };
  return { ok: false, retryAfter: Math.ceil((entry.resetAt - now) / 1000) };
}

export function reset(bucket: Bucket, key: string): void {
  bucketWindows(bucket).delete(key);
}

// limit counts one request from the current client against bucket
export async function limit(bucket: Bucket): Promise<RateLimitResult> {
  const forwardedFor = (await headers()).get("x-forwarded-for");
  const ip = resolveClientIp(forwardedFor);
  if (ip === "unknown" && Date.now() - unknownLoggedAt > minute) {
    unknownLoggedAt = Date.now();
    console.warn("no client ip for rate limit", { bucket, forwardedFor });
  }
  const result = consume(bucket, rateLimitKey(ip));
  if (!result.ok) {
    console.warn("rate limited", { bucket, ip, forwardedFor });
  }
  return result;
}

export async function limitAction(bucket: Bucket): Promise<Limited | null> {
  const result = await limit(bucket);
  return result.ok
    ? null
    : { rateLimited: true, retryAfter: result.retryAfter };
}
