import { baseURL } from "@lib/constants";
import { NextResponse } from "next/server";

const imageHost = "d3i73ktnzbi69i.cloudfront.net";

const extensions = {
  "image/jpeg": "jpg",
  "image/png": "png",
  "image/gif": "gif",
};

export async function GET(request, { params }) {
  const { pid } = await params;
  const id = Number(pid);

  if (!Number.isInteger(id) || id <= 0) {
    return NextResponse.json({ error: "Invalid post id" }, { status: 400 });
  }

  try {
    const postResponse = await fetch(`${baseURL}/post/${id}`);
    if (postResponse.status === 404) {
      return NextResponse.json({ error: "Post not found" }, { status: 404 });
    }
    if (!postResponse.ok) {
      throw new Error(`Failed to fetch post: ${postResponse.status}`);
    }

    const post = await postResponse.json();
    const imageUrl = new URL(post.images[3].url);
    if (imageUrl.protocol !== "https:" || imageUrl.hostname !== imageHost) {
      throw new Error("Unexpected image host");
    }

    const upstream = await fetch(imageUrl, { redirect: "error" });
    if (!upstream.ok || !upstream.body) {
      throw new Error(`Failed to fetch image: ${upstream.status}`);
    }

    const contentType = (upstream.headers.get("content-type") || "")
      .split(";")[0]
      .trim()
      .toLowerCase();
    const ext = extensions[contentType] || "jpg";

    const headers = {
      "Content-Type": extensions[contentType]
        ? contentType
        : "application/octet-stream",
      "Content-Disposition": `attachment; filename="analogdb-${id}.${ext}"`,
      "Cache-Control": "public, max-age=86400",
    };
    const contentLength = upstream.headers.get("content-length");
    if (contentLength) headers["Content-Length"] = contentLength;

    return new Response(upstream.body, { headers });
  } catch (error) {
    console.error("Download error:", error);
    return NextResponse.json({ error: "Download failed" }, { status: 502 });
  }
}
