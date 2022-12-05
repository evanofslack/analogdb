import styles from "./imageTag.module.css";
import { FaChevronDown, FaChevronUp } from "react-icons/fa";
import { useState } from "react";
import { baseURL } from "../constants.ts";

export default function ImageTag(props) {
    let post = props.post;
    const api_endpoint = baseURL + "/post/";
    const [isOpen, setIsOpen] = useState(false);
    const toggle = () => setIsOpen((value) => !value);

    return (
        <div className={styles.padding}>
            <div className={styles.container} onClick={toggle}>
                <p className={styles.title}>{post.title}</p>
                {!isOpen && (
                    <div className={styles.icon}>
                        <FaChevronDown />
                    </div>
                )}
                {isOpen && (
                    <div className={styles.icon}>
                        <FaChevronUp />
                    </div>
                )}
            </div>
            {isOpen && (
                <div className={styles.info}>
                    <a href={post.permalink}>
                        <p className={styles.title}>{post.author}</p>
                    </a>
                    <a href={post.images[3].url}>
                        <p className={styles.title}>
                            {post.images[3].width} x {post.images[3].height}
                        </p>
                    </a>
                    <a href={api_endpoint + post.id}>
                        <p className={styles.title}>#{post.id}</p>
                    </a>
                </div>
            )}
        </div>
    );
}
