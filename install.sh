#!/bin/sh
set -eu

repo=${CAUTION_TERM_REPO:-nullentropy/caution-term}
url=${CAUTION_TERM_URL:-}
dest=${CAUTION_TERM_DEST:-/Applications}
asset=CautionTerm-macos-universal.zip

fail() {
	echo "install.sh: $*" >&2
	exit 1
}

[ "$(uname -s)" = Darwin ] || fail "Caution Term is a macOS app"

if [ -z "$url" ]; then
	case $repo in
	*/*) ;;
	*) fail "CAUTION_TERM_REPO must be in owner/name form, such as nullentropy/caution-term, got: $repo" ;;
	esac
	url="https://github.com/$repo/releases/latest/download/$asset"
fi

[ -d "$dest" ] || fail "$dest does not exist"
[ -w "$dest" ] || fail "cannot write to $dest; run as an admin user or set CAUTION_TERM_DEST"

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

echo "Downloading $url"
curl -fSL --progress-bar -o "$work/$asset" "$url" || fail "download failed: $url"
ditto -x -k "$work/$asset" "$work/unzipped" || fail "could not unzip $asset"
[ -d "$work/unzipped/Caution Term.app" ] || fail "$asset has no Caution Term.app"

app="$dest/Caution Term.app"
exe="$app/Contents/MacOS/"
if pgrep -f "$exe" >/dev/null 2>&1; then
	echo "Quitting the running Caution Term"
	osascript -e "tell application \"$app\" to quit" >/dev/null 2>&1 || true
	i=0
	while pgrep -f "$exe" >/dev/null 2>&1 && [ $i -lt 20 ]; do
		sleep 0.5
		i=$((i + 1))
	done
	if pgrep -f "$exe" >/dev/null 2>&1; then
		pkill -TERM -f "$exe" || true
		sleep 2
		pkill -KILL -f "$exe" 2>/dev/null || true
	fi
fi

rm -rf "$app"
mv "$work/unzipped/Caution Term.app" "$app"
echo "Installed Caution Term to $app"
