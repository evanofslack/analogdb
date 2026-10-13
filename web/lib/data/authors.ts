import "server-only";
import { authorsApi } from "@lib/client";

export async function getAuthorsTotalCount(): Promise<number> {
  try {
    const response = await authorsApi.authorsGet({
      next: { revalidate: 60 },
    } as RequestInit);
    return Array.from(new Set(response.authors)).length;
  } catch (error) {
    console.error("get posts total count request failed:", error);
    throw error;
  }
}
