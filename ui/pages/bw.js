import Template from "../components/template";

export async function getServerSideProps(context) {
    const url = "https://analogdb.herokuapp.com/bw?page_size=20&nsfw=true";
    const response = await fetch(url);
    const data = await response.json();
    return {
        props: {
            data,
        },
    };
}

export default function Bw({ data }) {
    return <Template data={data}></Template>;
}
