export type MasonryPost = {
  images?: { width?: number; height?: number }[];
};

export type MasonryItem<T> = { post: T; index: number };

const gap = 15;

const columnWidth: Record<number, number> = { 2: 170, 3: 340, 4: 440 };

export function ratio(post: MasonryPost): number {
  const image = post.images?.[0];
  if (!image?.width || !image?.height) return 1;
  return image.height / image.width;
}

function shortest(heights: number[]): number {
  return heights.indexOf(Math.min(...heights));
}

// Pages are placed in order and each page only adds to the columns, so
// appending a page never moves items that are already placed
export function toColumns<T extends MasonryPost>(
  pages: T[][],
  count: number
): MasonryItem<T>[][] {
  const columns: MasonryItem<T>[][] = Array.from({ length: count }, () => []);
  const heights: number[] = Array(count).fill(0);
  const gapRatio = gap / (columnWidth[count] ?? columnWidth[4]);
  const tail = 2 * count;

  const place = (item: MasonryItem<T>, height: number) => {
    const column = shortest(heights);
    columns[column].push(item);
    heights[column] += height + gapRatio;
  };

  let index = 0;
  for (const page of pages) {
    const items = page.map((post) => ({
      item: { post, index: index++ },
      height: ratio(post),
    }));
    const split = Math.max(0, items.length - tail);
    items.slice(0, split).forEach(({ item, height }) => place(item, height));
    items
      .slice(split)
      .sort((a, b) => b.height - a.height)
      .forEach(({ item, height }) => place(item, height));
  }
  return columns;
}
