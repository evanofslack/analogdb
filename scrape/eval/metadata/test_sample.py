from .sample import (
    assign_strata,
    is_negative,
    names_uncatalogued_camera,
    op_comments,
    stratum_rules,
)

CATALOG = {
    "cameras": [
        {"id": 1, "make": "nikon", "model": "f3"},
        {"id": 2, "make": "mamiya", "model": "645"},
        {"id": 3, "make": "mamiya", "model": "645 super"},
        {"id": 4, "make": "kodak", "model": "retina"},
    ],
    "films": [
        {
            "id": 1,
            "make": "kodak",
            "type": "portra 400",
            "speed": 400,
            "color_type": "color",
        },
        {
            "id": 2,
            "make": "ilford",
            "type": "hp5 plus",
            "speed": 400,
            "color_type": "bw",
        },
    ],
}
RULES = stratum_rules(CATALOG)


def post(id, title, timestamp=1_600_000_000, **fields):
    return {"id": id, "title": title, "timestamp": timestamp, **fields}


def test_negative():
    brands = {"nikon", "portra"}
    assert is_negative(post(1, "Sunset at the beach"), brands, {"hp5"})
    assert not is_negative(post(1, "Sunset [35mm]"), brands, set())
    assert not is_negative(post(1, "Sunset on Portra"), brands, set())
    assert not is_negative(post(1, "HP5+ at dusk"), brands, {"hp5"})
    assert not RULES["negative"](post(1, "Tri X 400 and Portra400"))
    assert RULES["negative"](post(1, "My first roll"))


def test_names_uncatalogued_camera():
    assert not RULES["not_in_catalog"](post(1, "Nikon F3 | Portra 400"))
    assert not RULES["not_in_catalog"](post(1, "Mamiya 645 Super, 80mm"))
    assert RULES["not_in_catalog"](post(1, "Nikon N80 / Provia"))
    assert RULES["not_in_catalog"](post(1, "Lubitel 166 in the snow"))
    assert not RULES["not_in_catalog"](post(1, "Kodak 400 film"))
    assert not RULES["not_in_catalog"](post(1, "Shot on my Nikon"))
    assert not names_uncatalogued_camera(post(1, "Nikon"), {"nikon": {"f3"}})


def test_assign_strata_counts_and_order():
    posts = [post(i, "Nikon N80 test", timestamp=1_600_000_000) for i in range(1, 6)]
    posts += [post(37500, "Nikon F3", timestamp=1_750_000_000)]
    posts += [
        post(38001 + i, "Nikon F3", timestamp=1_780_000_000, camera_make="nikon")
        for i in range(3)
    ]
    quotas = {
        "negative": 1,
        "not_in_catalog": 2,
        "ids_37k": 1,
        "ids_38k_meta": 2,
        "pre_2025": 2,
    }
    picked = assign_strata(posts, RULES, quotas, fewshot=1)
    counts = {}
    for p in picked:
        counts[p["stratum"]] = counts.get(p["stratum"], 0) + 1
    assert counts == {
        "not_in_catalog": 2,
        "pre_2025": 2,
        "fewshot": 1,
        "ids_37k": 1,
        "ids_38k_meta": 2,
    }
    assert [p["id"] for p in picked if p["stratum"] == "not_in_catalog"] == [1, 2]
    assert all(
        p["set"] == ("fewshot" if p["stratum"] == "fewshot" else "test") for p in picked
    )
    assert picked == assign_strata(posts, RULES, quotas, fewshot=1)


def test_op_comments():
    comments = [
        {"author": "u/me", "body": "second", "time": 2},
        {"author": "u/other", "body": "not mine", "time": 1},
        {"author": "u/me", "body": "first  line\nnext", "time": 1},
        {"author": "u/me", "body": "[deleted]", "time": 3},
    ]
    assert op_comments(comments, "me") == ["first line next", "second"]
    long = [{"author": "u/me", "body": "x" * 1000, "time": i} for i in range(3)]
    assert sum(len(c) for c in op_comments(long, "u/me")) == 1500
