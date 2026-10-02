import CatalogDetail from "@components/catalogDetail";
import {
  detailJsonLd,
  detailMetadata,
  getEntry,
  loadDetail,
} from "@lib/catalogPage";
import { jsonLd } from "@lib/seo";
import { notFound } from "next/navigation";

const kind = "films";

export async function generateMetadata({ params }) {
  const { slug } = await params;
  const { entry } = await getEntry(kind, slug);
  return detailMetadata(kind, entry);
}

export default async function Page({ params, searchParams }) {
  const { slug } = await params;
  const { list, entry } = await getEntry(kind, slug);
  if (!entry) notFound();

  const detail = await loadDetail(kind, entry, list, await searchParams);

  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: jsonLd(detailJsonLd(kind, entry)) }}
      />
      <CatalogDetail kind={kind} entry={entry} {...detail} />
    </>
  );
}
