#!/usr/bin/env bash
# The macOS downloads of a release (docs/developer/releasing.md#the-release-pipeline), made from
# what `make dist-app` and `make dist-cli` built: the app in a disk image and cogpit-cli for macOS,
# signed with the Developer ID and notarized by Apple when the signing secrets are there. The
# release pipeline's job "Sign and notarize (macOS)" runs it through `make dist-macos`; on a Mac it
# runs the same, signed ad-hoc.
#
#   hack/macos/release.sh <version> <in dir> <out dir>
#   hack/macos/release.sh cleanup
#
# <in dir> holds Cogpit_<version>_darwin_<arch>.zip and cogpit-cli_<version>_darwin_<arch>.tar.gz;
# <out dir> receives the downloads. What happens depends on the environment:
#
#   Developer ID  MACOS_SIGN_P12 (the .p12, base64), MACOS_SIGN_P12_PASSWORD, APPLE_API_KEY_P8 (the
#                 App Store Connect key, base64), APPLE_API_KEY_ID and APPLE_API_ISSUER_ID all set,
#                 with APPLE_TEAM_ID and Apple's intermediate pinned in MACOS_SIGN_CA_URL and
#                 MACOS_SIGN_CA_SHA256: a temporary keychain; the app signed with the hardened
#                 runtime and a secure timestamp; Cogpit-<version>-macos-<arch>.dmg built and signed;
#                 cogpit-cli signed and its archives rebuilt; the disk image and cogpit-cli
#                 notarized; the disk image's ticket stapled (a bare binary cannot carry one); every
#                 signature checked, and the disk image and the app in it as Gatekeeper checks them.
#   ad-hoc        MACOS_SIGN=adhoc and none of the secrets (a Mac without the material): the same
#                 steps signed ad-hoc, without notarization and Gatekeeper's check.
#   unsigned      none of the secrets: the app zip and the archives pass through unchanged.
#
# Some of the secrets but not all is an error, never a quiet fallback to an unsigned download; so is
# every failed signature, notarization or check. `cleanup` deletes the temporary keychain and the
# key files; the pipeline runs it with if: always() after a cancelled run. Secrets never reach the
# log: no set -x, nothing derived from them is echoed, the keychain's password is masked.
#
# Entitlements: none, for the app and Sparkle's parts. Go generates no code at run time (no JIT, no
# unsigned executable memory), the app loads only Apple's frameworks and Sparkle.framework, which
# carries the app's own signature (library validation holds), and uses no camera, microphone,
# location, contacts or Apple Events. `make dist-macos MACOS_SIGN=adhoc` and starting the app
# signed this way is the check (docs/developer/releasing.md#signing-and-notarization).
set -euo pipefail

work="${RUNNER_TEMP:-${TMPDIR:-/tmp}}"
work="${work%/}/cogpit-macos-release"
keychain="$work/signing.keychain-db"
secrets="MACOS_SIGN_P12 MACOS_SIGN_P12_PASSWORD APPLE_API_KEY_P8 APPLE_API_KEY_ID APPLE_API_ISSUER_ID"
cli_identifier="io.guided-traffic.cogpit-cli"
# How long to wait for Apple's notary service, for the disk image and cogpit-cli together.
notary_minutes="${NOTARY_TIMEOUT_MINUTES:-45}"

log() { printf '%s\n' "$*"; }
err() {
	if [ -n "${GITHUB_ACTIONS:-}" ]; then printf '::error::%s\n' "$*" >&2; else printf 'error: %s\n' "$*" >&2; fi
}
die() { err "$@"; exit 1; }
warn() {
	if [ -n "${GITHUB_ACTIONS:-}" ]; then printf '::warning::%s\n' "$*" >&2; else printf 'warning: %s\n' "$*" >&2; fi
}
group() { if [ -n "${GITHUB_ACTIONS:-}" ]; then printf '::group::%s\n' "$*"; else printf '== %s\n' "$*"; fi; }
endgroup() { if [ -n "${GITHUB_ACTIONS:-}" ]; then printf '::endgroup::\n'; fi; }
mask() { if [ -n "${GITHUB_ACTIONS:-}" ]; then printf '::add-mask::%s\n' "$1"; fi; }
output() {
	if [ -n "${GITHUB_OUTPUT:-}" ]; then printf '%s=%s\n' "$1" "$2" >>"$GITHUB_OUTPUT"; fi
	log "$1=$2"
}
summary() { if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then printf '%s\n' "$*" >>"$GITHUB_STEP_SUMMARY"; fi; }

