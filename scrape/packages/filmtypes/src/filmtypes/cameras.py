import json
import time
from dataclasses import asdict, dataclass
from typing import Any, Dict, List, Optional
from urllib.parse import urljoin

import requests
from bs4 import BeautifulSoup

from .constants import FILMTYPES_URL, USER_AGENT


@dataclass
class CameraData:
    url: str
    slug: str
    make: str
    type: str
    manufacturer_country: Optional[str] = None
    description: Optional[str] = None


class CameraScraper:
    def __init__(self):
        self.base_url = FILMTYPES_URL
        self.cameras_url = f"{self.base_url}/cameras"
        self.session = requests.Session()
        self.session.headers.update(
            {
                "User-Agent": USER_AGENT,
                "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
                "Accept-Language": "en-US,en;q=0.5",
                "Accept-Encoding": "gzip, deflate",
                "Connection": "keep-alive",
            }
        )

    def get_page(self, url: str) -> Optional[BeautifulSoup]:
        """Fetch and parse a single page"""
        try:
            print(f"Fetching: {url}")
            response = self.session.get(url, timeout=10)
            response.raise_for_status()
            return BeautifulSoup(response.content, "html.parser")
        except requests.RequestException as e:
            print(f"Error fetching {url}: {e}")
            return None

    def get_all_links(self) -> List[dict]:
        """Extract all camera links from the main films page"""
        soup = self.get_page(self.cameras_url)
        if not soup:
            return []

        camera_links = []
        all_links = soup.find_all("a", href=True)

        for link in all_links:
            href = link.get("href")
            if (
                href
                and "/cameras/" in href
                and href != "/cameras"
                and href != "/cameras/"
            ):
                full_url = urljoin(self.base_url, href)
                film_name = link.get_text(strip=True)
                slug = href.split("/")[-1]

                camera_links.append(
                    {"name": film_name, "url": full_url, "href": href, "slug": slug}
                )

        print(f"Found {len(camera_links)} camera links")
        return camera_links

    def extract_json(self, json: Dict[str, Any]) -> Optional[CameraData]:
        if not json or "@graph" not in json:
            return None

        types = json.get("@type", [])
        if not isinstance(types, list):
            types = [types] if types else []

        if "Product" not in types or not json.get("name"):
            return None

        name = json.get("name", "")
        brand = json.get("brand", "")
        url = json.get("url", "")
        description = json.get("description")

        slug = self.slug(name, brand)

        manufacturer_country = None
        manufacturer = json.get("manufacturer", {})
        if isinstance(manufacturer, dict):
            address = manufacturer.get("address", {})
            if isinstance(address, dict):
                manufacturer_country = address.get("addressCountry")

        return CameraData(
            type=name,
            make=brand,
            url=url,
            slug=slug,
            description=description,
            manufacturer_country=manufacturer_country,
        )

    def slug(self, camera_make: str, camera_type: str) -> str:
        return (
            f"{camera_make}-{camera_type}".lower().replace(" ", "-")
            if camera_make and camera_type
            else ""
        )

    def scrape_individual(self, link: dict) -> Optional[CameraData]:
        soup = self.get_page(link["url"])
        if not soup:
            return None

        json_ld = soup.find("script", type="application/ld+json")
        if json_ld:
            try:
                structured_data = json.loads(json_ld.string)
                data = self.extract_json(structured_data)
                return data

            except json.JSONDecodeError:
                pass

    def normalize(self, camera: CameraData) -> CameraData:
        return camera

    def scrape_all(self, limit: Optional[int] = None) -> List[CameraData]:
        """Scrape all cameras with rate limiting"""
        links = self.get_all_links()

        if limit:
            links = links[:limit]
            print(f"Scraping {limit} cameras")

        datas: List[CameraData] = []

        # scrape from filmtypes
        for i, link in enumerate(links, 1):
            print(f"Scraping camera {i}/{len(links)}: {link['name']}")

            data = self.scrape_individual(link)
            if data:
                normalized = self.normalize(data)
                datas.append(normalized)

            if i < len(links):
                time.sleep(1)

        return datas

    def save_data(self, cameras: List[CameraData]):
        file = "cameras.json"
        with open(file, "w", encoding="utf-8") as f:
            json.dump(
                [asdict(camera) for camera in cameras],
                f,
                indent=2,
                ensure_ascii=False,
            )
        print(f"Saved data to {file}")


def main():
    scraper = CameraScraper()

    cameras = scraper.scrape_all(limit=100)
    if cameras:
        print(f"\nSuccessfully scraped {len(cameras)} films")
        scraper.save_data(cameras)

    else:
        print("No cameras scraped successfully")


if __name__ == "__main__":
    main()
