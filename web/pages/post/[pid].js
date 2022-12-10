import Image from "next/image";
import { baseURL } from "../../constants.ts";
import ImagePage from "../../components/ImagePage"


export async function getStaticPaths() {

    return {
        paths: [],
        fallback: "blocking" //indicates the type of fallback
    }
}

export async function getStaticProps({ params }) {
    const url = `${baseURL}/post/${params.pid}`
    const response = await fetch(url);
    const post = await response.json();
    return {
        props: {
            post,
        },
        revalidate: 10,
    };
}

export default function Post({ post }) {
    return ImagePage(post={post})
}

