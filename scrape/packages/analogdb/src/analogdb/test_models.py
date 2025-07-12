from .models import PhotoMetadata, PostPatch


class TestPostPatch:
    def test_is_empty_when_all_none(self):
        """Test that is_empty returns True when all optional fields are None (ignoring id)"""
        patch = PostPatch(
            id=1,
            score=None,
            descripton=None,
            nsfw=None,
            grayscale=None,
            sprocket=None,
            colors=None,
            keywords=None,
            metadata=None,
        )
        assert patch.is_empty() is True

    def test_is_empty_when_score(self):
        """Test that is_empty returns False when score is set"""
        patch = PostPatch(
            id=1,
            score=85,
            descripton=None,
            nsfw=None,
            grayscale=None,
            sprocket=None,
            colors=None,
            keywords=None,
            metadata=None,
        )
        assert patch.is_empty() is False

    def test_is_empty_when_multiple_fields(self):
        """Test that is_empty returns False when multiple fields are set"""
        patch = PostPatch(
            id=1,
            score=90,
            descripton=None,
            nsfw=True,
            grayscale=None,
            sprocket=None,
            colors=None,
            keywords=None,
            metadata=None,
        )
        assert patch.is_empty() is False


class TestPhotoMetadata:
    def test_is_empty_when_all_none(self):
        """Test that is_empty returns True when all optional fields are None (ignoring id)"""
        metadata = PhotoMetadata(
            camera_make=None,
            camera_model=None,
            film_make=None,
            film_type=None,
            film_speed=None,
            focal_length=None,
            aperture=None,
        )
        assert metadata.is_empty() is True

    def test_is_not_empty_when_camera_make(self):
        """Test that is_empty returns False when one optional fields is not None"""
        metadata = PhotoMetadata(
            camera_make="make",
            camera_model=None,
            film_make=None,
            film_type=None,
            film_speed=None,
            focal_length=None,
            aperture=None,
        )
        assert metadata.is_empty() is False
