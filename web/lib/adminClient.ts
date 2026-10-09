import "server-only";
import { headers } from "next/headers";
import pkg from "../package.json";
import { baseURL } from "./constants";

// Admin endpoints are not in the OpenAPI spec, so this client is written by hand.

const userAgent = `analogdb-web/${pkg.version}`;

export class AdminError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "AdminError";
    this.status = status;
  }
}

export type DependencyStatus = "ok" | "down" | "disabled";

export interface AdminOverview {
  status: Record<string, DependencyStatus>;
  app: {
    version: string;
    env: string;
    hostname: string;
    uptime_seconds: number;
  };
  counts: {
    posts: number;
    authors: number;
    cameras: number;
    films: number;
    keywords: number;
  };
  posted: { day: number; week: number; month: number };
  freshness: {
    newest_post: number | null;
    score_update: number | null;
    keywords_update: number | null;
    colors_update: number | null;
  };
  database: {
    bytes: number;
    tables: { name: string; bytes: number }[];
    migration_version: number;
    migration_dirty: boolean;
  };
  vectors: { objects: number | null; posts: number };
}

export interface AdminCoverage {
  total: number;
  camera: number;
  film: number;
  description: number;
  keywords: number;
  colors: number;
  focal_length: number;
  aperture: number;
}

export interface UnmatchedCamera {
  camera_make: string;
  camera_model: string;
  post_count: number;
  sample_post_id: number;
}

export interface UnmatchedFilm {
  film_make: string;
  film_type: string;
  film_speed: number | null;
  post_count: number;
  sample_post_id: number;
}

export interface AdminQuality {
  coverage: AdminCoverage;
  unmatched_cameras: UnmatchedCamera[];
  unmatched_films: UnmatchedFilm[];
}

export const missingFields = [
  "camera",
  "film",
  "description",
  "keywords",
  "colors",
  "caption",
  "vector",
] as const;
export type MissingField = (typeof missingFields)[number];

export interface AdminPost {
  id: number;
  title: string;
  author: string;
  time: number;
  low_url: string;
  camera_make: string | null;
  camera_model: string | null;
  film_make: string | null;
  film_type: string | null;
  film_speed: number | null;
}

export interface MissingPosts {
  posts: AdminPost[];
  next_before_id: number | null;
}

export const trafficRanges = ["24h", "7d", "30d"] as const;
export type TrafficRange = (typeof trafficRanges)[number];

export interface TrafficCount {
  name: string;
  client?: string;
  requests: number;
}

export interface Traffic {
  range: TrafficRange;
  bucket: "hour" | "day";
  series: {
    time: string;
    web: number;
    scraper: number;
    other: number;
    status_4xx: number;
    status_5xx: number;
  }[];
  totals: {
    requests: number;
    unique_ips: number;
    status_2xx: number;
    status_3xx: number;
    status_4xx: number;
    status_5xx: number;
  };
  routes: {
    route: string;
    requests: number;
    p50_ms: number;
    p95_ms: number;
    errors: number;
  }[];
  posts: { post_id: number; requests: number }[];
  legacy: { client: string; requests: number; legacy: number }[];
  params: TrafficCount[];
  user_agents: TrafficCount[];
  ips: TrafficCount[];
  errors: {
    time: string;
    method: string;
    path: string;
    status: number;
    request_id: string;
  }[];
}

export interface AuditEntry {
  time: string;
  start_ms: number;
  method: string;
  path: string;
  status: number;
  remote_ip: string;
  user_agent: string;
  request_id: string;
  client: string;
}

export interface AuditPage {
  entries: AuditEntry[];
  next_before: number | null;
}

export interface ReviewPost {
  id: number;
  title: string;
  author: string;
  permalink: string;
  timestamp: number;
  nsfw: boolean;
  grayscale: boolean;
  sprocket: boolean;
  images: { resolution: string; url: string; width: number; height: number }[];
}

export interface ReviewPage {
  posts: ReviewPost[];
  next_cursor: string | null;
}

export interface PostPatch {
  description?: string;
  nsfw?: boolean;
  grayscale?: boolean;
  sprocket?: boolean;
  camera_make?: string;
  camera_model?: string;
  film_make?: string;
  film_type?: string;
  film_speed?: number;
  focal_length?: number;
  aperture?: string;
}

