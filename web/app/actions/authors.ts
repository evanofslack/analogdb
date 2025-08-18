import { authorized_fetch } from "@lib/client";

interface AuthorsResponse {
  authors: string[];
}

export async function getAuthorsTotalCount(): Promise<number> {
  try {
    const authorsRoute = "/authors";
    const authorsResponse = await authorized_fetch(authorsRoute, "GET");
    const authorsData: AuthorsResponse = await authorsResponse.json();
    return Array.from(new Set(authorsData.authors)).length;
  } catch (error) {
    console.error("get posts total count request failed:", error);
    throw error;
  }
}
