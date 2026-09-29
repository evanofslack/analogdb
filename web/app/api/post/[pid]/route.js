import { removePost } from "@app/actions/admin";
import { checkAdminAuth } from "@lib/auth";

export async function DELETE(request, { params }) {
  const isAdmin = await checkAdminAuth();

  if (!isAdmin) {
    return new Response("Unauthorized", { status: 401 });
  }

  try {
    const { pid } = await params;
    const id = Number(pid);
    if (Number.isNaN(id)) {
      return new Response("Invalid post id", { status: 400 });
    }
    const result = await removePost(id);
    if (result.ok) {
      return new Response("Post deleted successfully", { status: 200 });
    } else {
      return new Response("Failed to delete post", { status: 500 });
    }
  } catch (error) {
    console.error("Error deleting post:", error);
    return new Response("Server error", { status: 500 });
  }
}
