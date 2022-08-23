import Template from "../components/template";

export async function getStaticProps(context) {
    const url = "https://analogdb.herokuapp.com/latest?page_size=50&nsfw=true";
    const response = await fetch(url);
    const data = await response.json();
    return {
        props: {
            data,
        },
        revalidate: 10,
    };
}

export default function Nsfw({ data }) {
    return <Template data={data}></Template>;
}