# retry <attempts> <command...>: Apple's timestamp and ticket services, and hdiutil on CI runners,
# fail now and then and succeed seconds later.
retry() {
	local attempt=1 attempts=$1
	shift
	until "$@"; do
		[ "$attempt" -lt "$attempts" ] || return 1
		log "$1 failed; again in $((attempt * 10)) s"
		sleep $((attempt * 10))
		attempt=$((attempt + 1))
	done
}

# signing_mode prints developer-id, adhoc or unsigned.
signing_mode() {
	local have=0 missing="" name
	for name in $secrets; do
		if [ -n "${!name:-}" ]; then have=$((have + 1)); else missing="$missing $name"; fi
	done
	if [ "$have" -eq 0 ]; then
		if [ "${MACOS_SIGN:-}" = adhoc ]; then echo adhoc; else echo unsigned; fi
		return
	fi
	[ -z "$missing" ] || die "the Developer ID signing needs every secret; missing:$missing"
	for name in APPLE_TEAM_ID MACOS_SIGN_CA_URL MACOS_SIGN_CA_SHA256; do
		[ -n "${!name:-}" ] || die "the Developer ID signing needs $name"
	done
	echo developer-id
}

# cleanup detaches the disk image, deletes the keychain, restores the keychain search list and
# removes the work directory with the key files; it runs in a subshell so that its set +e stays
# there.
cleanup() (
	set +e
	if [ -d "$work/mnt" ] && mount | grep -q " on $work/mnt "; then
		hdiutil detach -quiet -force "$work/mnt"
	fi
	if [ -f "$keychain" ]; then security delete-keychain "$keychain"; fi
	if [ -f "$work/keychains" ]; then
		saved=()
		while IFS= read -r line; do saved+=("$line"); done <"$work/keychains"
		security list-keychains -d user -s ${saved[@]+"${saved[@]}"}
	fi
	rm -rf "$work"
)

# setup_keychain creates the temporary keychain with the Developer ID identity and Apple's
# intermediate, and sets identity to the identity's SHA-1.
setup_keychain() {
	group "Temporary keychain with the Developer ID identity"
	local password p12="$work/signing.p12" ca="$work/DeveloperIDCA.cer" line identities count
	password="$(openssl rand -hex 32)"
	mask "$password"

	# Apple's intermediate, pinned by its SHA-256: without it beside the .p12 the identity is not
	# valid ("0 valid identities found").
	curl -fsSL --proto '=https' --tlsv1.2 --retry 3 -o "$ca" "$MACOS_SIGN_CA_URL"
	[ "$(shasum -a 256 "$ca" | cut -d' ' -f1)" = "$MACOS_SIGN_CA_SHA256" ] ||
		die "Apple's intermediate from $MACOS_SIGN_CA_URL does not match its pinned SHA-256"

	(umask 077 && printf '%s' "$MACOS_SIGN_P12" | base64 --decode >"$p12")
	# The user's keychain search list as it was, for cleanup to restore.
	security list-keychains -d user | sed -e 's/^[[:space:]]*"//' -e 's/"[[:space:]]*$//' |
		grep -vxF "$keychain" >"$work/keychains" || true
	security create-keychain -p "$password" "$keychain"
	security set-keychain-settings -lut 21600 "$keychain"
	security unlock-keychain -p "$password" "$keychain"
	# The keychain first in the search list, so codesign and the trust evaluation find the identity
	# and the intermediate.
	local search=("$keychain")
	while IFS= read -r line; do search+=("$line"); done <"$work/keychains"
	security list-keychains -d user -s "${search[@]}"

	security import "$p12" -k "$keychain" -f pkcs12 -x -P "$MACOS_SIGN_P12_PASSWORD" -T /usr/bin/codesign >/dev/null
	rm -f "$p12"
	security import "$ca" -k "$keychain" >/dev/null
	security set-key-partition-list -S apple-tool:,apple:,codesign: -s -k "$password" "$keychain" >/dev/null

	identities="$(security find-identity -v -p codesigning "$keychain")"
	log "$identities"
	identity="$(printf '%s\n' "$identities" |
		awk -v team="($APPLE_TEAM_ID)\"" '/"Developer ID Application: / && index($0, team) { print $2 }')"
	count="$(printf '%s' "$identity" | grep -c . || true)"
	[ "$count" = 1 ] || die "expected one valid Developer ID Application identity of team $APPLE_TEAM_ID, found $count (none: Apple's intermediate missing, or the certificate expired or revoked)"
	endgroup
}

