#!/bin/bash

# Configuration
BASE_URL="https://api.analogdb.com/posts"
PAGE_SIZE=200
MAX_POSTS=10000
OUTPUT_FILE="titles.txt"

# Initialize variables
total_posts=0
next_page_id=""
page_count=0

# Clear the output file
>"$OUTPUT_FILE"

echo "Starting to fetch posts..."

while [ $total_posts -lt $MAX_POSTS ]; do
    page_count=$((page_count + 1))

    # Build the URL
    if [ -z "$next_page_id" ]; then
        # First request
        url="${BASE_URL}?page_size=${PAGE_SIZE}"
    else
        # Subsequent requests with page_id
        url="${BASE_URL}?page_size=${PAGE_SIZE}&page_id=${next_page_id}"
    fi

    echo "Fetching page $page_count from: $url"

    # Make the API call and store response
    response=$(curl -s "$url")

    # Check if curl was successful
    if [ $? -ne 0 ]; then
        echo "Error: Failed to fetch data from API"
        exit 1
    fi

    # Extract titles and append to file
    echo "$response" | jq -r '.posts[].title' >>"$OUTPUT_FILE"

    # Check if jq was successful
    if [ $? -ne 0 ]; then
        echo "Error: Failed to parse JSON response"
        exit 1
    fi

    # Get the count of posts in this response
    posts_in_response=$(echo "$response" | jq '.posts | length')

    # Update total count
    total_posts=$((total_posts + posts_in_response))

    echo "Fetched $posts_in_response posts. Total so far: $total_posts"

    # Get next page ID for next iteration
    next_page_id=$(echo "$response" | jq -r '.meta.next_page_id')

    # Check if we have a next page ID (if null, we're done)
    if [ "$next_page_id" = "null" ]; then
        echo "No more pages available. Stopping."
        break
    fi

    # Optional: Add a small delay to be respectful to the API
    sleep 0.5
done

echo "Finished! Total posts fetched: $total_posts"
echo "Titles saved to: $OUTPUT_FILE"
echo "Total pages fetched: $page_count"
