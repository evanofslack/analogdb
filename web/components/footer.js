import { FiGithub } from "react-icons/fi";
import styles from "./footer.module.css";

export default function Footer() {
    return (
        <footer className={styles.footer}>
            <p> &copy; 2022 AnalogDB </p>
            <a href="https://github.com/evanofslack/analogdb">
                <FiGithub size="1.2rem" />
            </a>
        </footer>
    );
}