import pytest

from .client import Client
from .models import PostsFilter


@pytest.fixture
def client():
    return Client()


class TestFilterToParams:
    def test_filter_none_returns_defaults(self, client):
        """When filter is None, should return default parameters."""
        params = client._filter_to_params(None)

        expected = {
            "page_size": 20,
            "sort": "time",
        }
        assert params == expected

    def test_filter_with_all_params_set(self, client):
        """When filter has all parameters, should use filter values."""
        filter_obj = PostsFilter(
            count=None,
            nsfw=False,
            grayscale=False,
            sprocket=False,
            time_start=None,
            time_end=None,
        )
        params = client._filter_to_params(filter_obj)

        assert params["nsfw"] is False
        assert params["grayscale"] is False
        assert params["sprocket"] is False
        # Defaults should still be present for other params
        assert params["page_size"] == 20
        assert params["sort"] == "time"

    def test_filter_with_partial_params(self, client):
        """When filter has some parameters, should mix filter and defaults."""
        filter_obj = PostsFilter(
            count=None,
            nsfw=False,
            grayscale=None,
            sprocket=True,
            time_start=None,
            time_end=None,
        )
        params = client._filter_to_params(filter_obj)

        assert params["nsfw"] is False  # from filter
        assert params["sprocket"] is True  # from filter
        assert params["page_size"] == 20  # default
        assert params["sort"] == "time"  # default


class TestParsePost:
    """Test the _parse_post private method."""

    def test_parse_post_basic(self, client):
        """Should parse a basic post with all required fields."""
        post_data = {
            "id": 123,
            "title": "Test Post",
            "author": "test_user",
            "permalink": "https://example.com/post/123",
            "score": 42,
            "timestamp": "2024-01-01T00:00:00Z",
            "nsfw": False,
            "grayscale": True,
            "sprocket": False,
            "images": [],
        }

        post = client._parse_post(post_data)

        assert post.id == 123
        assert post.title == "Test Post"
        assert post.author == "test_user"
        assert post.permalink == "https://example.com/post/123"
        assert post.score == 42
        assert post.timestamp == "2024-01-01T00:00:00Z"
        assert post.nsfw is False
        assert post.grayscale is True
        assert post.sprocket is False
        assert post.images == []

    def test_parse_post_with_images(self, client):
        """Should parse a post with images."""
        post_data = {
            "id": 456,
            "title": "Post with Images",
            "author": "photographer",
            "permalink": "https://example.com/post/456",
            "score": 100,
            "timestamp": "2024-01-02T12:00:00Z",
            "nsfw": True,
            "grayscale": False,
            "sprocket": True,
            "images": [
                {
                    "url": "https://example.com/img1.jpg",
                    "resolution": "low",
                    "width": 800,
                    "height": 600,
                },
                {
                    "url": "https://example.com/img2.jpg",
                    "resolution": "medium",
                    "width": 1024,
                    "height": 768,
                },
            ],
        }

        post = client._parse_post(post_data)

        assert post.id == 456
        assert len(post.images) == 2
        assert post.images[0].url == "https://example.com/img1.jpg"
        assert post.images[0].resolution == "low"
        assert post.images[1].url == "https://example.com/img2.jpg"
        assert post.images[1].resolution == "medium"


class TestParsePostsResponse:
    def test_parse_posts_response_basic(self, client):
        """Should parse a posts response with meta and posts."""
        response_data = {
            "posts": [
                {
                    "id": 1,
                    "title": "First Post",
                    "author": "user1",
                    "permalink": "https://example.com/1",
                    "score": 10,
                    "timestamp": "2024-01-01T00:00:00Z",
                    "nsfw": False,
                    "grayscale": False,
                    "sprocket": False,
                    "images": [],
                },
                {
                    "id": 2,
                    "title": "Second Post",
                    "author": "user2",
                    "permalink": "https://example.com/2",
                    "score": 20,
                    "timestamp": "2024-01-02T00:00:00Z",
                    "nsfw": True,
                    "grayscale": True,
                    "sprocket": True,
                    "images": [],
                },
            ],
            "meta": {
                "total_posts": 100,
                "page_size": 20,
                "next_page_id": 42,
                "next_page_url": "test_url",
            },
        }

        posts_response = client._parse_posts_response(response_data)

        assert len(posts_response.posts) == 2
        assert posts_response.posts[0].id == 1
        assert posts_response.posts[0].title == "First Post"
        assert posts_response.posts[1].id == 2
        assert posts_response.posts[1].title == "Second Post"
        assert posts_response.meta.total_posts == 100
        assert posts_response.meta.page_size == 20
        assert posts_response.meta.next_page_id == 42
        assert posts_response.meta.next_page_url == "test_url"

    def test_parse_posts_response_empty(self, client):
        """Should handle empty posts response."""
        response_data = {
            "posts": [],
            "meta": {
                "total_posts": 0,
                "page_size": 20,
                "next_page_id": 0,
                "next_page_url": "",
            },
        }

        posts_response = client._parse_posts_response(response_data)

        assert len(posts_response.posts) == 0
        assert posts_response.meta.next_page_id == 0
        assert posts_response.meta.total_posts == 0
