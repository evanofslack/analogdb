import json
from unittest.mock import MagicMock, patch

import analogdb.models as adb
import dagster as dg
import pytest
from scrape.catalog_match import CatalogMatcher
from scrape.models import MetadataPost

from .assets import upload_cameras, upload_films
from .catalog import (
    camera_upload_plan,
    catalog_for_matching,
    film_upload_plan,
    retired_entries,
    validate_aliases,
)
from .constants import CAMERAS_PATH, FILMS_PATH
from .resources import AnalogDBResource, CamerasJsonResource, FilmsJsonResource


def camera(id, make, model, description="d"):
    return adb.Camera(id=id, make=make, model=model, description=description)


def film(id, make, type, speed, description="d", color_type="color"):
    return adb.Film(
        id=id,
        make=make,
        type=type,
        speed=speed,
        color_type=color_type,
        description=description,
    )


CAMERA_ENTRIES = [
    {"make": "nikon", "model": "f501", "aliases": ["n2020"], "description": "d"},
    {"make": "nikon", "model": "fm", "description": "d"},
    {"make": "argus", "model": "c4", "description": "d"},
]
FILM_ENTRIES = [
    {
        "make": "kodak",
        "type": "colorplus 200",
        "speed": 200,
        "aliases": ["kodacolor 200"],
        "color_type": "color",
        "description": "d",
    },
]
LIVE_CAMERAS = [
    camera(1, "nikon", "f501"),
    camera(2, "nikon", "n2020"),
    camera(3, "nikon", "fm"),
    camera(4, "pentax", "k1000"),
]
LIVE_FILMS = [film(1, "kodak", "colorplus 200", 200)]


class TestCatalogForMatching:
    def test_live_entries_and_aliases(self):
        c = catalog_for_matching(CAMERA_ENTRIES, FILM_ENTRIES, LIVE_CAMERAS, LIVE_FILMS)
        assert [(x.make, x.model) for x in c.cameras] == [
            ("nikon", "f501"),
            ("nikon", "fm"),
            ("pentax", "k1000"),
        ]
        assert c.not_live == ["argus c4"]
        assert c.not_in_json == ["pentax k1000"]
        assert [(a.kind, a.name, a.alias) for a in c.aliases] == [
            ("camera", "f501", "n2020"),
            ("film", "colorplus 200", "kodacolor 200"),
        ]

    def test_alias_of_entry_not_live_is_skipped(self):
        c = catalog_for_matching(CAMERA_ENTRIES, FILM_ENTRIES, LIVE_CAMERAS[2:], [])
        assert c.aliases == []

    def test_matcher_uses_aliases(self):
        c = catalog_for_matching(CAMERA_ENTRIES, FILM_ENTRIES, LIVE_CAMERAS, LIVE_FILMS)
        m = CatalogMatcher(c.cameras, c.films, c.aliases)
        title = "Nikon N2020, Kodacolor 200"
        r = m.match(
            {
                "cameras": [{"text": "Nikon N2020", "make": "Nikon", "model": "N2020"}],
                "films": [
                    {
                        "text": "Kodacolor 200",
                        "make": None,
                        "type": "Kodacolor 200",
                        "box_speed": 200,
                    }
                ],
            },
            MetadataPost(title=title),
        )
        p = r.proposed
        assert (p.camera_make, p.camera_model, p.film_make, p.film_type) == (
            "nikon",
            "f501",
            "kodak",
            "colorplus 200",
        )


class TestRetiredEntries:
    def test_live_aliases_are_retired(self):
        live_films = LIVE_FILMS + [film(2, "Kodak", "Kodacolor 200", 200)]
        retired = retired_entries(
            CAMERA_ENTRIES, FILM_ENTRIES, LIVE_CAMERAS, live_films
        )
        assert [(r.kind, r.id, r.make, r.name) for r in retired] == [
            ("camera", 2, "nikon", "n2020"),
            ("film", 2, "kodak", "kodacolor 200"),
        ]
        assert retired[0].target["model"] == "f501"
        assert retired[1].target["type"] == "colorplus 200"

    def test_target_not_live_is_not_retired(self):
        live = [camera(2, "nikon", "n2020"), camera(3, "nikon", "fm")]
        assert retired_entries(CAMERA_ENTRIES, [], live, []) == []

    def test_entries_and_unknown_rows_are_not_retired(self):
        live = [camera(1, "nikon", "f501"), camera(4, "pentax", "k1000")]
        assert retired_entries(CAMERA_ENTRIES, FILM_ENTRIES, live, LIVE_FILMS) == []


