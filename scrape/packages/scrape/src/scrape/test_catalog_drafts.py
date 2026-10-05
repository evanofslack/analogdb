import json

import pytest

from .catalog_drafts import (
    CAMERA_WIKI,
    WIKIPEDIA,
    Source,
    apply_drafts,
    candidates,
    draft_prompt,
    find_source,
    focus_text,
    html_text,
    parse_draft,
    pick_examples,
    review_markdown,
    title_fits,
)

CAMERAS = [
    {"make": "mamiya", "model": "645", "description": "The Mamiya 645 is a camera."},
    {"make": "mamiya", "model": "7", "description": "The Mamiya 7 is a camera."},
    {"make": "nikon", "model": "fm2", "description": "The Nikon FM2 is a camera."},
    {"make": "pentax", "model": "645", "description": "The Pentax 645 is a camera."},
    {"make": "pentax", "model": "67", "description": "The Pentax 67 is a camera."},
]
FILMS = [
    {
        "type": "portra 400",
        "make": "kodak",
        "speed": 400,
        "color_type": "color",
        "description": "Kodak Portra 400 is a film.",
    },
]


def extraction(post_id, *unmatched):
    return {"post_id": post_id, "unmatched": list(unmatched)}


def camera_mention(key, raw, make, model):
    return {"kind": "camera", "key": key, "raw": raw, "make": make, "model": model}


def test_candidates_rank_by_posts_with_nearest_names():
    n645 = camera_mention("pentax645n", "Pentax 645N", "pentax", "645n")
    gold = {
        "kind": "film",
        "key": "kodakgold400",
        "raw": "Gold 400",
        "make": "kodak",
        "type": "gold 400",
        "speed": 400,
    }
    rows = [
        extraction(1, n645),
        extraction(2, n645, gold),
        extraction(3, {**n645, "raw": "645n"}),
        extraction(4, gold),
        extraction(5, camera_mention("leicam6", "M6", "leica", "m6")),
    ]
    found = candidates(rows, CAMERAS, FILMS, min_posts=2)
    assert [(c.kind, c.key, c.posts) for c in found] == [
        ("camera", "pentax645n", 3),
        ("film", "kodakgold400", 2),
    ]
    first = found[0]
    assert first.spellings[0] == "Pentax 645N"
    assert first.examples == [3, 2, 1]
    assert "pentax 645" in first.nearest
    assert found[1].speed == 400


def test_html_text_drops_scripts_edit_links_and_footnotes():
    page = (
        '<div><h2>History<span class="mw-editsection">[<a>edit</a>]</span></h2>'
        "<p>The 645N<sup>[1]</sup> came out in 1997.<br/>It has autofocus.</p>"
        "<script>var x = 1;</script><ul><li>120 film</li></ul></div>"
    )
    assert html_text(page) == (
        "History\nThe 645N came out in 1997.\nIt has autofocus.\n120 film"
    )


@pytest.mark.parametrize(
    "title,kind,name,speed,expected",
    [
        ("Pentax 645N", "camera", "645n", None, True),
        ("Pentax 645", "camera", "645n", None, False),
        ("Mamiya M645 1000S", "camera", "645 1000s", None, True),
        ("Fujifilm Superia", "film", "superia 200", 200, True),
        ("Kodak Portra", "film", "portra 400nc", 400, False),
        ("Kodak Gold", "film", "gold 400", 400, True),
    ],
)
def test_title_fits(title, kind, name, speed, expected):
    assert title_fits(title, kind, name, speed) is expected


@pytest.mark.parametrize(
    "title,make,name,expected",
    [
        ("Fujica GS645S", "fujifilm", "gs645s", True),
        ("501(c) organization", "hasselblad", "501c", False),
        ("Fujitsu ARROWS A 201F", "hasselblad", "201f", False),
        ("Rolleiflex Hy6", "sinar", "f", False),
        ("Bessa R", "voigtländer", "bessa r", True),
    ],
)
def test_title_fits_needs_the_make(title, make, name, expected):
    assert title_fits(title, "camera", name, None, make) is expected


