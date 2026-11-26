#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
if [ "$(uname -s)" != Darwin ]; then
	echo "scripts/app.sh builds the macOS app; on other systems use go run ." >&2
	exit 1
fi
bin=$(mktemp -d)
trap 'rm -rf "$bin"' EXIT
go build -o "$bin/caution-term" .
go run github.com/nullentropy/caution/cmd/appbundle -bin "$bin/caution-term" -name "Caution Term" \
	-id dev.caution.term -icon icon.png -o dist/
codesign --force -s - "dist/Caution Term.app"
echo "dist/Caution Term.app"
