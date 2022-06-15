import React, { useState, useEffect } from "react";
import { BsChevronDoubleUp } from "react-icons/bs";
import styles from "./scrollTop.module.css";

const ScrollTop = () => {
    const [visible, setVisible] = useState(false);

    const toggleVisible = () => {
        const scrolled = document.documentElement.scrollTop;
        if (scrolled > 300) {
            setVisible(true);
        } else if (scrolled <= 300) {
            setVisible(false);
        }
    };

    const scrollToTop = () => {
        window.scrollTo({
            top: 0,
            behavior: "smooth",
        });
    };

    useEffect(() => {
        window.addEventListener("scroll", toggleVisible);
        return () => window.removeEventListener("scroll", toggleVisible);
    });

    return (
        <BsChevronDoubleUp
            size="3rem"
            onClick={scrollToTop}
            className={visible ? styles.visible : styles.hidden}
            title="Back to top"
        />
    );
};

export default ScrollTop;
