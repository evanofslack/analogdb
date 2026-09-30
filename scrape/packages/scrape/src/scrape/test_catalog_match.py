import pytest
from analogdb.models import Camera, Film

from .catalog_match import CatalogAlias, CatalogMatcher, film_type_key, valid_aperture
from .models import MetadataPost

CAMERAS = [
    ("canon", "ae-1"),
    ("canon", "ae-1 program"),
    ("canon", "eos 1v"),
    ("canon", "a-1"),
    ("contax", "g1"),
    ("hasselblad", "500c/m"),
    ("hasselblad", "500c"),
    ("mamiya", "7"),
    ("mamiya", "7ii"),
    ("mamiya", "6"),
    ("mamiya", "645"),
    ("mamiya", "645 super"),
    ("mamiya", "rb67 pro-s"),
    ("minolta", "maxxum 7000"),
    ("nikon", "f"),
    ("nikon", "f3"),
    ("nikon", "f90"),
    ("nikon", "f100"),
    ("nikon", "l35af"),
    ("nikon", "n6006"),
    ("olympus", "om-1"),
    ("olympus", "om-1n"),
    ("olympus", "om-2"),
    ("olympus", "mju ii"),
    ("pentax", "67"),
    ("pentax", "kx"),
    ("pentax", "spotmatic"),
    ("rollei", "rolleiflex 2.8f"),
    ("bronica", "sq-a"),
    ("voigtländer", "bessa r2"),
    ("yashica", "mat-124g"),
    ("yashica", "electro 35"),
    ("rollei", "rolleicord iv"),
    ("graflex", "speed graphic"),
]
FILMS = [
    ("kodak", "portra 160", 160),
    ("kodak", "portra 400", 400),
    ("kodak", "portra 800", 800),
    ("kodak", "gold 200", 200),
    ("kodak", "tri-x 400", 400),
    ("kodak", "t-max p3200", 3200),
    ("kodak", "vision3 500t", 500),
    ("ilford", "hp5 plus", 400),
    ("ilford", "delta 400", 400),
    ("fujifilm", "400", 400),
    ("fujifilm", "fujicolor pro 400h", 400),
    ("fujifilm", "fujicolor superia x-tra 400", 400),
    ("fujifilm", "fujichrome provia 100", 100),
    ("cinestill", "800t", 800),
    ("foma", "200 creative", 200),
    ("lomography", "400", 400),
]


@pytest.fixture(scope="module")
def matcher():
    cameras = [
        Camera(id=i, make=m, model=n, description="")
        for i, (m, n) in enumerate(CAMERAS)
    ]
    films = [
        Film(id=i, make=m, type=t, speed=s, color_type="color", description="")
        for i, (m, t, s) in enumerate(FILMS)
    ]
    return CatalogMatcher(cameras, films)


def cam(matcher, make, model, text=None):
    text = text or f"{make or ''} {model or ''}"
    raw = {"cameras": [{"text": text, "make": make, "model": model}]}
    return matcher.match(raw, MetadataPost(title=text))


def film(matcher, make, type, box_speed=None, text=None):
    text = text or f"{make or ''} {type or ''} {box_speed or ''}"
    raw = {
        "films": [
            {
                "text": text,
                "make": make,
                "type": type,
                "box_speed": box_speed,
                "shot_at": None,
            }
        ]
    }
    return matcher.match(raw, MetadataPost(title=text))


@pytest.mark.parametrize(
    "make,model,expected",
    [
        ("Olympus", "om2", ("olympus", "om-2")),
        ("Olympus", "OM-1N", ("olympus", "om-1n")),
        ("Hasselblad", "500CM", ("hasselblad", "500c/m")),
        ("Canon", "1V", ("canon", "eos 1v")),
        ("Canon", "AE1", ("canon", "ae-1")),
        ("Canon", "AE-1P", ("canon", "ae-1 program")),
        ("Nikon", "F-3", ("nikon", "f3")),
        ("Nikon", "F3HP", ("nikon", "f3")),
        ("Nikon", "N90", ("nikon", "f90")),
        ("Nikon", "F601", ("nikon", "n6006")),
        ("Pentax", "6x7", ("pentax", "67")),
        (None, "Pentax6x7", ("pentax", "67")),
        ("Pentax", "Spotmatic SP", ("pentax", "spotmatic")),
        ("Mamiya", "7 mk II", ("mamiya", "7ii")),
        ("Mamiya", "RB67 Pro S", ("mamiya", "rb67 pro-s")),
        (None, "RB67 Pro-S", ("mamiya", "rb67 pro-s")),
        ("Minolta", "Dynax 7000", ("minolta", "maxxum 7000")),
        ("Rolleiflex", "2.8F", ("rollei", "rolleiflex 2.8f")),
        ("Zenza Bronica", "SQ-A", ("bronica", "sq-a")),
        ("Voigtlander", "Bessa R2", ("voigtländer", "bessa r2")),
        ("Olympus", "μ-II", ("olympus", "mju ii")),
        ("Yashica", "Mat 124G", ("yashica", "mat-124g")),
        ("Yashica", "124G", ("yashica", "mat-124g")),
        ("Rollei", "Rolleicord IV", ("rollei", "rolleicord iv")),
        ("Asahi", "Pentax KX", ("pentax", "kx")),
        ("Nikon", "F3 Titan", ("nikon", "f3")),
        ("Yashica", "Electro 35GSN", ("yashica", "electro 35")),
        ("Graflex", "Speedgraphic 2X3", ("graflex", "speed graphic")),
        ("Mamiya", "RB67 Pro S", ("mamiya", "rb67 pro-s")),
    ],
)
def test_camera_matches(matcher, make, model, expected):
    r = cam(matcher, make, model)
    assert (r.proposed.camera_make, r.proposed.camera_model) == expected
    assert r.unmatched == []