# sign <path> [codesign options...]
sign() {
	local path=$1
	shift
	if [ "$mode" = developer-id ]; then
		retry 3 codesign --force --sign "$identity" --keychain "$keychain" --timestamp "$@" "$path"
	else
		codesign --force --sign - "$@" "$path"
	fi
}

# check_signature <path> <code|image>: the seal, the hardened runtime and no entitlements for code;
# for the Developer ID also the certificate's team and Apple's timestamp.
check_signature() {
	local path=$1 kind=$2 info
	codesign --verify --strict --deep --verbose=2 "$path"
	info="$(codesign --display --verbose=4 "$path" 2>&1)"
	printf '%s\n' "$info" | grep -E '^(Identifier=|Format=|CodeDirectory |Authority=|TeamIdentifier=|Timestamp=|Runtime Version=)' || true
	if [ "$kind" = code ]; then
		printf '%s\n' "$info" | grep -Eq '^CodeDirectory .*flags=0x[0-9a-f]+\([^)]*runtime' ||
			die "$path: signed without the hardened runtime"
		[ -z "$(codesign --display --entitlements - --xml "$path" 2>/dev/null)" ] ||
			die "$path: signed with entitlements; Cogpit needs none (hack/macos/release.sh, header)"
	fi
	if [ "$mode" = developer-id ]; then
		printf '%s\n' "$info" | grep -q "^Authority=Developer ID Application: .*($APPLE_TEAM_ID)\$" ||
			die "$path: not signed with team $APPLE_TEAM_ID's Developer ID Application certificate"
		printf '%s\n' "$info" | grep -qx "TeamIdentifier=$APPLE_TEAM_ID" || die "$path: team identifier is not $APPLE_TEAM_ID"
		printf '%s\n' "$info" | grep -q '^Timestamp=' || die "$path: no secure timestamp"
	fi
}

# sign_app signs the bundle inside-out, never with --deep. Its only nested code is Sparkle
# (D-4; hack/macos/sparkle.sh put it there without its XPC services, which only sandboxed apps
# use): Autoupdate, Updater.app, then the framework, the order of Sparkle's documentation for
# signing outside Xcode (sparkle-project.org/documentation/sandboxing#code-signing), then the app;
# each with the hardened runtime, Apple's timestamp and no entitlements (Sparkle's own ad-hoc
# signature of Autoupdate carries com.apple.application-identifier; its documented command does not
# keep it). Every Mach-O file in the bundle must then carry the bundle's signature authority, so a
# helper a newer Sparkle adds cannot reach Apple ad-hoc signed. Anything else nested is refused.
sign_app() {
	group "Sign $(basename "$app")"
	local dir part fw="$app/Contents/Frameworks/Sparkle.framework"
	for dir in PlugIns XPCServices Helpers Library; do
		[ ! -e "$app/Contents/$dir" ] ||
			die "$app/Contents/$dir holds nested code: sign it inside-out before the bundle (no --deep)"
	done
	if [ -e "$app/Contents/Frameworks" ]; then
		[ "$(ls -A "$app/Contents/Frameworks")" = Sparkle.framework ] ||
			die "$app/Contents/Frameworks holds more than Sparkle.framework: sign it inside-out before the bundle (no --deep)"
		[ ! -e "$fw/XPCServices" ] && [ ! -e "$fw/Versions/B/XPCServices" ] ||
			die "Sparkle's XPC services are in the bundle; make app leaves them out (hack/macos/sparkle.sh)"
		for part in "$fw/Versions/B/Autoupdate" "$fw/Versions/B/Updater.app" "$fw"; do
			[ -e "$part" ] || die "$part is missing: not the Sparkle layout this script signs"
			sign "$part" --options runtime
			check_signature "$part" code
		done
	fi
	[ "$(ls -A "$app/Contents/MacOS")" = "$exe" ] || die "$app/Contents/MacOS holds more than $exe"
	sign "$app" --options runtime
	check_signature "$app" code
	check_nested "$app"
	endgroup
}

