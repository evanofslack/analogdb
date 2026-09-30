from unittest.mock import MagicMock, patch

import dagster as dg
from scrape.models import MatchResult, PhotoMetadata, RedditComment, RedditPost

from .assets import patch_post_metadata, title_metadatas
from .resources import (
    AnalogDBResource,
    CamerasJsonResource,
    FilmsJsonResource,
    MetadataResource,
)
from .result import Result, Status
from .test_metadata_flow import CAMERAS_JSON, LIVE_CAMERAS, post


def match(**fields):
    return MatchResult(
        proposed=PhotoMetadata(**fields),
        raw={"cameras": [{"text": "Nikon F3"}], "films": [], "lenses": []},
    )


def patched(client, extractor):
    return (
        patch.object(AnalogDBResource, "client", return_value=client),
        patch.object(MetadataResource, "client", return_value=extractor),
        patch.object(CamerasJsonResource, "client", return_value=CAMERAS_JSON),
        patch.object(FilmsJsonResource, "client", return_value=[]),
    )


def catalog_client():
    client = MagicMock()
    client.get_cameras.return_value = LIVE_CAMERAS
    client.get_films.return_value = []
    client.upsert_extractions.return_value = (1, [])
    return client


def resources():
    return dict(
        analogdb=AnalogDBResource(),
        metadata=MetadataResource(openai_model="model-x"),
        cameras_json=CamerasJsonResource(file_path="unused"),
        films_json=FilmsJsonResource(file_path="unused"),
    )


def test_title_metadatas_uses_title_description_and_catalog():
    reddit_post = RedditPost(
        image_url="u",
        subreddit="analog",
        title="Dusk [Nikon F3]",
        selftext="desc",
        author="u/me",
        permalink="p1",
        score=1,
        nsfw=False,
        time=0,
        sprocket=False,
    )
    reddit_posts = Result(data={"p1": reddit_post}, status={"p1": Status.SUCCESS})
    extractor = MagicMock()
    extractor.extract.return_value = [match(camera_make="nikon", camera_model="f3")]
    a, b, c, d = patched(catalog_client(), extractor)
    with a, b, c, d:
        result = title_metadatas(
            dg.build_asset_context(), reddit_posts=reddit_posts, **resources()
        )
    inputs, cameras, films, aliases = extractor.extract.call_args.args
    assert (inputs[0].title, inputs[0].description, inputs[0].op_comments) == (
        "Dusk [Nikon F3]",
        "desc",
        [],
    )
    assert [c.model for c in cameras] == ["f3", "f501"]
    assert [a.alias for a in aliases] == ["n2020"]
    assert result.data["p1"].camera_model == "f3"


def test_patch_post_metadata_day_two():
    client = catalog_client()
    extractor = MagicMock()
    extractor.extract.return_value = [match(camera_make="nikon", camera_model="f3")]
    comments = [
        RedditComment(
            body="shot on my F3", score=1, author="u/me", time=1, permalink="p"
        )
    ]
    a, b, c, d = patched(client, extractor)
    with a, b, c, d:
        result = patch_post_metadata(
            dg.build_asset_context(partition_key="2026-09-28"),
            updated_reddit_comments=[(post(1, author="me"), comments)],
            **resources(),
        )
    inputs = extractor.extract.call_args.args[0]
    assert inputs[0].op_comments == ["shot on my F3"]
    assert client.patch_post.call_args.args[0] == 1
    record = client.upsert_extractions.call_args.args[0][0]
    assert (record.post_id, record.model) == (1, "model-x")
    assert result.metadata["patched"] == 1


def test_patch_post_metadata_no_posts():
    result = patch_post_metadata(
        dg.build_asset_context(partition_key="2026-09-28"),
        updated_reddit_comments=[],
        **resources(),
    )
    assert result.metadata == {"posts": 0}
