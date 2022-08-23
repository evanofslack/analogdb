import Template from "../components/template";

export async function getStaticProps(context) {
    const url = "https://analogdb.herokuapp.com/top?page_size=50&nsfw=false";
    const response = await fetch(url);
    const data = await response.json();
    return {
        props: {
            data,
        },
        revalidate: 10,
    };
}

export default function Top({ data }) {
    return <Template data={data}></Template>;
}
