#!/usr/bin/env bash
set -e

# Version : tag git > hash court > "dev"
VERSION=${CI_COMMIT_TAG:-${CI_COMMIT_SHORT_SHA:-$(git rev-parse --short HEAD 2>/dev/null || echo "dev")}}

echo "Build version : $VERSION"

mkdir -p build/bin

PLATFORMS=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64")

for PLATFORM in "${PLATFORMS[@]}"; do
  GOOS="${PLATFORM%/*}"
  GOARCH="${PLATFORM#*/}"

  SUFFIX=""
  [ "$GOOS" = "windows" ] && SUFFIX=".exe"

  echo "  Compilation $GOOS/$GOARCH..."

  # Flick API
  GOOS=$GOOS GOARCH=$GOARCH go build \
    -ldflags="-X main.Version=$VERSION" \
    -o "build/bin/flick-api-${GOOS}-${GOARCH}${SUFFIX}" \
    ./cmd/api

  # Flick CLI
  GOOS=$GOOS GOARCH=$GOARCH go build \
    -ldflags="-X main.Version=$VERSION" \
    -o "build/bin/flick-cli-${GOOS}-${GOARCH}${SUFFIX}" \
    ./cmd/cli
done

BASE_URL=${RELEASE_BASE_URL:-"https://flick.d3l.tech/releases"}

cat > build/bin/version.json << EOF
{
  "version": "$VERSION",
  "url_linux_amd64":   "$BASE_URL/$VERSION/flick-cli-linux-amd64",
  "url_linux_arm64":   "$BASE_URL/$VERSION/flick-cli-linux-arm64",
  "url_darwin_amd64":  "$BASE_URL/$VERSION/flick-cli-darwin-amd64",
  "url_darwin_arm64":  "$BASE_URL/$VERSION/flick-cli-darwin-arm64",
  "url_windows_amd64": "$BASE_URL/$VERSION/flick-cli-windows-amd64.exe"
}
EOF

# Checksums
cd build/bin
sha256sum flick-* > checksums.txt
cd -

echo "Build finished: $VERSION"
