from .score import compute_metrics, field_stats, norm, pair, single


def label(post_id, stratum="s", unmatched=None, **fields):
    base = {
        "post_id": post_id,
        "stratum": stratum,
        "camera_make": None,
        "camera_model": None,
        "film_make": None,
        "film_type": None,
        "film_speed": None,
        "focal_length": None,
        "aperture": None,
        "unmatched": unmatched or [],
    }
    return {**base, **fields}


def pred(unmatched=None, **fields):
    return {"fields": fields, "unmatched": unmatched or [], "flags": []}


GOLD = [
    label(
        1,
        camera_make="nikon",
        camera_model="f3",
        film_make="kodak",
        film_type="portra 400",
        film_speed=400,
    ),
    label(2, camera_make="contax", unmatched=[{"kind": "camera", "raw": "Contax G2"}]),
    label(3, "neg"),
    label(4, "neg", camera_make="canon", camera_model="ae-1", aperture="f/2.8"),
]
PREDS = {
    1: pred(
        camera_make="Nikon",
        camera_model="F3",
        film_make="kodak",
        film_type="portra 160",
        film_speed=400,
    ),
    2: pred(
        camera_make="contax",
        camera_model="g1",
        unmatched=[{"kind": "camera", "raw": "G2", "key": "contaxg2"}],
    ),
    3: pred(camera_make="leica", camera_model="m6"),
    4: pred(aperture="2.8"),
}


def test_camera_pair():
    m = compute_metrics(GOLD, PREDS, with_unmatched=True)["fields"]["camera"]
    assert m["gold"] == 2
    assert m["set"] == 3
    assert m["correct"] == 1
    assert m["wrong"] == 1  # g1 for a Contax G2 is wrong, not spurious
    assert m["spurious"] == 1  # leica m6 on a post with no camera
    assert m["precision"] == 1 / 3
    assert m["recall"] == 1 / 2


def test_film_and_speed():
    m = compute_metrics(GOLD, PREDS, with_unmatched=False)["fields"]
    assert (m["film"]["correct"], m["film"]["wrong"]) == (0, 1)
    assert (m["film_speed"]["correct"], m["film_speed"]["wrong"]) == (1, 0)
    assert m["aperture"]["correct"] == 1


def test_unmatched_recall_and_strata():
    m = compute_metrics(GOLD, PREDS, with_unmatched=True)
    assert m["unmatched"] == {"gold": 1, "hits": 1, "recall": 1.0}
    assert m["strata"]["neg"]["posts"] == 2
    assert m["strata"]["s"]["fields"]["camera"]["correct"] == 1
    assert compute_metrics(GOLD, PREDS, with_unmatched=False)["unmatched"] is None


def test_missing_prediction_counts_as_unset():
    m = compute_metrics(GOLD, {}, with_unmatched=False)["fields"]["camera_make"]
    assert (m["set"], m["gold"], m["recall"], m["precision"]) == (0, 3, 0.0, None)


def test_norm():
    assert norm("aperture", "f/2.8") == norm("aperture", "2.8") == 2.8
    assert norm("film_speed", "400") == 400
    assert norm("camera_make", " Nikon ") == "nikon"
    assert norm("camera_make", "") is None


def test_field_stats_single():
    s = field_stats([({"x": "a"}, {"x": "a"}), ({"x": None}, {"x": "b"})], single("x"))
    assert (s["correct"], s["spurious"], s["wrong"]) == (1, 1, 0)
    assert pair("a", "b")({"a": "x", "b": None}) is None
