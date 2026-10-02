import json
from types import SimpleNamespace

import eval_tags


def api_post(id, title="A street", nsfw=False, grayscale=False):
    return {
        "id": id,
        "title": title,
        "description": None,
        "nsfw": nsfw,
        "grayscale": grayscale,
        "images": [
            {"resolution": "low", "url": f"https://cdn/{id}-low.jpg"},
            {"resolution": "medium", "url": f"https://cdn/{id}-medium.jpg"},
        ],
        "keywords": [{"word": "great", "weight": 1.0}],
    }


def fake_fetch(params):
    if params.get("nsfw") == "true":
        return [api_post(i, nsfw=True) for i in range(1, 4)]
    if params.get("grayscale") == "true":
        return [api_post(i, grayscale=True) for i in range(10, 13)]
    return [api_post(20, "Morning in Lisbon"), api_post(21), api_post(22), api_post(1)]


class FakeCompletions:
    def __init__(self):
        self.calls = []

    def create(self, **kwargs):
        self.calls.append(kwargs)
        image = any(
            part.get("type") == "image_url"
            for m in kwargs["messages"]
            if isinstance(m["content"], list)
            for part in m["content"]
        )
        if image and "nsfw" in kwargs["model"]:
            message = SimpleNamespace(content=None, refusal="no")
        else:
            content = json.dumps({"caption": "A street", "tags": ["streets", "photo"]})
            message = SimpleNamespace(content=content, refusal=None)
        usage = SimpleNamespace(prompt_tokens=100, completion_tokens=20, cost=0.0001)
        choice = SimpleNamespace(message=message, finish_reason="stop")
        return SimpleNamespace(choices=[choice], usage=usage)


def test_sample_groups_and_dedupes():
    posts = eval_tags.sample_posts(fake_fetch, nsfw=2, bw=2, places=1, total=7)

    assert [p["group"] for p in posts] == [
        "nsfw",
        "nsfw",
        "bw",
        "bw",
        "place",
        "random",
        "random",
    ]
    assert [p["id"] for p in posts] == [1, 2, 10, 11, 20, 21, 22]
    assert posts[0]["url"] == "https://cdn/1-medium.jpg"


def test_main_runs_models_and_renders(tmp_path):
    ai = SimpleNamespace(chat=SimpleNamespace(completions=FakeCompletions()))
    argv = [
        "--work-dir",
        str(tmp_path),
        "--total",
        "4",
        "--per-group",
        "1",
        "--models",
        "good",
        "nsfw-refuser",
        "--concurrency",
        "2",
    ]

    summary = eval_tags.main(argv, fetch=fake_fetch, openai=ai)

    assert summary["good"]["posts"] == 4
    assert summary["good"]["image_ok"] == 4
    assert summary["good"]["cost_per_image"] == 0.0001
    assert summary["nsfw-refuser"]["text_only"] == 4
    assert summary["nsfw-refuser"]["refusal_rate"] == 1.0
    results = json.loads((tmp_path / "results.json").read_text())
    first = results["good"]["1"]
    assert first["tags"] == ["street"]
    assert results["good"]["10"]["tags"] == ["street", "monochrome"]
    assert first["calls"][0]["image"] is True
    assert ai.chat.completions.calls[0]["extra_body"] == {"usage": {"include": True}}
    page = (tmp_path / "review.html").read_text()
    assert "nsfw-refuser" in page and "https://cdn/1-medium.jpg" in page

    calls = len(ai.chat.completions.calls)
    eval_tags.main(argv, fetch=fake_fetch, openai=ai)
    assert len(ai.chat.completions.calls) == calls

    results["good"]["1"]["tags"] = ["stale"]
    (tmp_path / "results.json").write_text(json.dumps(results))
    eval_tags.main(
        argv[:-4] + ["good", "--concurrency", "2", "--force"],
        fetch=fake_fetch,
        openai=ai,
    )
    assert len(ai.chat.completions.calls) == calls + 4
    rerun = json.loads((tmp_path / "results.json").read_text())
    assert rerun["good"]["1"]["tags"] == ["street"]
    assert rerun["nsfw-refuser"] == results["nsfw-refuser"]


def test_sample_only_makes_no_calls(tmp_path):
    ai = SimpleNamespace(chat=SimpleNamespace(completions=FakeCompletions()))

    eval_tags.main(
        ["--work-dir", str(tmp_path), "--sample-only"], fetch=fake_fetch, openai=ai
    )

    assert (tmp_path / "sample.json").exists()
    assert ai.chat.completions.calls == []
