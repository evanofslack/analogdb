import { deletePost } from "@app/actions/posts";
import { checkAdminAuth } from "@lib/auth";

export async function DELETE(request, { params }) {
  const isAdmin = await checkAdminAuth();

  if (!isAdmin) {
    return new Response("Unauthorized", { status: 401 });
  }

  try {
    const { pid } = await params;
    const res = await deletePost(pid);
    if (res.ok) {
      return new Response("Post deleted successfully", { status: 200 });
    } else {
      return new Response("Failed to delete post", { status: 500 });
    }
  } catch (error) {
    console.error("Error deleting post:", error);
    return new Response("Server error", { status: 500 });
  }
}
