import "server-only";
import { BlockList, isIP } from "net";
import { headers } from "next/headers";

const mappedPrefix = "::ffff:";

function family(ip: string): "ipv4" | "ipv6" | null {
  const version = isIP(ip);
  if (version === 4) return "ipv4";
  if (version === 6) return "ipv6";
  return null;
}

function parseTrustedProxies(value: string | undefined): BlockList {
  const list = new BlockList();
  for (const raw of (value ?? "").split(",")) {
    const entry = raw.trim();
    if (!entry) continue;
    const [address, prefix, ...rest] = entry.split("/");
    const type = family(address);
    const bits = Number(prefix);
    const maxBits = type === "ipv6" ? 128 : 32;
    if (!type || rest.length > 0) {
      console.warn(`ignoring invalid trusted proxy ${entry}`);
    } else if (prefix === undefined) {
      list.addAddress(address, type);
    } else if (Number.isInteger(bits) && bits >= 0 && bits <= maxBits) {
      list.addSubnet(address, bits, type);
    } else {
      console.warn(`ignoring invalid trusted proxy ${entry}`);
    }
  }
  return list;
}

const trustedProxies = parseTrustedProxies(process.env.TRUSTED_PROXIES);

function normalize(ip: string): string {
  const lower = ip.toLowerCase();
  if (lower.startsWith(mappedPrefix)) {
    const v4 = lower.slice(mappedPrefix.length);
    if (isIP(v4) === 4) return v4;
  }
  return lower;
}

// resolveClientIp walks X-Forwarded-For from the right, skipping trusted
// proxies, the same way the backend's resolveClientIP does
export function resolveClientIp(forwardedFor: string | null): string {
  const hops = (forwardedFor ?? "")
    .split(",")
    .map((hop) => hop.trim())
    .filter(Boolean);
  for (let i = hops.length - 1; i >= 0; i--) {
    const ip = normalize(hops[i]);
    const type = family(ip);
    if (!type) return "unknown";
    if (!trustedProxies.check(ip, type)) return ip;
  }
  return "unknown";
}

export async function clientIp(): Promise<string> {
  return resolveClientIp((await headers()).get("x-forwarded-for"));
}

function expandIpv6(ip: string): string[] {
  const [head, tail] = ip.split("::");
  const groups = (part: string) =>
    part
      ? part.split(":").flatMap((g) => (g.includes(".") ? ["0", "0"] : [g]))
      : [];
  const left = groups(head);
  if (tail === undefined) return left;
  const right = groups(tail);
  const zeros = Array(8 - left.length - right.length).fill("0");
  return [...left, ...zeros, ...right];
}

// rateLimitKey groups IPv6 clients by /64, since one host usually gets a whole /64
export function rateLimitKey(ip: string): string {
  if (family(ip) !== "ipv6") return ip;
  const prefix = expandIpv6(ip)
    .slice(0, 4)
    .map((group) => parseInt(group, 16).toString(16))
    .join(":");
  return `${prefix}::/64`;
}
