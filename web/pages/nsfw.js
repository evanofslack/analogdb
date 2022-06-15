import Template from "../components/template";

export async function getServerSideProps(context) {
    const url = "https://analogdb.herokuapp.com/nsfw?page_size=20";
    const response = await fetch(url);
    const data = await response.json();
    return {
        props: {
            data,
        },
    };
}

export default function Nsfw({ data }) {
    return <Template data={data}></Template>;
}
