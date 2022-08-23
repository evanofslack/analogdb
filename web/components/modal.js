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
    let fullImage = post.images[3];

    if (breakpoints["xs"]) {
        image = post.images[1];
        fullImage = post.images[3];
    }
    // } else if (breakpoints["lg"]) {
    //     image = post.images[3];
    //     fullImage = post.images[3];
    // }
    const [isOpen, setIsOpen] = useState(false);
    const [isLoaded, setIsLoaded] = useState(false);
    const toggle = () => setIsOpen((value) => !value);

    useDisableScroll(isOpen);

    return (
        <div>
            <Image
                src={image.url}
                width={image.width}
                height={image.height}
                alt={`Image ${post.id} by ${post.author}`}
                quality={100}
                layout="responsive"
                placeholder="blur"
                blurDataURL={props.post.images[0].url} // low res image
                onClick={toggle}
            />
            {isOpen && (
                <div className={styles.modal}>
                    <ImageTag post={post}></ImageTag>
                    <div className={styles.imageContainer} onClick={toggle}>
                        {/* Preview picture of lower resolution */}
                        <Image
                            style={isLoaded ? { display: "none" } : {}}
                            src={image.url}
                            width={image.width}
                            height={image.height}
                            alt={`Image ${post.id} by ${post.author}`}
                            quality={100}
                            layout="fill"
                            objectFit="contain"
                        />
                        {/* Replace with full resolution picture when loaded */}
                        <Image
                            style={isLoaded ? {} : { display: "none" }}
                            src={fullImage.url}
                            width={fullImage.width}
                            height={fullImage.height}
                            alt={`Image ${post.id} by ${post.author}`}
                            quality={100}
                            layout="fill"
                            objectFit="contain"
                            onLoad={() => setIsLoaded(true)}
                        />
                    </div>
                </div>
            )}
        </div>
    );
}
