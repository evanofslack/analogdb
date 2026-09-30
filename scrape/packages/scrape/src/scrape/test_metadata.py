import json
from types import SimpleNamespace

import httpx
from analogdb.models import Camera, Film
from openai import BadRequestError

from .metadata_llm import (
    MetadataExtractorV2,
    Transcriber,
    parse_transcripts,
    post_input,
)
from .models import MetadataPost


def transcript(post_id, cameras=(), films=(), lenses=()):
    return {
        "post_id": post_id,
        "cameras": list(cameras),
        "films": list(films),
        "lenses": list(lenses),
    }


class FakeCompletions:
    def __init__(self, responses):
        self.responses = list(responses)
        self.calls = []

    def create(self, **kwargs):
        self.calls.append(kwargs)
        response = self.responses.pop(0)
        if isinstance(response, Exception):
            raise response
        message = SimpleNamespace(content=response)
        return SimpleNamespace(choices=[SimpleNamespace(message=message)], usage=None)


class FakeOpenAI:
    def __init__(self, responses):
        self.chat = SimpleNamespace(completions=FakeCompletions(responses))


def bad_request():
    request = httpx.Request("POST", "https://example.com")
    return BadRequestError(
        "no schema", response=httpx.Response(400, request=request), body=None
    )


def test_parse_transcripts():
    content = (
        "```json\n"
        + json.dumps({"posts": [transcript(2), transcript(1), transcript(9), "x"]})
        + "\n```"
    )
    by_id = parse_transcripts(content, [1, 2])
    assert set(by_id) == {1, 2}
    assert by_id[1] == {"cameras": [], "films": [], "lenses": []}


def test_parse_transcripts_bare_list_and_string_ids():
    content = json.dumps(
        [{"post_id": "1", "cameras": [{"text": "F3"}, "bad"]}, {"post_id": True}]
    )
    assert parse_transcripts(content, [1]) == {
        1: {"cameras": [{"text": "F3"}], "films": [], "lenses": []}
    }


def test_post_input_caps_comments_first():
    post = MetadataPost(
        title="Title  here", description="desc\nline", op_comments=["a" * 5000]
    )
    text = post_input(3, post, limit=200)
    assert text.startswith(
        "post_id: 3\ntitle: Title here\ndescription: desc line\nop comments: aaa"
    )
    assert len(text) <= 200


def test_schema_rejected_falls_back_to_json_object():
    client = FakeOpenAI([bad_request(), json.dumps({"posts": [transcript(1)]})])
    t = Transcriber(client, "model")
    out = t.transcribe([MetadataPost(title="a")], ["nikon"], ["kodak"])
    calls = client.chat.completions.calls
    assert calls[0]["response_format"]["type"] == "json_schema"
    assert calls[1]["response_format"] == {"type": "json_object"}
    assert out == [{"cameras": [], "films": [], "lenses": []}]
    assert t.use_schema is False


def test_failed_batch_returns_none_and_batches_continue():
    client = FakeOpenAI(
        ["not json", "still not", json.dumps({"posts": [transcript(1)]})]
    )
    t = Transcriber(client, "model", batch_size=2)
    out = t.transcribe([MetadataPost(title=x) for x in "abc"], [], [])
    assert out[:2] == [None, None]
    assert out[2] is not None
    assert len(client.chat.completions.calls) == 3


def test_prompt_has_makes_and_examples_not_catalog():
    client = FakeOpenAI([json.dumps({"posts": [transcript(1)]})])
    Transcriber(client, "model").transcribe(
        [MetadataPost(title="a")], ["nikon"], ["kodak"]
    )
    messages = client.chat.completions.calls[0]["messages"]
    assert "nikon" in messages[0]["content"] and "Lomography" in messages[0]["content"]
    assert [m["role"] for m in messages] == ["system", "user", "assistant", "user"]
    assert messages[-1]["content"] == "post_id: 1\ntitle: a"


def test_extractor_v2_end_to_end():
    response = json.dumps(
        {
            "posts": [
                transcript(
                    1,
                    cameras=[{"text": "Contax G2", "make": "Contax", "model": "G2"}],
                    films=[
                        {
                            "text": "Portra400",
                            "make": None,
                            "type": "Portra400",
                            "box_speed": None,
                            "shot_at": None,
                        }
                    ],
                ),
                transcript(
                    2, cameras=[{"text": "Nikon F-3", "make": "Nikon", "model": "F-3"}]
                ),
            ]
        }
    )
    cameras = [
        Camera(id=1, make="contax", model="g1", description=""),
        Camera(id=2, make="nikon", model="f3", description=""),
    ]
    films = [
        Film(
            id=1,
            make="kodak",
            type="portra 400",
            speed=400,
            color_type="color",
            description="",
        )
    ]
    extractor = MetadataExtractorV2(FakeOpenAI([response]), "model")
    posts = [
        MetadataPost(title="Rain [Contax G2, Portra400]"),
        MetadataPost(title="Nikon F-3"),
    ]
    first, second = extractor.extract(posts, cameras, films)
    assert (first.proposed.camera_make, first.proposed.camera_model) == ("contax", None)
    assert first.unmatched[0].key == "contaxg2"
    assert (first.proposed.film_type, first.proposed.film_speed) == ("portra 400", 400)
    assert second.proposed.camera_model == "f3"
    assert first.raw["cameras"][0]["model"] == "G2"
