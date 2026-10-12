"use client";

import { loadPost } from "@app/actions/admin";
import { EditablePost } from "@lib/adminClient";
import { Button } from "@mantine/core";
import { useRouter } from "next/navigation";
import { useState } from "react";
import PostEditor from "./postEditor";

interface EditPostButtonProps {
  id: number;
}

export default function EditPostButton({ id }: EditPostButtonProps) {
  const router = useRouter();
  const [post, setPost] = useState<EditablePost | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function open() {
    setLoading(true);
    setError(null);
    const result = await loadPost(id);
    setLoading(false);
    if (result.ok === true) {
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
