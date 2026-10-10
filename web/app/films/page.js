import CatalogIndex from "@components/catalogIndex";
import { groupFilms } from "@lib/catalog";
import { getCatalog, indexJsonLd } from "@lib/catalogPage";
import { jsonLd } from "@lib/seo";
import { connection } from "next/server";

const title = "Film stocks";
const description =
  "Film photographs grouped by film stock, from Kodak Portra to Ilford HP5, with a description of each stock.";

export const metadata = {
  title,
  description,
  alternates: { canonical: "/films" },
  openGraph: { siteName: "AnalogDB", title, description, url: "/films" },
};

export default async function Page() {
  await connection();
  const list = await getCatalog("films");

  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: jsonLd(indexJsonLd("films", title, description, list)),
        }}
      />
      <CatalogIndex
        kind="films"
        title="FILM STOCKS"
        intro="See how every film stock looks in real photos"
        placeholder="filter film stocks..."
        groups={groupFilms(list)}
      />
    </>
  );
}
