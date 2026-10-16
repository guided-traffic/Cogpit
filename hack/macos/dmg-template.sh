#!/usr/bin/env bash
# The template of the macOS disk image's window (D-70; docs/developer/releasing.md#the-disk-images-window):
# a writable disk image with the window the owner chose in the DMG lab (packaging/dmg/: the
# background, the icon size and places, the window's size), the app's icon as the volume's icon and
# the link to /Applications, but no app. The release pipeline makes it in the job that builds the
# app ("Cogpit.app (macOS arm64)", no secrets), because dmgbuild is code from PyPI: it never runs in
# the job with the signing identity. That job (hack/macos/release.sh build_dmg) checks the
# template's contents, copies the signed app in and converts it with Apple's tools only.
#
#   hack/macos/dmg-template.sh <version> <Cogpit.app> <out dir>     (make dist-dmg-template)
#
# Writes <out dir>/Cogpit-<version>-template.dmg. dmgbuild and its dependencies are pinned with
# their hashes in hack/macos/dmgbuild-requirements.txt and installed into a throwaway virtual
# environment; dmgbuild writes Finder's view settings (.DS_Store) itself, without Finder or
# AppleScript, so it runs on a CI runner. It needs Python 3.10 or newer (PYTHON chooses another).
set -euo pipefail

die() {
	if [ -n "${GITHUB_ACTIONS:-}" ]; then printf '::error::%s\n' "$*" >&2; else printf 'error: %s\n' "$*" >&2; fi
	exit 1
}

[ $# -eq 3 ] || die "usage: $0 <version> <Cogpit.app> <out dir>"
version=$1 app=$2 out=$3
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
[ -d "$app" ] || die "no app at $app (make dist-app first)"
py="${PYTHON:-python3}"
"$py" -c 'import sys; sys.exit(sys.version_info < (3, 10))' ||
	die "dmgbuild needs Python 3.10 or newer, $py is $("$py" --version 2>&1) (set PYTHON)"

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
"$py" -m venv "$work/venv"
"$work/venv/bin/pip" install --quiet --disable-pip-version-check --require-hashes --only-binary :all: \
	--no-deps -r "$root/hack/macos/dmgbuild-requirements.txt"
tiffutil -cathidpicheck "$root/packaging/dmg/background.png" "$root/packaging/dmg/background@2x.png" \
	-out "$work/background.tiff" >/dev/null
icon="$(plutil -extract CFBundleIconFile raw "$app/Contents/Info.plist")"
icon="$app/Contents/Resources/${icon%.icns}.icns"
[ -f "$icon" ] || die "the app has no icon file $icon"

mkdir -p "$out"
template="$out/Cogpit-$version-template.dmg"
"$work/venv/bin/dmgbuild" -s "$root/hack/macos/dmg_settings.py" -D lab="$root/packaging/dmg/lab.json" \
	-D background="$work/background.tiff" -D volicon="$icon" "Cogpit $version" "$template" >/dev/null
printf 'disk image template: %s (%s bytes)\n' "$template" "$(stat -f %z "$template")"
