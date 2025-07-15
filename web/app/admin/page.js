import AdminPanel from '@components/admin/adminPanel';
import LoginForm from '@components/admin/loginForm';
import { checkAdminAuth } from '@lib/auth';

export default async function AdminPage({ searchParams }) {
  const isAdmin = await checkAdminAuth();

  if (!isAdmin) {
    return <LoginForm error={await searchParams?.error} />;
  }

  return <AdminPanel isAdmin={isAdmin} />;
}
