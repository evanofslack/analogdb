type Keyword = { word?: string };

export type SeoPost = {
  id?: number;
  title?: string;
  caption?: string;
  author?: string;
  camera_make?: string;
  camera_model?: string;
  film_make?: string;
  film_type?: string;
  cameraMake?: string;
  cameraModel?: string;
  filmMake?: string;
  filmType?: string;
  keywords?: Keyword[];
};

const maxDescription = 160;
const maxKeywords = 5;

function join(...parts: (string | undefined)[]): string | null {
  const name = parts.filter(Boolean).join(" ");
  return name || null;
}

export function authorName(post: SeoPost): string {
  return (post.author ?? "").replace("u/", "");
}

export function cameraName(post: SeoPost): string | null {
  return join(
    post.camera_make ?? post.cameraMake,
    post.camera_model ?? post.cameraModel
  );
}

export function filmName(post: SeoPost): string | null {
  return join(post.film_make ?? post.filmMake, post.film_type ?? post.filmType);
}

function shotOn(post: SeoPost): string {
  const camera = cameraName(post);
  const film = filmName(post);
  let text = "";
  if (camera) text += ` on ${camera}`;
  if (film) text += ` with ${film}`;
  return text;
}

export function postAlt(post: SeoPost): string {
  const caption = post.caption?.trim();
  if (caption) return caption;
  const title = post.title?.trim();
  if (!title) {
    const author = authorName(post);
    return author ? `photo by ${author}` : "photo";
  }
  const shot = shotOn(post);
  return shot ? `${title}, shot${shot}` : title;
}

export function truncate(text: string, max: number): string {
  if (text.length <= max) return text;
  const cut = text.slice(0, max - 1);
  const space = cut.lastIndexOf(" ");
  const words = space > 0 ? cut.slice(0, space) : cut;
  return `${words.replace(/[\s,.:;]+$/, "")}…`;
}

export function postDescription(post: SeoPost): string {
  const parts: string[] = [];
  const title = post.title?.trim();
  if (title) parts.push(/[.!?]$/.test(title) ? title : `${title}.`);

  const author = authorName(post);
  let shot = shotOn(post);
  if (author) shot += ` by ${author}`;
  if (shot) parts.push(`Shot${shot}.`);

  if (!cameraName(post) && !filmName(post)) {
    const words = (post.keywords ?? [])
      .map((keyword) => keyword.word)
      .filter(Boolean)
      .slice(0, maxKeywords);
    if (words.length > 0) parts.push(`Keywords: ${words.join(", ")}.`);
  }

  return truncate(parts.join(" "), maxDescription);
}

export function jsonLd(data: unknown): string {
  return JSON.stringify(data).replace(/</g, "\\u003c");
}
