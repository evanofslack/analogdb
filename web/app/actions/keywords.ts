"use server";

import { CatalogEntry } from "@lib/catalog";
import * as data from "@lib/data/keywords";
import { limitAction } from "@lib/rateLimit";
import { Limited } from "@lib/rateLimited";

export async function getKeywordCatalog(): Promise<CatalogEntry[] | Limited> {
  return (await limitAction("browse")) ?? data.getKeywordCatalog();
}
