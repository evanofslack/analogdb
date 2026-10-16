import "server-only";
import { errorStatus, statsApi } from "@lib/client";
import { Stats } from "@lib/stats";
import { unstable_cache } from "next/cache";

const getStatsCached = unstable_cache(
  async (): Promise<Stats | null> => {
    try {
      return await statsApi.statsGet();
    } catch (error) {
      // no snapshot built yet
      if (errorStatus(error) === 404) return null;
      throw error;
    }
  },
  ["stats"],
  { revalidate: 3600, tags: ["stats"] }
);

export async function getStats(): Promise<Stats | null> {
  try {
    return await getStatsCached();
  } catch (error) {
    console.error("get stats request failed:", error);
    return null;
  }
}
