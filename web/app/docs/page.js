import Documentation from '@components/documentation';
import styles from '@components/gallery.module.css';
import Header from '@components/header';
import { checkAdminAuth } from '@lib/auth';

export const metadata = {
  title: 'AnalogDB',
  description: 'Film photography database',
};

export default async function Docs() {
  const isAdmin = await checkAdminAuth();
  return (
    <div className={styles.container}>
      <Header isAdmin={isAdmin} />
      <Documentation />
    </div>
  );
}
