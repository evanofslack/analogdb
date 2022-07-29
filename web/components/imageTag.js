import styles from "./imageTag.module.css";
import { IoLogoReddit } from "react-icons/io5";
import { ImReddit } from "react-icons/im";
import { FaChevronDown, FaChevronUp } from "react-icons/fa";
import { useState } from "react";

export default function ImageTag(props) {
    let post = props.post;
    const base_user = "https://www.reddit.com/";
    const api_endpoint = "https://analogdb.herokuapp.com/post/";
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
                    <a href={base_user + post.author}>
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
                    <a href={post.permalink}>
                        {/* <IoLogoReddit size="1.2rem" title="View on Reddit" /> */}
                        <ImReddit size="1.2rem" title="View on Reddit" />
                    </a>
                </div>
            )}
        </div>
    );
}
