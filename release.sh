#!/bin/sh
set -eu

APP_NAME="ComicDaysGoDownloader"
DIST_DIR="${DIST_DIR:-dist}"
DEFAULT_PLATFORMS="darwin/amd64 darwin/arm64 dragonfly/amd64 freebsd/386 freebsd/amd64 freebsd/arm freebsd/arm64 linux/386 linux/amd64 linux/arm linux/arm64 linux/riscv64 netbsd/386 netbsd/amd64 netbsd/arm netbsd/arm64 openbsd/386 openbsd/amd64 openbsd/arm openbsd/arm64 windows/386 windows/amd64 windows/arm64"

if [ -n "${1-}" ]; then
  VERSION="$1"
else
  printf 'Version: '
  IFS= read -r VERSION || {
    printf '\nVersion is required\n' >&2
    exit 1
  }

  if [ -z "$VERSION" ]; then
    printf 'Version is required\n' >&2
    exit 1
  fi
fi

case "$VERSION" in
  v*) ;;
  *) VERSION="v$VERSION" ;;
esac

command -v go >/dev/null 2>&1 || {
  printf '%s\n' "go is required" >&2
  exit 1
}

PLATFORMS="${PLATFORMS:-$DEFAULT_PLATFORMS}"

command -v zip >/dev/null 2>&1 || {
  printf '%s\n' "zip is required" >&2
  exit 1
}

SCRIPT_DIR=$(CDPATH= cd "$(dirname "$0")" && pwd)
cd "$SCRIPT_DIR"

case "$DIST_DIR" in
  /*) OUT_DIR="$DIST_DIR" ;;
  *) OUT_DIR="$SCRIPT_DIR/$DIST_DIR" ;;
esac

BUILD_DIR=$(mktemp -d "${TMPDIR:-/tmp}/${APP_NAME}.release.XXXXXX")
cleanup() {
  rm -rf "$BUILD_DIR"
}
trap cleanup EXIT INT TERM

mkdir -p "$OUT_DIR"
rm -f "$OUT_DIR/$APP_NAME-$VERSION-"*.zip

success_count=0
failure_count=0
failed_platforms=""

for platform in $PLATFORMS; do
  goos=${platform%/*}
  goarch=${platform#*/}
  target="${APP_NAME}-${VERSION}-${goos}-${goarch}"
  staging_dir="$BUILD_DIR/$target"
  binary_name="$APP_NAME"

  if [ "$goos" = "windows" ]; then
    binary_name="$APP_NAME.exe"
  fi

  mkdir -p "$staging_dir"

  printf 'Building %s...\n' "$target"
  if ! CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build -trimpath -ldflags="-s -w" -o "$staging_dir/$binary_name" .; then
    failure_count=$((failure_count + 1))
    failed_platforms="$failed_platforms
$platform"
    rm -rf "$staging_dir"
    printf 'Failed %s\n' "$target" >&2
    continue
  fi

  cp README.md LICENSE "$staging_dir/"

  archive="$OUT_DIR/$target.zip"
  rm -f "$archive"
  (cd "$staging_dir" && zip -q -9 -r "$archive" .)
  success_count=$((success_count + 1))
  printf 'Created %s\n' "$archive"
done

printf 'Release artifacts are in %s\n' "$OUT_DIR"
printf 'Built %d target(s).\n' "$success_count"

if [ "$failure_count" -gt 0 ]; then
  printf 'Skipped %d failed target(s):%s\n' "$failure_count" "$failed_platforms" >&2
fi

if [ "$success_count" -eq 0 ]; then
  exit 1
fi
