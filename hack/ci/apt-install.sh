#!/usr/bin/env bash
# apt-install.sh <package...>: installs Debian packages on a CI runner without letting a hanging
# mirror take the job with it. On 2026-10-15 `apt-get update` hung 13 minutes on the shared
# self-hosted node until the job's timeout cancelled the job, and with it the run's release. Each
# try here has a deadline, the network timeouts are short, and there are three tries.
set -euo pipefail

apt() {
	sudo timeout 240 apt-get -qq -o Acquire::Retries=3 -o Acquire::http::Timeout=20 \
		-o Acquire::https::Timeout=20 -o DPkg::Lock::Timeout=60 "$@"
}

# try <what> <command...>: the command, up to three times, 10 and 20 seconds apart.
try() {
	local what=$1 n
	shift
	for n in 1 2 3; do
		if "$@"; then return 0; fi
		[ "$n" -lt 3 ] || break
		echo "apt-get ${what} failed or hung (try ${n} of 3); again in $((n * 10)) s"
		sleep $((n * 10))
	done
	echo "::error::apt-get ${what} failed three times"
	return 1
}

[ "$#" -gt 0 ] || { echo "usage: $0 <package...>" >&2; exit 2; }
try update apt update
try install apt install -y "$@" >/dev/null
