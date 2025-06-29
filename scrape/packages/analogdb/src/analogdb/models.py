from dataclasses import dataclass, fields
from typing import Dict, List, Optional


@dataclass
class Image:
    url: str
    resolution: str
    width: int
    height: int

    def to_json(self) -> Dict:
        return {
            "url": self.url,
            "resolution": self.resolution,
            "width": self.width,
            "height": self.height,
        }


@dataclass
class Color:
    hex: str
    css: str
    html: str
    percent: float

    def to_json(self) -> Dict:
        return {
            "hex": self.hex,
            "css": self.css,
            "html": self.html,
            "percent": self.percent,
        }


@dataclass
class Keyword:
    word: str
    weight: float

    def to_json(self) -> Dict:
        return {
            "word": self.word,
            "weight": self.weight,
        }


@dataclass
class PhotoMetadata:
    camera_make: Optional[str] = None
    camera_model: Optional[str] = None
    film_make: Optional[str] = None
    film_type: Optional[str] = None
    film_speed: Optional[int] = None
    focal_length: Optional[int] = None
    aperture: Optional[str] = None

    def is_empty(self) -> bool:
        return all(getattr(self, field.name) is None for field in fields(self))


@dataclass
class Meta:
    total_posts: int
    page_size: int
    next_page_id: Optional[int]
    next_page_url: Optional[str]


@dataclass
class Post:
    id: int
    title: str
    author: str
    permalink: str
    score: int
    timestamp: int
    nsfw: bool
    grayscale: bool
    sprocket: bool
    images: List[Image]


@dataclass
class Posts:
    posts: List[Post]
    meta: Meta


@dataclass
class PostsFilter:
    count: Optional[int]
    nsfw: Optional[bool]
    grayscale: Optional[bool]
    sprocket: Optional[bool]
    time_start: Optional[int]
    time_end: Optional[int]


@dataclass
class PostPatch:
    id: int
    score: Optional[int]
    nsfw: Optional[bool]
    greyscale: Optional[bool]
    sprocket: Optional[bool]
    colors: Optional[List[Color]]
    keywords: Optional[List[Keyword]]
    metadata: Optional[PhotoMetadata]

    def is_empty(self) -> bool:
        return all(getattr(self, field.name) is None for field in fields(self))

    def to_json(self) -> Dict:
        body = {}
        if self.score is not None:
            body["upvotes"] = self.score
        if self.nsfw is not None:
            body["nsfw"] = self.nsfw
        if self.greyscale is not None:
            body["grayscale"] = self.greyscale
        if self.sprocket is not None:
            body["sprocket"] = self.sprocket
        if self.colors is not None:
            body["colors"] = [c.to_json for c in self.colors]
        if self.keywords is not None:
            body["keywords"] = [kw.to_json for kw in self.keywords]
        if self.metadata is not None:
            if self.metadata.camera_make is not None:
                body["camera_make"] = self.metadata.camera_make
            if self.metadata.camera_model is not None:
                body["camera_model"] = self.metadata.camera_model
            if self.metadata.film_make is not None:
                body["film_make"] = self.metadata.film_make
            if self.metadata.film_type is not None:
                body["film_type"] = self.metadata.film_type
            if self.metadata.film_speed is not None:
                body["film_speed"] = self.metadata.film_speed
            if self.metadata.focal_length is not None:
                body["focal_length"] = self.metadata.focal_length
            if self.metadata.aperture is not None:
                body["aperture"] = self.metadata.aperture
        return body


def create_post_patch(
    id: int,
    score: Optional[int] = None,
    nsfw: Optional[bool] = None,
    greyscale: Optional[bool] = None,
    sprocket: Optional[bool] = None,
    colors: Optional[List[Color]] = None,
    keywords: Optional[List[Keyword]] = None,
    metadata: Optional[PhotoMetadata] = None,
):
    return PostPatch(id, score, nsfw, greyscale, sprocket, colors, keywords, metadata)


@dataclass
class PostCreate:
    title: str
    author: str
    permalink: str
    score: int
    nsfw: bool
    grayscale: bool
    time: int
    sprocket: bool

    camera_make: Optional[str]
    camera_model: Optional[str]
    film_make: Optional[str]
    film_type: Optional[str]
    film_speed: Optional[int]
    focal_length: Optional[int]
    aperture: Optional[str]

    images: List[Image]
    keywords: List[Keyword]
    colors: List[Color]

    def to_json(self) -> Dict:
        body = {}

        body["title"] = self.title
        body["author"] = self.author
        body["permalink"] = self.permalink
        body["nsfw"] = self.nsfw
        body["grayscale"] = self.grayscale
        body["unix_time"] = self.time
        body["sprocket"] = self.sprocket
        body["upvotes"] = self.score

        if self.camera_make is not None:
            body["camera_make"] = self.camera_make
        if self.camera_model is not None:
            body["camera_model"] = self.camera_model
        if self.film_make is not None:
            body["film_make"] = self.film_make
        if self.film_type is not None:
            body["film_type"] = self.film_type
        if self.film_speed is not None:
            body["film_speed"] = self.film_speed
        if self.focal_length is not None:
            body["focal_length"] = self.focal_length
        if self.aperture is not None:
            body["aperture"] = self.aperture

        body["images"] = [img.to_json() for img in self.images]
        body["keywords"] = [keyword.to_json() for keyword in self.keywords]
        body["colors"] = [color.to_json() for color in self.colors]

        return body


@dataclass
class Film:
    id: int
    type: str
    make: str
    speed: int
    color_type: str
    description: str

    def to_json_minimal(self) -> Dict:
        body = {
            "type": self.type,
            "make": self.make,
            "speed": self.speed,
        }
        return body


@dataclass
class FilmCreate:
    type: str
    make: str
    speed: int
    color_type: str
    description: str

    def to_json(self) -> Dict:
        body = {
            "type": self.type,
            "make": self.make,
            "speed": self.speed,
            "color_type": self.color_type,
            "description": self.description,
        }
        return body


@dataclass
class Camera:
    id: str
    make: str
    model: str
    description: str

    def to_json_minimal(self) -> Dict:
        body = {
            "make": self.make,
            "model": self.model,
        }
        return body


@dataclass
class CameraCreate:
    make: str
    model: str
    description: str

    def to_json(self) -> Dict:
        body = {
            "make": self.make,
            "model": self.model,
            "description": self.description,
        }
        return body
