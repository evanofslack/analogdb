import re
import json
from typing import Optional, Tuple, List, Dict
from openai import OpenAI
from .models import PhotoMetadata


class MetadataExtractor:
    CAMERAS: List[str] = [
        "hasselblad",
        "mamiya",
        "pentax",
        "canon",
        "nikon",
        "leica",
        "contax",
        "olympus",
        "minolta",
        "fuji",
        "fujifilm",
        "yashica",
        "bronica",
        "rollei",
        "zeiss",
        "zenza",
    ]

    FILMS: Dict[str, str] = {
        "cms": "adox",
        "scala": "adox",
        "silvermax": "adox",
        "apx": "agfa",
        "vista plus": "agfa",
        "vistaplus": "agfa",
        "800tungsten": "cinestill",
        "800t": "cinestill",
        "50daylight": "cinestill",
        "50d": "cinestill",
        "400dynamic": "cinestill",
        "400d": "cinestill",
        "delta": "ilford",
        "fp4": "ilford",
        "hp5": "ilford",
        "pan f": "ilford",
        "pan": "ilford",
        "sfx": "ilford",
        "xp2": "ilford",
        "class": "fomopan",
        "creative": "fomopan",
        "action": "fomopan",
        "c200": "fuji",
        "neopan": "fuji",
        "acros": "fuji",
        "provia": "fuji",
        "superia x-tra": "fuji",
        "superia premium": "fuji",
        "superia": "fuji",
        "velvia": "fuji",
        "pro400h": "fuji",
        "fujicolor": "fuji",
        "colorplus": "kodak",
        "kodacolor": "kodak",
        "ektachrome": "kodak",
        "ektar": "kodak",
        "gold": "kodak",
        "porta": "kodak",
        "proimage": "kodak",
        "t-max": "kodak",
        "tri-x": "kodak",
        "ultramax": "kodak",
        "purple": "lomography",
        "turquoise": "lomography",
        "metropolis": "lomography",
        "cn": "lomography",
        "earl grey": "lomography",
        "lady grey": "lomography",
        "lomochrome": "lomography",
        "lomo": "lomography",
    }

    GENERIC_FILM_TYPES_LOMO: List[str] = ["purple", "turquoise", "metropolis"]
    GENERIC_FILM_TYPES: List[str] = ["gold", "pan", "class", "creative", "action"]

    FILM_TYPES = set(FILMS.keys())
    FILM_MAKES = set(FILMS.values())

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
You are a photo metadata extraction assistant. Extract specific technical information from photo post titles and return as JSON. Only extract explicitly mentioned or clearly implied information. Leave fields blank rather than guess. Accuracy with fewer fields is better than inaccuracy.

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

Rules:
- If multiple values exist, use only the first one
- camera_make: Brand only (e.g., "Hasselblad", "Canon", "Nikon", "Mamiya")
- camera_model: Model after make (e.g., "500cm", "AE-1", "F4", "RB67 Pro-S")
- film_make: Manufacturer (e.g., "Kodak", "Fuji", "Ilford")
- film_type: Specific film (e.g., "Portra", "Ektachrome", "HP5")
- film_speed: ISO as integer (e.g., 400, 800). For push/pull like "400@800", use base speed (400)
- focal_length: Lens focal length in mm as integer (e.g., 50, 85). For ranges like "20-35mm", use first number (20)
- aperture: F-stop as "f/X.X" format (e.g., "f/2.8", "f/4")
- Use null for missing data
"""

    def __init__(self, openai: OpenAI, llm_model: str):
        self.openai = openai
        self.llm_model = llm_model

    def extract(self, titles: List[str]) -> List[PhotoMetadata]:
        prompt = self._create_prompt(titles)
        posts_raw = self._query_metadata_llm(prompt)
        posts = []
        for post, title in zip(posts_raw, titles):
            posts.append(self._validate_metadata(post, title))
        return posts

    def _create_prompt(self, titles: List[str]) -> str:
        prompt = self.PROMT
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

    def _validate_metadata(self, metadata: PhotoMetadata, title: str) -> PhotoMetadata:
        cleaned = PhotoMetadata()
        title = title.lower()

        # Camera, model
        cleaned.camera_make = self._validate_camera_make(metadata.camera_make)
        cleaned.camera_model = self._validate_camera_model(metadata.camera_model)

        # Film
        cleaned.film_make, cleaned.film_type = self._validate_film_info(
            metadata.film_make, metadata.film_type, title
        )

        # Numbers
        cleaned.film_speed = self._validate_film_speed(metadata.film_speed)
        cleaned.focal_length = self._validate_focal_length(metadata.focal_length)
        cleaned.aperture = self._validate_aperture(metadata.aperture)

        return cleaned

    def _validate_camera_make(self, make: Optional[str]) -> Optional[str]:
        if not make:
            return None

        return make.lower().strip()

    def _validate_camera_model(self, model: Optional[str]) -> Optional[str]:
        """Basic validation and cleaning of camera model."""
        if not model:
            return None

        cleaned_model = model.lower().strip()

        cleaned_model = re.sub(r"\s+", " ", cleaned_model)  # Normalize spaces
        cleaned_model = re.sub(
            r"[|\[\],/]+$", "", cleaned_model
        ).strip()  # Remove trailing separators

        # Reject if too short or all digits
        if len(cleaned_model) <= 1 or cleaned_model.isdigit():
            return None

        # Reject if too long (probably garbage)
        if len(cleaned_model) > 60:
            return None

        return cleaned_model

    def _validate_film_info(
        self, film_make: Optional[str], film_type: Optional[str], title: str = ""
    ) -> Tuple[Optional[str], Optional[str]]:
        """Validate film make and type consistency."""
        if not film_type and not film_make:
            return None, None

        if film_make:
            film_make = film_make.lower().strip()
        if film_type:
            film_type = film_type.lower().strip()

        # If we have a film type, validate it
        if film_type:
            if film_type in self.FILM_TYPES:
                expected_make = self.FILMS[film_type]

                # Handle generic film types that need make validation
                if film_type in self.GENERIC_FILM_TYPES_LOMO:
                    if title and "lomo" not in title:
                        return None, None

                if film_type in self.GENERIC_FILM_TYPES:
                    if title and expected_make not in title:
                        return None, None

                # Return the expected make for this film type
                return expected_make, film_type

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
