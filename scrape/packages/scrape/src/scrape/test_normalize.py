import pytest

from .normalize import normalize_key, normalize_tokens


@pytest.mark.parametrize(
    "text,key",
    [
        ("Nikon F-3", "nikonf3"),
        ("500 C/M", "500cm"),
        ("HP5+", "hp5"),
        ("Tri-X 400", "trix400"),
        ("Voigtländer", "voigtlander"),
        ("Pentax6x7", "pentax6x7"),
        ("Chinon Industries Inc.", "chinonindustriesinc"),
        ("Yes!Star", "yesstar"),
        ("Fuji & Kodak", "fujiandkodak"),
        ("  RB67 Pro-S  ", "rb67pros"),
        ("T-MAX P3200", "tmaxp3200"),
        ("", ""),
    ],
)
def test_normalize_key(text, key):
    assert normalize_key(text) == key


def test_normalize_tokens():
    assert normalize_tokens("Nikon F-3, Portra 400!") == [
        "nikon",
        "f",
        "3",
        "portra",
        "400",
    ]
    assert normalize_tokens("Voigtländer") == ["voigtlander"]
