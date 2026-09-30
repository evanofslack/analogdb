import re
from dataclasses import dataclass
from itertools import combinations
from typing import Dict, Iterable, List, Optional

from analogdb.models import Camera, Film
from rapidfuzz import fuzz

from .metadata import MetadataExtractor
from .models import MatchResult, MetadataPost, PhotoMetadata, UnmatchedMention
from .normalize import normalize_key, normalize_tokens

FUZZY_MIN = 92
FUZZY_GAP = 5

CAMERA_MAKE_ALIASES = {
    "asahi": "pentax",
    "asahipentax": "pentax",
    "fuji": "fujifilm",
    "fujica": "fujifilm",
    "rolleicord": "rollei",
    "rolleiflex": "rollei",
    "zenza": "bronica",
    "zenzabronica": "bronica",
}

FILM_MAKE_ALIASES = {
    "cine": "cinestill",
    "ferrania": "film ferrania",
    "fomapan": "foma",
    "fuji": "fujifilm",
    "fujichrome": "fujifilm",
    "fujicolor": "fujifilm",
    "harman": "harmon",
    "lomo": "lomography",
    "orwo": "original wolfen",
    "wolfen": "original wolfen",
}

# Words dropped from film types on both sides, so "Pro 400H" meets "fujicolor pro 400h"
# and "HP5+" meets "hp5 plus"
FILM_OPTIONAL_WORDS = {
    "action",
    "agfacolor",
    "classic",
    "creative",
    "film",
    "fomapan",
    "fujichrome",
    "fujicolor",
    "lomochrome",
    "plus",
    "pro",
    "professional",
    "soft",
    "super",
    "vision",
    "vision3",
    "xtra",
}

# Product lines written stuck to the name: "Fomapan200", "FujicolorC200"
FILM_LINE_WORDS = {"agfacolor", "fomapan", "fujichrome", "fujicolor", "lomochrome"}

FILM_TYPE_ALIASES = {
    "acros": "neopanacros",
    "acros100": "neopanacros100",
    "astia100": "astia100f",
    "provia100f": "provia100",
    "tmax3200": "tmaxp3200",
}

FILM_WORD_ALIASES = {"colour": "color", "ir": "infrared", "tx": "trix"}

# Trailing words that say nothing about which film it was
FILM_TRAILING_WORDS = {"disposable", "expired", "mix", "roll", "rolls"}

# Suffixes that name a variant of a catalog camera: F3HP is an F3
MODEL_MODIFIERS = {"hp", "md", "ttl"}

# Electro 35 GSN is an Electro 35
MODEL_MODIFIERS |= {"gs", "gsn", "gt"}

# The same, only as a separate word: Spotmatic SP is a Spotmatic, OM-2SP is not an OM-2
MODEL_MODIFIER_WORDS = {"n", "pro", "sp", "titan"}

# Trailing words that say nothing about which camera it was
CAMERA_TRAILING_WORDS = {"camera", "frame", "half", "rangefinder", "slr", "tlr"}

NIKON_US_NAMES = {
    "n2000": "f301",
    "n2020": "f501",
    "n4004": "f401",
    "n6006": "f601",
    "n8008": "f801",
    "n8008s": "f801s",
}

MINOLTA_PREFIXES = ("maxxum", "dynax", "alpha")


def _key(text: Optional[str]) -> str:
    if not text:
        return ""
    return normalize_key(text.replace("μ", "mju").replace("µ", "mju"))


def _tokens(text: Optional[str]) -> List[str]:
    return normalize_tokens((text or "").replace("μ", "mju").replace("µ", "mju"))


def _has_digit(key: str) -> bool:
    return any(c.isdigit() for c in key)


def _digits(key: str) -> List[str]:
    return re.findall(r"\d+", key)


@dataclass
class CatalogAlias:
    kind: str
    key: str
    make: str
    name: str


@dataclass
class _Hit:
    make: Optional[str] = None
    name: Optional[str] = None
    speed: Optional[int] = None
    flag: Optional[str] = None
    unmatched: Optional[UnmatchedMention] = None


def _unique(items: Iterable) -> List:
    seen, out = set(), []
    for item in items:
        if id(item) not in seen:
            seen.add(id(item))
            out.append(item)
    return out


