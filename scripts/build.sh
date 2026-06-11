#!/usr/bin/env bash
set -euo pipefail

VERSION="${CI_COMMIT_TAG:-${CI_COMMIT_SHORT_SHA:-$(git rev-parse --short HEAD 2>/dev/null || echo dev)}}"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

export CGO_ENABLED=0

echo "Build version: $VERSION (commit=$COMMIT)"
mkdir -p build/bin

CLI_PKG="github.com/matteoepitech/flick/internal/cli"
LDFLAGS="-s -w -X ${CLI_PKG}.CLIVersion=$VERSION -X ${CLI_PKG}.CLICommit=$COMMIT -X ${CLI_PKG}.CLIBuildDate=$BUILD_DATE"

PLATFORMS=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64")

for PLATFORM in "${PLATFORMS[@]}"; do
  GOOS="${PLATFORM%/*}"
  GOARCH="${PLATFORM#*/}"
  SUFFIX=""
  [ "$GOOS" = "windows" ] && SUFFIX=".exe"

  echo "  → flick ${GOOS}/${GOARCH}"
  GOOS=$GOOS GOARCH=$GOARCH go build \
    -ldflags="$LDFLAGS" \
    -o "build/bin/flick-${GOOS}-${GOARCH}${SUFFIX}" \
    "./cmd/cli"
done

BASE_URL="${RELEASE_BASE_URL:-https://apt.d3l.tech/releases}"

cat > build/bin/version.json << EOF
{
  "version": "$VERSION",
  "commit": "$COMMIT",
  "build_date": "$BUILD_DATE",
  "url_linux_amd64":   "$BASE_URL/$VERSION/flick-linux-amd64",
  "url_linux_arm64":   "$BASE_URL/$VERSION/flick-linux-arm64",
  "url_darwin_amd64":  "$BASE_URL/$VERSION/flick-darwin-amd64",
  "url_darwin_arm64":  "$BASE_URL/$VERSION/flick-darwin-arm64",
  "url_windows_amd64": "$BASE_URL/$VERSION/flick-windows-amd64.exe"
}
EOF

cd build/bin
sha256sum flick-* > checksums.txt
cd -

echo "Build finished: $VERSION"
