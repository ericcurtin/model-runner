#!/bin/bash

set -eux -o pipefail

echo "Installing curl..."
apt-get update -qq 1>/dev/null
apt-get install -y curl 1>/dev/null

echo "Installing Docker CE..."
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh 1>/dev/null

echo "Testing docker model version..."
version_output=$(docker model version 2>&1 || true)
echo "Output: $version_output"

if echo "$version_output" | grep -q "version $EXPECTED_VERSION"; then
  echo "✓ Success: Found expected version $EXPECTED_VERSION"
  exit 0
else
  echo "✗ Error: Expected version $EXPECTED_VERSION not found in output"
  exit 1
fi
