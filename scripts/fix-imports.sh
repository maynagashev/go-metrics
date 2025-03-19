#!/bin/bash

# Script to fix import ordering in all Go files
# Uses gci to organize imports according to Go conventions
# Skips files with blank imports to avoid issues

# Ensure the script exits if any command fails
set -e

# Check if gci is installed
if ! command -v gci &> /dev/null; then
    echo "gci is not installed. Installing..."
    go install github.com/daixiang0/gci@latest
fi

# Fix import ordering in all Go files
echo "Fixing import ordering in all Go files..."

# Find all Go files and process them
find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" | while read -r file; do
    # Skip files that have blank imports (imports with underscore prefix) to avoid issues
    if grep -q "_[[:space:]]*\"" "$file" || grep -q "import[[:space:]]*(" "$file" | grep -q "_[[:space:]]"; then
        echo "Skipping file with blank imports: $file"
    else
        gci write --skip-generated -s standard -s blank -s default -s blank -s "prefix(github.com/maynagashev/go-metrics)" "$file"
    fi
done

echo "Import ordering fixed successfully!" 