"use server";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

export async function loginAction(formData) {
  const username = formData.get("username");
  const password = formData.get("password");

  if (
    username === process.env.ADMIN_USERNAME &&
    password === process.env.ADMIN_PASSWORD
  ) {
    cookies().set("admin-token", process.env.ADMIN_SECRET, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      maxAge: 60 * 60 * 24 * 7, // 1 week
      path: "/",
    });

    redirect("/admin");
  } else {
    // For simplicity, we'll redirect back with error in URL
    redirect("/admin?error=invalid");
  }
}

export async function logoutAction() {
  cookies().set("admin-token", "", {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    maxAge: 0,
    path: "/",
  });

  redirect("/admin");
}

export async function checkAdminAuth() {
  const cookieStore = await cookies();
  const adminToken = cookieStore.get("admin-token");
  return adminToken?.value === process.env.ADMIN_SECRET;
}
