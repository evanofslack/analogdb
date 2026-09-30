from dataclasses import dataclass, field
from typing import Any, Dict, List, Tuple

import analogdb.models as adb
from scrape.catalog_match import CatalogAlias
from scrape.normalize import normalize_key

Entry = Dict[str, Any]


def _clean(value: Any) -> Any:
    return value.strip().lower() if isinstance(value, str) else value


def camera_key(make: str, model: str) -> Tuple[str, str]:
    return (_clean(make), _clean(model))


def film_key(make: str, type: str, speed: int) -> Tuple[str, str, int]:
    return (_clean(make), _clean(type), speed)


@dataclass
class MatchingCatalog:
    cameras: List[adb.Camera]
    films: List[adb.Film]
    aliases: List[CatalogAlias]
    # names of JSON entries that aren't live yet, left out of matching
    not_live: List[str] = field(default_factory=list)
    # names of live entries the JSON doesn't have, still matched
    not_in_json: List[str] = field(default_factory=list)


def catalog_for_matching(
    camera_entries: List[Entry],
    film_entries: List[Entry],
    live_cameras: List[adb.Camera],
    live_films: List[adb.Film],
) -> MatchingCatalog:
    """The live catalog plus the JSON aliases. Posts must only get values the
    site has, so JSON entries that aren't uploaded yet are left out. Live entries
    that the JSON lists as an alias of another entry are left out too, so the
    alias target wins (n2020 matches f501)."""
    json_cameras = {camera_key(e["make"], e["model"]): e for e in camera_entries}
    json_films = {film_key(e["make"], e["type"], e["speed"]): e for e in film_entries}
    live_camera_keys = {camera_key(c.make, c.model) for c in live_cameras}
    live_film_keys = {film_key(f.make, f.type, f.speed) for f in live_films}

    aliases: List[CatalogAlias] = []
    merged_cameras = set()
    merged_films = set()
    for key, e in json_cameras.items():
        if key not in live_camera_keys:
            continue
        for alias in e.get("aliases") or []:
            aliases.append(CatalogAlias("camera", key[0], key[1], alias))
            merged_cameras.add(camera_key(key[0], alias))
    for key, e in json_films.items():
        if key not in live_film_keys:
            continue
        for alias in e.get("aliases") or []:
            aliases.append(CatalogAlias("film", key[0], key[1], alias))
            merged_films.add((key[0], _clean(alias)))

    cameras = [
        c for c in live_cameras if camera_key(c.make, c.model) not in merged_cameras
    ]
    films = [
        f for f in live_films if (_clean(f.make), _clean(f.type)) not in merged_films
    ]

    not_live = [" ".join(k) for k in json_cameras if k not in live_camera_keys]
    not_live += [f"{k[0]} {k[1]}" for k in json_films if k not in live_film_keys]
    not_in_json = [
        f"{c.make} {c.model}"
        for c in cameras
        if camera_key(c.make, c.model) not in json_cameras
    ]
    not_in_json += [
        f"{f.make} {f.type}"
        for f in films
        if film_key(f.make, f.type, f.speed) not in json_films
    ]
    return MatchingCatalog(
        cameras, films, aliases, sorted(not_live), sorted(not_in_json)
    )


def validate_aliases(
    camera_entries: List[Entry], film_entries: List[Entry]
) -> List[str]:
    """Problems with the JSON aliases: an alias used twice, or one that is
    another entry's own name, or the entry's own name."""
    problems: List[str] = []
    for kind, entries, name_field in (
        ("camera", camera_entries, "model"),
        ("film", film_entries, "type"),
    ):
        names = {
            normalize_key(f"{e['make']} {e[name_field]}"): e[name_field]
            for e in entries
        }
        seen: Dict[str, str] = {}
        for e in entries:
            for alias in e.get("aliases") or []:
                key = normalize_key(f"{e['make']} {alias}")
                target = f"{e['make']} {e[name_field]}"
                if not isinstance(alias, str) or not alias.strip():
                    problems.append(f"{kind} {target}: empty alias")
                elif key in names:
                    problems.append(
                        f"{kind} {target}: alias {alias!r} is the name of {e['make']} {names[key]}"
                    )
                elif key in seen:
                    problems.append(
                        f"{kind} {target}: alias {alias!r} also used by {seen[key]}"
                    )
                else:
                    seen[key] = target
    return problems


@dataclass
class UploadPlan:
    new: List[Entry]
    updated: List[Entry]
    unchanged: int
    # live entries the JSON doesn't have
    not_in_json: List[str]


def camera_upload_plan(entries: List[Entry], live: List[adb.Camera]) -> UploadPlan:
    live_by_key = {camera_key(c.make, c.model): c for c in live}
    return _upload_plan(
        entries,
        live_by_key,
        lambda e: camera_key(e["make"], e["model"]),
        lambda e, c: e["description"] != c.description,
        lambda k: " ".join(k),
    )


def film_upload_plan(entries: List[Entry], live: List[adb.Film]) -> UploadPlan:
    live_by_key = {film_key(f.make, f.type, f.speed): f for f in live}
    return _upload_plan(
        entries,
        live_by_key,
        lambda e: film_key(e["make"], e["type"], e["speed"]),
        lambda e, f: e["description"] != f.description
        or e["color_type"] != f.color_type,
        lambda k: f"{k[0]} {k[1]} {k[2]}",
    )


def _upload_plan(entries, live_by_key, key_of, changed, name_of) -> UploadPlan:
    new: List[Entry] = []
    updated: List[Entry] = []
    unchanged = 0
    seen = set()
    for e in entries:
        key = key_of(e)
        if key in seen:
            continue
        seen.add(key)
        current = live_by_key.get(key)
        if current is None:
            new.append(e)
        elif changed(e, current):
            updated.append(e)
        else:
            unchanged += 1
    not_in_json = sorted(name_of(k) for k in live_by_key if k not in seen)
    return UploadPlan(new, updated, unchanged, not_in_json)
