import styles from "./imageTag.module.css";
import { FaChevronDown, FaChevronUp } from "react-icons/fa";
import { useState } from "react";
import { baseURL } from "../constants.ts";

export default function ImageTag(props) {
    let post = props.post;
    const api_endpoint = baseURL + "/post/";

    return (
        <div className={styles.padding}>
            <div className={styles.container} >
                <a href={api_endpoint + post.id}>
                    <p className={styles.id}>#{post.id}</p>
                </a>
                <a href={post.permalink}>
                    <p className={styles.title}>{post.title}</p>
                </a>
            </div>
        </div>
    );
}
