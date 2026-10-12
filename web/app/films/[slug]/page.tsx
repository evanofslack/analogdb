import CatalogDetail from "@components/catalogDetail";
import { CatalogKind } from "@lib/catalog";
import {
  detailJsonLd,
  detailMetadata,
  getEntry,
  loadDetail,
} from "@lib/catalogPage";
import { jsonLd } from "@lib/seo";
import { Metadata } from "next";
import { notFound } from "next/navigation";

const kind: CatalogKind = "films";

type PageProps = {
  params: Promise<{ slug: string }>;
  searchParams: Promise<Record<string, string | string[] | undefined>>;
};

export async function generateMetadata({
  params,
}: PageProps): Promise<Metadata> {
  const { slug } = await params;
  const { entry } = await getEntry(kind, slug);
  return detailMetadata(kind, entry);
}

export default async function Page({ params, searchParams }: PageProps) {
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
