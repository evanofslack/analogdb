'use client';

import { baseURL } from "../../../../constants.js";
import { useEffect } from "react";
import { useRouter } from "next/navigation";

async function deletePost(pid) {
  const url = `${baseURL}/post/${pid}`;
  const response = await fetch(url, {
    method: "DELETE",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Basic ${Buffer.from("user:pw").toString("base64")}`,
    },
  });
  console.log(response);
}

export default function DeletePage({ params }) {
  const router = useRouter();
  const { pid } = params;

  useEffect(() => {
    if (pid) {
      deletePost(pid);
      // Navigate back to homepage after deletion
      setTimeout(() => {
        router.push('/');
      }, 1000);
    }
  }, [pid, router]);
  
  return <div>Deleting post...</div>;
}
