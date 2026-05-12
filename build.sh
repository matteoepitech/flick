#!/usr/bin/env bash
set -euo pipefail

VERSION="${CI_COMMIT_TAG:-${CI_COMMIT_SHORT_SHA:-$(git rev-parse --short HEAD 2>/dev/null || echo dev)}}"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

export CGO_ENABLED=0

echo "Build version: $VERSION (commit=$COMMIT)"
mkdir -p build/bin

LDFLAGS="-s -w -X main.CLIVersion=$VERSION -X main.CLICommit=$COMMIT -X main.CLIBuildDate=$BUILD_DATE"

PLATFORMS=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64")
BINARIES=("api" "cli")

for PLATFORM in "${PLATFORMS[@]}"; do
  GOOS="${PLATFORM%/*}"
  GOARCH="${PLATFORM#*/}"
  SUFFIX=""
  [ "$GOOS" = "windows" ] && SUFFIX=".exe"

  for BIN in "${BINARIES[@]}"; do
    echo "  → flick-${BIN} ${GOOS}/${GOARCH}"
    GOOS=$GOOS GOARCH=$GOARCH go build \
      -ldflags="$LDFLAGS" \
      -o "build/bin/flick-${BIN}-${GOOS}-${GOARCH}${SUFFIX}" \
      "./cmd/${BIN}"
  done
done

BASE_URL="${RELEASE_BASE_URL:-https://flick.d3l.tech/releases}"

cat > build/bin/version.json << EOF
{
  "version": "$VERSION",
  "commit": "$COMMIT",
  "build_date": "$BUILD_DATE",
  "url_linux_amd64":   "$BASE_URL/$VERSION/flick-cli-linux-amd64",
  "url_linux_arm64":   "$BASE_URL/$VERSION/flick-cli-linux-arm64",
  "url_darwin_amd64":  "$BASE_URL/$VERSION/flick-cli-darwin-amd64",
  "url_darwin_arm64":  "$BASE_URL/$VERSION/flick-cli-darwin-arm64",
  "url_windows_amd64": "$BASE_URL/$VERSION/flick-cli-windows-amd64.exe"
}
EOF

cd build/bin
sha256sum flick-* > checksums.txt
cd -

echo "Build finished: $VERSION"
