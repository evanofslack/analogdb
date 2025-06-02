from openai import OpenAI
from models import PhotoMetadata
from loguru import logger
import json


def extract_metadata(title: str, client: OpenAI, model: str) -> PhotoMetadata:
    prompt = f"""
system prompt:
You are a photo metadata extraction assistant. Your task is to extract specific technical information from photo post titles and return it as JSON. You should only extract information that is explicitly mentioned or clearly implied in the title. Title will not always have all fields (sometimes may have all or none or some). It is ok to leave fields blank. Accuracy with less fields is better than inaccurary with more fields extracted. 

user prompt:
Extract the following information from this photo post title and return it as JSON:

Title: "{title}"

Return only this JSON structure with extracted values (null if not found):

{{
  "camera_make": "camera brand name",
  "camera_model": "camera model name", 
  "film_make": "film brand name",
  "film_type": "film type name",
  "film_speed": integer_iso,
  "focal_length": integer_mm,
  "aperture": "f/X.X"
}}

Rules: Use standard brand names, integer values for speed/focal length, f/X.X format for aperture, null for missing data.


Extraction Guidelines:
If there are multiple cameras, models, films, speeds, apertures etc, use the first one and only the first one.
camera_make: Brand name only (e.g., "Hasselblad", "Canon", "Nikon", "Mamiya")
camera_model: Specific model after the make (e.g., "500cm", "AE-1", "F4", "RB67 Pro-S")
film_make: Film manufacturer (e.g., "Kodak", "Fuji", "Ilford", "Lomography")
film_type: Specific film name (e.g., "Portra", "Ektachrome", "HP5", "Gold")
film_speed: ISO number only as integer (e.g., 400, 800, 100). If push/pull processing like "400@800", use the base speed (400)
focal_length: Lens focal length in mm as integer (e.g., 50, 85, 200). For ranges like "20-35mm", use the first number (20)
aperture: F-stop value as string in format "f/X.X" (e.g., "f/2.8", "f/4", "f/1.4")
"""

    resp = client.chat.completions.create(
        extra_headers={
            "HTTP-Referer": "analogdb.com",
            "X-Title": "analogdb.com",
        },
        model=model,
        messages=[{"role": "user", "content": prompt}],
        temperature=0,  # Low temperature for consistent extraction
    )

    content = resp.choices[0].message.content
    if content is None:
        logger.warning("llm resp content is none")
        return PhotoMetadata()

    try:
        j = json.loads(content)
        return json_to_metadata(j)
    except json.JSONDecodeError as e:
        logger.warning(f"llm resp content json decode, content={content}, err={e}")
        return PhotoMetadata()


def json_to_metadata(json_str: str) -> PhotoMetadata:
    """Parse JSON string to PhotoMetadata with manual field mapping."""
    try:
        data = json.loads(json_str) if isinstance(json_str, str) else json_str

        return PhotoMetadata(
            camera_make=data.get("camera_make"),
            camera_model=data.get("camera_model"),
            film_make=data.get("film_make"),
            film_type=data.get("film_type"),
            film_speed=data.get("film_speed"),
            focal_length=data.get("focal_length"),
            aperture=data.get("aperture"),
        )
    except (json.JSONDecodeError, TypeError, KeyError) as e:
        print(f"llm parse json, content={json_str}, err={e}")
        return PhotoMetadata()
