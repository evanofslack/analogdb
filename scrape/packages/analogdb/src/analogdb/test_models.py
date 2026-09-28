from .models import Keyword, PostPatch, create_posts_filter


class TestPostPatch:
    def test_to_dict_empty_when_nothing_set(self):
        assert PostPatch().to_dict() == {}

    def test_to_dict_only_set_fields(self):
        assert PostPatch(score=5).to_dict() == {"score": 5}

    def test_to_dict_metadata_fields(self):
        patch = PostPatch(camera_make="nikon", film_speed=400, aperture=None)
        assert patch.to_dict() == {"camera_make": "nikon", "film_speed": 400}

    def test_to_dict_keywords(self):
        patch = PostPatch(keywords=[Keyword(word="field", weight=0.5)])
        assert patch.to_dict() == {"keywords": [{"word": "field", "weight": 0.5}]}


class TestPostsFilter:
    def test_create_posts_filter(self):
        f = create_posts_filter(time_start=1, time_end=2)
        assert f.time_start == 1
        assert f.time_end == 2
        assert f.nsfw is None
