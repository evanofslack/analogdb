import json
import re
from typing import List, Optional, Tuple

from analogdb.models import Camera, Film
from openai import OpenAI

from .models import PhotoMetadata


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

    def __init__(self, openai: OpenAI, llm_model: str):
        self.openai = openai
        self.llm_model = llm_model

    def extract(
        self,
        titles: List[str],
        films: List[Film],
        cameras: List[Camera],
    ) -> Tuple[List[PhotoMetadata], str]:
        ids: list[int] = list(range(1, len(titles) + 1))
        prompt = self._create_prompt(ids, titles, films, cameras)
        posts_raw = self._query_metadata_llm(prompt)
        posts = []
        for id, post, title in zip(ids, posts_raw, titles):
            posts.append(self._validate_metadata(post, id, title, films, cameras))
        return posts, prompt

    def _create_prompt(
        self,
        ids: List[int],
        titles: List[str],
        films: List[Film],
        cameras: List[Camera],
    ) -> str:
        prompt = "valid cameras:\n"
        prompt += json.dumps([camera.to_json_minimal() for camera in cameras])
        prompt += "\nvalid films:\n"
        prompt += json.dumps([film.to_json_minimal() for film in films])
        prompt += f"\nvalid film speeds: {sorted(self.VALID_FILM_SPEEDS)}\n"
        for id, title in zip(ids, titles):
            clean_title = title.replace("\n", " ").replace("\r", " ")
            prompt += "\n" + f"post_id: {id}, {clean_title}"
        return prompt

    def _query_metadata_llm(self, prompt: str) -> List[PhotoMetadata]:
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
            return [PhotoMetadata()]

        content = content.strip()
        if content.startswith("```"):
            content = re.sub(r"^```(?:json)?\s*", "", content)
            content = re.sub(r"\s*```$", "", content)

        try:
            j = json.loads(content)
            return self._parse_metadata_llm(j)
        except json.JSONDecodeError:
            return [PhotoMetadata()]

    def _parse_metadata_llm(self, json_str: str) -> list[PhotoMetadata]:
        """Parse JSON string to list of PhotoMetadata with manual field mapping."""
        try:
            data = json.loads(json_str) if isinstance(json_str, str) else json_str
            # response_format=json_object wraps arrays in an object — unwrap any list value
            if isinstance(data, dict):
                for v in data.values():
                    if isinstance(v, list):
                        data = v
                        break
                else:
                    return [PhotoMetadata()]
            if not isinstance(data, list):
                return [PhotoMetadata()]

            metadata_list = []
            for item in data:
                raw_id = item.get("post_id")
                metadata = PhotoMetadata(
                    post_id=int(raw_id) if raw_id is not None else None,
                    camera_make=item.get("camera_make"),
                    camera_model=item.get("camera_model"),
                    film_make=item.get("film_make"),
                    film_type=item.get("film_type"),
                    film_speed=item.get("film_speed"),
                    focal_length=item.get("focal_length"),
                    aperture=item.get("aperture"),
                )
                metadata_list.append(metadata)
            return metadata_list
        except (json.JSONDecodeError, TypeError, KeyError):
            return [PhotoMetadata()]

    def _validate_metadata(
        self,
        metadata: PhotoMetadata,
        id: int,
        title: str,
        films: List[Film],
        cameras: List[Camera],
    ) -> PhotoMetadata:
        clean = PhotoMetadata()
        # parse error, post id sanity check didn't match
        if metadata.post_id != id:
            return clean

        title = title.lower()

        clean.camera_make = self._validate_camera_make(metadata.camera_make, cameras)
        clean.camera_model = self._validate_camera_model(metadata.camera_model, cameras)
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

        # lookup make from type
        if film_type and film_make is None:
            for film in films:
                if film.type == film_type:
                    film_make = film.make

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
