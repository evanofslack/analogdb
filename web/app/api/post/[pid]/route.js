import { checkAdminAuth } from '@lib/auth';
import { authorized_fetch } from '@lib/cient';

export async function DELETE(request, { params }) {
  const isAdmin = await checkAdminAuth();

  if (!isAdmin) {
    return new Response('Unauthorized', { status: 401 });
  }

  try {
    const { pid } = await params;
    const route = `/post/${pid}`;
    const res = await authorized_fetch(route, 'DELETE');
    if (res.ok) {
      return new Response('Post deleted successfully', { status: 200 });
    } else {
      return new Response('Failed to delete post', { status: 500 });
    }
  } catch (error) {
    console.error('Error deleting post:', error);
    return new Response('Server error', { status: 500 });
  }
}