# check_nested: every Mach-O file in the bundle is signed by the authority the bundle is signed
# by (ad-hoc locally), with the hardened runtime.
check_nested() {
	local bundle=$1 want file info n=0
	want="$(codesign --display --verbose=2 "$bundle" 2>&1 | sed -n 's/^Authority=//p' | head -n 1)"
	while IFS= read -r file; do
		file -b "$file" | grep -q '^Mach-O' || continue
		n=$((n + 1))
		info="$(codesign --display --verbose=4 "$file" 2>&1)" || die "$file is not signed"
		printf '%s\n' "$info" | grep -Eq '^CodeDirectory .*flags=0x[0-9a-f]+\([^)]*runtime' ||
			die "$file: signed without the hardened runtime"
		if [ -n "$want" ]; then
			printf '%s\n' "$info" | grep -qxF "Authority=$want" || die "$file is not signed by $want"
		else
			printf '%s\n' "$info" | grep -qx 'Signature=adhoc' || die "$file is not signed ad-hoc like the bundle"
		fi
	done < <(find "$bundle/Contents" -type f -perm -u+x)
	log "$n Mach-O files in $(basename "$bundle"), each signed ${want:-ad-hoc} with the hardened runtime"
}

# sign_cli signs cogpit-cli in each macOS archive and writes the archive again into <out dir>.
sign_cli() {
	local archive name binary
	mkdir -p "$work/cli"
	for archive in "${cli_archives[@]}"; do
		name="$(basename "$archive" .tar.gz)"
		group "Sign $name"
		tar -xzf "$archive" -C "$work/cli"
		binary="$work/cli/$name/cogpit-cli"
		[ -f "$binary" ] || die "$archive holds no $name/cogpit-cli"
		sign "$binary" --options runtime --identifier "$cli_identifier"
		check_signature "$binary" code
		COPYFILE_DISABLE=1 tar --no-mac-metadata -C "$work/cli" -czf "$out/$name.tar.gz" "$name"
		[ "$(tar -tzf "$archive" | sort)" = "$(tar -tzf "$out/$name.tar.gz" | sort)" ] ||
			die "$out/$name.tar.gz does not hold the files $archive holds"
		endgroup
	done
}

# build_dmg puts the app and a link to /Applications into the disk image and signs it.
build_dmg() {
	group "Disk image $(basename "$dmg")"
	local stage="$work/dmg"
	mkdir "$stage"
	ditto "$app" "$stage/Cogpit.app"
	ln -s /Applications "$stage/Applications"
	retry 3 hdiutil create -quiet -ov -volname "Cogpit $version" -srcfolder "$stage" -fs HFS+ \
		-format UDZO -imagekey zlib-level=9 "$dmg"
	sign "$dmg"
	check_signature "$dmg" image
	endgroup
}

notary() {
	xcrun notarytool "$@" --key "$work/notary.p8" --key-id "$APPLE_API_KEY_ID" --issuer "$APPLE_API_ISSUER_ID"
}

