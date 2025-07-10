import { logoutAction } from "@lib/auth";
import styles from "./adminPanel.module.css";
import Header from "@components/header";
import Footer from "@components/footer";

export default function AdminPanel() {
  return (
    <div className={styles.main}>
      <Header />
      <div className={styles.container}>
        <div className={styles.wrapper}>
          <div className={styles.header}>
            <h1 className={styles.title}>Admin Panel</h1>

            <form action={logoutAction}>
              <button type="submit" className={styles.logoutButton}>
                Logout
              </button>
            </form>
          </div>

          <div className={styles.content}>
            <div className={styles.card}>
              <div className={styles.cardHeader}>
                <h3 className={styles.cardTitle}>Welcome to Admin Dashboard</h3>
                <p className={styles.cardDescription}>
                  You are logged in as an administrator.
                </p>
              </div>

              <div className={styles.cardContent}>
                <div className={styles.grid}>
                  <div className={styles.gridItem}>
                    <h4 className={styles.itemTitle}>Posts Management</h4>
                    <p className={styles.itemDescription}>
                      You can now see delete buttons on all posts.
                    </p>
                  </div>

                  <div className={styles.gridItem}>
                    <h4 className={styles.itemTitle}>Site Administration</h4>
                    <p className={styles.itemDescription}>
                      Access to administrative features.
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <Footer />
    </div>
  );
}