@pytest.mark.parametrize(
    "make,model,expected_make,key",
    [
        ("Contax", "G2", "contax", "contaxg2"),
        ("Nikon", "L35AD", "nikon", "nikonl35ad"),
        ("Mamiya", "M645", "mamiya", "mamiyam645"),
        ("Nikon", "F100s", "nikon", "nikonf100s"),
        ("Pentax", "KM", "pentax", "pentaxkm"),
        ("Plaubel", "Makina 67", None, "plaubelmakina67"),
        ("Olympus", "OM-2SP", "olympus", "olympusom2sp"),
        ("Mamiya", "645N", "mamiya", "mamiya645n"),
        ("Canon", "EOS 3000", "canon", "canoneos3000"),
    ],
)
def test_camera_not_in_catalog_is_unmatched(matcher, make, model, expected_make, key):
    r = cam(matcher, make, model)
    assert r.proposed.camera_make == expected_make
    assert r.proposed.camera_model is None
    assert [u.key for u in r.unmatched] == [key]
    assert ("catalog_make_only" in r.flags) == (expected_make is not None)


def test_nikon_f_is_not_a_prefix_of_f100s(matcher):
    assert cam(matcher, "Nikon", "F100s").proposed.camera_model is None


@pytest.mark.parametrize(
    "make,type,speed,expected",
    [
        ("Kodak", "Portra 400", None, ("kodak", "portra 400", 400)),
        (None, "Portra400", None, ("kodak", "portra 400", 400)),
        (None, "Portra", 160, ("kodak", "portra 160", 160)),
        ("Ilford", "HP5+", 400, ("ilford", "hp5 plus", 400)),
        (None, "HP5", None, ("ilford", "hp5 plus", 400)),
        ("Fuji", "400", 400, ("fujifilm", "400", 400)),
        ("Fuji", "Pro400H", 400, ("fujifilm", "fujicolor pro 400h", 400)),
        (
            "Fujifilm",
            "Superia 400",
            400,
            ("fujifilm", "fujicolor superia x-tra 400", 400),
        ),
        ("Fujifilm", "Provia 100F", 100, ("fujifilm", "fujichrome provia 100", 100)),
        ("Kodak", "Tri-X", None, ("kodak", "tri-x 400", 400)),
        ("Kodak", "Tmax 3200", 3200, ("kodak", "t-max p3200", 3200)),
        (None, "Gold", None, ("kodak", "gold 200", 200)),
        ("Cine", "800T", 800, ("cinestill", "800t", 800)),
        (None, "800T", None, ("cinestill", "800t", 800)),
        ("Kodak", "500T", None, ("kodak", "vision3 500t", 500)),
        (None, "Fomapan 200", 200, ("foma", "200 creative", 200)),
        ("Foma", "Fomapan200", 200, ("foma", "200 creative", 200)),
        ("Kodak Professional", "Portra 800", 800, ("kodak", "portra 800", 800)),
        ("Kodak", "Portra 400/800 Mix", None, ("kodak", "portra 400", 400)),
        ("Ilford", "HP5 400", 400, ("ilford", "hp5 plus", 400)),
        ("Kodak", "Tx 400", 400, ("kodak", "tri-x 400", 400)),
        ("Fujifilm", "400 Disposable Film", 400, ("fujifilm", "400", 400)),
    ],
)
def test_film_matches(matcher, make, type, speed, expected):
    r = film(matcher, make, type, speed)
    p = r.proposed
    assert (p.film_make, p.film_type, p.film_speed) == expected
    assert r.unmatched == []


