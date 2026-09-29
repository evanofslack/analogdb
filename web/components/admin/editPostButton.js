"use client";

import { loadPost } from "@app/actions/admin";
import { Button } from "@mantine/core";
import { useRouter } from "next/navigation";
import { useState } from "react";
import PostEditor from "./postEditor";

export default function EditPostButton({ id }) {
  const router = useRouter();
  const [post, setPost] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  async function open() {
    setLoading(true);
    setError(null);
    const result = await loadPost(id);
    setLoading(false);
    if (result.ok) {
      setPost(result.post);
    } else {
      setError(result.error);
    }
  }

  return (
    <>
      <Button
        size="xs"
        variant="default"
        onClick={open}
        loading={loading}
        color={error ? "red" : undefined}
        title={error ?? undefined}
      >
        {error ? "Retry" : "Edit"}
      </Button>
      {post && (
        <PostEditor
          post={post}
          opened={!!post}
          onClose={() => setPost(null)}
          onSaved={() => router.refresh()}
        />
      )}
    </>
  );
}
