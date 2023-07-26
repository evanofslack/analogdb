import { baseURL } from "../../../constants.js";
import { useEffect } from "react";
import { useRouter } from "next/router";

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

export default function DeletePost() {
  const router = useRouter();
  const { pid } = router.query;

  useEffect(() => {
    if (pid != "") {
      deletePost(pid);
    }
  }, [pid]);
}
