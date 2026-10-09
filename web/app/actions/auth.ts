"use server";

import {
  adminCookieName,
  checkCredentials,
  createSessionToken,
  sessionMaxAge,
} from "@lib/auth";
import { clientIp, rateLimitKey } from "@lib/clientIp";
import { consume, reset } from "@lib/rateLimit";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

const cookieOptions = {
  httpOnly: true,
  secure: true,
  sameSite: "strict" as const,
  path: "/",
};

const adminHintName = "admin-hint";

const hintOptions = {
  httpOnly: false,
  secure: true,
  sameSite: "strict" as const,
  path: "/",
};

export async function loginAction(formData: FormData): Promise<void> {
  const key = rateLimitKey(await clientIp());
  if (!consume("login", key).ok) {
    redirect("/admin?error=invalid");
  }

  const username = formData.get("username");
  const password = formData.get("password");
  const valid =
    typeof username === "string" &&
    typeof password === "string" &&
    checkCredentials(username, password);
  const token = valid ? createSessionToken() : null;

  if (!token) {
    redirect("/admin?error=invalid");
  }

  reset("login", key);
  const cookieStore = await cookies();
  cookieStore.set(adminCookieName, token, {
    ...cookieOptions,
    maxAge: sessionMaxAge,
  });
  cookieStore.set(adminHintName, "1", {
    ...hintOptions,
    maxAge: sessionMaxAge,
  });
  redirect("/admin");
}

export async function logoutAction(): Promise<void> {
  const cookieStore = await cookies();
  cookieStore.set(adminCookieName, "", {
    ...cookieOptions,
    maxAge: 0,
  });
  cookieStore.set(adminHintName, "", {
    ...hintOptions,
    maxAge: 0,
  });
  redirect("/admin");
}
