"use server";

import {
  adminCookieName,
  checkCredentials,
  clientIp,
  consumeLoginAttempt,
  createSessionToken,
  resetLoginAttempts,
  sessionMaxAge,
} from "@lib/auth";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

const cookieOptions = {
  httpOnly: true,
  secure: true,
  sameSite: "strict" as const,
  path: "/",
};

export async function loginAction(formData: FormData): Promise<void> {
  const ip = await clientIp();
  if (!consumeLoginAttempt(ip)) {
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

  resetLoginAttempts(ip);
  (await cookies()).set(adminCookieName, token, {
    ...cookieOptions,
    maxAge: sessionMaxAge,
  });
  redirect("/admin");
}

export async function logoutAction(): Promise<void> {
  (await cookies()).set(adminCookieName, "", {
    ...cookieOptions,
    maxAge: 0,
  });
  redirect("/admin");
}
