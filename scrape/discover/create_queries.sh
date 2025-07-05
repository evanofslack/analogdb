#!/bin/bash

# Configuration
BATCHES_DIR="batches"
FILMS_FILE="films.csv"
CAMERAS_FILE="cameras.csv"
QUERY_FILE="query.txt"
OUTPUT_DIR="queries"

# Check if required files exist
check_file() {
    if [ ! -f "$1" ]; then
        echo "Error: $1 not found!"
        exit 1
    fi
}

echo "Checking required files..."
check_file "$FILMS_FILE"
check_file "$CAMERAS_FILE"
check_file "$QUERY_FILE"

if [ ! -d "$BATCHES_DIR" ]; then
    echo "Error: $BATCHES_DIR directory not found!"
    exit 1
fi

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Count batch files
batch_count=$(ls -1 "$BATCHES_DIR"/*.csv 2>/dev/null | wc -l)
if [ $batch_count -eq 0 ]; then
    echo "Error: No CSV files found in $BATCHES_DIR directory!"
    exit 1
fi

echo "Found $batch_count batch files to process"

# Process each batch file
for batch_file in "$BATCHES_DIR"/*.csv; do
    # Extract batch filename without path and extension
    batch_name=$(basename "$batch_file" .csv)
    output_file="$OUTPUT_DIR/query_${batch_name}.txt"

    echo "Processing $batch_file → $output_file"

    # Create the aggregate query file
    {
        echo "# MAIN QUERY"
        echo "============"
        echo ""
        cat "$QUERY_FILE"
        echo ""
        echo ""

        echo "# POST TITLES DATA"
        echo "=================="
        echo ""
        echo "Post titles to analyze:"
        echo '```'
        cat "$batch_file"
        echo '```'
        echo ""
        echo ""

        echo "# EXISTING FILMS DATABASE"
        echo "========================"
        echo ""
        echo "Current films in database:"
        echo '```'
        cat "$FILMS_FILE"
        echo '```'
        echo ""
        echo ""

        echo "# EXISTING CAMERAS DATABASE"
        echo "=========================="
        echo ""
        echo "Current cameras in database:"
        echo '```'
        cat "$CAMERAS_FILE"
        echo '```'
        echo ""
        echo ""

        echo "Please analyze the post titles above and identify any missing films or cameras as instructed."

    } >"$output_file"

    # Show file size for reference
    file_size=$(wc -c <"$output_file")
    title_count=$(wc -l <"$batch_file")

    echo "  → Created $output_file (${file_size} bytes, ~${title_count} titles)"
done

echo ""
echo "=== SUMMARY ==="
echo "Created query files in ./$OUTPUT_DIR/ directory:"
ls -la "$OUTPUT_DIR"/*.txt | while read -r line; do
    file=$(echo "$line" | awk '{print $NF}')
    size=$(echo "$line" | awk '{print $5}')
    echo "  $(basename "$file"): $size bytes"
done

echo ""
echo "Total query files created: $(ls -1 "$OUTPUT_DIR"/*.txt | wc -l)"
echo ""
echo "You can now send each query file to your LLM for analysis."
echo "Example: cat queries/query_titles_batch_001.txt | your_llm_command"
