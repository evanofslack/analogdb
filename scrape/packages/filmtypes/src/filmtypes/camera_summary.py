import asyncio
import json
import re
from dataclasses import dataclass, asdict
from typing import List
import logging
import os

from openai import AsyncOpenAI
from dotenv import load_dotenv

load_dotenv()

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)


@dataclass
class CameraInfo:
    make: str
    model: str
    description: str
    found: bool


class CameraResearcher:
    def __init__(self):
        self.client = AsyncOpenAI(
            api_key=os.getenv("OPENROUTER_API_KEY"),
            base_url=os.getenv("OPENROUTER_BASE_URL"),
        )
        self.model = os.getenv("OPENROUTER_MODEL")
        self.results: List[CameraInfo] = []

    def create_research_prompt(self, camera_name: str) -> str:
        return f"""You are a film camera database researcher. Research the film camera: "{camera_name}"

TASK:
1. Search for technical information about this camera model
2. Prioritize official sources: camera-wiki.org, manufacturer websites, Wikipedia
3. If not found as written, note that you cannot find it
4. Create a factual 6 sentence technical summary for a camera database

REQUIREMENTS:
- Focus on: manufacturer, year released, format, key technical specifications, notable features
- Do not mention price information
- Be concise and factual
- If camera name is unclear or not found, state "Camera not found as specified"

OUTPUT FORMAT:
Return ONLY a JSON object with these exact fields:
{{
  "make": "manufacturer_name_lowercase",
  "model": "model_name_lowercase", 
  "description": "5-6 sentence technical summary",
  "found": true/false
}}

Camera to research: {camera_name}"""

    def parse_llm_response(
        self, response_text: str, original_camera: str
    ) -> CameraInfo:
        try:
            json_match = re.search(r"\{.*\}", response_text, re.DOTALL)
            if json_match:
                json_str = json_match.group()
                data = json.loads(json_str)

                if all(
                    key in data for key in ["make", "model", "description", "found"]
                ):
                    return CameraInfo(
                        make=str(data["make"]).lower().strip(),
                        model=str(data["model"]).lower().strip(),
                        description=str(data["description"]).strip(),
                        found=bool(data["found"]),
                    )

            logger.warning(
                f"Could not parse JSON for {original_camera}, attempting fallback"
            )
            return CameraInfo(
                make="unknown",
                model=original_camera.lower().strip(),
                description="Could not retrieve camera information",
                found=False,
            )

        except (json.JSONDecodeError, KeyError) as e:
            logger.error(f"Error parsing response for {original_camera}: {e}")
            return CameraInfo(
                make="unknown",
                model=original_camera.lower().strip(),
                description="Error retrieving camera information",
                found=False,
            )

    async def research_camera(self, camera_name: str) -> CameraInfo:
        if self.model is None:
            raise Exception("model must be set")

        try:
            prompt = self.create_research_prompt(camera_name)
            logger.info(f"Researching: {camera_name}")

            response = await self.client.chat.completions.create(
                model=self.model,
                messages=[
                    {
                        "role": "system",
                        "content": "You are a technical camera database researcher with web search capabilities. Provide accurate, factual information about camera specifications.",
                    },
                    {"role": "user", "content": prompt},
                ],
                max_tokens=800,
                temperature=0.1,
            )

            response_text = response.choices[0].message.content
            if response_text is None:
                raise Exception("response text is empty")

            result = self.parse_llm_response(response_text, camera_name)

            if result.found:
                logger.info(f"✓ Found: {camera_name}")
            else:
                logger.warning(f"✗ Not found: {camera_name}")

            return result

        except Exception as e:
            logger.error(f"Error researching {camera_name}: {e}")
            return CameraInfo(
                make="error",
                model=camera_name.lower().strip(),
                description=f"API error: {str(e)}",
                found=False,
            )

    async def process_camera_list(
        self, camera_list: List[str], max_concurrent: int = 3
    ) -> List[CameraInfo]:
        semaphore = asyncio.Semaphore(max_concurrent)

        async def bounded_research(camera):
            async with semaphore:
                return await self.research_camera(camera)

        tasks = [
            bounded_research(camera.strip()) for camera in camera_list if camera.strip()
        ]
        results = await asyncio.gather(*tasks, return_exceptions=True)

        valid_results = []
        for i, result in enumerate(results):
            if isinstance(result, Exception):
                logger.error(f"Exception for camera {camera_list[i]}: {result}")
                valid_results.append(
                    CameraInfo(
                        make="error",
                        model=camera_list[i].lower().strip(),
                        description="Processing error occurred",
                        found=False,
                    )
                )
            else:
                valid_results.append(result)

        return valid_results


def load_camera_list(file_path: str) -> List[str]:
    try:
        with open(file_path, "r", encoding="utf-8") as f:
            cameras = [line.strip() for line in f if line.strip()]
        logger.info(f"Loaded {len(cameras)} cameras from {file_path}")
        return cameras
    except FileNotFoundError:
        logger.error(f"File not found: {file_path}")
        return []
    except Exception as e:
        logger.error(f"Error loading camera list: {e}")
        return []


def save_results(results: List[CameraInfo], output_path: str):
    try:
        results_dict = [asdict(r) for r in results]

        with open(output_path, "w", encoding="utf-8") as f:
            json.dump(results_dict, f, indent=2, ensure_ascii=False)

        found_count = sum(1 for r in results if r.found)
        logger.info(
            f"Saved {len(results)} results to {output_path} ({found_count} found)"
        )

    except Exception as e:
        logger.error(f"Error saving results: {e}")


async def main():
    input_file = "cameras.todo"
    output_file = "camera_database.json"
    max_concurrent_requests = 3

    cameras = load_camera_list(input_file)
    if not cameras:
        logger.error("No cameras to process")
        return

    researcher = CameraResearcher()

    logger.info(f"Starting research on {len(cameras)} cameras...")
    results = await researcher.process_camera_list(cameras, max_concurrent_requests)

    save_results(results, output_file)

    found = sum(1 for r in results if r.found)
    logger.info(f"Research complete: {found}/{len(results)} cameras found")


if __name__ == "__main__":
    asyncio.run(main())
