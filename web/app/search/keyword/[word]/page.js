import { getKeyword } from "@app/actions/keywords";
import CatalogDetail from "@components/catalogDetail";
import { loadKeywordDetail } from "@lib/catalogPage";
import {
  keywordLabel,
  keywordPageHref,
  keywordSearchHref,
} from "@lib/keywords";
import { notFound } from "next/navigation";

const robots = { index: false, follow: true };

function decodeWord(word) {
  try {
    return decodeURIComponent(word);
  } catch {
    return word;
  }
}

export async function generateMetadata({ params }) {
  const { word } = await params;
  const keyword = await getKeyword(decodeWord(word));
  if (!keyword) notFound();
  const { entry } = keyword;
  const title = `${entry.make} photos`;
  const description = `${entry.postCount.toLocaleString(
    "en-US"
  )} analog photographs tagged ${entry.make}, from the AnalogDB archive.`;
  const url = keywordPageHref(entry.make);
  const images = entry.cover
    ? [
        {
          url: entry.cover.url,
          width: entry.cover.width,
          height: entry.cover.height,
          alt: title,
        },
      ]
    : undefined;
  return {
    title,
    description,
    robots,
    alternates: { canonical: url },
    openGraph: {
      siteName: "AnalogDB",
      type: "website",
      title,
      description,
      url,
      images,
    },
    twitter: { card: "summary_large_image", title, description, images },
  };
}

export default async function Page({ params, searchParams }) {
  const { word } = await params;
  const keyword = await getKeyword(decodeWord(word));
  if (!keyword) notFound();

  const { entry, related } = keyword;
  const detail = await loadKeywordDetail(entry, await searchParams);

  return (
    <CatalogDetail
      entry={entry}
      {...detail}
      title={keywordLabel(entry.make)}
      titleLink={{
        href: keywordSearchHref(entry.make),
        label: `search for ${entry.make}`,
      }}
      relatedTitle="related keywords"
      relatedLinks={related.map((other) => ({
        href: keywordPageHref(other.word),
        label: other.word,
      }))}
    />
  );
}
