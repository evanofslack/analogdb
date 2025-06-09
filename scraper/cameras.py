import re
from typing import Optional, Tuple
from models import PhotoMetadata

camera_makes = [
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

films = {
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

generic_film_types_lomo = ["purple", "turquoise", "metropolis"]
generic_film_types = ["gold", "pan", "class", "creative", "action"]

film_types = set(films.keys())
film_makes = set(films.values())

# Valid ranges for validation
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


def validate_metadata(metadata: PhotoMetadata, title: str) -> PhotoMetadata:
    cleaned = PhotoMetadata()
    title = title.lower()

    # Validate and clean camera make
    cleaned.camera_make = _validate_camera_make(metadata.camera_make)

    # Validate camera model (basic cleanup)
    cleaned.camera_model = _validate_camera_model(metadata.camera_model)

    # Validate and clean film information
    cleaned.film_make, cleaned.film_type = _validate_film_info(
        metadata.film_make, metadata.film_type, title
    )

    # Validate film speed
    cleaned.film_speed = _validate_film_speed(metadata.film_speed)

    # Validate focal length
    cleaned.focal_length = _validate_focal_length(metadata.focal_length)

    # Validate aperture
    cleaned.aperture = _validate_aperture(metadata.aperture)

    return cleaned


def _validate_camera_make(make: Optional[str]) -> Optional[str]:
    if not make:
        return None

    return make.lower().strip()


def _validate_camera_model(model: Optional[str]) -> Optional[str]:
    """Basic validation and cleaning of camera model."""
    if not model:
        return None

    cleaned_model = model.lower().strip()

    # Basic cleaning
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
    film_make: Optional[str], film_type: Optional[str], title: str = ""
) -> Tuple[Optional[str], Optional[str]]:
    """Validate film make and type consistency."""
    if not film_type and not film_make:
        return None, None

    # Clean inputs
    if film_make:
        film_make = film_make.lower().strip()
    if film_type:
        film_type = film_type.lower().strip()

    # If we have a film type, validate it
    if film_type:
        if film_type in film_types:
            expected_make = films[film_type]

            # Handle generic film types that need make validation
            if film_type in generic_film_types_lomo:
                if title and "lomo" not in title:
                    return None, None

            if film_type in generic_film_types:
                if title and expected_make not in title:
                    return None, None

            # Return the expected make for this film type
            return expected_make, film_type

    return film_make, film_type


def _validate_film_speed(speed: Optional[int]) -> Optional[int]:
    """Validate film speed against known values."""
    if speed is None:
        return None

    if isinstance(speed, str):
        try:
            speed = int(speed)
        except ValueError:
            return None

    if speed in VALID_FILM_SPEEDS:
        return speed

    if 25 <= speed <= 6400:
        return speed

    return None


def _validate_focal_length(focal_length: Optional[int]) -> Optional[int]:
    """Validate focal length is within reasonable range."""
    if focal_length is None:
        return None

    if isinstance(focal_length, str):
        try:
            focal_length = int(focal_length)
        except ValueError:
            return None

    if VALID_FOCAL_LENGTH_RANGE[0] <= focal_length <= VALID_FOCAL_LENGTH_RANGE[1]:
        return focal_length

    return None


def _validate_aperture(aperture: Optional[str]) -> Optional[str]:
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
                if VALID_APERTURE_RANGE[0] <= f_number <= VALID_APERTURE_RANGE[1]:
                    # Normalize format
                    if f_number == int(f_number):
                        return f"f/{int(f_number)}"
                    else:
                        return f"f/{f_number}"

            except ValueError:
                continue

    return None
