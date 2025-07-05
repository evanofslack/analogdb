#!/usr/bin/env python3

import json
import os
import glob
from collections import defaultdict
from typing import Dict, List, Any


def load_json_file(filepath: str) -> Dict[str, Any]:
    """Load and validate a JSON file."""
    try:
        with open(filepath, "r") as f:
            data = json.load(f)

        # Validate structure
        if not isinstance(data, dict):
            raise ValueError("Root is not a dictionary")

        if "missing_cameras" not in data or "missing_films" not in data:
            raise ValueError("Missing required keys: missing_cameras, missing_films")

        if not isinstance(data["missing_cameras"], list) or not isinstance(
            data["missing_films"], list
        ):
            raise ValueError("missing_cameras and missing_films must be lists")

        return data

    except json.JSONDecodeError as e:
        raise ValueError(f"Invalid JSON: {e}")
    except Exception as e:
        raise ValueError(f"Error loading file: {e}")


def aggregate_cameras(all_cameras: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
    """Aggregate cameras by make and model."""
    camera_dict = defaultdict(
        lambda: {
            "count": 0,
            "camera_make": "",
            "camera_model": "",
            "confidences": set(),
            "notes": set(),
        }
    )

    for camera in all_cameras:
        # Create unique key
        make = camera.get("camera_make", "").lower().strip()
        model = camera.get("camera_model", "").lower().strip()
        key = f"{make}|{model}"

        # Update aggregated data
        camera_dict[key]["count"] += 1
        camera_dict[key]["camera_make"] = camera.get("camera_make", "")
        camera_dict[key]["camera_model"] = camera.get("camera_model", "")

        # Add confidence and notes
        if "confidence" in camera:
            camera_dict[key]["confidences"].add(camera["confidence"])
        if "notes" in camera:
            camera_dict[key]["notes"].add(camera["notes"])

    # Convert sets to sorted lists and return as list
    result = []
    for data in camera_dict.values():
        result.append(
            {
                "count": data["count"],
                "camera_make": data["camera_make"],
                "camera_model": data["camera_model"],
                "confidences": sorted(list(data["confidences"])),
                "notes": sorted(list(data["notes"])),
            }
        )

    return sorted(result, key=lambda x: x["count"], reverse=True)


def aggregate_films(all_films: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
    """Aggregate films by make and type."""
    film_dict = defaultdict(
        lambda: {
            "count": 0,
            "film_make": "",
            "film_type": "",
            "confidences": set(),
            "notes": set(),
        }
    )

    for film in all_films:
        # Create unique key
        make = film.get("film_make", "") or ""
        film_type = film.get("film_type", "") or ""

        make = make.lower().strip()
        film_type = film_type.lower().strip()

        key = f"{make}|{film_type}"

        # Update aggregated data
        film_dict[key]["count"] += 1
        film_dict[key]["film_make"] = film.get("film_make", "")
        film_dict[key]["film_type"] = film.get("film_type", "")

        # Add confidence and notes
        if "confidence" in film:
            film_dict[key]["confidences"].add(film["confidence"])
        if "notes" in film:
            film_dict[key]["notes"].add(film["notes"])

    # Convert sets to sorted lists and return as list
    result = []
    for data in film_dict.values():
        result.append(
            {
                "count": data["count"],
                "film_make": data["film_make"],
                "film_type": data["film_type"],
                "confidences": sorted(list(data["confidences"])),
                "notes": sorted(list(data["notes"])),
            }
        )

    return sorted(result, key=lambda x: x["count"], reverse=True)


def main():
    missing_dir = "missing"
    output_file = "aggregated_results.json"

    # Check if missing directory exists
    if not os.path.exists(missing_dir):
        print(f"Error: {missing_dir} directory not found!")
        return 1

    # Find all JSON files
    json_files = glob.glob(os.path.join(missing_dir, "*.json"))

    if not json_files:
        print(f"Error: No JSON files found in {missing_dir} directory!")
        return 1

    print(f"Found {len(json_files)} JSON files to aggregate")
    print(f"Output file: {output_file}")
    print()

    # Collect all cameras and films
    all_cameras = []
    all_films = []
    processed_files = 0
    invalid_files = 0

    for json_file in sorted(json_files):
        print(f"Processing {os.path.basename(json_file)}...")

        try:
            data = load_json_file(json_file)

            # Extract cameras and films
            cameras = data.get("missing_cameras", [])
            films = data.get("missing_films", [])

            all_cameras.extend(cameras)
            all_films.extend(films)

            print(f"  ✓ Processed: {len(cameras)} cameras, {len(films)} films")
            processed_files += 1

        except ValueError as e:
            print(f"  ⚠ Skipping invalid file: {e}")
            invalid_files += 1
        except Exception as e:
            print(f"  ⚠ Error processing file: {e}")
            invalid_files += 1

    print()
    print("=== AGGREGATING DATA ===")

    # Aggregate the data
    aggregated_cameras = aggregate_cameras(all_cameras)
    aggregated_films = aggregate_films(all_films)

    # Create final output
    result = {"missing_cameras": aggregated_cameras, "missing_films": aggregated_films}

    # Save to file
    with open(output_file, "w") as f:
        json.dump(result, f, indent=2)

    print("=== AGGREGATION COMPLETE ===")
    print(f"Processed files: {processed_files}")
    print(f"Invalid files skipped: {invalid_files}")
    print(f"Output saved to: {output_file}")
    print()

    # Summary statistics
    total_cameras = len(aggregated_cameras)
    total_films = len(aggregated_films)

    print("Summary:")
    print(f"  Unique missing cameras: {total_cameras}")
    print(f"  Unique missing films: {total_films}")
    print()

    # Show top items by count
    print("Top 5 most frequently missing cameras:")
    for camera in aggregated_cameras[:5]:
        print(
            f"  {camera['camera_make']} {camera['camera_model']} (count: {camera['count']})"
        )

    print()
    print("Top 5 most frequently missing films:")
    for film in aggregated_films[:5]:
        print(f"  {film['film_make']} {film['film_type']} (count: {film['count']})")

    print()
    print(f"Aggregation complete! Check {output_file} for full results.")

    return 0


if __name__ == "__main__":
    exit(main())
