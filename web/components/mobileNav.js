import { AiOutlineMenu } from "react-icons/ai";
import { useState } from "react";
import styles from "./mobileNav.module.css";
import { useRouter } from "next/router";
import Link from "next/link";
import { AiOutlineTrophy } from "react-icons/ai";
import { BiTimeFive } from "react-icons/bi";
import { BiShuffle } from "react-icons/bi";
import { RiCameraLensFill } from "react-icons/ri";
import { BiCheck } from "react-icons/bi";
import { GrClose } from "react-icons/gr";
import { FiGithub } from "react-icons/fi";

export default function MobileNav() {
    const router = useRouter();
    const [isOpen, setIsOpen] = useState(false);
    const toggle = () => setIsOpen((value) => !value);

    return (
        <div>
            <AiOutlineMenu size="1.8rem" onClick={toggle} />
            {isOpen && (
                <div className={styles.blur}>
                    <div className={styles.headerContainer}>
                        <div className={styles.close}>
                            <GrClose size="1.5rem" onClick={toggle} />
                        </div>

                        <nav className={styles.navContainer}>
                            <Link href="/">
                                <a className={styles.link}>
                                    <div className={styles.icon}>
                                        <AiOutlineTrophy size="1.8rem" />
                                        <div className={styles.check}>
                                            <p className={styles.iconText}>latest</p>
                                            {router.pathname == "/" && <BiCheck size="1.8rem" />}
                                        </div>
                                    </div>
                                </a>
                            </Link>
                            <Link href="/top">
                                <a className={styles.link}>
                                    <div className={styles.icon}>
                                        <BiTimeFive size="1.8rem" />
                                        <div className={styles.check}>
                                            <p className={styles.iconText}>top</p>
                                            {router.pathname == "/top" && <BiCheck size="1.8rem" />}
                                        </div>
                                    </div>
                                </a>
                            </Link>
                            <Link href="/random">
                                <a className={styles.link}>
                                    <div className={styles.icon}>
                                        <BiShuffle size="1.8rem" />
                                        <div className={styles.check}>
                                            <p className={styles.iconText}>random</p>
                                            {router.pathname == "/random" && (
                                                <BiCheck size="1.8rem" />
                                            )}
                                        </div>
                                    </div>
                                </a>
                            </Link>
                            <Link href="/bw">
                                <a className={styles.link}>
                                    <div className={styles.icon}>
                                        <RiCameraLensFill size="1.8rem" />
                                        <div className={styles.check}>
                                            <p className={styles.iconText}>b&w</p>
                                            {router.pathname == "/bw" && <BiCheck size="1.8rem" />}
                                        </div>
                                    </div>
                                </a>
                            </Link>
                        </nav>
                        <div className={styles.footer}>
                            <p> &copy; 2022 analogdb </p>
                            <a href="https://github.com/evanofslack/analog-gallery">
                                <FiGithub size="1.2rem" />
                            </a>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
