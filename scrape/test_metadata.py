"""
Standalone test for LLM metadata extraction.
Usage: uv run python test_metadata.py
Requires: OPENROUTER_API_KEY, OPENROUTER_BASE_URL, OPENROUTER_MODEL env vars (or defaults)
"""
import os

from analogdb.client import Client
from openai import OpenAI

from scrape.metadata import MetadataExtractor

TEST_TITLES = [
    # camera + film, typical format
    "title: Golden hour at the Golden Gate. 📸: Pentax MJU 67. 🎞️: Cinestill800T",
    # abbreviated camera name, film with speed embedded
    "title: Street shot [Canon AE1] [Kodak Portra 400]",
    # Nikon F/N alias, push/pull notation
    "title: Late night (Nikon F90) (Fuji 400H @ 800)",
    # film only, no camera
    "title: Morning light on HP5+ 400",
    # Hasselblad + Ektar, focal length and aperture present
    "title: Desert dunes | Hasselblad 500cm | 80mm | f/2.8 | Kodak Ektar 100",
    # no metadata at all — should return all nulls
    "title: Just a nice sunset today",
    # partial/ambiguous — camera make only, no model match expected
    "title: Shot this on my old Mamiya with some Tri X",
]

PASS = "\033[92mPASS\033[0m"
FAIL = "\033[91mFAIL\033[0m"


def check(label, value, expected):
    ok = value == expected
    status = PASS if ok else FAIL
    print(f"  [{status}] {label}: got={value!r} expected={expected!r}")
    return ok


def main():
    api = Client(base_url="https://api.analogdb.com")
    films = api.get_films()
    cameras = api.get_cameras()
    print(f"Loaded {len(films)} films, {len(cameras)} cameras\n")

    client = OpenAI(
        base_url=os.environ.get("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1"),
        api_key=os.environ["OPENROUTER_API_KEY"],
    )
    model = os.environ.get("OPENROUTER_MODEL", "google/gemini-2.5-flash")
    extractor = MetadataExtractor(client, model)
    print(f"Using model: {model}\n")

    metadatas, _ = extractor.extract(TEST_TITLES, films, cameras)

    cases = [
        # (title_snippet, metadata, field_checks: [(label, actual, expected), ...])
        (
            "Pentax MJU 67 + Cinestill800T",
            metadatas[0],
            [
                ("camera_make", metadatas[0].camera_make, "pentax"),
                ("film_make", metadatas[0].film_make, "cinestill"),
                ("film_type", metadatas[0].film_type, "800t"),
                ("film_speed", metadatas[0].film_speed, 800),
            ],
        ),
        (
            "Canon AE1 + Portra 400",
            metadatas[1],
            [
                ("camera_make", metadatas[1].camera_make, "canon"),
                ("film_make", metadatas[1].film_make, "kodak"),
                ("film_type", metadatas[1].film_type, "portra 400"),
                ("film_speed", metadatas[1].film_speed, 400),
            ],
        ),
        (
            "Nikon F90 (N90 alias) + Fuji 400H push",
            metadatas[2],
            [
                ("camera_make", metadatas[2].camera_make, "nikon"),
                ("film_speed", metadatas[2].film_speed, 400),  # base speed, not pushed
            ],
        ),
        (
            "HP5+ film only",
            metadatas[3],
            [
                ("camera_make", metadatas[3].camera_make, None),
                ("film_make", metadatas[3].film_make, "ilford"),
                ("film_type", metadatas[3].film_type, "hp5 plus"),
            ],
        ),
        (
            "Hasselblad 500cm + focal + aperture + Ektar",
            metadatas[4],
            [
                ("camera_make", metadatas[4].camera_make, "hasselblad"),
                ("camera_model", metadatas[4].camera_model, "500c/m"),
                ("focal_length", metadatas[4].focal_length, 80),
                ("aperture", metadatas[4].aperture, "f/2.8"),
                ("film_make", metadatas[4].film_make, "kodak"),
                ("film_type", metadatas[4].film_type, "ektar 100"),
            ],
        ),
        (
            "No metadata",
            metadatas[5],
            [
                ("camera_make", metadatas[5].camera_make, None),
                ("film_make", metadatas[5].film_make, None),
                ("film_speed", metadatas[5].film_speed, None),
            ],
        ),
        (
            "Mamiya make only + Tri-X",
            metadatas[6],
            [
                ("camera_make", metadatas[6].camera_make, "mamiya"),
                ("film_make", metadatas[6].film_make, "kodak"),
                ("film_type", metadatas[6].film_type, "tri-x 400"),
            ],
        ),
    ]

    total = 0
    passed = 0
    for title, _, checks in cases:
        print(f"{title}")
        for check_args in checks:
            total += 1
            if check(*check_args):
                passed += 1
        print()

    print(f"Results: {passed}/{total} checks passed")


if __name__ == "__main__":
    main()
