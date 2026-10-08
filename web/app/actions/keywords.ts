"use server";

import { CatalogEntry } from "@lib/catalog";
import * as data from "@lib/data/keywords";

export async function getKeywordCatalog(): Promise<CatalogEntry[]> {
  return data.getKeywordCatalog();
}
