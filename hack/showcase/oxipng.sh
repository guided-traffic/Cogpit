#!/usr/bin/env bash
# oxipng, the lossless PNG optimiser for the README's showcase pictures (A112;
# docs/developer/testing.md): the pinned release binary and its run. `make oxipng` and
# `make showcase` run it.
#
#   hack/showcase/oxipng.sh fetch <version> <target> <sha256> <dir>
#   hack/showcase/oxipng.sh optimise <oxipng> <flags> <png>...
#
# fetch downloads oxipng-<version>-<target>.tar.gz from oxipng's GitHub release, refuses it unless
# its SHA-256 is the pinned one (the Makefile's OXIPNG_SHA256_<platform>), and keeps the binary and
# its licence in <dir> (.cache/oxipng/<version>). Nothing is installed anywhere else.
#
# optimise rewrites the PNGs in place with <flags> (the Makefile's OXIPNG_FLAGS) and prints their
# total size before and after. oxipng is lossless: it recompresses, chooses filters and drops
# metadata chunks (--strip safe keeps the ones that change how a picture looks) and reduces the
# colour type only where no pixel changes; the flags never include --alpha, its one lossy option.
set -euo pipefail

die() {
	if [ -n "${GITHUB_ACTIONS:-}" ]; then printf '::error::%s\n' "$*" >&2; else printf 'error: %s\n' "$*" >&2; fi
	exit 1
}

fetch() {
	[ "$#" -eq 4 ] || die "usage: $0 fetch <version> <target> <sha256> <dir>"
	local version=$1 target=$2 sha=$3 dir=$4 name url got
	case "$version" in "" | *[!0-9.]*) die "oxipng version $version: digits and dots" ;; esac
	case "$target" in "" | *[!a-z0-9_-]*) die "oxipng target $target: a Rust target triple" ;; esac
	[ -n "$sha" ] || die "no pinned SHA-256 for this platform: oxipng is pinned for aarch64-apple-darwin and x86_64-unknown-linux-gnu (the Makefile's OXIPNG_SHA256_*)"
	case "$sha" in *[!0-9a-f]*) die "the pinned SHA-256 is not lower-case hex" ;; esac
	[ "${#sha}" = 64 ] || die "the pinned SHA-256 has ${#sha} characters, not 64"
	name="oxipng-$version-$target"
	url="https://github.com/shssoichiro/oxipng/releases/download/v$version/$name.tar.gz"
	mkdir -p "$(dirname "$dir")"
	# Global, for the trap that removes it when the script ends.
	tmp="$(mktemp -d "$(dirname "$dir")/.oxipng.XXXXXX")"
	trap 'rm -rf "$tmp"' EXIT
	curl -fsSL --proto '=https' --tlsv1.2 --retry 3 -o "$tmp/oxipng.tar.gz" "$url"
	got="$(shasum -a 256 "$tmp/oxipng.tar.gz" | cut -d' ' -f1)"
	[ "$got" = "$sha" ] || die "$name.tar.gz has SHA-256 $got, the pin is $sha"
	mkdir "$tmp/x"
	tar -xzf "$tmp/oxipng.tar.gz" -C "$tmp/x" "$name/oxipng" "$name/LICENSE"
	[ -f "$tmp/x/$name/oxipng" ] || die "the archive holds no oxipng"
	chmod 0755 "$tmp/x/$name/oxipng"
	printf '%s  %s.tar.gz\n' "$sha" "$name" >"$tmp/x/$name/.verified"
	rm -rf "$dir"
	mv "$tmp/x/$name" "$dir"
	echo "oxipng $version in $dir (SHA-256 $sha)"
}

# size prints the files' total size in bytes.
size() {
	local total=0 n f
	for f in "$@"; do
		n="$(wc -c <"$f" | tr -d ' ')"
		total=$((total + n))
	done
	echo "$total"
}

optimise() {
	[ "$#" -ge 3 ] || die "usage: $0 optimise <oxipng> <flags> <png>..."
	local bin=$1 flags=$2 before after
	shift 2
	[ -x "$bin" ] || die "no oxipng at $bin: make oxipng"
	case " $flags " in *" -a "* | *" --alpha "*) die "--alpha alters pixels: the showcase stays lossless" ;; esac
	before="$(size "$@")"
	# One run over every file: oxipng works on several files at once.
	# shellcheck disable=SC2086 # the flags are words
	"$bin" -q $flags -- "$@"
	after="$(size "$@")"
	echo "oxipng: $# pictures, $before bytes before, $after after"
}

case "${1:-}" in
fetch | optimise)
	cmd=$1
	shift
	"$cmd" "$@"
	;;
*) die "usage: $0 fetch <version> <target> <sha256> <dir> | optimise <oxipng> <flags> <png>..." ;;
esac
