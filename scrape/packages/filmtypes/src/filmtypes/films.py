import json
import re
import time
from dataclasses import asdict, dataclass
from typing import List, Optional
from urllib.parse import urljoin

import requests
from bs4 import BeautifulSoup

from .constants import FILMTYPES_URL, USER_AGENT


@dataclass
class FilmData:
    type: str
    slug: str
    make: Optional[str] = None
    speed: Optional[int] = None
    color_type: Optional[str] = None
    description: Optional[str] = None


class FilmScraper:
    def __init__(self):
        self.base_url = FILMTYPES_URL
        self.films_url = f"{self.base_url}/films"
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
        """Extract all film links from the main films page"""
        soup = self.get_page(self.films_url)
        if not soup:
            return []

        links = []
        all_links = soup.find_all("a", href=True)

        for link in all_links:
            href = link.get("href")
            if href and "/films/" in href and href != "/films" and href != "/films/":
                full_url = urljoin(self.base_url, href)
                film_name = link.get_text(strip=True)
                slug = href.split("/")[-1]

                links.append(
                    {"name": film_name, "url": full_url, "href": href, "slug": slug}
                )

        print(f"Found {len(links)} film links")
        return links

    def extract_json(self, json: dict) -> dict:
        if not json or "@graph" not in json:
            return {}

        for item in json["@graph"]:
            if item.get("@type") and "Product" in item["@type"]:
                data = {
                    "manufacturer": item.get("brand"),
                    "description": item.get("description"),
                    "color_type": item.get("category"),
                    "country": None,
                }

                if "manufacturer" in item and "address" in item["manufacturer"]:
                    addr = item["manufacturer"]["address"]
                    data["country"] = addr.get("addressCountry")

                props = item.get("additionalProperty", [])
                for prop in props:
                    name = prop.get("name", "").lower()
                    value = prop.get("value")

                    if "iso" in name:
                        data["iso_speed"] = value
                    elif "grain" in name:
                        data["grain"] = value
                    elif "contrast" in name:
                        data["contrast"] = value
                    elif "type" in name and "film" in name:
                        data["film_type"] = value

                return data
        return {}

    def fetch_filmapi(
        self,
        url: str = "https://filmapi.vercel.app/api/films",
    ) -> List[FilmData]:
        """
        Fetches film data from the API and converts it to FilmData dataclass instances.
        """
        try:
            print(f"Fetching: {url}")
            response = self.session.get(url, timeout=10)
            response.raise_for_status()

            films_json = response.json()
            film_data_list = []
            for film in films_json:
                film_data = self.convert_filmapi(film)
                film_data_list.append(film_data)

            return film_data_list

        except requests.RequestException as e:
            print(f"Error fetching data: {e}")
            raise
        except json.JSONDecodeError as e:
            print(f"Error parsing JSON: {e}")
            raise

    def convert_filmapi(self, film_json: dict) -> FilmData:
        """
        Converts a single film JSON object to FilmData dataclass.
        """
        brand = film_json.get("brand", "").strip()
        name = film_json.get("name", "").strip()
        slug = self.slug(brand, name)

        is_color = film_json.get("color", False)
        process = film_json.get("process", "").lower()
        if is_color:
            color_type = "color"
        elif "c-41" in process:
            color_type = "color"
        elif "b&w" in process or "black" in process:
            color_type = "bw"
        else:
            color_type = "unknown"

        return FilmData(
            type=name,
            slug=slug,
            make=brand if brand else None,
            speed=film_json.get("iso"),
            color_type=color_type,
            description=film_json.get("description"),
        )

    def extract_fallback_data(self, text: str) -> dict:
        data = {}

        iso_match = re.search(r"ISO\s*(\d+)", text, re.IGNORECASE)
        if iso_match:
            data["iso_speed"] = iso_match.group(1)

        if re.search(r"black.{0,5}white|b.{0,2}w|monochrome", text, re.IGNORECASE):
            data["color_type"] = "bw"
        elif re.search(r"color|colour", text, re.IGNORECASE):
            data["color_type"] = "color"

        return data

    def slug(self, film_make: Optional[str], film_type: Optional[str]) -> str:
        return (
            f"{film_make}-{film_type}".lower().replace(" ", "-")
            if film_make and film_type
            else ""
        )

    def scrape_individual(self, film_link: dict) -> Optional[FilmData]:
        soup = self.get_page(film_link["url"])
        if not soup:
            return None

        film = FilmData(type=film_link["name"], slug=film_link["slug"])

        json_ld = soup.find("script", type="application/ld+json")
        if json_ld:
            try:
                structured_data = json.loads(json_ld.string)
                jsonld_data = self.extract_json(structured_data)

                film.make = jsonld_data.get("manufacturer")
                film.description = jsonld_data.get("description")
                film.color_type = jsonld_data.get("color_type")
                film.speed = jsonld_data.get("iso_speed")

            except json.JSONDecodeError:
                pass

        if not film.speed or not film.color_type:
            text = soup.get_text()
            fallback_data = self.extract_fallback_data(text)

            film.speed = film.speed or fallback_data.get("iso_speed")
            film.color_type = film.color_type or fallback_data.get("color_type")

        return film

    def normalize(self, film: FilmData) -> FilmData:
        if film.make:
            film.make = film.make.lower()

        if film.type:
            film.type = film.type.lower()

        if film.speed:
            film.speed = int(film.speed)

        if film.color_type and film.color_type.casefold() == "black & white":
            film.color_type = "bw"
        if film.color_type and film.color_type.casefold() == "color negative":
            film.color_type = "color"

        if film.make and film.type:
            pattern = re.escape(film.make) + r"\s*"
            new_type = re.sub(pattern, "", film.type, flags=re.IGNORECASE).strip()
            film.type = new_type

        # drop 'professional' from films
        new_type = film.type.strip("professional").strip()
        film.type = new_type

        # rename fomapan -> foma
        if film.make and film.make.casefold() == "fomapan":
            film.make = "foma"

        # drop 'fuji' from fujifilm
        if film.make and film.make.casefold() == "fujifilm":
            new_type = film.type.strip("fuji ").strip()
            film.type = new_type

        film.slug = self.slug(film.make, film.type)
        return film

    def scrape_all(self, limit: Optional[int] = None) -> List[FilmData]:
        """Scrape all films with rate limiting"""
        film_links = self.get_all_links()

        if limit:
            film_links = film_links[:limit]
            print(f"Scraping {limit} films")

        films_data = []

        # scrape from filmtypes
        for i, film_link in enumerate(film_links, 1):
            print(f"Scraping film {i}/{len(film_links)}: {film_link['name']}")

            film_data = self.scrape_individual(film_link)
            if film_data:
                normalized = self.normalize(film_data)
                films_data.append(normalized)

            if i < len(film_links):
                time.sleep(1)

        # scrape from filmapi
        films = self.fetch_filmapi()
        for f in films:
            normalized = self.normalize(f)
            films_data.append(normalized)

        return films_data

    def deduplicate(self, films: List[FilmData]) -> List[FilmData]:
        out: List[FilmData] = []
        seen = set()

        for f in films:
            key = (f.type, f.make, f.speed)

            if key not in seen:
                seen.add(key)
                out.append(f)

        out.sort(key=lambda film: film.slug)
        return out

    def save_data(self, films_data: List[FilmData]):
        file = "films.json"
        with open(file, "w", encoding="utf-8") as f:
            json.dump(
                [asdict(film) for film in films_data],
                f,
                indent=2,
                ensure_ascii=False,
            )
        print(f"Saved data to {file}")


def main():
    scraper = FilmScraper()

    films = scraper.scrape_all()
    if films:
        print(f"\nSuccessfully scraped {len(films)} films")
        dedupe = scraper.deduplicate(films)
        print(f"\nAfter deduplicating {len(dedupe)} films")
        scraper.save_data(dedupe)

    else:
        print("No films were scraped successfully")


if __name__ == "__main__":
    main()