def camera_model_variants(model: str, make: Optional[str]) -> List[str]:
    keys = [model]

    def add(k: str):
        if k and k not in keys:
            keys.append(k)

    add(re.sub(r"(mk|mark)(ii|2)$", "ii", model))
    add(re.sub(r"(mk|mark)(iii|3)$", "iii", model))
    add(re.sub(r"6x(45|6|7|8|9)", r"6\1", model))
    if make == "nikon":
        add(NIKON_US_NAMES.get(model, ""))
        add({v: k for k, v in NIKON_US_NAMES.items()}.get(model, ""))
        m = re.fullmatch(r"([nf])(\d+)([a-z]*)", model)
        if m:
            add(("f" if m.group(1) == "n" else "n") + m.group(2) + m.group(3))
    if make == "minolta":
        for prefix in MINOLTA_PREFIXES:
            if model.startswith(prefix):
                for other in MINOLTA_PREFIXES:
                    add(other + model[len(prefix) :])
    if make == "canon":
        add(model[3:] if model.startswith("eos") else "eos" + model)
        if model.endswith("p"):
            add(model[:-1] + "program")
    if make == "olympus":
        add(re.sub(r"mju(2|ii)$", "mjuii", model))
        add(re.sub(r"mju(1|i)?$", "mjui", model))
    return keys


def _film_tokens(text: Optional[str], make_words: set) -> List[str]:
    if not text:
        return []
    text = re.sub(r"x-?tra", " ", text.lower())
    tokens = normalize_tokens(text.replace("+", " plus "))
    tokens = [FILM_WORD_ALIASES.get(t, t) for t in tokens]
    out = []
    for t in tokens:
        if t in make_words:
            continue
        for w in (
            []
            if t in FILM_OPTIONAL_WORDS
            else sorted(make_words | FILM_LINE_WORDS, key=len, reverse=True)
        ):
            if len(w) >= 4 and t.startswith(w) and len(t) > len(w):
                t = t[len(w) :]
                break
        out.append(t)
    return out


def film_type_key(text: Optional[str], make_words: set) -> str:
    """The type without make words and optional words: "Fujicolor Pro 400H" is "400h"."""
    key = "".join(
        t for t in _film_tokens(text, make_words) if t not in FILM_OPTIONAL_WORDS
    )
    return FILM_TYPE_ALIASES.get(key, key)


def film_type_keys(
    text: Optional[str], make_words: set, trim: bool = False
) -> List[str]:
    """Every key with some optional words dropped, most specific first. With trim, then
    the same with trailing speeds and filler cut: "HP5 400" and "Portra 400/800 Mix"."""
    tokens = _film_tokens(text, make_words)
    keys: List[str] = []
    while tokens:
        optional = [i for i, t in enumerate(tokens) if t in FILM_OPTIONAL_WORDS]
        for n in range(len(optional) + 1):
            for drop in combinations(optional, n):
                key = "".join(t for i, t in enumerate(tokens) if i not in drop)
                key = FILM_TYPE_ALIASES.get(key, key)
                if key and key not in keys:
                    keys.append(key)
        last = tokens[-1] if tokens else ""
        trailing = (
            last.isdigit() or last in FILM_TRAILING_WORDS or last in FILM_OPTIONAL_WORDS
        )
        if not trim or len(tokens) < 2 or not trailing:
            break
        tokens = tokens[:-1]
    return keys


