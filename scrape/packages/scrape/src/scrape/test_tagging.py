import json
from pathlib import Path
from types import SimpleNamespace

import pytest

from .models import Keyword, S3Image
from .tagging import (
    MAX_TAGS,
    TAGGER_VERSION,
    ImageTagger,
    TaggingError,
    TagInput,
    catalog_words,
    load_stoplist,
    medium_url,
    named_in,
    normalize_tags,
    parse_tags,
    post_text,
    tag_keywords,
    with_monochrome,
)
from .test_metadata import FakeCompletions, bad_request

STOPLIST = Path(__file__).parents[4] / "data" / "tag_stoplist.txt"

CAMERAS = [
    {"make": "canon", "model": "ae-1", "description": "d"},
    {"make": "nikon", "model": "f3", "aliases": ["n2020"], "description": "d"},
]
FILMS = [
    {"make": "kodak", "type": "gold 200", "speed": 200},
    {"make": "ilford", "type": "hp5 plus", "speed": 400},
]


def reply(caption="A dog on a beach", tags=("dog", "beach")):
    return json.dumps({"caption": caption, "tags": list(tags)})


REFUSAL = object()


class FakeTagCompletions(FakeCompletions):
    def create(self, **kwargs):
        if self.responses and self.responses[0] is REFUSAL:
            self.responses.pop(0)
            self.calls.append(kwargs)
            message = SimpleNamespace(content=None, refusal="I can't help with that")
            return SimpleNamespace(choices=[SimpleNamespace(message=message)])
        return super().create(**kwargs)


class FakeOpenAI:
    def __init__(self, responses):
        self.chat = SimpleNamespace(completions=FakeTagCompletions(responses))


class TestNormalize:
    def test_plural_and_case(self):
        tags = ["Dogs", "  Cities ", "LEAVES", "people", "glass", "bus", "trees"]
        assert normalize_tags(tags, set()) == [
            "dog",
            "city",
            "leaf",
            "people",
            "glass",
            "bus",
            "tree",
        ]

    def test_punctuation_and_spaces(self):
        assert normalize_tags(["new   york!", "#sunset", "red-brick", "-"], set()) == [
            "new york",
            "sunset",
            "red-brick",
        ]

    def test_catalog_words(self):
        blocked = catalog_words(CAMERAS, FILMS)
        tags = ["canon", "ae-1", "nikon f3", "n2020", "gold 200", "gold", "hp5 plus"]
        assert normalize_tags(tags, blocked) == ["gold"]

    def test_stoplist(self, tmp_path):
        path = tmp_path / "stop.txt"
        path.write_text("photo\nfilm\n\nmedium format\n")
        blocked = load_stoplist(str(path))
        tags = ["photo", "Photos", "film", "medium format", "street"]
        assert normalize_tags(tags, blocked) == ["street"]

    def test_missing_stoplist_is_empty(self, tmp_path):
        assert load_stoplist(str(tmp_path / "missing.txt")) == set()

    def test_monochrome(self):
        tags = ["Black and White", "black-and-white", "B&W", "bw", "grayscale"]
        assert normalize_tags(tags + ["Greyscale", "women"], set()) == [
            "monochrome",
            "woman",
        ]

    def test_dedupe_after_singular(self):
        assert normalize_tags(["dog", "dogs", "Dog", 3, None], set()) == ["dog"]

    def test_cap(self):
        tags = [f"tag{i}" for i in range(30)]
        assert normalize_tags(tags, set()) == tags[:MAX_TAGS]

    def test_named_places_kept(self):
        named = named_in("Sunday in Brussels\nwalking around Los Angeles with dogs")
        tags = ["brussels", "los angeles", "dogs", "towns"]
        assert normalize_tags(tags, set(), named) == [
            "brussels",
            "los angeles",
            "dog",
            "town",
        ]

    def test_real_stoplist_allows_common_tags(self):
        blocked = load_stoplist(str(STOPLIST))
        assert {"photo", "film", "camera", "kodak"} <= blocked
        assert {"day", "daylight", "outdoors", "indoor", "scene", "thing"} <= blocked
        assert not {"people", "light", "color", "place", "monochrome"} & blocked
        assert normalize_tags(["outdoors", "black and white", "dusk"], blocked) == [
            "monochrome",
            "dusk",
        ]


class TestMonochrome:
    def test_added_last_for_grayscale(self):
        assert with_monochrome(["woman", "street"], True) == [
            "woman",
            "street",
            "monochrome",
        ]

    def test_not_added_for_color(self):
        assert with_monochrome(["woman"], False) == ["woman"]

    def test_deduped(self):
        assert with_monochrome(["monochrome", "woman"], True) == [
            "monochrome",
            "woman",
        ]

    def test_replaces_last_when_full(self):
        tags = [f"tag{i}" for i in range(MAX_TAGS)]
        got = with_monochrome(tags, True)
        assert len(got) == MAX_TAGS
        assert got == tags[:-1] + ["monochrome"]


