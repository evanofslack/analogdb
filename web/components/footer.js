import { FiGithub } from "react-icons/fi";
import styles from "./footer.module.css";

export default function Footer({ children = null }) {
  return (
    <footer className={styles.footer}>
      {children && <p className={styles.about}>{children}</p>}
      <div className={styles.row}>
        <p> &copy; 2025 AnalogDB </p>
        <a href="https://github.com/evanofslack/analogdb">
          <FiGithub size="18px" />
        </a>
      </div>
    </footer>
  );
}