class CatalogMatcher:
    """Matches transcribed camera and film mentions to catalog entries."""

    def __init__(
        self,
        cameras: List[Camera],
        films: List[Film],
        aliases: Optional[List[CatalogAlias]] = None,
    ):
        self.cameras = cameras
        self.films = films

        self.camera_makes: Dict[str, str] = {}
        for c in cameras:
            make = c.make.lower().strip()
            self.camera_makes[_key(make)] = make
            first = normalize_tokens(make)[0]
            self.camera_makes.setdefault(first, make)
        for alias, make in CAMERA_MAKE_ALIASES.items():
            if make in self.camera_makes.values():
                self.camera_makes.setdefault(alias, make)

        self.film_makes: Dict[str, str] = {}
        for f in films:
            make = f.make.lower().strip()
            self.film_makes[_key(make)] = make
            tokens = [
                t for t in normalize_tokens(make) if t not in ("film", "original")
            ]
            if tokens:
                self.film_makes.setdefault(tokens[0], make)
        for alias, make in FILM_MAKE_ALIASES.items():
            if make in self.film_makes.values():
                self.film_makes.setdefault(alias, make)
        self.film_make_words = set(self.film_makes) | set(FILM_MAKE_ALIASES)

        self.models: Dict[str, Dict[str, List[Camera]]] = {}
        self.models_any: Dict[str, List[Camera]] = {}
        for c in cameras:
            make, key = c.make.lower().strip(), _key(c.model)
            self.models.setdefault(make, {}).setdefault(key, []).append(c)
            self.models_any.setdefault(key, []).append(c)

        self.types: Dict[str, Dict[str, List[Film]]] = {}
        self.types_any: Dict[str, List[Film]] = {}
        for f in films:
            make = f.make.lower().strip()
            for key in {_key(f.type), *film_type_keys(f.type, self.film_make_words)}:
                if key:
                    self.types.setdefault(make, {}).setdefault(key, []).append(f)
                    self.types_any.setdefault(key, []).append(f)

        self.camera_aliases: Dict[str, Camera] = {}
        self.film_aliases: Dict[str, Film] = {}
        for a in aliases or []:
            if a.kind == "camera":
                c = self._find_camera(a.make, a.name)
                if c:
                    self.camera_aliases[a.key] = c
            elif a.kind == "film":
                f = self._find_film(a.make, a.name)
                if f:
                    self.film_aliases[a.key] = f

    def _find_camera(self, make: str, model: str) -> Optional[Camera]:
        hits = self.models.get(make.lower(), {}).get(_key(model), [])
        return hits[0] if hits else None

    def _find_film(self, make: str, type: str) -> Optional[Film]:
        hits = self.types.get(make.lower(), {}).get(_key(type), [])
        return hits[0] if hits else None

    def match(self, raw: Optional[Dict], post: MetadataPost) -> MatchResult:
        """raw is the stage 1 output for one post, None when the LLM failed."""
        result = MatchResult(proposed=PhotoMetadata(), raw=raw)
        if raw is None:
            result.flags.append("llm_failed")
            return result
        text = " ".join(
            [post.title or "", post.description or ""] + list(post.op_comments or [])
        )
        if not text.strip():
            result.flags.append("no_text")
        squashed = _key(text)
        tokens = set(normalize_tokens(text.replace("μ", "mju")))

        cameras = [c for c in raw.get("cameras") or [] if isinstance(c, dict)]
        if len({_key(f"{c.get('make')} {c.get('model')}") for c in cameras}) > 1:
            result.flags.append("multiple_cameras")
        if cameras:
            hit = self.match_camera(cameras[0])
            self._apply(result, hit, "camera", squashed, tokens, cameras[0])

        films = [f for f in raw.get("films") or [] if isinstance(f, dict)]
        if len({_key(f"{f.get('make')} {f.get('type')}") for f in films}) > 1:
            result.flags.append("multiple_films")
        if films:
            hit = self.match_film(films[0])
            self._apply(result, hit, "film", squashed, tokens, films[0])

        for lens in raw.get("lenses") or []:
            if not isinstance(lens, dict):
                continue
            if result.proposed.focal_length is None:
                result.proposed.focal_length = valid_focal_length(
                    lens.get("focal_length")
                )
            if result.proposed.aperture is None:
                result.proposed.aperture = valid_aperture(lens.get("aperture"))
        return result

    def _apply(
        self,
        result: MatchResult,
        hit: _Hit,
        kind: str,
        squashed: str,
        tokens: set,
        mention: Dict,
    ):
        if hit.flag:
            result.flags.append(hit.flag)
        if hit.unmatched:
            result.unmatched.append(hit.unmatched)
        name_field = "camera_model" if kind == "camera" else "film_type"
        if hit.name and not grounded(hit.name, mention, squashed, tokens, kind):
            result.flags.append(f"ungrounded_{kind}")
            hit.name = None
            hit.speed = None
        p = result.proposed
        setattr(p, f"{kind}_make", hit.make)
        setattr(p, name_field, hit.name)
        if kind == "film":
            p.film_speed = hit.speed

    def match_camera(self, mention: Dict) -> _Hit:
        raw_make = mention.get("make") or ""
        raw_model = mention.get("model") or ""
        make_key, model_key = _key(raw_make), _key(raw_model)
        make = self.camera_makes.get(make_key)

        if make is None and not make_key:
            for key, m in self.camera_makes.items():
                if (
                    len(key) >= 4
                    and model_key.startswith(key)
                    and len(model_key) > len(key)
                ):
                    make, make_key, model_key = m, key, model_key[len(key) :]
                    break
        if make is None and make_key:
            for word in normalize_tokens(raw_make):
                if word in self.camera_makes:
                    make = self.camera_makes[word]
                    break

        alias = self.camera_aliases.get(_key(f"{make or raw_make} {raw_model}"))
        if alias:
            return _Hit(make=alias.make.lower(), name=alias.model.lower())
        if not model_key:
            if make:
                return _Hit(make=make)
            if make_key:
                return self._camera_unmatched(mention, None, make_key, model_key)
            return _Hit()

        words = _tokens(raw_model)
        while len(words) > 1 and (
            words[-1] in CAMERA_TRAILING_WORDS or re.fullmatch(r"\d+x\d+", words[-1])
        ):
            words = words[:-1]
        if words and _key(raw_model) != "".join(words):
            model_key = "".join(words)
        keys = camera_model_variants(model_key, make)
        for key in self.camera_makes:
            if (
                len(key) >= 4
                and model_key.startswith(key)
                and len(model_key) > len(key)
            ):
                keys += camera_model_variants(model_key[len(key) :], make)
        if make and make_key != _key(make):
            keys += [make_key + k for k in list(keys)]

        if make:
            models = self.models.get(make, {})
            for k in keys:
                if k in models:
                    return self._camera_hit(models[k][0])
            if len(words) > 1 and words[-1] in MODEL_MODIFIER_WORDS:
                keys.append("".join(words[:-1]))
            prefix = _unique(
                c
                for k in keys
                for mk, cs in models.items()
                if k == mk or (k.startswith(mk) and k[len(mk) :] in MODEL_MODIFIERS)
                for c in cs
            )
            if len(prefix) == 1:
                return self._camera_hit(prefix[0])
            suffix = _unique(
                c
                for mk, cs in models.items()
                if len(model_key) >= 3
                and _has_digit(model_key)
                and mk.endswith(model_key)
                for c in cs
            )
            if len(suffix) == 1:
                return self._camera_hit(suffix[0])
            best = self._fuzzy(model_key, models)
            if best == "ambiguous":
                return _Hit(make=make, flag="ambiguous_camera")
            if best:
                return self._camera_hit(best)
        elif not make_key and (len(model_key) >= 3 or _has_digit(model_key)):
            hits = _unique(c for k in keys for c in self.models_any.get(k, []))
            if len(hits) == 1:
                return self._camera_hit(hits[0])
            if len(hits) > 1:
                return _Hit(flag="ambiguous_camera")

        return self._camera_unmatched(mention, make, make_key, model_key)

    def _camera_hit(self, c: Camera) -> _Hit:
        return _Hit(make=c.make.lower().strip(), name=c.model.lower().strip())

    def _camera_unmatched(
        self, mention: Dict, make: Optional[str], make_key: str, model_key: str
    ) -> _Hit:
        raw = mention.get("text") or " ".join(
            p for p in (mention.get("make"), mention.get("model")) if p
        )
        mention_obj = UnmatchedMention(
            kind="camera",
            raw=raw,
            key=_key(make or "") + model_key if make else make_key + model_key,
            make=make or (mention.get("make") or "").lower() or None,
            model=(mention.get("model") or "").lower() or None,
        )
        return _Hit(
            make=make,
            flag="catalog_make_only" if make else None,
            unmatched=mention_obj,
        )

    def match_film(self, mention: Dict) -> _Hit:
        raw_make = mention.get("make") or ""
        raw_type = mention.get("type") or ""
        make = self.film_makes.get(_key(raw_make))
        if make is None:
            for word in normalize_tokens(raw_make) + normalize_tokens(raw_type):
                if word in self.film_makes:
                    make = self.film_makes[word]
                    break
        speed = valid_speed(mention.get("box_speed"))
        tkey = film_type_key(raw_type, self.film_make_words)

        alias = self.film_aliases.get(_key(f"{make or raw_make} {raw_type}"))
        if alias:
            return self._film_hit(alias)
        if not tkey:
            if make:
                return _Hit(make=make, speed=speed)
            if raw_make:
                return self._film_unmatched(mention, None, tkey, speed)
            return _Hit(speed=speed)

        if make:
            types = self.types.get(make, {})
        elif (not raw_make or not self._is_other_make(raw_make)) and (
            len(tkey) >= 3 or _has_digit(tkey)
        ):
            types = self.types_any
        else:
            types = {}

        hits = _unique(types.get(tkey, []))
        for k in film_type_keys(raw_type, self.film_make_words, trim=True):
            if hits:
                break
            hits = _unique(types.get(k, []))
        if not hits and not _has_digit(tkey):
            hits = _unique(
                f for k, fs in types.items() if k.startswith(tkey) for f in fs
            )
            if speed:
                hits = [f for f in hits if f.speed == speed]
            elif len(hits) > 1:
                with_speed = _unique(f for f in hits if str(f.speed) in tkey)
                hits = with_speed or hits
        if not hits and make and not tkey.isdigit():
            best = self._fuzzy(tkey, types)
            if best == "ambiguous":
                return _Hit(make=make, speed=speed, flag="ambiguous_film")
            if best:
                hits = [best]
        if len(hits) == 1:
            return self._film_hit(hits[0])
        if len(hits) > 1:
            makes = {f.make.lower() for f in hits}
            return _Hit(
                make=make or (makes.pop() if len(makes) == 1 else None),
                speed=speed,
                flag="ambiguous_film",
            )
        return self._film_unmatched(mention, make, tkey, speed)

    def _is_other_make(self, raw_make: str) -> bool:
        """A make we can't place, like Svema, as opposed to no make at all."""
        return bool(_key(raw_make)) and _key(raw_make) not in self.film_make_words

    def _film_hit(self, f: Film) -> _Hit:
        return _Hit(
            make=f.make.lower().strip(), name=f.type.lower().strip(), speed=f.speed
        )

    def _film_unmatched(
        self, mention: Dict, make: Optional[str], tkey: str, speed: Optional[int]
    ) -> _Hit:
        raw = mention.get("text") or " ".join(
            p for p in (mention.get("make"), mention.get("type")) if p
        )
        mention_obj = UnmatchedMention(
            kind="film",
            raw=raw,
            key=_key(
                f"{make or mention.get('make') or ''} {mention.get('type') or ''}"
            ),
            make=make or (mention.get("make") or "").lower() or None,
            type=(mention.get("type") or "").lower() or None,
            speed=speed,
        )
        return _Hit(
            make=make,
            speed=speed,
            flag="catalog_make_only" if make else None,
            unmatched=mention_obj,
        )

    def _fuzzy(self, key: str, index: Dict[str, List]):
        scores = sorted(
            ((fuzz.ratio(key, k), k) for k in index if k),
            reverse=True,
        )
        scores = [
            (score, k)
            for score, k in scores
            if _digits(k) == _digits(key)
            and not k.startswith(key)
            and not key.startswith(k)
        ]
        if not scores or scores[0][0] < FUZZY_MIN:
            return None
        if len(scores) > 1 and scores[0][0] - scores[1][0] < FUZZY_GAP:
            return "ambiguous"
        entries = _unique(index[scores[0][1]])
        return entries[0] if len(entries) == 1 else "ambiguous"