class TestWeights:
    def test_linear_from_one_to_half(self):
        assert tag_keywords(["a", "b", "c"]) == [
            Keyword("a", 1.0),
            Keyword("b", 0.75),
            Keyword("c", 0.5),
        ]

    def test_single_tag(self):
        assert tag_keywords(["a"]) == [Keyword("a", 1.0)]

    def test_no_tags(self):
        assert tag_keywords([]) == []


class TestParse:
    def test_valid(self):
        assert parse_tags(reply()) == {
            "caption": "A dog on a beach",
            "tags": ["dog", "beach"],
        }

    def test_fenced(self):
        assert parse_tags(f"```json\n{reply()}\n```")["tags"] == ["dog", "beach"]

    def test_missing_caption(self):
        assert parse_tags(json.dumps({"tags": ["dog"]})) == {"tags": ["dog"]}

    @pytest.mark.parametrize(
        "content",
        [
            None,
            "",
            "I'm sorry, I can't describe this image.",
            json.dumps({"caption": "A dog"}),
            json.dumps({"caption": "A dog", "tags": []}),
            json.dumps(["dog"]),
        ],
    )
    def test_invalid(self, content):
        with pytest.raises(ValueError):
            parse_tags(content)


class TestTagger:
    def tagger(self, responses):
        ai = FakeOpenAI(responses)
        return ImageTagger(ai, "model-x", catalog_words(CAMERAS, FILMS), {"photo"}), ai

    def test_tag_image(self):
        tagger, ai = self.tagger([reply(tags=["Dogs", "photo", "Canon", "beach"])])

        tags = tagger.tag("https://cdn/medium.jpg", "Beach day", "x" * 2000)

        assert tags.caption == "A dog on a beach"
        assert tags.tags == ["dog", "beach"]
        assert tags.version == TAGGER_VERSION
        assert tags.model == "model-x"
        assert tags.raw["tags"] == ["Dogs", "photo", "Canon", "beach"]
        assert tags.post_caption().version == "v1"
        call = ai.chat.completions.calls[0]
        image, text = call["messages"][1]["content"]
        assert image == {
            "type": "image_url",
            "image_url": {"url": "https://cdn/medium.jpg"},
        }
        assert text["text"].startswith("title: Beach day\ndescription: x")
        assert len(text["text"]) == 1000
        assert call["response_format"]["type"] == "json_schema"

    def test_refusal_retries_then_title_only(self):
        tagger, ai = self.tagger(
            [
                REFUSAL,
                "I can't help with that.",
                reply(caption="ignored", tags=["new york"]),
            ]
        )

        tags = tagger.tag("https://cdn/medium.jpg", "New York", None)

        assert tags.caption is None
        assert tags.tags == ["new york"]
        assert tags.version == "v1-text"
        assert tags.text_only
        calls = ai.chat.completions.calls
        assert len(calls) == 3
        assert all(p["type"] == "text" for p in calls[2]["messages"][1]["content"])

    def test_grayscale_forces_monochrome(self):
        tagger, _ = self.tagger([reply(), "no", "no", reply(tags=["street"])])

        image = tagger.tag("https://cdn/m.jpg", "t", None, grayscale=True)
        text = tagger.tag("https://cdn/m.jpg", "t", None, grayscale=True)

        assert image.tags == ["dog", "beach", "monochrome"]
        assert text.version == "v1-text"
        assert text.tags == ["street", "monochrome"]

    def test_tag_all_passes_grayscale(self):
        tagger, _ = self.tagger([reply()])

        results = tagger.tag_all(
            [TagInput("https://cdn/1.jpg", "a", grayscale=True)], concurrency=1
        )

        assert results[0].tags[-1] == "monochrome"

    def test_empty_after_normalizing_retries(self):
        tagger, _ = self.tagger([reply(tags=["photo"]), reply(tags=["dog"])])

        assert tagger.tag("https://cdn/m.jpg", "t", None).tags == ["dog"]

    def test_schema_rejected_falls_back_to_json_object(self):
        tagger, ai = self.tagger([bad_request(), reply()])

        assert tagger.tag("https://cdn/m.jpg", "t", None).version == "v1"
        assert ai.chat.completions.calls[1]["response_format"] == {
            "type": "json_object"
        }

    def test_all_attempts_fail(self):
        tagger, _ = self.tagger(["no", "no", "no", "no"])

        with pytest.raises(TaggingError):
            tagger.tag("https://cdn/m.jpg", "t", None)

    def test_tag_all_returns_errors_in_order(self):
        tagger, _ = self.tagger([reply(), "no", "no", "no", "no"])

        results = tagger.tag_all(
            [TagInput("https://cdn/1.jpg", "a"), TagInput("https://cdn/2.jpg", "b")],
            concurrency=1,
        )

        assert results[0].tags == ["dog", "beach"]
        assert isinstance(results[1], TaggingError)


def test_post_text():
    assert post_text(" A  title ", None) == "title: A title"
    assert post_text("t", "line\n\nbreak") == "title: t\ndescription: line break"


def test_medium_url():
    images = [S3Image("low", "https://l", 1, 1), S3Image("medium", "https://m", 1, 1)]
    assert medium_url(images) == "https://m"
    with pytest.raises(ValueError):
        medium_url(images[:1])
