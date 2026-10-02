from dataclasses import dataclass
from typing import Optional

from analogdb_generated.models.analogdb_camera import AnalogdbCamera as Camera
from analogdb_generated.models.analogdb_color import AnalogdbColor as Color
from analogdb_generated.models.analogdb_create_camera import (
    AnalogdbCreateCamera as CameraCreate,
)
from analogdb_generated.models.analogdb_create_film import (
    AnalogdbCreateFilm as FilmCreate,
)
from analogdb_generated.models.analogdb_create_post import (
    AnalogdbCreatePost as PostCreate,
)
from analogdb_generated.models.analogdb_film import AnalogdbFilm as Film
from analogdb_generated.models.analogdb_image import AnalogdbImage as Image
from analogdb_generated.models.analogdb_keyword import AnalogdbKeyword as Keyword
from analogdb_generated.models.analogdb_patch_post import (
    AnalogdbPatchPost as PostPatch,
)
from analogdb_generated.models.analogdb_post import AnalogdbPost as Post
from analogdb_generated.models.analogdb_post_caption import (
    AnalogdbPostCaption as PostCaption,
)
from analogdb_generated.models.analogdb_post_extraction import (
    AnalogdbPostExtraction as PostExtraction,
)

__all__ = [
    "Camera",
    "CameraCreate",
    "Color",
    "Film",
    "FilmCreate",
    "Image",
    "Keyword",
    "Post",
    "PostCaption",
    "PostCreate",
    "PostExtraction",
    "PostPatch",
    "PostsFilter",
    "create_posts_filter",
]


@dataclass
class PostsFilter:
    count: Optional[int]
    nsfw: Optional[bool]
    grayscale: Optional[bool]
    sprocket: Optional[bool]
    time_start: Optional[int]
    time_end: Optional[int]


def create_posts_filter(
    count: Optional[int] = None,
    nsfw: Optional[bool] = None,
    grayscale: Optional[bool] = None,
    sprocket: Optional[bool] = None,
    time_start: Optional[int] = None,
    time_end: Optional[int] = None,
):
    return PostsFilter(count, nsfw, grayscale, sprocket, time_start, time_end)
