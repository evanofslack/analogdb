import re
import unicodedata
from typing import List

_NON_ALNUM = re.compile(r"[^a-z0-9]+")


def _fold(text: str) -> str:
    text = unicodedata.normalize("NFKD", text.lower())
    text = "".join(c for c in text if not unicodedata.combining(c))
    return text.replace("&", " and ")


def normalize_key(text: str) -> str:
    """Lowercase, & to and, strip accents and everything but letters and digits."""
    return _NON_ALNUM.sub("", _fold(text))


def normalize_tokens(text: str) -> List[str]:
    """Lowercase words of letters and digits, split on everything else."""
    return [t for t in _NON_ALNUM.split(_fold(text)) if t]
