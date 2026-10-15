"""Extract the palette of every thumbnail in a folder and write them as JSON.

Each file's stem is taken as the post id. The output maps ids to colors:
    {"<id>": [{"hex": "#rrggbb", "percent": 0.42}, ...]}

Usage:
    uv run python scripts/dump_palettes.py <thumb_dir> <out.json>
"""

import argparse
import json
import os
from multiprocessing import Pool
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple

from PIL import Image
from scrape.image import extract_palette

SUFFIXES = {".jpg", ".jpeg", ".png"}


def palette(path: Path) -> Tuple[str, Optional[List[Dict[str, Any]]]]:
    try:
        with Image.open(path) as image:
            colors = extract_palette(image)
    except Exception as e:
        print(f"skipping {path.name}: {e}")
        return path.stem, None
    return path.stem, [{"hex": h, "percent": p} for h, p in colors]


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("thumb_dir", type=Path)
    parser.add_argument("out", type=Path)
    parser.add_argument("--workers", type=int, default=os.cpu_count())
    args = parser.parse_args()

    paths = sorted(p for p in args.thumb_dir.iterdir() if p.suffix.lower() in SUFFIXES)
    with Pool(args.workers) as pool:
        results = pool.imap(palette, paths, chunksize=16)
        palettes = {i: colors for i, colors in results if colors is not None}

    args.out.parent.mkdir(parents=True, exist_ok=True)
    with open(args.out, "w") as f:
        f.write(json.dumps(palettes, indent=2) + "\n")
    print(f"wrote {len(palettes)} of {len(paths)} palettes to {args.out}")


if __name__ == "__main__":
    main()
