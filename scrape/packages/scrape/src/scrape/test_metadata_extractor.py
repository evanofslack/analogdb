import json
import re
from types import SimpleNamespace

from analogdb.models import Camera, Film
from openai import OpenAIError

from .metadata import MetadataExtractor

CAMERAS = [
    Camera(id=1, make="Canon", model="AE-1", description=""),
    Camera(id=2, make="Nikon", model="F3", description=""),
]

FILMS = [
    Film(
        id=1,
        type="Portra",
        make="Kodak",
        speed=400,
        color_type="color",
        description="",
    ),
    Film(
        id=2,
        type="Gold",
        make="Kodak",
        speed=200,
        color_type="color",
        description="",
    ),
    Film(
        id=3,
        type="Gold",
        make="Fuji",
        speed=200,
        color_type="color",
        description="",
    ),
]


class FakeCompletions:
    def __init__(self, responses):
        self.responses = list(responses)
        self.prompts = []

    def create(self, **kwargs):
        prompt = kwargs["messages"][-1]["content"]
        self.prompts.append(prompt)
        response = self.responses.pop(0)
        content = response(prompt) if callable(response) else response
        message = SimpleNamespace(content=content)
        return SimpleNamespace(choices=[SimpleNamespace(message=message)])


class FakeOpenAI:
    def __init__(self, responses):
        self.chat = SimpleNamespace(completions=FakeCompletions(responses))

    @property
    def calls(self):
        return len(self.chat.completions.prompts)


def new_extractor(responses, batch_size=25):
    client = FakeOpenAI(responses)
    return MetadataExtractor(client, "test-model", batch_size), client


def test_out_of_order_items():
    content = json.dumps(
        [
            {"post_id": 2, "camera_make": "Nikon", "camera_model": "F3"},
            {"post_id": 1, "camera_make": "Canon", "camera_model": "AE-1"},
        ]
    )
    extractor, _ = new_extractor([content])
    result = extractor.extract(["title: a", "title: b"], FILMS, CAMERAS)
    assert result.failed == 0
    assert result.metadata[0].camera_make == "canon"
    assert result.metadata[0].camera_model == "ae-1"
    assert result.metadata[1].camera_make == "nikon"
    assert result.metadata[1].camera_model == "f3"


def test_missing_id():
    content = json.dumps(
        [
            {"post_id": 1, "camera_make": "Canon"},
            {"post_id": 3, "camera_make": "Nikon"},
        ]
    )
    extractor, _ = new_extractor([content])
    result = extractor.extract(["title: a", "title: b", "title: c"], FILMS, CAMERAS)
    assert result.failed == 1
    assert len(result.metadata) == 3
    assert result.metadata[0].camera_make == "canon"
    assert result.metadata[1].is_empty()
    assert result.metadata[2].camera_make == "nikon"


def test_garbage_response_retried_once():
    extractor, client = new_extractor(["not json", "still not json"])
    result = extractor.extract(["title: a", "title: b"], FILMS, CAMERAS)
    assert client.calls == 2
    assert result.failed == 2
    assert len(result.metadata) == 2
    assert all(m.is_empty() for m in result.metadata)


def test_retry_recovers():
    content = json.dumps([{"post_id": 1, "camera_make": "Canon"}])
    extractor, client = new_extractor(['{"posts": "none"}', content])
    result = extractor.extract(["title: a"], FILMS, CAMERAS)
    assert client.calls == 2
    assert result.failed == 0
    assert result.metadata[0].camera_make == "canon"


def raise_api_error(prompt):
    raise OpenAIError("unavailable")


def test_api_error_retried_once():
    extractor, client = new_extractor([raise_api_error, raise_api_error])
    result = extractor.extract(["title: a"], FILMS, CAMERAS)
    assert client.calls == 2
    assert result.failed == 1
    assert result.metadata[0].is_empty()


def test_wrapped_response():
    content = json.dumps({"posts": [{"post_id": 1, "film_type": "Portra"}]})
    extractor, _ = new_extractor([content])
    result = extractor.extract(["title: a"], FILMS, CAMERAS)
    assert result.failed == 0
    assert result.metadata[0].film_make == "kodak"
    assert result.metadata[0].film_type == "portra"
    assert result.metadata[0].film_speed == 400


def test_string_post_id():
    content = json.dumps(
        [
            {"post_id": "1", "camera_make": "Canon"},
            {"post_id": "2", "camera_make": "Nikon"},
        ]
    )
    extractor, _ = new_extractor([content])
    result = extractor.extract(["title: a", "title: b"], FILMS, CAMERAS)
    assert result.failed == 0
    assert result.metadata[1].camera_make == "nikon"


def echo_focal_length(prompt):
    items = []
    for id, n in re.findall(r"post_id: (\d+), title: (\d+)", prompt):
        items.append({"post_id": int(id), "focal_length": 10 + int(n)})
    return json.dumps(list(reversed(items)))


def test_chunks():
    titles = [f"title: {n}" for n in range(60)]
    extractor, client = new_extractor([echo_focal_length] * 3, batch_size=25)
    result = extractor.extract(titles, FILMS, CAMERAS)
    assert client.calls == 3
    assert result.failed == 0
    assert len(result.metadata) == 60
    assert [m.focal_length for m in result.metadata] == [10 + n for n in range(60)]


def test_mismatched_camera_pair():
    content = json.dumps([{"post_id": 1, "camera_make": "Canon", "camera_model": "F3"}])
    extractor, _ = new_extractor([content])
    result = extractor.extract(["title: a"], FILMS, CAMERAS)
    assert result.metadata[0].camera_make == "canon"
    assert result.metadata[0].camera_model is None


def test_film_type_with_two_makes():
    content = json.dumps([{"post_id": 1, "film_type": "Gold"}])
    extractor, _ = new_extractor([content])
    result = extractor.extract(["title: a"], FILMS, CAMERAS)
    assert result.metadata[0].film_type == "gold"
    assert result.metadata[0].film_make is None


def test_empty_titles():
    extractor, client = new_extractor([])
    result = extractor.extract([], FILMS, CAMERAS)
    assert client.calls == 0
    assert result.metadata == []
    assert result.failed == 0