# await <id> <what>: waits for Apple's verdict on a submission until the deadline and prints its
# log (the files Apple checked and any issue); fails unless Apple accepted it. Called with ||, so
# errexit is off in here: every step checks its own result.
await() {
	local id=$1 what=$2 left status
	group "Notarization of $what (submission $id)"
	left=$((deadline - $(date +%s)))
	[ "$left" -gt 60 ] || left=60
	notary wait "$id" --timeout "$left" || log "notarytool wait ended with an error; asking for the submission's status"
	if ! notary info "$id" --output-format json >"$work/info.json" ||
		! status="$(plutil -extract status raw -o - "$work/info.json")"; then
		endgroup
		err "$what: cannot read the status of submission $id from Apple's notary service"
		return 1
	fi
	summary "| $what | \`$id\` | $status |"
	if [ "$status" = "In Progress" ]; then
		endgroup
		err "$what: Apple has not finished notarizing submission $id within $notary_minutes minutes and keeps working on it. Check it with xcrun notarytool info $id (or history), then re-run the failed jobs (docs/developer/releasing.md#when-signing-or-notarization-fails)"
		return 1
	fi
	notary log "$id" || log "no log for submission $id"
	endgroup
	if [ "$status" != Accepted ]; then
		err "$what: Apple's notary service says $status for submission $id; its log is above"
		return 1
	fi
	log "$what: Accepted (submission $id)"
}

# notarize submits the disk image and cogpit-cli (zipped) together, waits for both, staples the
# disk image's ticket and checks it as Gatekeeper does.
notarize() {
	local cli_zip="$work/cogpit-cli-macos.zip" dmg_id cli_id failed=0
	# The key as base64, or as the .p8 file's PEM text itself.
	case "$APPLE_API_KEY_P8" in
	-----BEGIN*) (umask 077 && printf '%s\n' "$APPLE_API_KEY_P8" >"$work/notary.p8") ;;
	*) (umask 077 && printf '%s' "$APPLE_API_KEY_P8" | base64 --decode >"$work/notary.p8") ;;
	esac
	ditto -c -k --keepParent "$work/cli" "$cli_zip"

	group "Submit to Apple's notary service"
	notary submit "$dmg" --no-wait --output-format json >"$work/submit-dmg.json"
	dmg_id="$(plutil -extract id raw -o - "$work/submit-dmg.json")"
	log "$(basename "$dmg"): submission $dmg_id"
	notary submit "$cli_zip" --no-wait --output-format json >"$work/submit-cli.json"
	cli_id="$(plutil -extract id raw -o - "$work/submit-cli.json")"
	log "cogpit-cli: submission $cli_id"
	endgroup

	summary "| Notarized | Submission | Status |"
	summary "|---|---|---|"
	deadline=$(($(date +%s) + notary_minutes * 60))
	await "$dmg_id" "$(basename "$dmg")" || failed=1
	await "$cli_id" "cogpit-cli for macOS" || failed=1
	rm -f "$work/notary.p8"
	[ "$failed" = 0 ] || exit 1

	group "Staple and check $(basename "$dmg")"
	retry 3 xcrun stapler staple "$dmg"
	xcrun stapler validate "$dmg"
	gatekeeper "$dmg" -t open --context context:primary-signature
	endgroup
}

# gatekeeper <path> <spctl options...>: Gatekeeper's verdict, which must be a notarized Developer ID.
# A machine with Gatekeeper turned off accepts everything without a verdict; then the notary
# service's Accepted and the stapled ticket stand alone, and the log says so.
gatekeeper() {
	local path=$1 verdict
	shift
	verdict="$(spctl --assess -vv "$@" "$path" 2>&1)" || die "Gatekeeper rejects $path: $verdict"
	log "$verdict"
	if printf '%s\n' "$verdict" | grep -q '^override=security disabled'; then
		warn "Gatekeeper is turned off on this machine ($(spctl --status 2>&1)): no verdict on $(basename "$path")"
		return 0
	fi
	printf '%s\n' "$verdict" | grep -q '^source=Notarized Developer ID$' ||
		die "Gatekeeper accepts $path, but not as notarized: $verdict"
}

