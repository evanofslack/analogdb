import json
import re
from typing import Dict, List, Optional, Tuple

from analogdb.models import Camera, Film
from openai import OpenAI

from .models import PhotoMetadata


class MetadataExtractor:
    VALID_FILM_SPEEDS = {
        25,
        50,
        64,
        100,
        125,
        160,
        200,
        320,
        400,
        800,
        1000,
        1600,
        3200,
        6400,
    }
    VALID_FOCAL_LENGTH_RANGE = (8, 800)  # 8mm to 800mm
    VALID_APERTURE_RANGE = (0.7, 32.0)  # f/0.7 to f/32

    PROMT = """
system prompt:
You are a photo metadata extraction assistant. Extract specific technical information from photo post titles and return as JSON. Only extract explicitly mentioned or clearly implied information. Leave fields blank rather than guess. Accuracy with fewer fields is better than inaccuracy. Metadata is more likely to be inside of containers like '[]' or '()' and may be separated by space or | characters. You will be provided with a list of valid cameras in json form, valid films in json form, and then a list of post titles to extract metadata from.

Extract the following information and return as array of JSON:
{{
  "camera_make": "camera brand name",
  "camera_model": "camera model name", 
  "film_make": "film brand name",
  "film_type": "film type name",
  "film_speed": integer_iso,
  "focal_length": integer_mm,
  "aperture": "f/X.X"
}}

Extraction rules:
- If multiple values exist, use only the first one
- camera_make: Brand only (e.g., "Hasselblad", "Canon", "Nikon", "Mamiya"). Match valid camera list json.
- camera_model: Model after make (e.g., "500cm", "AE-1", "F4", "RB67 Pro-S"). Match valid camera list json.
- film_make: Manufacturer (e.g., "Kodak", "Fuji", "Ilford"). Match valid film list json.
- film_type: Specific film (e.g., "Portra", "Ektachrome", "HP5"). Match valid film list json.
- film_speed: ISO as integer (e.g., 400, 800). For push/pull like "400@800", use base speed (400)
- focal_length: Lens focal length in mm as integer (e.g., 50, 85). For ranges like "20-35mm", use first number (20)
- aperture: F-stop as "f/X.X" format (e.g., "f/2.8", "f/4")
- Use null for missing data

Validation rules:
- Exact matching first: Always prioritize exact matches from the valid lists
- Fuzzy matching: If no exact match, use these strategies:
- Handle common abbreviations: "AE1" → "AE-1", "RB67" → "RB67 Pro-S"
- Ignore case differences: "portra" → "Portra"
- Handle missing/extra spaces: "Tri X" → "Tri-X"
- Accept partial model names if unambiguous: "500c" → "500cm" (only if one match exists)
- Handle common typos: "Hasselblad" variations, "Mamiya" vs "Mamya"
- Cross-reference completion: Use valid lists to fill missing information
- If camera model found but not make, match from valid camera list
- If film type found but not make, match from valid film list

"""

    def __init__(self, openai: OpenAI, llm_model: str):
        self.openai = openai
        self.llm_model = llm_model

    def extract(
        self, titles: List[str], films: List[Film], cameras: List[Camera]
    ) -> Tuple[List[PhotoMetadata], str]:
        prompt = self._create_prompt(titles, films, cameras)
        posts_raw = self._query_metadata_llm(prompt)
        posts = []
        for post, title in zip(posts_raw, titles):
            posts.append(self._validate_metadata(post, title, films, cameras))
        return posts, prompt

    def _create_prompt(
        self, titles: List[str], films: List[Film], cameras: List[Camera]
    ) -> str:
        prompt = self.PROMT
        prompt += "\n valid cameras:"
        for camera in cameras:
            prompt += str(camera.to_json_minimal())
        prompt += "\n valid films:"
        for film in films:
            prompt += str(film.to_json_minimal())
        for i, title in enumerate(titles):
            prompt += "\n" + f"title #{i}: {title}"
        return prompt

    def _query_metadata_llm(self, prompt: str) -> List[PhotoMetadata]:
        resp = self.openai.chat.completions.create(
            extra_headers={
                "HTTP-Referer": "analogdb.com",
                "X-Title": "analogdb.com",
            },
            model=self.llm_model,
            messages=[{"role": "user", "content": prompt}],
            temperature=0,  # Low temperature for consistent extraction
        )

        content = resp.choices[0].message.content
        if content is None:
            return [PhotoMetadata()]

        try:
            j = json.loads(content)
            return self._parse_metadata_llm(j)
        except json.JSONDecodeError:
            return [PhotoMetadata()]

    def _parse_metadata_llm(self, json_str: str) -> list[PhotoMetadata]:
        """Parse JSON string to list of PhotoMetadata with manual field mapping."""
        try:
            data = json.loads(json_str) if isinstance(json_str, str) else json_str
            if not isinstance(data, list):
                return [PhotoMetadata()]

            metadata_list = []
            for item in data:
                metadata = PhotoMetadata(
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
        title: str,
        films: List[Film],
        cameras: List[Camera],
    ) -> PhotoMetadata:
        clean = PhotoMetadata()
        title = title.lower()

        clean.camera_make = self._validate_camera_make(metadata.camera_make, cameras)
        clean.camera_model = self._validate_camera_model(metadata.camera_model, cameras)
        # lookup make from model
        if clean.camera_model is not None and clean.camera_make is None:
            for camera in cameras:
                if camera.model.lower().strip() == clean.camera_model:
                    clean.camera_make = camera.make

        clean.film_make, clean.film_type = self._validate_film_info(
            metadata.film_make, metadata.film_type, films
        )

        clean.film_speed = self._validate_film_speed(metadata.film_speed)
        # lookup speed from film type
        if clean.film_type is not None and clean.film_speed is None:
            for film in films:
                if film.type.lower().strip() == clean.film_type:
                    clean.film_speed = film.speed

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

    def _validate_film_speed(self, speed: Optional[int]) -> Optional[int]:
        """Validate film speed against known values."""
        if speed is None:
            return None

        if isinstance(speed, str):
            try:
                speed = int(speed)
            except ValueError:
                return None

        if speed in self.VALID_FILM_SPEEDS:
            return speed

        if 25 <= speed <= 6400:
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
