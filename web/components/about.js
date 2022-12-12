import styles from "./about.module.css";
import Image from "next/image";
import Link from "next/link";
import { useEffect, useState } from "react";
import Footer from "./footer";
import { useBreakpoint } from "../providers/breakpoint.js";
import { baseURL } from "../constants.ts";

export default function About() {
    const [post, setPost] = useState();
    const [loaded, setIsLoaded] = useState(false);
    const postIDs = [3070, 3059, 2930, 2226, 1997, 1912, 1810, 1775, 1668, 1421, 1262, 359];
    const random = Math.floor(Math.random() * postIDs.length);
    const endpoint = baseURL + "/post/" + postIDs[random];

    const breakpoints = useBreakpoint();
    let isMobile = false;
    if (breakpoints["sm"]) {
        isMobile = true;
    }

    useEffect(() => {
        fetch(endpoint)
            .then((response) => response.json())
            .then((resp) => {
                setPost(resp);
                setIsLoaded(true);
                console.log(post);
            });
    }, []);

    return (
        <main>
            <div className={styles.sectionOne}>
                <div className={styles.subSection}>
                    <div className={styles.title}>Film for all</div>
                    <p className={styles.subtitle}>
                        AnalogDB is a curated database featuring thousands of film photographs. And
                        it is always growing, with new pictures added every day.
                    </p>
                    <Link href="/" className={styles.link}>
                        view latest
                    </Link>
                </div>
                {loaded && !isMobile && (
                    <div className={styles.imageOne}>
                        <Image
                            src={post.images[2].url}
                            width={post.images[2].width}
                            height={post.images[2].height}
                            alt={`Image ${post.id} by ${post.author}`}
                            quality={100}
                            placeholder="blur"
                            blurDataURL={post.images[0].url} // low res image
                        />
                        {/* <p>{postIDs[random]}</p> */}
                    </div>
                )}
            </div>

            <div className={styles.sectionTwo}>
                <div className={styles.imageTwo}>
                    {!isMobile && (
                        <Image
                            src={"/analogdb_curl.png"}
                            alt={`example AnalogDB API call`}
                            width="1064"
                            height="1224"
                            quality={100}
                            className={styles.imageTwoBorder}
                        />
                    )}
                </div>
                <div>
                    <div className={styles.title}>Accesible API</div>
                    <p className={styles.subtitle}>
                        The entire collection of film is exposed through a simple and intuitive API.
                        Embedded any of our photos in your next project with ease.
                    </p>
                    <Link href="/api" className={styles.link}>
                        read the docs
                    </Link>
                </div>
            </div>

            <div className={styles.sectionThree}>
                <div>
                    <div className={styles.title}>Free and open-source</div>
                    <p className={styles.subtitle}>
                        All code made publically avaliable on Github with flexible licensing.
                    </p>
                    <a className={styles.link} href="https://github.com/evanofslack/analogdb">
                        view source
                    </a>
                </div>
                <div className={styles.imageThree}>
                    {!isMobile && (
                        <Image
                            src={"/github_logo.png"}
                            alt={`example AnalogDB API call`}
                            width="3840"
                            height="2160"
                            quality={100}
                        />
                    )}
                </div>
            </div>
            <Footer />
        </main>
    );
}