class FakeWikis:
    def __init__(self, camera_wiki=None, wikipedia=None):
        self.pages = {CAMERA_WIKI: camera_wiki or {}, WIKIPEDIA: wikipedia or {}}
        self.calls = []

    def __call__(self, url, params):
        self.calls.append((url, dict(params)))
        pages = self.pages[url]
        if params.get("list") == "search":
            word = params["srsearch"].split()[0].lower()
            hits = [{"title": t} for t in pages if word in t.lower()]
            return {"query": {"search": hits}}
        text = pages.get(params["page"])
        if text is None:
            return {"error": {"code": "missingtitle"}}
        return {"parse": {"title": params["page"], "text": text}}


def test_find_source_prefers_camera_wiki_page_about_the_model():
    fetch = FakeWikis(
        camera_wiki={
            "Pentax 645": "<p>The 645.</p>",
            "Pentax 645N": "<p>The 645N.</p>",
        },
        wikipedia={"Pentax 645N": "Wikipedia text."},
    )
    source = find_source({"kind": "camera", "make": "pentax", "model": "645n"}, fetch)
    assert source.site == "camera-wiki"
    assert source.title == "Pentax 645N"
    assert source.text == "The 645N."
    assert source.url == "https://camera-wiki.org/wiki/Pentax_645N"


def test_find_source_falls_back_to_wikipedia_and_takes_a_pin():
    fetch = FakeWikis(wikipedia={"Kodak Gold": "Kodak Gold is a film."})
    item = {"kind": "film", "make": "kodak", "type": "gold 400", "speed": 400}
    assert find_source(item, fetch).site == "wikipedia"

    pinned = {**item, "source": "wikipedia:Kodak Gold"}
    fetch.calls.clear()
    assert find_source(pinned, fetch).title == "Kodak Gold"
    assert [c[1].get("list") for c in fetch.calls] == [None]

    missing = {"kind": "camera", "make": "kiev", "model": "60"}
    assert find_source(missing, FakeWikis()) is None


def test_pick_examples_same_make_first():
    item = {"kind": "camera", "make": "pentax", "model": "645n"}
    picks = pick_examples(item, CAMERAS)
    assert [p["model"] for p in picks[:2]] == ["645", "67"]
    assert len(picks) == 3 and picks[2]["make"] != "pentax"
    assert pick_examples(item, CAMERAS) == picks


def test_draft_prompt_has_examples_source_and_film_fields():
    item = {"kind": "film", "make": "kodak", "type": "gold 400", "speed": 400}
    source = Source("wikipedia", "Kodak Gold", "https://w/Kodak_Gold", "Gold facts.")
    system, user = draft_prompt(item, source, FILMS)
    assert "at most 60 words" in system and "color_type" in system
    assert "Entry: kodak gold 400" in user and "ISO 400" in user
    assert "Kodak Portra 400 is a film." in user and "Gold facts." in user


def words(n):
    return " ".join(["word"] * n)


def test_parse_draft():
    reply = json.dumps({"description": words(60), "color_type": "Color"})
    assert parse_draft(f"```json\n{reply}\n```", "film") == {
        "description": words(60),
        "color_type": "color",
    }
    assert parse_draft(json.dumps({"description": words(130)}), "camera") == {
        "description": words(130)
    }
    for bad, kind in [
        ("not json", "camera"),
        ("[]", "camera"),
        (json.dumps({"description": ""}), "camera"),
        (json.dumps({"description": words(400)}), "camera"),
        (json.dumps({"description": words(60), "color_type": "slide"}), "film"),
    ]:
        with pytest.raises(ValueError):
            parse_draft(bad, kind)


