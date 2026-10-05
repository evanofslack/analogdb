import { ServerSearchResponse } from "analogdb-generated";

export type ImageSearch = {
  blob: Blob;
  thumb: string;
  flagsKey: string;
  response: ServerSearchResponse;
};

export const imageAccept = ["image/jpeg", "image/png", "image/webp"];
export const imageMaxSize = 10 * 1024 * 1024;

export function rejectMessage(
  rejections: { errors?: { code?: string }[] }[]
): string {
  const code = rejections?.[0]?.errors?.[0]?.code;
  if (code === "file-too-large") return "that image is over 10 MB";
  if (code === "file-invalid-type") return "use a JPEG, PNG or WebP image";
  if (code === "too-many-files") return "drop one image at a time";
  return "couldn't use that file";
}

const maxEntries = 10;
const searches = new Map<string, ImageSearch>();

export function saveImageSearch(entry: ImageSearch): string {
  const token = Math.random().toString(36).slice(2, 10);
  searches.set(token, entry);
  if (searches.size > maxEntries) {
    const [oldest, old] = searches.entries().next().value;
    URL.revokeObjectURL(old.thumb);
    searches.delete(oldest);
  }
  return token;
}

export function getImageSearch(token: string | null): ImageSearch | null {
  return (token && searches.get(token)) || null;
}

export function updateImageSearch(token: string, entry: ImageSearch): void {
  if (searches.has(token)) searches.set(token, entry);
}

export async function postImageSearch(
  blob: Blob,
  flags: Record<string, boolean>
): Promise<ServerSearchResponse> {
  const form = new FormData();
  form.append("image", blob, "image.jpg");
  const params = new URLSearchParams(
    Object.entries(flags).map(([key, value]) => [key, String(value)])
  );
  const response = await fetch(`/api/search/image?${params}`, {
    method: "POST",
    body: form,
  });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(body.error?.toLowerCase() || "image search failed");
  }
  return body;
}
