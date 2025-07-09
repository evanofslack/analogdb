import { checkAdminAuth } from "@lib/auth";
import LoginForm from "@components/admin/loginForm";
import AdminPanel from "@components/admin/adminPanel";

export default async function AdminPage({ searchParams }) {
  const isAdmin = await checkAdminAuth();

  if (!isAdmin) {
    return <LoginForm error={await searchParams?.error} />;
  }

  return <AdminPanel />;
}