def test_apply_drafts_inserts_by_make_and_adds_aliases():
    items = [
        {
            "kind": "camera",
            "action": "new",
            "make": "pentax",
            "model": "645n",
            "aliases": ["645 n"],
            "description": "The Pentax 645N.",
        },
        {
            "kind": "camera",
            "action": "new",
            "make": "plaubel",
            "model": "makina 67",
            "description": "The Plaubel Makina 67.",
        },
        {
            "kind": "film",
            "action": "new",
            "make": "kodak",
            "type": "gold 400",
            "speed": 400,
            "color_type": "color",
            "description": "Kodak Gold 400.",
        },
        {
            "kind": "camera",
            "action": "alias",
            "make": "nikon",
            "target": "fm2",
            "aliases": ["fm2n"],
        },
        {"kind": "camera", "action": "skip", "make": "kodak", "model": "400"},
    ]
    cameras, films = apply_drafts(CAMERAS, FILMS, items)
    assert [(c["make"], c["model"]) for c in cameras] == [
        ("mamiya", "645"),
        ("mamiya", "7"),
        ("nikon", "fm2"),
        ("pentax", "645"),
        ("pentax", "67"),
        ("pentax", "645n"),
        ("plaubel", "makina 67"),
    ]
    assert list(cameras[5]) == ["make", "model", "aliases", "description"]
    assert cameras[2] == {**CAMERAS[2], "aliases": ["fm2n"]}
    assert list(cameras[2]) == ["make", "model", "aliases", "description"]
    assert list(films[1]) == ["type", "make", "speed", "color_type", "description"]
    assert CAMERAS[2] == {
        "make": "nikon",
        "model": "fm2",
        "description": "The Nikon FM2 is a camera.",
    }


@pytest.mark.parametrize(
    "item",
    [
        {"kind": "camera", "action": "new", "make": "pentax", "model": "645"},
        {
            "kind": "camera",
            "action": "new",
            "make": "pentax",
            "model": "645",
            "description": "Again.",
        },
        {
            "kind": "camera",
            "action": "alias",
            "make": "nikon",
            "target": "f9",
            "aliases": ["x"],
        },
    ],
)
def test_apply_drafts_rejects(item):
    with pytest.raises(ValueError):
        apply_drafts(CAMERAS, FILMS, [item])


def test_review_markdown_flags_missing_sources():
    md = review_markdown(
        [
            {
                "kind": "camera",
                "action": "new",
                "make": "kiev",
                "model": "60",
                "description": "The Kiev 60.",
            },
            {
                "kind": "film",
                "action": "new",
                "make": "kodak",
                "type": "gold 400",
                "speed": 400,
                "color_type": "color",
                "source_title": "Kodak Gold",
                "source_url": "https://w/Kodak_Gold",
                "description": "Kodak Gold 400.",
            },
        ]
    )
    assert "## camera: kiev 60\nSource: **no source**" in md
    assert "Source: [Kodak Gold](https://w/Kodak_Gold)" in md
    assert "ISO 400, color" in md


def test_find_source_searches_a_distinctive_model_on_its_own():
    fetch = FakeWikis(camera_wiki={"Fujica GS645S": "<p>The GS645S.</p>"})
    item = {"kind": "camera", "make": "fujifilm", "model": "gs645s"}
    assert find_source(item, fetch).title == "Fujica GS645S"
    short = {"kind": "camera", "make": "fujifilm", "model": "tx"}
    assert find_source(short, FakeWikis(camera_wiki={"TX": "<p>x</p>"})) is None


def test_focus_text_keeps_the_start_and_passages_about_the_entry():
    rows = [f"Film {i}\nKodak\n{1980 + i}\nnotes {i}" for i in range(400)]
    text = (
        "Intro to the list.\n"
        + "\n".join(rows[:200])
        + "\nPortra 400NC\nKodak\n1998\n"
        + "\n".join(rows[200:])
    )
    out = focus_text(text, ["Portra 400NC"], limit=3000)
    assert out.startswith("Intro to the list.")
    assert "Portra 400NC\nKodak\n1998" in out
    assert len(out) <= 3000
    assert focus_text("short", ["x"]) == "short"
    assert focus_text(text, ["Not There"], limit=100) == text[:100]


def test_find_source_focuses_long_pinned_pages():
    page = (
        "<p>Intro.</p>" + "<p>filler</p>" * 2000 + "<p>Acros II: a 100 speed film.</p>"
    )
    fetch = FakeWikis(wikipedia={"Neopan": page})
    item = {
        "kind": "film",
        "make": "fujifilm",
        "type": "neopan 100 acros ii",
        "speed": 100,
        "source": "wikipedia:Neopan",
        "focus": ["Acros II"],
    }
    text = find_source(item, fetch).text
    assert text.startswith("Intro.") and "Acros II: a 100 speed film." in text
