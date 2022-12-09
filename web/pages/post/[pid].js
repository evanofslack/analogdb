import { baseURL } from "../../constants.ts";


export async function getStaticPaths() {

    return {
        paths: [],
        fallback: "blocking" //indicates the type of fallback
    }
}

export async function getStaticProps({ params }) {
    const url = `${baseURL}/post/${params.pid}`
    const response = await fetch(url);
    const data = await response.json();
    return {
        props: {
            data,
        },
        revalidate: 10,
    };
}

export default function Post({ data }) {
	return <p> Post: {data.title}</p>;
}

