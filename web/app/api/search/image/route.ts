import { searchApi } from "@lib/client";
import { limit } from "@lib/rateLimit";
import { NextResponse } from "next/server";

const maxBytes = 5 * 1024 * 1024;
const imageTypes = ["image/jpeg", "image/png", "image/webp"];
const pageSize = 50;

function flag(value: string | null): boolean | undefined {
  if (value === "true") return true;
  if (value === "false") return false;
  return undefined;
}

function error(message: string, status: number, init: ResponseInit = {}) {
  return NextResponse.json({ error: message }, { ...init, status });
}

export async function POST(request: Request) {
  const limited = await limit("imageSearch");
  if (!limited.ok) {
    return error("Too many searches, try again in a minute", 429, {
      headers: { "Retry-After": String(limited.retryAfter) },
    });
  }

  const contentType = request.headers.get("content-type") ?? "";
  if (!contentType.startsWith("multipart/form-data")) {
    return error("Expected multipart form data", 415);
  }
  const contentLength = Number(request.headers.get("content-length"));
  if (contentLength > maxBytes + 64 * 1024) {
    return error("Image is too large", 413);
  }

  let form: FormData;
  try {
    form = await request.formData();
  } catch {
    return error("Invalid form data", 400);
  }

  const keys = Array.from(new Set(form.keys()));
  const image = form.get("image");
  if (
    keys.length !== 1 ||
    keys[0] !== "image" ||
    form.getAll("image").length !== 1
  ) {
    return error("Expected one image field", 400);
  }
  if (!(image instanceof Blob)) {
    return error("Expected an image file", 400);
  }
  if (!imageTypes.includes(image.type)) {
    return error("Image must be JPEG, PNG or WebP", 415);
  }
  if (image.size === 0 || image.size > maxBytes) {
    return error("Image is too large", 413);
  }

  const params = new URL(request.url).searchParams;

  try {
    const response = await searchApi.searchImagePost({
      image,
      pageSize,
      nsfw: flag(params.get("nsfw")),
      grayscale: flag(params.get("grayscale")),
      sprocket: flag(params.get("sprocket")),
    });
    return NextResponse.json(response, {
      headers: { "Cache-Control": "no-store" },
    });
  } catch (err) {
    const status = err?.response?.status;
    if (status === 400 || status === 413 || status === 415) {
      return error("Couldn't read that image", status);
    }
    console.error("image search request failed:", err);
    return error("Image search failed", 502);
  }
}
