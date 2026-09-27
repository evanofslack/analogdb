import "server-only";

import { createHash, createHmac, timingSafeEqual } from "crypto";
import { cookies, headers } from "next/headers";

export const adminCookieName = "admin-token";
export const sessionMaxAge = 60 * 60 * 24 * 7;

const loginLimit = 5;
const loginWindowMs = 15 * 60 * 1000;
const loginAttempts = new Map<string, { count: number; resetAt: number }>();

function adminConfig() {
  const username = process.env.ADMIN_USERNAME;
  const password = process.env.ADMIN_PASSWORD;
  const secret = process.env.ADMIN_SECRET;
  if (!username || !password || !secret) return null;
  return { username, password, secret };
}

function safeEqual(a: string, b: string): boolean {
  const hashA = createHash("sha256").update(a).digest();
  const hashB = createHash("sha256").update(b).digest();
  return timingSafeEqual(hashA, hashB);
}

function sign(value: string, secret: string): string {
  return createHmac("sha256", secret).update(value).digest("base64url");
}

export function createSessionToken(): string | null {
  const config = adminConfig();
  if (!config) return null;
  const exp = String(Date.now() + sessionMaxAge * 1000);
  const encodedExp = Buffer.from(exp).toString("base64url");
  return `${encodedExp}.${sign(encodedExp, config.secret)}`;
}

function verifySessionToken(token: string, secret: string): boolean {
  const [encodedExp, signature, ...rest] = token.split(".");
  if (!encodedExp || !signature || rest.length > 0) return false;
  if (!safeEqual(signature, sign(encodedExp, secret))) return false;
  const exp = Number(Buffer.from(encodedExp, "base64url").toString());
  return Number.isFinite(exp) && exp > Date.now();
}

export function checkCredentials(username: string, password: string): boolean {
  const config = adminConfig();
  if (!config) return false;
  const usernameOk = safeEqual(username, config.username);
  const passwordOk = safeEqual(password, config.password);
  return usernameOk && passwordOk;
}

export async function checkAdminAuth(): Promise<boolean> {
  const config = adminConfig();
  if (!config) return false;
  const token = (await cookies()).get(adminCookieName)?.value;
  if (!token) return false;
  return verifySessionToken(token, config.secret);
}

export async function clientIp(): Promise<string> {
  const forwarded = (await headers()).get("x-forwarded-for");
  return forwarded?.split(",")[0].trim() || "unknown";
}

export function consumeLoginAttempt(ip: string): boolean {
  const now = Date.now();
  if (loginAttempts.size > 10000) {
    loginAttempts.forEach((entry, key) => {
      if (entry.resetAt <= now) loginAttempts.delete(key);
    });
  }
  const entry = loginAttempts.get(ip);
  if (!entry || entry.resetAt <= now) {
    loginAttempts.set(ip, { count: 1, resetAt: now + loginWindowMs });
    return true;
  }
  entry.count += 1;
  return entry.count <= loginLimit;
}

export function resetLoginAttempts(ip: string): void {
  loginAttempts.delete(ip);
}
