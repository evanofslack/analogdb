import pytest
from camera import extract_metadata
from models import PhotoMetadata


class TestExtractMetadata:
    """Test cases for photo metadata extraction."""

    def test_camera_extraction(self):
        """Test camera make and model extraction."""
        title = "Shot with Canon EOS 5D | 50mm f/1.8"
        result = extract_metadata(title)

        assert result.camera_make == "Canon"
        assert result.camera_model == "EOS 5D"
        assert result.focal_length == 50
        assert result.aperture == "f/1.8"

    def test_film_extraction(self):
        """Test film type and speed extraction."""
        title = "Nikon F3 | Kodak Portra 400 | 85mm"
        result = extract_metadata(title)

        assert result.camera_make == "Nikon"
        assert result.camera_model == "F3"
        assert result.film_make == "Kodak"
        assert result.film_type == "Portra"
        assert result.film_speed == 400
        assert result.focal_length == 85

    def test_multiple_cameras(self):
        """Test that first camera found is extracted."""
        title = "Canon AE-1 vs Nikon FM2 comparison"
        result = extract_metadata(title)

        assert result.camera_make == "Canon"
        assert result.camera_model == "AE-1"

    def test_film_without_speed(self):
        """Test film extraction without speed."""
        title = "Leica M6 | Ilford HP5 | Street photography"
        result = extract_metadata(title)

        assert result.camera_make == "Leica"
        assert result.camera_model == "M6"
        assert result.film_make == "Ilford"
        assert result.film_type == "Hp5"
        assert result.film_speed is None

    def test_standalone_film_speed(self):
        """Test film speed extraction without film type."""
        title = "Hasselblad 500CM | Some unknown film 800"
        result = extract_metadata(title)

        assert result.camera_make == "Hasselblad"
        assert result.camera_model == "500CM"
        assert result.film_speed == 800

    def test_aperture_variations(self):
        """Test different aperture format variations."""
        test_cases = [
            ("Shot at f/2.8 today", "f/2.8"),
            ("Used f4 for this shot", "f/4"),
            ("Beautiful bokeh at f/1.4", "f/1.4"),
            ("Landscape shot f22", "f/22"),
        ]

        for title, expected_aperture in test_cases:
            result = extract_metadata(title)
            assert result.aperture == expected_aperture

    def test_focal_length_variations(self):
        """Test focal length extraction."""
        test_cases = [
            ("Shot with 35mm lens", 35),
            ("Portrait with 85mm", 85),
            ("Wide angle 14mm", 14),
            ("Telephoto 200mm shot", 200),
        ]

        for title, expected_focal_length in test_cases:
            result = extract_metadata(title)
            assert result.focal_length == expected_focal_length

    def test_complex_title(self):
        """Test extraction from complex, realistic title."""
        title = "[OC] Street photography | Pentax K1000 | Kodak Tri-X 400 | 50mm f/2.0"
        result = extract_metadata(title)

        assert result.camera_make == "Pentax"
        assert result.camera_model == "K1000"
        assert result.film_make == "Kodak"
        assert result.film_type == "Tri-X"
        assert result.film_speed == 400
        assert result.focal_length == 50
        assert result.aperture == "f/2.0"

    def test_empty_title(self):
        """Test with empty title."""
        result = extract_metadata("")

        assert result.camera_make is None
        assert result.camera_model is None
        assert result.film_make is None
        assert result.film_type is None
        assert result.film_speed is None
        assert result.focal_length is None
        assert result.aperture is None

    def test_no_metadata_title(self):
        """Test title with no extractable metadata."""
        title = "Beautiful sunset landscape photography"
        result = extract_metadata(title)

        assert result.camera_make is None
        assert result.camera_model is None
        assert result.film_make is None
        assert result.film_type is None

    def test_case_insensitive_extraction(self):
        """Test that extraction works regardless of case."""
        title = "CANON EOS 1V | KODAK PORTRA 160 | 35MM F/1.4"
        result = extract_metadata(title)

        assert result.camera_make == "Canon"
        assert result.camera_model == "EOS 1V"
        assert result.film_make == "Kodak"
        assert result.film_type == "Portra"
        assert result.focal_length == 35
        assert result.aperture == "f/1.4"

    def test_film_make_inference(self):
        """Test that film make is correctly inferred from film type."""
        test_cases = [
            ("Fuji Provia 100", "Fuji", "Provia"),
            ("Ilford Delta 100", "Ilford", "Delta"),
            ("Shot on Ektar 100", "Kodak", "Ektar"),
            ("Lomography film", "Lomography", "Lomography"),
        ]

        for title, expected_make, expected_type in test_cases:
            result = extract_metadata(title)
            assert result.film_make == expected_make
            assert result.film_type == expected_type

    def test_partial_matches_avoided(self):
        """Test that partial word matches are avoided."""
        # "scan" contains "canon" but shouldn't match
        title = "Scanned with Epson scanner"
        result = extract_metadata(title)

        assert result.camera_make is None
        assert result.camera_model is None


if __name__ == "__main__":
    pytest.main([__file__])
