#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
root=$(pwd)

if [ "$(uname -s)" != Darwin ]; then
	echo "scripts/release.sh runs on macOS" >&2
	exit 1
fi
if [ $# -gt 1 ]; then
	echo "usage: scripts/release.sh [version]" >&2
	exit 2
fi
if [ -n "$(git status --porcelain)" ]; then
	echo "working tree is not clean; commit first" >&2
	exit 1
fi

if [ $# -eq 1 ]; then
	version=$1
else
	version=$(git describe --tags --exact-match 2>/dev/null) || {
		echo "HEAD has no tag; tag it or pass a version: scripts/release.sh 1.2.3" >&2
		exit 1
	}
fi
version=${version#v}
case $version in
'' | *[!0-9A-Za-z.-]*)
	echo "bad version: $version" >&2
	exit 1
	;;
esac
bundle_version=$(printf '%s' "$version" | sed 's/[^0-9.].*//; s/\.$//')
[ -n "$bundle_version" ] || bundle_version=0
min_macos=13.0

out="$root/release"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
mkdir -p "$out"

for arch in arm64 amd64; do
	CGO_ENABLED=1 GOOS=darwin GOARCH=$arch \
		MACOSX_DEPLOYMENT_TARGET=$min_macos \
		CGO_CFLAGS="-O2 -g -mmacosx-version-min=$min_macos" \
		CGO_LDFLAGS="-mmacosx-version-min=$min_macos" \
		go build -trimpath -o "$work/caution-term-$arch" .
done
lipo -create -output "$work/caution-term" "$work/caution-term-arm64" "$work/caution-term-amd64"

go run github.com/nullentropy/caution/cmd/appbundle -bin "$work/caution-term" -name "Caution Term" \
	-id dev.caution.term -icon icon.png -o "$work/stage"
app="$work/stage/Caution Term.app"
plutil -replace CFBundleShortVersionString -string "$version" "$app/Contents/Info.plist"
plutil -replace CFBundleVersion -string "$bundle_version" "$app/Contents/Info.plist"
plutil -replace LSMinimumSystemVersion -string "$min_macos" "$app/Contents/Info.plist"
codesign --force -s - "$app"
codesign --verify --strict "$app"

zip="$out/CautionTerm-$version-macos-universal.zip"
latest="$out/CautionTerm-macos-universal.zip"
rm -f "$zip" "$latest"
ditto -c -k --sequesterRsrc --keepParent "$app" "$zip"
cp "$zip" "$latest"
echo "$zip"
echo "$latest"