export interface CreateCamera {
  make: string;
  model: string;
  description: string;
}

export interface CreateFilm {
  make: string;
  type: string;
  speed: number;
  color_type: string;
  description: string;
}

async function adminFetch(
  route: string,
  init: { method?: string; body?: unknown } = {}
): Promise<Response> {
  const requestHeaders: Record<string, string> = { "User-Agent": userAgent };

  const username = process.env.API_ADMIN_USERNAME;
  const password = process.env.API_ADMIN_PASSWORD;
  if (username && password) {
    const auth = Buffer.from(`${username}:${password}`).toString("base64");
    requestHeaders["Authorization"] = `Basic ${auth}`;
  }

  const forwarded = (await headers()).get("x-forwarded-for");
  if (forwarded) {
    requestHeaders["X-Forwarded-For"] = forwarded;
  }

  if (init.body !== undefined) {
    requestHeaders["Content-Type"] = "application/json";
  }

  const response = await fetch(`${baseURL}${route}`, {
    method: init.method ?? "GET",
    headers: requestHeaders,
    body: init.body !== undefined ? JSON.stringify(init.body) : undefined,
    cache: "no-store",
  });

  if (!response.ok) {
    let message = response.statusText;
    try {
      const body = await response.json();
      if (body?.error) message = body.error;
    } catch {}
    throw new AdminError(response.status, message);
  }
  return response;
}

async function adminGet<T>(route: string): Promise<T> {
  const response = await adminFetch(route);
  return (await response.json()) as T;
}

function query(params: Record<string, string | number | null | undefined>) {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== null && value !== undefined && value !== "") {
      search.set(key, String(value));
    }
  }
  const str = search.toString();
  return str ? `?${str}` : "";
}

export function getOverview(): Promise<AdminOverview> {
  return adminGet("/admin/overview");
}

export function getQuality(): Promise<AdminQuality> {
  return adminGet("/admin/quality");
}

export function getMissingPosts(
  field: MissingField,
  beforeId?: number | null
): Promise<MissingPosts> {
  return adminGet(
    `/admin/posts/missing${query({ field, before_id: beforeId, limit: 40 })}`
  );
}

export function getTraffic(range: TrafficRange): Promise<Traffic> {
  return adminGet(`/admin/traffic${query({ range })}`);
}

export function getAudit(before?: number | null): Promise<AuditPage> {
  return adminGet(`/admin/audit${query({ before, limit: 50 })}`);
}

export async function getReviewPosts(
  cursor?: string | null
): Promise<ReviewPage> {
  const body = await adminGet<{
    posts: ReviewPost[];
    meta: { next_cursor?: string };
  }>(`/posts${query({ sort: "time", page_size: 40, cursor })}`);
  return {
    posts: body.posts ?? [],
    next_cursor: body.meta?.next_cursor || null,
  };
}

export interface EditablePost {
  id: number;
  description?: string;
  nsfw: boolean;
  grayscale: boolean;
  sprocket: boolean;
  camera_make?: string;
  camera_model?: string;
  film_make?: string;
  film_type?: string;
  film_speed?: number;
  focal_length?: number;
  aperture?: string;
}

export async function getPost(id: number): Promise<EditablePost> {
  const post = await adminGet<EditablePost & Record<string, unknown>>(
    `/post/${id}`
  );
  return {
    id: post.id,
    description: post.description,
    nsfw: post.nsfw,
    grayscale: post.grayscale,
    sprocket: post.sprocket,
    camera_make: post.camera_make,
    camera_model: post.camera_model,
    film_make: post.film_make,
    film_type: post.film_type,
    film_speed: post.film_speed,
    focal_length: post.focal_length,
    aperture: post.aperture,
  };
}

export async function patchPost(id: number, patch: PostPatch): Promise<void> {
  await adminFetch(`/post/${id}`, { method: "PATCH", body: patch });
}

export async function deletePost(id: number): Promise<void> {
  await adminFetch(`/post/${id}`, { method: "DELETE" });
}

export async function createCamera(camera: CreateCamera): Promise<void> {
  await adminFetch("/camera", { method: "PUT", body: camera });
}

export async function createFilm(film: CreateFilm): Promise<void> {
  await adminFetch("/film", { method: "PUT", body: film });
}
