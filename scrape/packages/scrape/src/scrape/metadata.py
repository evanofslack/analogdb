import json
import re
from typing import Dict, List, Optional, Tuple

from analogdb.models import Camera, Film
from openai import OpenAI, OpenAIError

from .models import ExtractResult, PhotoMetadata


def _film_json(f: Film) -> Dict:
    return {"type": f.type, "make": f.make, "speed": f.speed}


def _camera_json(c: Camera) -> Dict:
    return {"make": c.make, "model": c.model}


class MetadataExtractor:
    VALID_FILM_SPEEDS = {
        1,
        2,
        3,
        6,
        12,
        20,
        25,
        50,
        64,
        80,
        100,
        125,
        160,
        200,
        250,
        320,
        400,
        500,
        800,
        1000,
        1600,
        3200,
        6400,
    }
    VALID_FOCAL_LENGTH_RANGE = (8, 800)  # 8mm to 800mm
    VALID_APERTURE_RANGE = (0.7, 32.0)  # f/0.7 to f/32
    MAX_ATTEMPTS = 2

    SYSTEM_PROMPT = """You are a photo metadata extraction assistant. Extract specific technical information from photo post titles and return as a JSON array. Only extract explicitly mentioned or clearly implied information. Leave fields null rather than guess. Accuracy with fewer fields is better than inaccuracy. Metadata is more likely to be inside containers like '[]' or '()' and may be separated by space, commas, /, or | characters.

A numerical post_id is provided at the start of each title — repeat it back as an integer in the output for cross-reference.

Return a JSON array of objects with this shape:
[
  {
    "post_id": 1,
    "camera_make": "camera brand name",
    "camera_model": "camera model name",
    "film_make": "film brand name",
    "film_type": "film type name",
    "film_speed": 400,
    "focal_length": 50,
    "aperture": "f/2.8"
  }
]

Extraction rules:
- If multiple values exist, use only the first one
- camera_make: Brand only (e.g., "Hasselblad", "Canon", "Nikon", "Mamiya"). Match valid camera list.
- camera_model: Model after make (e.g., "500cm", "AE-1", "F4", "RB67 Pro-S"). Match valid camera list.
- film_make: Manufacturer (e.g., "Kodak", "Fuji", "Ilford"). Match valid film list.
- film_type: Specific film (e.g., "Portra", "Ektachrome", "HP5"). Match valid film list.
- film_speed: ISO as integer (e.g., 400, 800). For push/pull like "400@800", use base speed (400)
- focal_length: Lens focal length in mm as integer (e.g., 50, 85). For ranges like "20-35mm", use first number (20)
- aperture: F-stop as "f/X.X" format (e.g., "f/2.8", "f/4")
- Use null for missing data

Validation rules:
- Exact matching first: Always prioritize exact matches from the valid lists
- Fuzzy matching: If no exact match, use these strategies:
- Handle common abbreviations: "AE1" → "AE-1", "RB67" → "RB67 Pro-S", "Lomo" -> "Lomography"
- Ignore case differences: "portra" → "Portra"
- Handle missing/extra spaces: "Tri X" → "Tri-X"
- Accept partial model names if unambiguous: "500c" → "500cm" (only if one match exists)
- Handle common typos: "Hasselblad" variations, "Mamiya" vs "Mamya"
- Handle assumed or unspecified makes: "Gold" -> "Kodak Gold"
- Understand common alternative names: Nikon cameras prefix F is the same as N (F90 == N90)
- Understand common abbreviations: P for Program (cameras)
- Understand common alternative names: color == colour, minolta Maxxum, Dynax, Alpha (a) are equivalent
- Understand common alternative film names: Portrait -> Porta, ORWO -> Original Wolfen
- Cross-reference completion: Use valid lists to fill missing information
- If camera model found but not make, match from valid camera list
- If film type found but not make, match from valid film list
- If camera make found but camera model not matched, ok to just set camera make
- If film make found but film type not matched, ok to just set film make"""

    def __init__(self, openai: OpenAI, llm_model: str, batch_size: int = 25):
        self.openai = openai
        self.llm_model = llm_model
        self.batch_size = batch_size

    def extract(
        self,
        titles: List[str],
        films: List[Film],
        cameras: List[Camera],
    ) -> ExtractResult:
        results = [PhotoMetadata() for _ in titles]
        failed = 0
        for start in range(0, len(titles), self.batch_size):
            chunk = titles[start : start + self.batch_size]
            ids = list(range(1, len(chunk) + 1))
            by_id = self._query_chunk(ids, chunk, films, cameras)
            for id, title in zip(ids, chunk):
                m = by_id.get(id)
                if m is None:
                    failed += 1
                    continue
                results[start + id - 1] = self._validate_metadata(
                    m, title, films, cameras
                )
        return ExtractResult(metadata=results, failed=failed)

    def _create_prompt(
        self,
        ids: List[int],
        titles: List[str],
        films: List[Film],
        cameras: List[Camera],
    ) -> str:
        prompt = "valid cameras:\n"
        prompt += json.dumps([_camera_json(camera) for camera in cameras])
        prompt += "\nvalid films:\n"
        prompt += json.dumps([_film_json(film) for film in films])
        prompt += f"\nvalid film speeds: {sorted(self.VALID_FILM_SPEEDS)}\n"
        for id, title in zip(ids, titles):
            clean_title = title.replace("\n", " ").replace("\r", " ")
            prompt += "\n" + f"post_id: {id}, {clean_title}"
        return prompt

    def _query_chunk(
        self,
        ids: List[int],
        titles: List[str],
        films: List[Film],
        cameras: List[Camera],
    ) -> Dict[int, PhotoMetadata]:
        prompt = self._create_prompt(ids, titles, films, cameras)
        for _ in range(self.MAX_ATTEMPTS):
            try:
                items = self._query_metadata_llm(prompt)
            except (OpenAIError, json.JSONDecodeError):
                continue
            if items is not None:
                return self._parse_metadata_llm(items, ids)
        return {}

    def _query_metadata_llm(self, prompt: str) -> Optional[List]:
        resp = self.openai.chat.completions.create(
            extra_headers={
                "HTTP-Referer": "analogdb.com",
                "X-Title": "analogdb.com",
            },
            model=self.llm_model,
            messages=[
                {"role": "system", "content": self.SYSTEM_PROMPT},
                {"role": "user", "content": prompt},
            ],
            temperature=0,
            response_format={"type": "json_object"},
        )

        content = resp.choices[0].message.content
        if content is None:
            return None

        content = content.strip()
        if content.startswith("```"):
            content = re.sub(r"^```(?:json)?\s*", "", content)
            content = re.sub(r"\s*```$", "", content)

        data = json.loads(content)
        # response_format=json_object wraps arrays in an object — unwrap any list value
        if isinstance(data, dict):
            data = next((v for v in data.values() if isinstance(v, list)), None)
        if not isinstance(data, list):
            return None
        return data

    def _parse_metadata_llm(
        self, items: List, ids: List[int]
    ) -> Dict[int, PhotoMetadata]:
        by_id: Dict[int, PhotoMetadata] = {}
        for item in items:
            if not isinstance(item, dict):
                continue
            id = self._parse_post_id(item.get("post_id"))
            if id not in ids or id in by_id:
                continue
            by_id[id] = PhotoMetadata(
                post_id=id,
                camera_make=item.get("camera_make"),
                camera_model=item.get("camera_model"),
                film_make=item.get("film_make"),
                film_type=item.get("film_type"),
                film_speed=item.get("film_speed"),
                focal_length=item.get("focal_length"),
                aperture=item.get("aperture"),
            )
        return by_id

    def _parse_post_id(self, raw_id) -> Optional[int]:
        if isinstance(raw_id, bool):
            return None
        if isinstance(raw_id, int):
            return raw_id
        if isinstance(raw_id, str) and raw_id.strip().isdigit():
            return int(raw_id.strip())
        return None

    def _validate_metadata(
        self,
        metadata: PhotoMetadata,
        title: str,
        films: List[Film],
        cameras: List[Camera],
    ) -> PhotoMetadata:
        clean = PhotoMetadata()

        title = title.lower()

        clean.camera_make = self._validate_camera_make(metadata.camera_make, cameras)
        clean.camera_model = self._validate_camera_model(metadata.camera_model, cameras)
        # drop model if the make/model pair is not a known camera
        if clean.camera_make is not None and clean.camera_model is not None:
            pairs = {
                (camera.make.lower().strip(), camera.model.lower().strip())
                for camera in cameras
            }
            if (clean.camera_make, clean.camera_model) not in pairs:
                clean.camera_model = None
        # lookup make from model (only if exact make match)
        if clean.camera_model is not None and clean.camera_make is None:
            matching_cameras = [
                camera
                for camera in cameras
                if camera.model.lower().strip() == clean.camera_model
            ]
            if len(matching_cameras) == 1:
                clean.camera_make = matching_cameras[0].make

        clean.film_make, clean.film_type = self._validate_film_info(
            metadata.film_make, metadata.film_type, films
        )

        clean.film_speed = self._validate_film_speed(
            metadata.film_speed, clean.film_make, clean.film_type, films
        )

        clean.focal_length = self._validate_focal_length(metadata.focal_length)
        clean.aperture = self._validate_aperture(metadata.aperture)

        return clean

    def _validate_camera_make(
        self, make: Optional[str], cameras: List[Camera]
    ) -> Optional[str]:
        if not make:
            return None

        clean_make = make.lower().strip()
        makes = {camera.make.lower().strip() for camera in cameras}
        if clean_make in makes:
            return clean_make
        return None

    def _validate_camera_model(
        self, model: Optional[str], cameras: List[Camera]
    ) -> Optional[str]:
        if not model:
            return None

        clean_model = model.lower().strip()
        clean_model = re.sub(r"\s+", " ", clean_model)  # Normalize spaces
        clean_model = re.sub(
            r"[|\[\],/]+$", "", clean_model
        ).strip()  # Remove trailing separators

        models = {camera.model.lower().strip() for camera in cameras}
        if clean_model in models:
            return clean_model

        return None

    def _validate_film_info(
        self,
        film_make: Optional[str],
        film_type: Optional[str],
        films: List[Film],
    ) -> Tuple[Optional[str], Optional[str]]:
        """Validate film make and type consistency."""
        if not film_type and not film_make:
            return None, None

        makes = {film.make.lower().strip() for film in films}
        types = {film.type.lower().strip() for film in films}

        if film_make:
            film_make = film_make.lower().strip()
        if film_type:
            film_type = film_type.lower().strip()

        if film_make and film_make not in makes:
            film_make = None

        if film_type and film_type not in types:
            film_type = None

        # lookup make from type (only if exactly one make has that type)
        if film_type and film_make is None:
            matching_makes = {
                film.make.lower().strip()
                for film in films
                if film.type.lower().strip() == film_type
            }
            if len(matching_makes) == 1:
                film_make = matching_makes.pop()

        return film_make, film_type

    def _validate_film_speed(
        self,
        speed: Optional[int],
        film_make: Optional[str],
        film_type: Optional[str],
        films: List[Film],
    ) -> Optional[int]:
        """Validate film speed against known values."""
        # lookup speed from film make/type (only if exact type match)
        if film_type is not None and film_make is not None:
            matching_films = [
                film
                for film in films
                if film.type.lower().strip() == film_type
                and film.make.lower().strip() == film_make
            ]
            if len(matching_films) == 1:
                return matching_films[0].speed

        # Didn't get any speed
        if speed is None:
            return None

        if isinstance(speed, str):
            try:
                speed = int(speed)
            except ValueError:
                return None

        if speed in self.VALID_FILM_SPEEDS:
            return speed

        return None

    def _validate_focal_length(self, focal_length: Optional[int]) -> Optional[int]:
        """Validate focal length is within reasonable range."""
        if focal_length is None:
            return None

        if isinstance(focal_length, str):
            try:
                focal_length = int(focal_length)
            except ValueError:
                return None

        if (
            self.VALID_FOCAL_LENGTH_RANGE[0]
            <= focal_length
            <= self.VALID_FOCAL_LENGTH_RANGE[1]
        ):
            return focal_length

        return None

    def _validate_aperture(self, aperture: Optional[str]) -> Optional[str]:
        """Validate and normalize aperture value."""
        if not aperture:
            return None

        aperture_str = str(aperture).strip().lower()

        # Try to extract numeric value from various formats
        patterns = [
            r"f[/]?(\d+(?:\.\d+)?)",  # f/2.8, f2.8, f 2.8
            r"(\d+(?:\.\d+)?)$",  # Just the number
        ]

        for pattern in patterns:
            match = re.search(pattern, aperture_str)
            if match:
                try:
                    f_number = float(match.group(1))

                    # Validate range
                    if (
                        self.VALID_APERTURE_RANGE[0]
                        <= f_number
                        <= self.VALID_APERTURE_RANGE[1]
                    ):
                        # Normalize format
                        if f_number == int(f_number):
                            return f"f/{int(f_number)}"
                        else:
                            return f"f/{f_number}"

                except ValueError:
                    continue

        return None
