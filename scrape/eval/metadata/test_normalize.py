import json
from pathlib import Path

from scrape.normalize import normalize_key, normalize_tokens

CASES = json.loads((Path(__file__).parent / "normalize_cases.json").read_text())


def test_cases_have_input_and_key():
    assert CASES
    for case in CASES:
        assert set(case) == {"input", "key"}


def test_normalize_key_matches_fixture():
    for case in CASES:
        assert normalize_key(case["input"]) == case["key"], case


def test_normalize_tokens():
    assert normalize_tokens("Nikon F-3, Portra 400!") == [
        "nikon",
        "f",
        "3",
        "portra",
        "400",
    ]
    assert normalize_tokens("Voigtländer") == ["voigtlander"]
