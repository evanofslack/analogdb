import Template from "../components/template";
import { baseURL } from "../constants.ts";

export async function getStaticProps(context) {
    const url = baseURL + "/posts/random?page_size=50&nsfw=false";
    const response = await fetch(url);
    const data = await response.json();
    return {
        props: {
            data,
        },
        revalidate: 10,
    };
}

export default function Random({ data }) {
    return <Template data={data}></Template>;
}
