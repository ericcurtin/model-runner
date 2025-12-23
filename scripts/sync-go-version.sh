#!/usr/bin/env bash

set -euo pipefail

# Sync Go version from go.mod to other files in the repo
# Usage: ./scripts/sync-go-version.sh <check|sync>

usage() {
    echo "Usage: $0 <check|sync>"
    echo "  check  - Check if Go version is consistent across files"
    echo "  sync   - Sync Go version from go.mod to other files"
    exit 1
}

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

GO_VERSION=$(grep '^go ' go.mod | awk '{print $2}')
GO_VERSION_MINOR=$(echo "$GO_VERSION" | cut -d. -f1,2)

if [[ -z "$GO_VERSION" ]]; then
    echo "Error: Could not extract Go version from go.mod" >&2
    exit 1
fi

echo "Go version from go.mod: $GO_VERSION"
echo ""

update_file() {
    local file="$1"
    local pattern="$2"
    local replacement="$3"

    if [[ ! -f "$file" ]]; then
        return
    fi

    local current
    current=$(grep -E "$pattern" "$file" 2>/dev/null | head -1 || echo "")

    if [[ -z "$current" ]]; then
        return
    fi

    if [[ "$current" == "$replacement" ]]; then
        return
    fi

    if [[ "$(uname)" == "Darwin" ]]; then
        sed -i '' "s|$pattern|$replacement|" "$file"
    else
        sed -i "s|$pattern|$replacement|" "$file"
    fi
    echo "Updated: $file"
    echo "  $current -> $replacement"
}

NEEDS_UPDATE=0
check_file() {
    local file="$1"
    local expected_pattern="$2"
    local search_pattern="$3"
    local expected_value="$4"

    if [[ ! -f "$file" ]]; then
        return
    fi

    if ! grep -q "$expected_pattern" "$file"; then
        local current
        current=$(grep "$search_pattern" "$file" 2>/dev/null | head -1 | xargs)
        current=${current:-(not found)}
        echo "Mismatch: $(realpath "$file")"
        echo "  have: $current"
        echo "  want: $expected_value"
        NEEDS_UPDATE=1
    fi
}

# Files to update (excluding pkg/go-containerregistry)
GO_FILES=("go.work" "go.mod" "cmd/cli/go.mod")
MAKEFILE_FILES=()
while IFS= read -r -d '' file; do
    MAKEFILE_FILES+=("$file")
done < <(find . -name 'Makefile' -not -path './pkg/go-containerregistry/*' -print0)
DOCKERFILE_FILES=()
while IFS= read -r -d '' file; do
    DOCKERFILE_FILES+=("$file")
done < <(find . -name 'Dockerfile*' -not -path './pkg/go-containerregistry/*' -print0)
WORKFLOW_FILES=()
while IFS= read -r -d '' file; do
    WORKFLOW_FILES+=("$file")
done < <(find .github/workflows -name '*.yml' -print0 2>/dev/null || true)

case "${1:-}" in
    check)
        echo "Checking Go version consistency..."
        echo ""

        for gofile in "${GO_FILES[@]}"; do
            check_file "$gofile" "^go $GO_VERSION" "^go " "go $GO_VERSION"
        done

        for makefile in "${MAKEFILE_FILES[@]}"; do
            if grep -q "^GO_VERSION := " "$makefile" 2>/dev/null; then
                check_file "$makefile" "^GO_VERSION := $GO_VERSION" "^GO_VERSION := " "GO_VERSION := $GO_VERSION"
            fi
        done

        for dockerfile in "${DOCKERFILE_FILES[@]}"; do
            if grep -q "ARG GO_VERSION=" "$dockerfile" 2>/dev/null; then
                check_file "$dockerfile" "ARG GO_VERSION=$GO_VERSION_MINOR" "ARG GO_VERSION=" "ARG GO_VERSION=$GO_VERSION_MINOR"
            fi
        done

        for workflow in "${WORKFLOW_FILES[@]}"; do
            if grep -q "go-version:" "$workflow" 2>/dev/null; then
                check_file "$workflow" "go-version: $GO_VERSION" "go-version:" "go-version: $GO_VERSION"
            fi
        done

        if [[ "$NEEDS_UPDATE" -eq 1 ]]; then
            echo ""
            echo "Files are out of sync. Run: ./scripts/sync-go-version.sh sync"
            exit 1
        else
            echo "All files are in sync."
            exit 0
        fi
        ;;
    sync)
        echo "Syncing Go version to $GO_VERSION..."
        echo ""

        for gofile in "${GO_FILES[@]}"; do
            update_file "$gofile" "^go .*" "go $GO_VERSION"
        done

        for makefile in "${MAKEFILE_FILES[@]}"; do
            update_file "$makefile" "^GO_VERSION := .*" "GO_VERSION := $GO_VERSION"
        done

        for dockerfile in "${DOCKERFILE_FILES[@]}"; do
            update_file "$dockerfile" "ARG GO_VERSION=.*" "ARG GO_VERSION=$GO_VERSION_MINOR"
        done

        for workflow in "${WORKFLOW_FILES[@]}"; do
            update_file "$workflow" "go-version: .*" "go-version: $GO_VERSION"
        done

        echo ""
        echo "Done. Review changes with: git diff"
        ;;
    *)
        usage
        ;;
esac
