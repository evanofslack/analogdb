import { getPost } from "@lib/data/posts";
import { notFound } from "next/navigation";
import { ReactNode } from "react";

type LayoutProps = {
  children: ReactNode;
  params: Promise<{ pid: string }>;
};

// The missing post check lives here, outside loading.tsx's Suspense boundary,
// so notFound runs before the page streams and the response is a real 404
export default async function PostLayout({ children, params }: LayoutProps) {
  const { pid } = await params;
  if (!(await getPost(pid))) notFound();
  return children;
}
