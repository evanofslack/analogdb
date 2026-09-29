"use server";

import {
  AdminError,
  createCamera,
  createFilm,
  deletePost,
  EditablePost,
  getPost,
  patchPost,
  PostPatch,
} from "@lib/adminClient";
import { checkAdminAuth } from "@lib/auth";
import { revalidatePath, revalidateTag } from "next/cache";

export type ActionResult = { ok: true } | { ok: false; error: string };

const textFields = [
  "camera_make",
  "camera_model",
  "film_make",
  "film_type",
  "aperture",
] as const;
const numberFields = ["film_speed", "focal_length"] as const;
const boolFields = ["nsfw", "grayscale", "sprocket"] as const;

const maxTextLength = 255;
const maxDescriptionLength = 10000;

function fail(error: string): { ok: false; error: string } {
  return { ok: false, error };
}

async function run(action: () => Promise<void>): Promise<ActionResult> {
  if (!(await checkAdminAuth())) {
    return fail("Not logged in as admin");
  }
  try {
    await action();
    return { ok: true };
  } catch (error) {
    console.error("admin action failed:", error);
    if (error instanceof AdminError) {
      return fail(error.message || `Request failed (${error.status})`);
    }
    return fail("Request failed");
  }
}

// cleanPatch keeps only editable fields with the right types. Score is left
// out on purpose, the pipeline overwrites it.
function cleanPatch(input: Record<string, unknown>): PostPatch | string {
  const patch: Record<string, unknown> = {};

  if (input.description !== undefined) {
    if (typeof input.description !== "string")
      return "description must be text";
    if (input.description.length > maxDescriptionLength) {
      return "description is too long";
    }
    patch.description = input.description.trim();
  }
  for (const key of textFields) {
    const value = input[key];
    if (value === undefined) continue;
    if (typeof value !== "string") return `${key} must be text`;
    if (value.length > maxTextLength) return `${key} is too long`;
    patch[key] = value.trim().toLowerCase();
  }
  for (const key of numberFields) {
    const value = input[key];
    if (value === undefined) continue;
    if (typeof value !== "number" || !Number.isInteger(value) || value <= 0) {
      return `${key} must be a positive whole number`;
    }
    patch[key] = value;
  }
  for (const key of boolFields) {
    const value = input[key];
    if (value === undefined) continue;
    if (typeof value !== "boolean") return `${key} must be true or false`;
    patch[key] = value;
  }

  if (Object.keys(patch).length === 0) return "nothing to change";
  return patch as PostPatch;
}

function validId(id: unknown): id is number {
  return typeof id === "number" && Number.isInteger(id) && id > 0;
}

function revalidatePost(id: number) {
  revalidatePath(`/post/${id}`);
  revalidatePath("/");
  revalidateTag("posts");
}

export async function updatePost(
  id: number,
  input: Record<string, unknown>
): Promise<ActionResult> {
  if (!validId(id)) return fail("invalid post id");
  const patch = cleanPatch(input ?? {});
  if (typeof patch === "string") return fail(patch);

  return run(async () => {
    await patchPost(id, patch);
    revalidatePost(id);
  });
}

export async function loadPost(
  id: number
): Promise<{ ok: true; post: EditablePost } | { ok: false; error: string }> {
  if (!validId(id)) return { ok: false, error: "invalid post id" };
  let post: EditablePost | undefined;
  const result = await run(async () => {
    post = await getPost(id);
  });
  if ("error" in result) return fail(result.error);
  if (!post) return fail("post not found");
  return { ok: true, post };
}

export async function removePost(id: number): Promise<ActionResult> {
  if (!validId(id)) return fail("invalid post id");
  return run(async () => {
    await deletePost(id);
    revalidatePost(id);
  });
}

function requireText(value: unknown, name: string): string {
  if (typeof value !== "string" || value.trim() === "") {
    throw new AdminError(400, `${name} is required`);
  }
  if (value.length > maxTextLength) {
    throw new AdminError(400, `${name} is too long`);
  }
  return value.trim().toLowerCase();
}

export async function addCatalogCamera(input: {
  make: unknown;
  model: unknown;
  description: unknown;
}): Promise<ActionResult> {
  return run(async () => {
    await createCamera({
      make: requireText(input?.make, "make"),
      model: requireText(input?.model, "model"),
      description:
        typeof input?.description === "string" ? input.description.trim() : "",
    });
    revalidateTag("posts");
  });
}

const colorTypes = ["color", "bw"];

export async function addCatalogFilm(input: {
  make: unknown;
  type: unknown;
  speed: unknown;
  colorType: unknown;
  description: unknown;
}): Promise<ActionResult> {
  return run(async () => {
    const speed = input?.speed;
    if (typeof speed !== "number" || !Number.isInteger(speed) || speed <= 0) {
      throw new AdminError(400, "speed must be a positive whole number");
    }
    const colorType = input?.colorType;
    if (typeof colorType !== "string" || !colorTypes.includes(colorType)) {
      throw new AdminError(
        400,
        `color type must be one of ${colorTypes.join(", ")}`
      );
    }
    await createFilm({
      make: requireText(input?.make, "make"),
      type: requireText(input?.type, "type"),
      speed,
      color_type: colorType,
      description:
        typeof input?.description === "string" ? input.description.trim() : "",
    });
    revalidateTag("posts");
  });
}