def test_film_with_speed_is_not_a_prefix(matcher):
    cameras = []
    films = [
        Film(
            id=1,
            make="fujifilm",
            type="neopan 400c",
            speed=400,
            color_type="bw",
            description="",
        )
    ]
    r = CatalogMatcher(cameras, films).match(
        {
            "films": [
                {
                    "text": "Neopan 400",
                    "make": "Fuji",
                    "type": "Neopan 400",
                    "box_speed": 400,
                }
            ]
        },
        MetadataPost(title="Fuji Neopan 400"),
    )
    assert (r.proposed.film_make, r.proposed.film_type) == ("fujifilm", None)
    assert r.unmatched[0].key == "fujifilmneopan400"


def test_film_ambiguous_without_speed(matcher):
    r = film(matcher, "Kodak", "Portra")
    assert (r.proposed.film_make, r.proposed.film_type) == ("kodak", None)
    assert "ambiguous_film" in r.flags


def test_film_not_in_catalog(matcher):
    r = film(matcher, "Kodak", "Gold", 400)
    assert (r.proposed.film_make, r.proposed.film_type, r.proposed.film_speed) == (
        "kodak",
        None,
        400,
    )
    assert [u.key for u in r.unmatched] == ["kodakgold"]
    r = film(matcher, "Fuji", "Superia 200", 200)
    assert r.proposed.film_type is None and r.unmatched[0].kind == "film"


def test_bare_speed_without_make_is_not_guessed(matcher):
    r = film(matcher, None, "400", 400)
    assert (r.proposed.film_make, r.proposed.film_type, r.proposed.film_speed) == (
        None,
        None,
        400,
    )


def test_push_keeps_box_speed(matcher):
    raw = {
        "films": [
            {
                "text": "HP5 pushed to 1600",
                "make": "Ilford",
                "type": "HP5",
                "box_speed": 400,
                "shot_at": 1600,
            }
        ]
    }
    r = matcher.match(raw, MetadataPost(title="HP5 pushed to 1600"))
    assert (r.proposed.film_type, r.proposed.film_speed) == ("hp5 plus", 400)


def test_ungrounded_is_dropped(matcher):
    raw = {"cameras": [{"text": "Nikon F3", "make": "Nikon", "model": "F3"}]}
    r = matcher.match(raw, MetadataPost(title="Shot on my Nikon"))
    assert r.proposed.camera_make == "nikon"
    assert r.proposed.camera_model is None
    assert "ungrounded_camera" in r.flags


def test_grounded_by_comments(matcher):
    raw = {"cameras": [{"text": "F3", "make": "Nikon", "model": "F3"}]}
    r = matcher.match(raw, MetadataPost(title="Dusk", op_comments=["this was my F3"]))
    assert r.proposed.camera_model == "f3"


def test_first_mention_wins_and_multiple_flag(matcher):
    raw = {
        "cameras": [
            {"text": "Pentax 67", "make": "Pentax", "model": "67"},
            {"text": "Canon AE-1", "make": "Canon", "model": "AE-1"},
        ],
        "films": [
            {
                "text": "Portra 160",
                "make": "Kodak",
                "type": "Portra 160",
                "box_speed": 160,
                "shot_at": None,
            },
            {
                "text": "Portra 400",
                "make": "Kodak",
                "type": "Portra 400",
                "box_speed": 400,
                "shot_at": None,
            },
        ],
        "lenses": [{"text": "105mm f/2.4", "focal_length": 105, "aperture": "f/2.4"}],
    }
    r = matcher.match(
        raw, MetadataPost(title="Pentax 67 and Canon AE-1, Portra 160/400, 105mm f/2.4")
    )
    p = r.proposed
    assert (p.camera_make, p.camera_model, p.film_type, p.focal_length, p.aperture) == (
        "pentax",
        "67",
        "portra 160",
        105,
        "f/2.4",
    )
    assert {"multiple_cameras", "multiple_films"} <= set(r.flags)


def test_llm_failed(matcher):
    r = matcher.match(None, MetadataPost(title="x"))
    assert r.flags == ["llm_failed"] and r.proposed.is_empty()


def test_db_alias(matcher):
    cameras = [Camera(id=1, make="hasselblad", model="500c/m", description="")]
    m = CatalogMatcher(
        cameras,
        [],
        [CatalogAlias("camera", "hasselblad500classic", "hasselblad", "500c/m")],
    )
    r = m.match(
        {
            "cameras": [
                {
                    "text": "Hasselblad 500 Classic",
                    "make": "Hasselblad",
                    "model": "500 Classic",
                }
            ]
        },
        MetadataPost(title="Hasselblad 500 Classic"),
    )
    assert r.proposed.camera_model == "500c/m"


def test_helpers():
    assert film_type_key("HP5+", set()) == "hp5"
    assert film_type_key("Fujicolor Superia X-TRA 400", {"fujifilm"}) == "superia400"
    assert valid_aperture("F1.4") == "f/1.4"
    assert valid_aperture("1:2.8") == "f/2.8"
    assert valid_aperture("f/2") == "f/2"
    assert valid_aperture("0.2") is None
