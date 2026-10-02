import CatalogIndex from "@components/catalogIndex";
import { groupCameras } from "@lib/catalog";
import { getCatalog, indexJsonLd } from "@lib/catalogPage";
import { jsonLd } from "@lib/seo";

export const revalidate = 3600;

const title = "Cameras";
const description =
  "Film photographs grouped by the camera they were shot on, from Canon and Nikon SLRs to Hasselblad and Leica.";

export const metadata = {
  title,
  description,
  alternates: { canonical: "/cameras" },
  openGraph: { siteName: "AnalogDB", title, description, url: "/cameras" },
};

export default async function Page() {
  const list = await getCatalog("cameras");

  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: jsonLd(indexJsonLd("cameras", title, description, list)),
        }}
      />
      <CatalogIndex
        kind="cameras"
        title="CAMERAS"
        intro="See what people are shooting with every camera."
        placeholder="filter cameras..."
        groups={groupCameras(list)}
      />
    </>
  );
}
