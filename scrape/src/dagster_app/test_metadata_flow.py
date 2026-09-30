import hashlib
from unittest.mock import MagicMock

import analogdb.models as adb
import dagster as dg
from scrape.metadata import EXTRACTOR_VERSION
from scrape.models import MatchResult, PhotoMetadata, RedditComment, UnmatchedMention

from .catalog import MatchingCatalog
from .metadata_flow import (
    extract_and_write,
    extraction_record,
    metadata_post,
    op_comments,
    patch_groups,
    plan_patch,
    unmatched_report,
    write_results,
)


def post(id=1, **fields):
    base = dict(
        id=id,
        title="t",
        author="u/me",
        permalink=f"/r/analog/{id}",
        score=1,
        timestamp=0,
        nsfw=False,
        grayscale=False,
        sprocket=False,
        images=[],
    )
    return adb.Post(**{**base, **fields})


def comment(author, body, time):
    return RedditComment(body=body, score=1, author=author, time=time, permalink="p")


def result(raw=None, unmatched=(), flags=(), **fields):
    return MatchResult(
        proposed=PhotoMetadata(**fields),
        unmatched=list(unmatched),
        flags=list(flags),
        raw=raw if raw is not None else {"cameras": [], "films": [], "lenses": []},
    )


CAMERAS_JSON = [
    {"make": "nikon", "model": "f3", "description": "d"},
    {"make": "nikon", "model": "fm", "description": "d"},
    {"make": "nikon", "model": "f501", "aliases": ["n2020"], "description": "d"},
]
LIVE_CAMERAS = [
    adb.Camera(id=1, make="nikon", model="f3", description="d"),
    adb.Camera(id=2, make="nikon", model="f501", description="d"),
]


CAMERA = {"cameras": [{"text": "Nikon F3"}], "films": [], "lenses": []}
FILM = {"cameras": [], "films": [{"text": "Portra"}], "lenses": []}


class TestComments:
    def test_op_comments(self):
        comments = [
            comment("u/me", "second", 2),
            comment("u/other", "not mine", 1),
            comment("u/me", "first  one\nline", 1),
            comment("u/me", "[deleted]", 3),
        ]
        assert op_comments(comments, "me") == ["first one line", "second"]
        long = [comment("u/me", "x" * 1000, i) for i in range(3)]
        assert sum(len(c) for c in op_comments(long, "u/me")) == 1500

    def test_metadata_post(self):
        p = post(title="Dusk", description="desc", author="me")
        m = metadata_post(p, [comment("u/me", "shot on my F3", 1)])
        assert (m.title, m.description, m.op_comments) == (
            "Dusk",
            "desc",
            ["shot on my F3"],
        )


class TestPlanPatch:
    def test_camera_mention_decides_both_fields(self):
        current = post(camera_make="contax", camera_model="g1")
        patch = plan_patch(current, result(CAMERA, camera_make="contax"))
        assert patch.to_dict() == {"clear": ["camera_model"]}
        assert patch_groups(patch) == ["camera"]

    def test_camera_replaced(self):
        current = post(camera_make="canon", camera_model="ae-1")
        patch = plan_patch(
            current, result(CAMERA, camera_make="nikon", camera_model="f3")
        )
        assert patch.to_dict() == {"camera_make": "nikon", "camera_model": "f3"}

    def test_no_mention_keeps_old(self):
        current = post(camera_make="nikon", camera_model="f3", film_make="kodak")
        assert plan_patch(current, result()) is None

    def test_film_group_clears_speed(self):
        current = post(film_make="kodak", film_type="portra 400", film_speed=400)
        patch = plan_patch(current, result(FILM, film_make="kodak"))
        assert patch.to_dict() == {"clear": ["film_type", "film_speed"]}

    def test_lens_set_but_never_cleared(self):
        current = post(focal_length=50, aperture="f/2")
        assert plan_patch(current, result(focal_length=35)).to_dict() == {
            "focal_length": 35
        }
        assert plan_patch(current, result()) is None

    def test_same_values_and_empty_strings_are_unchanged(self):
        current = post(camera_make="Nikon", camera_model="f3", film_make="")
        assert (
            plan_patch(current, result(CAMERA, camera_make="nikon", camera_model="f3"))
            is None
        )


