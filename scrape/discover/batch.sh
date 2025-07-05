#!/bin/bash

# Configuration
INPUT_FILE="titles.csv"
BATCH_SIZE=200
OUTPUT_PREFIX="titles_batch_"

# Check if input file exists
if [ ! -f "$INPUT_FILE" ]; then
    echo "Error: $INPUT_FILE not found!"
    exit 1
fi

# Get total number of lines (excluding header if present)
total_lines=$(wc -l <"$INPUT_FILE")
echo "Total lines in $INPUT_FILE: $total_lines"

# Check if file has a header by looking at first line
first_line=$(head -n 1 "$INPUT_FILE")
if [[ "$first_line" =~ ^[[:space:]]*[\"\']*[Tt]itle[\"\']*[[:space:]]*$ ]] || [[ "$first_line" == *"title"* ]]; then
    echo "Header detected: $first_line"
    has_header=true
    data_lines=$((total_lines - 1))
else
    echo "No header detected"
    has_header=false
    data_lines=$total_lines
fi

echo "Data lines to split: $data_lines"

# Calculate number of batches needed
num_batches=$(((data_lines + BATCH_SIZE - 1) / BATCH_SIZE))
echo "Will create $num_batches batch files"

# Create output directory
mkdir -p batches
cd batches

# Function to create batch files
create_batches() {
    local start_line=1
    local batch_num=1

    if [ "$has_header" = true ]; then
        # Extract header
        header=$(head -n 1 "../$INPUT_FILE")
        start_line=2
    fi

    while [ $batch_num -le $num_batches ]; do
        local output_file="${OUTPUT_PREFIX}$(printf "%03d" $batch_num).csv"
        echo "Creating $output_file..."

        # Add header if it exists
        if [ "$has_header" = true ]; then
            echo "$header" >"$output_file"
        else
            >"$output_file"
        fi

        # Calculate line range for this batch
        local end_line=$((start_line + BATCH_SIZE - 1))

        # Extract lines for this batch
        sed -n "${start_line},${end_line}p" "../$INPUT_FILE" >>"$output_file"

        # Count actual lines added (excluding header)
        local lines_in_batch
        if [ "$has_header" = true ]; then
            lines_in_batch=$(($(wc -l <"$output_file") - 1))
        else
            lines_in_batch=$(wc -l <"$output_file")
        fi

        echo "  → $lines_in_batch titles in $output_file"

        # Update for next batch
        start_line=$((end_line + 1))
        batch_num=$((batch_num + 1))

        # Break if we've processed all data
        if [ $start_line -gt $total_lines ]; then
            break
        fi
    done
}

# Create the batch files
create_batches

# Summary
echo ""
echo "=== SUMMARY ==="
echo "Created batch files in ./batches/ directory:"
ls -la *.csv | while read -r line; do
    file=$(echo "$line" | awk '{print $NF}')
    count=$(wc -l <"$file")
    if [ "$has_header" = true ]; then
        count=$((count - 1))
    fi
    echo "  $file: $count titles"
done

echo ""
echo "Total batch files created: $(ls -1 *.csv | wc -l)"
echo "You can now process each batch file individually with your LLM."
