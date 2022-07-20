import Template from "../components/template";

const baseURL = "https://analogdb.herokuapp.com";

export async function getServerSideProps(context) {
    const url = baseURL + "/latest?page_size=50&bw=false&nsfw=false";
    const response = await fetch(url);
    const data = await response.json();
    return {
        props: {
            data,
        },
    };
}

export default function Home({ data }) {
    return <Template data={data}></Template>;
}