def grounded(name: str, mention: Dict, squashed: str, tokens: set, kind: str) -> bool:
    """The matched name, or what the LLM copied for it, is in the post's text."""
    field = "model" if kind == "camera" else "type"
    for candidate in (mention.get(field), mention.get("text"), name):
        key = _key(candidate)
        if not key:
            continue
        if len(key) >= 3 and key in squashed:
            return True
        if key in tokens:
            return True
    return False


def valid_speed(speed) -> Optional[int]:
    try:
        speed = int(speed)
    except (TypeError, ValueError):
        return None
    return speed if speed in MetadataExtractor.VALID_FILM_SPEEDS else None


def valid_focal_length(value) -> Optional[int]:
    try:
        value = int(float(value))
    except (TypeError, ValueError):
        return None
    low, high = MetadataExtractor.VALID_FOCAL_LENGTH_RANGE
    return value if low <= value <= high else None


def valid_aperture(value) -> Optional[str]:
    if value is None or value == "":
        return None
    text = str(value)
    m = re.search(r"1\s*:\s*(\d+(?:\.\d+)?)", text) or re.search(
        r"(\d+(?:\.\d+)?)", text
    )
    if not m:
        return None
    f = float(m.group(1))
    low, high = MetadataExtractor.VALID_APERTURE_RANGE
    if not low <= f <= high:
        return None
    return f"f/{int(f)}" if f == int(f) else f"f/{f}"