class TestValidateAliases:
    def test_real_catalog_json(self):
        with open(CAMERAS_PATH) as f:
            cameras = json.load(f)
        with open(FILMS_PATH) as f:
            films = json.load(f)
        assert validate_aliases(cameras, films) == []

    def test_problems(self):
        cameras = [
            {"make": "nikon", "model": "f501", "aliases": ["n2020", "fm", "f501"]},
            {"make": "nikon", "model": "fm", "aliases": ["N-2020", ""]},
        ]
        problems = validate_aliases(cameras, [])
        assert len(problems) == 4
        assert any("'fm' is the name of nikon fm" in p for p in problems)
        assert any("also used by nikon f501" in p for p in problems)
        assert any("empty alias" in p for p in problems)


class TestUploadPlan:
    def test_cameras(self):
        entries = [
            {"make": "Nikon", "model": "F501", "description": "d"},
            {"make": "nikon", "model": "fm", "description": "new text"},
            {"make": "argus", "model": "c4", "description": "d"},
            {"make": "argus", "model": "c4", "description": "dupe"},
        ]
        plan = camera_upload_plan(entries, LIVE_CAMERAS)
        assert [e["model"] for e in plan.new] == ["c4"]
        assert [e["model"] for e in plan.updated] == ["fm"]
        assert plan.unchanged == 1
        assert plan.not_in_json == ["nikon n2020", "pentax k1000"]

    def test_films_key_includes_speed(self):
        entries = [
            {
                "make": "kodak",
                "type": "colorplus 200",
                "speed": 200,
                "color_type": "bw",
                "description": "d",
            },
            {
                "make": "kodak",
                "type": "colorplus 200",
                "speed": 400,
                "color_type": "color",
                "description": "d",
            },
        ]
        plan = film_upload_plan(entries, LIVE_FILMS)
        assert [e["speed"] for e in plan.updated] == [200]
        assert [e["speed"] for e in plan.new] == [400]
        assert plan.unchanged == 0


def run_upload(asset, resource_cls, entries, client):
    context = dg.build_asset_context()
    with (
        patch.object(resource_cls, "client", return_value=entries),
        patch.object(AnalogDBResource, "client", return_value=client),
    ):
        return asset(context, resource_cls(file_path="unused"), AnalogDBResource())


class TestUploadAssets:
    def test_cameras_sends_only_new_and_changed(self):
        client = MagicMock()
        client.get_cameras.return_value = LIVE_CAMERAS
        entries = [
            {
                "make": "nikon",
                "model": "f501",
                "aliases": ["n2020"],
                "description": "d",
            },
            {"make": "nikon", "model": "fm", "description": "changed"},
            {"make": "argus", "model": "c4", "description": "d"},
        ]
        result = run_upload(upload_cameras, CamerasJsonResource, entries, client)
        sent = [c.args[0].model for c in client.upload_camera.call_args_list]
        assert sent == ["c4", "fm"]
        assert result.metadata["new"] == 1
        assert result.metadata["updated"] == 1
        assert result.metadata["unchanged"] == 1

    def test_films_nothing_to_send(self):
        client = MagicMock()
        client.get_films.return_value = LIVE_FILMS
        entries = [
            {
                "make": "kodak",
                "type": "colorplus 200",
                "speed": 200,
                "color_type": "color",
                "description": "d",
            }
        ]
        result = run_upload(upload_films, FilmsJsonResource, entries, client)
        client.upload_film.assert_not_called()
        assert result.metadata["unchanged"] == 1

    def test_failure_raises(self):
        client = MagicMock()
        client.get_cameras.return_value = []
        client.upload_camera.side_effect = RuntimeError("boom")
        entries = [{"make": "argus", "model": "c4", "description": "d"}]
        with pytest.raises(dg.Failure):
            run_upload(upload_cameras, CamerasJsonResource, entries, client)
