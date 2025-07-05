#!/bin/bash

# Configuration
QUERIES_DIR="queries"
OUTPUT_DIR="missing"
LLM_MODEL="openrouter/openai/gpt-4.1"

# Check if queries directory exists
if [ ! -d "$QUERIES_DIR" ]; then
    echo "Error: $QUERIES_DIR directory not found!"
    exit 1
fi

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Count query files
query_count=$(ls -1 "$QUERIES_DIR"/*.txt 2>/dev/null | wc -l)
if [ $query_count -eq 0 ]; then
    echo "Error: No query files found in $QUERIES_DIR directory!"
    exit 1
fi

echo "Found $query_count query files to process"
echo "Model: $LLM_MODEL"
echo "Output directory: $OUTPUT_DIR"
echo ""

# Process each query file
counter=1
for query_file in "$QUERIES_DIR"/*.txt; do
    # Extract query filename without path and extension
    query_name=$(basename "$query_file" .txt)
    output_file="$OUTPUT_DIR/missing_$(printf "%03d" $counter).json"

    echo "[$counter/$query_count] Processing $query_name..."
    echo "  Input: $query_file"
    echo "  Output: $output_file"

    # Run the LLM query and save output
    if cat "$query_file" | llm -m "$LLM_MODEL" >"$output_file" 2>&1; then
        # Check if the output is valid JSON
        if jq . "$output_file" >/dev/null 2>&1; then
            # Get counts from the JSON
            missing_cameras=$(jq '.missing_cameras | length' "$output_file" 2>/dev/null || echo "0")
            missing_films=$(jq '.missing_films | length' "$output_file" 2>/dev/null || echo "0")

            echo "  ✓ Success: $missing_cameras cameras, $missing_films films"
        else
            echo "  ⚠ Warning: Output is not valid JSON"
            echo "  First few lines of output:"
            head -n 3 "$output_file" | sed 's/^/    /'
        fi
    else
        echo "  ✗ Error: LLM command failed"
        echo "  Error output saved to $output_file"
    fi

    echo ""
    counter=$((counter + 1))

    # Optional: Add delay between requests to be respectful to API
    sleep 1
done

echo "=== SUMMARY ==="
echo "Processed $query_count query files"
echo "Output files created in ./$OUTPUT_DIR/"
echo ""

# Summary of results
echo "Results summary:"
total_cameras=0
total_films=0
valid_files=0
invalid_files=0

for output_file in "$OUTPUT_DIR"/*.json; do
    if [ -f "$output_file" ]; then
        if jq . "$output_file" >/dev/null 2>&1; then
            cameras=$(jq '.missing_cameras | length' "$output_file" 2>/dev/null || echo "0")
            films=$(jq '.missing_films | length' "$output_file" 2>/dev/null || echo "0")
            total_cameras=$((total_cameras + cameras))
            total_films=$((total_films + films))
            valid_files=$((valid_files + 1))
            echo "  $(basename "$output_file"): $cameras cameras, $films films"
        else
            invalid_files=$((invalid_files + 1))
            echo "  $(basename "$output_file"): INVALID JSON"
        fi
    fi
done

echo ""
echo "Total missing cameras found: $total_cameras"
echo "Total missing films found: $total_films"
echo "Valid JSON files: $valid_files"
echo "Invalid files: $invalid_files"

if [ $invalid_files -gt 0 ]; then
    echo ""
    echo "⚠ Some files contain invalid JSON. Check them manually:"
    for output_file in "$OUTPUT_DIR"/*.json; do
        if [ -f "$output_file" ] && ! jq . "$output_file" >/dev/null 2>&1; then
            echo "  $output_file"
        fi
    done
fi
