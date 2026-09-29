import json
import os
import time
from pathlib import Path
from typing import Any, Dict, List, Optional

import requests
from analogdb.models import Camera, Film
from dotenv import load_dotenv
from openai import OpenAI

ROOT = Path(__file__).parent
DATA = ROOT / "data"
REPORTS = DATA / "reports"
RUNS = DATA / "runs"

SAMPLE_PATH = DATA / "sample.json"
CATALOG_PATH = DATA / "catalog.json"
DRAFTS_PATH = DATA / "drafts.json"
GOLD_PATH = DATA / "gold.json"

API_URL = "https://api.analogdb.com/v1"
FIELDS = [
    "camera_make",
    "camera_model",
    "film_make",
    "film_type",
    "film_speed",
    "focal_length",
    "aperture",
]


def load_env() -> None:
    load_dotenv(ROOT.parents[1] / ".env")


def api_get(path: str, params: Optional[Dict[str, Any]] = None) -> Dict:
    for attempt in range(5):
        resp = requests.get(f"{API_URL}{path}", params=params, timeout=30)
        if resp.status_code == 429 or resp.status_code >= 500:
            time.sleep(2**attempt)
            continue
        resp.raise_for_status()
        return resp.json()
    resp.raise_for_status()
    return {}


def read_json(path: Path) -> Any:
    with open(path) as f:
        return json.load(f)


def write_json(path: Path, data: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with open(path, "w") as f:
        json.dump(data, f, indent=1, ensure_ascii=False)
        f.write("\n")


def fetch_catalog() -> Dict:
    cameras = api_get("/cameras")["cameras"]
    films = api_get("/films")["films"]
    return {
        "fetched": int(time.time()),
        "cameras": [
            {"id": c["id"], "make": c["make"], "model": c["model"]}
            for c in sorted(cameras, key=lambda c: c["id"])
        ],
        "films": [
            {
                "id": f["id"],
                "make": f["make"],
                "type": f["type"],
                "speed": f.get("speed"),
                "color_type": f.get("color_type"),
            }
            for f in sorted(films, key=lambda f: f["id"])
        ],
    }


def catalog_models(catalog: Dict) -> tuple[List[Camera], List[Film]]:
    cameras = [
        Camera(id=c["id"], make=c["make"], model=c["model"], description="")
        for c in catalog["cameras"]
    ]
    films = [
        Film(
            id=f["id"],
            make=f["make"],
            type=f["type"],
            speed=f["speed"],
            color_type=f["color_type"] or "",
            description="",
        )
        for f in catalog["films"]
    ]
    return cameras, films


def openrouter() -> OpenAI:
    return OpenAI(
        base_url=os.environ.get("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1"),
        api_key=os.environ["OPENROUTER_API_KEY"],
    )


class Usage:
    """Totals of tokens and OpenRouter cost across calls."""

    def __init__(self):
        self.calls = 0
        self.input_tokens = 0
        self.output_tokens = 0
        self.cost = 0.0

    def add(self, usage) -> None:
        self.calls += 1
        if usage is None:
            return
        self.input_tokens += usage.prompt_tokens or 0
        self.output_tokens += usage.completion_tokens or 0
        extra = getattr(usage, "model_extra", None) or {}
        self.cost += float(extra.get("cost") or 0)

    def to_dict(self) -> Dict:
        return {
            "calls": self.calls,
            "input_tokens": self.input_tokens,
            "output_tokens": self.output_tokens,
            "cost": round(self.cost, 4),
        }


class TrackedOpenAI:
    """Wraps an OpenAI client so every chat completion adds to a Usage."""

    def __init__(self, client: OpenAI, usage: Usage):
        self.chat = self
        self.completions = self
        self._client = client
        self._usage = usage

    def create(self, **kwargs):
        resp = self._client.chat.completions.create(**kwargs)
        self._usage.add(resp.usage)
        return resp


def llm_input(post: Dict, comments: bool = True) -> str:
    """Title, description and OP comments, labeled, on one line each."""
    parts = [f"title: {post['title']}"]
    if post.get("description"):
        parts.append(f"description: {post['description']}")
    if comments and post.get("op_comments"):
        parts.append("op comments: " + " / ".join(post["op_comments"]))
    return "\n".join(" ".join(p.split()) for p in parts)
