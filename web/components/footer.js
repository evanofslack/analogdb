import { FiGithub } from "react-icons/fi";
import styles from "./footer.module.css";

export default function Footer() {
    return (
        <footer className={styles.footer}>
            <p> &copy; 2025 AnalogDB </p>
            <a href="https://github.com/evanofslack/analogdb">
                <FiGithub size="18px" />
            </a>
        </footer>
    );
}