def test_extraction_record():
    r = result(
        CAMERA,
        unmatched=[
            UnmatchedMention(kind="camera", raw="Nikon FM", key="nikonfm", make="nikon")
        ],
    )
    record = extraction_record(7, r, "model-x", "title: t")
    assert record.post_id == 7
    assert record.extractor_version == EXTRACTOR_VERSION
    assert record.model == "model-x"
    assert record.input_hash == hashlib.sha256(b"title: t").hexdigest()
    assert record.raw == CAMERA
    assert record.unmatched == [
        {"kind": "camera", "raw": "Nikon FM", "key": "nikonfm", "make": "nikon"}
    ]


def test_unmatched_report():
    def rec(post_id, *mentions):
        return adb.PostExtraction(
            post_id=post_id,
            extractor_version="v",
            model="m",
            input_hash="h",
            unmatched=[{"kind": k, "raw": raw, "key": key} for k, raw, key in mentions],
        )

    records = [
        rec(1, ("camera", "Nikon FM", "nikonfm")),
        rec(
            2,
            ("camera", "nikon fm", "nikonfm"),
            ("film", "Kodacolor 200", "kodacolor200"),
        ),
        rec(3, ("camera", "Nikon FM", "nikonfm")),
    ]
    report = unmatched_report(records)
    lines = report.splitlines()
    assert lines[2] == "| camera | Nikon FM | 3 | 3, 2, 1 |"
    assert lines[3] == "| film | Kodacolor 200 | 1 | 2 |"
    assert unmatched_report([]) == "No unmatched mentions."


class TestWrite:
    def test_patches_stores_and_skips_llm_failures(self):
        client = MagicMock()
        client.upsert_extractions.return_value = (2, [])
        posts = [post(1, camera_make="canon"), post(2), post(3)]
        results = [
            result(CAMERA, camera_make="nikon"),
            result(),
            result(flags=["llm_failed"]),
        ]
        stats = write_results(
            dg.build_asset_context(), client, posts, results, ["a", "b", "c"], "m"
        )
        client.patch_post.assert_called_once()
        assert client.patch_post.call_args.args[0] == 1
        stored = client.upsert_extractions.call_args.args[0]
        assert [r.post_id for r in stored] == [1, 2]
        assert (stats.patched, stats.unchanged, stats.llm_failed, stats.stored) == (
            1,
            1,
            1,
            2,
        )
        assert stats.groups["camera"] == 1
        assert stats.failures() == 0

    def test_counts_failures(self):
        client = MagicMock()
        client.patch_post.side_effect = RuntimeError("boom")
        client.upsert_extractions.side_effect = RuntimeError("boom")
        stats = write_results(
            dg.build_asset_context(),
            client,
            [post(1)],
            [result(CAMERA, camera_make="nikon")],
            ["a"],
            "m",
        )
        assert (stats.patch_failed, stats.store_failed) == (1, 1)
        assert stats.failures() == 2

    def test_extract_and_write_uses_comments_and_catalog(self):
        client = MagicMock()
        client.upsert_extractions.return_value = (1, [])
        extractor = MagicMock()
        extractor.extract.return_value = [result()]
        catalog = MatchingCatalog(cameras=["c"], films=["f"], aliases=["a"])
        p = post(1, title="Dusk", author="me")
        extract_and_write(
            dg.build_asset_context(),
            client,
            extractor,
            catalog,
            "m",
            [p],
            {1: [comment("u/me", "Nikon F3", 1)]},
        )
        inputs, cameras, films, aliases = extractor.extract.call_args.args
        assert inputs[0].op_comments == ["Nikon F3"]
        assert (cameras, films, aliases) == (["c"], ["f"], ["a"])
        record = client.upsert_extractions.call_args.args[0][0]
        assert record.input == "title: Dusk\nop comments: Nikon F3"
