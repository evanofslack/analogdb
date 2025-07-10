import { loginAction } from "@lib/auth";
import styles from "./loginForm.module.css";
import Header from "@components/header";
import Footer from "@components/footer";

export default function LoginForm({ error }) {
  return (
    <div className={styles.main}>
      <Header />
      <div className={styles.container}>
        <div className={styles.formWrapper}>
          <div className={styles.header}>
            <h2 className={styles.title}>Admin Login</h2>
          </div>

          <form className={styles.form} action={loginAction}>
            <div className={styles.inputGroup}>
              <label htmlFor="username" className={styles.label}>
                Username
              </label>
              <input
                id="username"
                name="username"
                type="username"
                required
                className={styles.input}
                placeholder="admin username"
              />
            </div>
            <div className={styles.inputGroup}>
              <label htmlFor="password" className={styles.label}>
                Password
              </label>
              <input
                id="password"
                name="password"
                type="password"
                required
                className={styles.input}
                placeholder="admin password"
              />
            </div>

            {error === "invalid" && (
              <div className={styles.error}>Invalid username or password</div>
            )}

            <div className={styles.buttonGroup}>
              <button type="submit" className={styles.submitButton}>
                Sign In
              </button>
            </div>
          </form>
        </div>
      </div>
      <Footer />
    </div>
  );
}
