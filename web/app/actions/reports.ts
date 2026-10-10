"use server";

import { postApi } from "@lib/client";
import { limitAction } from "@lib/rateLimit";
import { Limited } from "@lib/rateLimited";
import { isReportReason, maxReportEmail, maxReportMessage } from "@lib/reports";
import { ResponseError } from "analogdb-generated";

export type ReportResult =
  | { ok: true }
  | { ok: false; error: string }
  | Limited;

export async function reportPost(
  id: number,
  input: { reason: unknown; message: unknown; email: unknown; website: unknown }
): Promise<ReportResult> {
  if (!Number.isInteger(id) || id <= 0) {
    return { ok: false, error: "invalid post id" };
  }
  const limited = await limitAction("report");
  if (limited) return limited;

  // website is a hidden field only bots fill in, pretend it worked
  if (typeof input?.website === "string" && input.website !== "") {
    return { ok: true };
  }

  const reason = input?.reason;
  if (!isReportReason(reason)) return { ok: false, error: "pick a reason" };
  const message =
    typeof input?.message === "string" ? input.message.trim() : "";
  if (message.length > maxReportMessage) {
    return { ok: false, error: "message is too long" };
  }
  const email = typeof input?.email === "string" ? input.email.trim() : "";
  if (email.length > maxReportEmail) {
    return { ok: false, error: "email is too long" };
  }

  try {
    await postApi.postIdReportPost({
      id,
      report: {
        reason,
        message: message || undefined,
        email: email || undefined,
      },
    });
    return { ok: true };
  } catch (error) {
    if (error instanceof ResponseError) {
      if (error.response.status === 429) {
        return { rateLimited: true, retryAfter: 60 };
      }
      if (error.response.status === 400) {
        try {
          const body = await error.response.json();
          if (body?.error) return { ok: false, error: body.error };
        } catch {}
      }
      if (error.response.status === 404) {
        return { ok: false, error: "this post no longer exists" };
      }
    }
    console.error("report post failed:", error);
    return { ok: false, error: "could not send the report, try again later" };
  }
}
