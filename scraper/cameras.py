import re
from dataclasses import dataclass
from typing import Optional, Dict
from models import PhotoMetadata


def extract_metadata(title: str) -> PhotoMetadata:
    """Extract photo metadata from a post title."""
    title_lower = title.lower()
    metadata = PhotoMetadata()

    # Extract camera make and model
    camera_info = _extract_camera_info(title, title_lower)
    metadata.camera_make = camera_info.get("make")
    metadata.camera_model = camera_info.get("model")

    # Extract film information
    film_info = _extract_film_info(title, title_lower)
    metadata.film_make = film_info.get("make")
    metadata.film_type = film_info.get("type")
    metadata.film_speed = film_info.get("speed")

    # Extract focal length
    metadata.focal_length = _extract_focal_length(title)

    # Extract aperture
    metadata.aperture = _extract_aperture(title)

    return metadata


def _get_camera_makes():
    """Get list of camera makes."""
    return [
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


def _get_film_types():
    """Get list of film types."""
    return [
        "portra",
        "kodak gold",
        "ektachrome",
        "ilford",
        "ektar",
        "hp5",
        "fp4",
        "tri-x",
        "tmax",
        "delta",
        "fomapan",
        "kentmere",
        "lomography",
        "fujipro",
        "superia",
        "provia",
        "velvia",
        "vision",
        "colorplus",
        "ultramax",
        "gold",
    ]


def _get_film_makes():
    """Get list of film makes."""
    return [
        "kodak",
        "ilford",
        "fuji",
        "fujifilm",
        "lomography",
        "foma",
        "kentmere",
        "agfa",
        "rollei",
    ]


def _extract_camera_info(title: str, title_lower: str) -> dict:
    """Extract camera make and model from title."""
    camera_info: Dict[str, Optional[str]] = {"make": None, "model": None}
    camera_makes = _get_camera_makes()

    # Find camera make
    for make in camera_makes:
        # Use word boundaries to avoid partial matches
        pattern = r"\b" + re.escape(make) + r"\b"
        match = re.search(pattern, title_lower)
        if match:
            camera_info["make"] = make.capitalize()

            # Extract model after the make
            model = _extract_camera_model(title, match.end(), make)
            if model:
                camera_info["model"] = model
            break

    return camera_info


def _extract_camera_model(title: str, start_pos: int, make: str) -> Optional[str]:
    """Extract camera model after the make."""
    # Look for model after the make
    remaining_text = title[start_pos:].strip()

    # Common patterns for camera models
    patterns = [
        r"^\s+([A-Za-z0-9\-\+\s]+?)(?=\s*[|\[\],/]|\s+\d+mm|\s+f[\d/.]|\s*$)",
        r"^\s+([A-Za-z0-9\-\+\s]{2,20}?)(?=\s*[|\[\],])",
    ]

    for pattern in patterns:
        match = re.search(pattern, remaining_text)
        if match:
            model = match.group(1).strip()
            # Clean up common artifacts
            model = re.sub(r"\s+", " ", model)
            if len(model) > 1 and not model.isdigit():
                return model

    return None


def _extract_film_info(title: str, title_lower: str) -> dict:
    """Extract film make, type, and speed."""
    film_info: Dict[str, Optional[str | int]] = {
        "make": None,
        "type": None,
        "speed": None,
    }

    film_types = _get_film_types()
    film_makes = _get_film_makes()

    # First, try to find film types (which often include the make)
    for film_type in sorted(film_types, key=len, reverse=True):
        pattern = r"\b" + re.escape(film_type) + r"\b"
        match = re.search(pattern, title_lower)
        if match:
            film_info["type"] = film_type.title()

            # Extract speed if it follows the film type
            speed = _extract_film_speed_near_position(title, match.end())
            if speed:
                film_info["speed"] = speed

            # Determine make from film type
            film_info["make"] = _get_film_make_from_type(film_type)
            break

    # If no film type found, look for standalone film makes
    if not film_info["type"]:
        for make in film_makes:
            pattern = r"\b" + re.escape(make) + r"\b"
            match = re.search(pattern, title_lower)
            if match:
                film_info["make"] = make.capitalize()
                # Try to extract speed near the make
                speed = _extract_film_speed_near_position(title, match.end())
                if speed:
                    film_info["speed"] = speed
                break

    # If still no speed found, look for standalone speed patterns
    if not film_info["speed"]:
        film_info["speed"] = _extract_film_speed(title)

    return film_info


def _extract_film_speed_near_position(title: str, pos: int) -> Optional[int]:
    """Extract film speed near a given position in the title."""
    # Look within 10 characters after the position
    search_area = title[pos : pos + 10]

    # Pattern for film speed (numbers 25-6400)
    speed_match = re.search(
        r"\b(100|200|400|800|1000|1600)\b",
        search_area,
    )
    if speed_match:
        return int(speed_match.group(1))

    return None


def _extract_film_speed(title: str) -> Optional[int]:
    """Extract film speed from anywhere in the title."""
    # Look for common film speeds
    speed_match = re.search(
        r"\b(100|200|400|800|1000|1600)\b",
        title,
    )
    if speed_match:
        return int(speed_match.group(1))
    return None


def _get_film_make_from_type(film_type: str) -> str:
    """Determine film make from film type."""
    kodak_films = [
        "portra",
        "ektar",
        "kodak gold",
        "ektachrome",
        "tri-x",
        "tmax",
        "vision",
        "gold",
        "colorplus",
        "ultramax",
    ]
    ilford_films = ["hp5", "fp4", "delta"]
    fuji_films = ["fujipro", "superia", "provia", "velvia"]

    if film_type.lower() in kodak_films:
        return "Kodak"
    elif film_type.lower() in ilford_films:
        return "Ilford"
    elif film_type.lower() in fuji_films:
        return "Fuji"
    elif film_type.lower() == "fomapan":
        return "Foma"
    elif film_type.lower() == "kentmere":
        return "Kentmere"
    elif film_type.lower() == "lomography":
        return "Lomography"

    return film_type.split()[0].capitalize()


def _extract_focal_length(title: str) -> Optional[int]:
    """Extract focal length in mm."""
    # Pattern for focal length (number followed by mm)
    focal_match = re.search(r"\b(\d+)mm\b", title)
    if focal_match:
        return int(focal_match.group(1))
    return None


def _extract_aperture(title: str) -> Optional[str]:
    """Extract aperture value."""
    # Patterns for aperture (f/2.8, f2.8, f/2, f4, etc.)
    aperture_patterns = [
        r"\bf/(\d+(?:\.\d+)?)\b",  # f/2.8, f/2
        r"\bf(\d+(?:\.\d+)?)\b",  # f2.8, f4
    ]

    for pattern in aperture_patterns:
        match = re.search(pattern, title)
        if match:
            return f"f/{match.group(1)}"

    return None

