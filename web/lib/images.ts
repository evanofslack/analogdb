type Img = {
  resolution?: string;
  url?: string;
  width?: number;
  height?: number;
};

function find<T extends Img>(images: T[], resolution: string, index: number) {
  return images.find((img) => img.resolution === resolution) ?? images[index];
}

// 1st gen low res is too small, use medium res
export function smallImage<T extends Img>(images: T[] = []): T | undefined {
  const low = find(images, "low", 0);
  const medium = find(images, "medium", 1);
  if (low && Math.max(low.width ?? 0, low.height ?? 0) >= 720) return low;
  return medium ?? low;
}
