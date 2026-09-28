import scrape.models as s

from .convert import convert_create

FULL = s.CreatePost(
    title="A day at the fields [Nikon FM2 | Portra 400]",
    author="u/thecameraman",
    permalink="https://www.reddit.com/r/analog/comments/abc/post",
    description="shot at f/2",
    score=1000,
    nsfw=False,
    grayscale=False,
    time=1752354541,
    sprocket=True,
    camera_make="nikon",
    camera_model="fm2",
    film_make="kodak",
    film_type="portra 400",
    film_speed=400,
    focal_length=50,
    aperture="f/2.0",
    images=[
        s.S3Image("low", "https://x/low.jpg", 320, 240),
        s.S3Image("raw", "https://x/raw.jpg", 4800, 3600),
    ],
    keywords=[s.Keyword("field", 0.43)],
    colors=[s.Color("#837d5c", "dimgray", "gray", 0.55)],
)

FULL_BODY = {
    "title": "A day at the fields [Nikon FM2 | Portra 400]",
    "author": "u/thecameraman",
    "permalink": "https://www.reddit.com/r/analog/comments/abc/post",
    "nsfw": False,
    "grayscale": False,
    "timestamp": 1752354541,
    "sprocket": True,
    "score": 1000,
    "description": "shot at f/2",
    "camera_make": "nikon",
    "camera_model": "fm2",
    "film_make": "kodak",
    "film_type": "portra 400",
    "film_speed": 400,
    "focal_length": 50,
    "aperture": "f/2.0",
    "images": [
        {"url": "https://x/low.jpg", "resolution": "low", "width": 320, "height": 240},
        {
            "url": "https://x/raw.jpg",
            "resolution": "raw",
            "width": 4800,
            "height": 3600,
        },
    ],
    "keywords": [{"word": "field", "weight": 0.43}],
    "colors": [{"hex": "#837d5c", "css": "dimgray", "html": "gray", "percent": 0.55}],
}

BARE = s.CreatePost(
    title="t",
    author="u/a",
    permalink="https://www.reddit.com/r/analog/comments/def/post",
    description=None,
    score=0,
    nsfw=True,
    grayscale=True,
    time=1,
    sprocket=False,
    camera_make=None,
    camera_model=None,
    film_make=None,
    film_type=None,
    film_speed=None,
    focal_length=None,
    aperture=None,
    images=[],
    keywords=[],
    colors=[],
)

BARE_BODY = {
    "title": "t",
    "author": "u/a",
    "permalink": "https://www.reddit.com/r/analog/comments/def/post",
    "nsfw": True,
    "grayscale": True,
    "timestamp": 1,
    "sprocket": False,
    "score": 0,
    "images": [],
    "keywords": [],
    "colors": [],
}


class TestConvertCreate:
    def test_full_post_body(self):
        assert convert_create(FULL).to_dict() == FULL_BODY

    def test_bare_post_body(self):
        assert convert_create(BARE).to_dict() == BARE_BODY