# check_dmg mounts the disk image and checks what opening it shows: the app and the link.
check_dmg() {
	group "Check the contents of $(basename "$dmg")"
	local mnt="$work/mnt" entries
	mkdir -p "$mnt"
	hdiutil attach -quiet -readonly -nobrowse -noautoopen -mountpoint "$mnt" "$dmg"
	entries="$(find "$mnt" -mindepth 1 -maxdepth 1 -exec basename {} \; | sort | tr '\n' ' ')"
	log "contents: $entries"
	[ "$entries" = "Applications Cogpit.app " ] || die "$(basename "$dmg") holds $entries, not Applications and Cogpit.app"
	[ "$(readlink "$mnt/Applications")" = /Applications ] || die "Applications in $(basename "$dmg") does not link to /Applications"
	codesign --verify --strict --deep --verbose=2 "$mnt/Cogpit.app"
	[ "$(plutil -extract CFBundleShortVersionString raw "$mnt/Cogpit.app/Contents/Info.plist")" = "$version" ] ||
		die "the app in $(basename "$dmg") is not version $version"
	if [ "$mode" = developer-id ]; then gatekeeper "$mnt/Cogpit.app" -t execute; fi
	hdiutil detach -quiet "$mnt"
	endgroup
}

main() {
	[ "$#" -eq 3 ] || die "usage: $0 <version> <in dir> <out dir> | $0 cleanup"
	version=$1
	local in=$2
	out=$3

	mode="$(signing_mode)"
	log "signing: $mode"

	local zips=() archive
	for archive in "$in"/Cogpit_"$version"_darwin_*.zip; do [ ! -f "$archive" ] || zips+=("$archive"); done
	[ "${#zips[@]}" -eq 1 ] || die "expected one Cogpit_${version}_darwin_<arch>.zip in $in (make dist-app), found ${#zips[@]}"
	cli_archives=()
	for archive in "$in"/cogpit-cli_"$version"_darwin_*.tar.gz; do [ ! -f "$archive" ] || cli_archives+=("$archive"); done
	[ "${#cli_archives[@]}" -ge 1 ] || die "no cogpit-cli_${version}_darwin_<arch>.tar.gz in $in (make dist-cli)"
	mkdir -p "$out"

	if [ "$mode" = unsigned ]; then
		cp "${zips[0]}" "${cli_archives[@]}" "$out/"
		output signed false
		summary "macOS downloads **unsigned**: the signing secrets are not set (a fork, or a run outside the environment \`release\`)."
		return
	fi

	case "$work" in */cogpit-macos-release) ;; *) die "unexpected work directory $work" ;; esac
	cleanup
	trap cleanup EXIT
	trap 'exit 130' INT
	trap 'exit 143' TERM
	mkdir -m 700 "$work"

	if [ "$mode" = developer-id ]; then setup_keychain; fi

	ditto -x -k "${zips[0]}" "$work/app"
	app="$work/app/Cogpit.app"
	[ -d "$app" ] || die "${zips[0]} holds no Cogpit.app"
	[ "$(plutil -extract CFBundleShortVersionString raw "$app/Contents/Info.plist")" = "$version" ] ||
		die "${zips[0]}: the app is not version $version"
	exe="$(plutil -extract CFBundleExecutable raw "$app/Contents/Info.plist")"
	local arch
	arch="$(lipo -archs "$app/Contents/MacOS/$exe")"
	case "$arch" in *" "*) arch=universal ;; esac
	dmg="$out/Cogpit-$version-macos-$arch.dmg"

	sign_app
	sign_cli
	build_dmg
	if [ "$mode" = developer-id ]; then notarize; fi
	check_dmg

	if [ "$mode" = developer-id ]; then
		local authority
		authority="$(codesign --display --verbose=2 "$app" 2>&1 | sed -n 's/^Authority=\(Developer ID Application: .*\)$/\1/p')"
		output signed true
		output identity "$authority"
		summary ""
		summary "\`$(basename "$dmg")\` and cogpit-cli for macOS signed by **$authority**, notarized; the disk image's ticket stapled."
	else
		output signed false
		log "ad-hoc signed, not notarized: a local check of the steps, not a download to publish"
	fi
	ls -l "$out"
}

# Sourced (to try a function with stubs in PATH): the functions only.
if [ "${BASH_SOURCE[0]}" != "$0" ]; then return 0; fi
if [ "${1:-}" = cleanup ]; then
	cleanup
	exit 0
fi
main "$@"
