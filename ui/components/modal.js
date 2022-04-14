import Image from "next/image";
import { useState } from "react";
import styles from "./modal.module.css";
import ImageTag from "./imageTag";
import useDisableScroll from "../hooks/useDisableScroll";
import { useBreakpoint } from "../providers/breakpoint.js";

export default function Modal(props) {
    const post = props.post;
    const breakpoints = useBreakpoint();

    let image = post.images[2];
    let fullImage = post.images[2];
    if (breakpoints["xs"]) {
        image = post.images[0];
        fullImage = post.images[0];
    } else if (breakpoints["sm"]) {
        image = post.images[1];
        fullImage = post.images[1];
    } else if (breakpoints["md"]) {
        image = post.images[1];
        fullImage = post.images[2];
    } else if (breakpoints["lg"]) {
        image = post.images[2];
        fullImage = post.images[2];
    }
    const [isOpen, setIsOpen] = useState(false);
    const toggle = () => setIsOpen((value) => !value);

    useDisableScroll(isOpen);

    return (
        <div>
            <Image
                src={image.url}
                width={image.width}
                height={image.height}
                alt={`Image ${post.id} by ${post.author}`}
                quality={80}
                layout="responsive"
                placeholder="blur"
                blurDataURL={props.post.images[0].url} // low res image
                onClick={toggle}
            />
            {isOpen && (
                <div className={styles.modal}>
                    <ImageTag post={post}></ImageTag>
                    <div className={styles.imageContainer} onClick={toggle}>
                        <Image
                            src={fullImage.url}
                            width={fullImage.width}
                            height={fullImage.height}
                            alt={`Image ${post.id} by ${post.author}`}
                            quality={100}
                            layout="fill"
                            objectFit="contain"
                            // placeholder="blur"
                            // blurDataURL={props.post.images[0].url} // low res image
                        />
                    </div>
                </div>
            )}
        </div>
    );
}
