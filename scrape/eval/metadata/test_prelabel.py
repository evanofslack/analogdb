from .prelabel import clean_draft, parse_labels
from .test_sample import CATALOG


def test_parse_labels():
    assert parse_labels('```json\n{"labels": [{"post_id": 1}, 3]}\n```') == [
        {"post_id": 1}
    ]
    assert parse_labels("") == []


def test_clean_draft_keeps_catalog_values_only():
    d = clean_draft(
        {
            "camera_make": "Nikon",
            "camera_model": "N80",
            "film_make": "kodak",
            "film_type": "portra 400",
            "film_speed": 800,
            "focal_length": "50",
            "aperture": "f/2",
            "unmatched": [{"kind": "lens", "raw": "Nikkor"}],
        },
        CATALOG,
    )
    assert d["camera_make"] == "nikon"
    assert d["camera_model"] is None
    assert d["unmatched"] == [{"kind": "camera", "raw": "nikon n80"}]
    assert (d["film_type"], d["film_speed"], d["focal_length"]) == (
        "portra 400",
        400,
        50,
    )


def test_clean_draft_unknown_make():
    d = clean_draft(
        {
            "camera_make": "Plaubel",
            "camera_model": "Makina 67",
            "unmatched": [{"kind": "camera", "raw": "Plaubel Makina 67"}],
        },
        CATALOG,
    )
    assert d["camera_make"] is None
    assert d["unmatched"] == [{"kind": "camera", "raw": "Plaubel Makina 67"}]
