# Updates

The update channel: how Cogpit learns about a new version and replaces itself, one of the two
places where Cogpit downloads code that then runs. No update code exists yet: releases exist since
2026-10-13 ([below](#what-a-release-holds-today)), but Cogpit neither checks for nor installs one;
the design is [D-4](../developer/decisions.md#d-4-packaging-and-updates) and
[releasing › updates](../developer/releasing.md#updates). The other place, built 2026-10-13, is the
`kubectl` a terminal offers for its cluster: downloaded only on the user's click and verified against
the Kubernetes release's own sigstore signature, not Cogpit's update key
([trust boundaries](trust-boundaries.md#boundaries-and-required-checks),
[H-35](trust-boundaries.md#h-35-kubectls-signature-is-checked-without-sigstores-transparency-log)).

## Chain of trust

```text
release workflow ─► artifacts + SHA256SUMS + update manifest (version, artifact per format, SHA-256)
                                   │
owner, offline or by approval ─────► manifest signature with the update key
                                   │
Cogpit ─► fetch manifest over HTTPS ─► verify signature with the public key built into the app
       ─► newer version than the running one? ─► download artifact ─► verify SHA-256
       ─► verify the OS code signature (macOS: Developer ID and notarization)
       ─► replace the app and relaunch
```

On macOS the chain is Sparkle 2's: the manifest is its appcast, the signature is its EdDSA
(ed25519) signature over the disk image, and Sparkle checks that signature and Apple's code
signature before it replaces the app ([D-4](../developer/decisions.md#d-4-packaging-and-updates)).

| Check | Prevents |
|---|---|
| Manifest signature verified with a public key compiled into Cogpit | a tampered manifest, a compromised download host, a compromised CI that cannot reach the update key |
| SHA-256 of the artifact against the signed manifest | a swapped artifact |
| Operating system code signature of the new app | an artifact that is not Cogpit's |
| Version strictly newer than the running one | downgrade to a version with known flaws |
| HTTPS | eavesdroppers learning which version a user runs (not integrity: the signature provides that) |
| Package-manager installs hand over to the manager | Cogpit overwriting files a package manager owns |

## What a release holds today

Since 2026-10-13 ([D-16](../developer/decisions.md#d-16-the-first-real-release)) semantic-release
creates a tag and a GitHub release once every gate on `main` is green, and the release pipeline
([`build.yml`](../../.github/workflows/build.yml)) builds the artifacts from that tag and attaches
them ([releasing › the release pipeline](../developer/releasing.md#the-release-pipeline)):

| Artifact | Built by | What vouches for it today |
|---|---|---|
| `cogpit-cli_<v>_<os>_<arch>.tar.gz` (`.zip` for Windows), six platforms | `make dist-cli` on a self-hosted Linux runner | its SHA-256 in `SHA256SUMS`; no code signature |
| `Cogpit_<v>_darwin_arm64.zip`, the app bundle | `make dist-app` on GitHub's `macos-latest` | its SHA-256 in `SHA256SUMS`; an ad-hoc signature (`codesign --sign -`), which seals the bundle but names nobody: **no Developer ID, not notarized**, so Gatekeeper blocks the first start until the user allows it |
| `SHA256SUMS` | the release pipeline's last job | nothing: it is not signed, so it shows that a download matches what CI attached to the release, not who attached it |

The release page says the app is unsigned. There is no update manifest and no update key yet, and
nothing in Cogpit fetches a release: the [chain of trust](#chain-of-trust) above is the design for
when there is.

### Keys and who holds them

| Key | Where it lives | What it can do | If it leaks |
|---|---|---|---|
| The organisation's GitHub App private key (`APP_CLIENT_ID`, `APP_PRIVATE_KEY`) | organisation secrets of `guided-traffic`, visible to this repository; used by the Semantic Release job (token with `contents: write`) and by Renovate | push to `main`, create tags and releases; a release it creates starts the release pipeline, which builds whatever the tag points at | rotate the App's private key in the App's settings and replace the organisation secret; delete the releases and tags it made. Secrets are not handed to workflows of pull requests from forks |
| The release pipeline's `GITHUB_TOKEN` | per job; `contents: write` only in its last job, to attach the artifacts | upload assets to and edit the text of this repository's releases | expires with the job |
| Developer ID Application certificate, App Store Connect API key (planned) | secrets of a GitHub environment `release` that only tag workflows may use (deployment rule: tags `v*`), so that no pull request workflow can read them | sign and notarize Cogpit | revoke the certificate at Apple, and the key in App Store Connect; releases notarized before stay valid (the owner's note, 2026-10-13; not checked against Apple's documentation) |
| The update key (Sparkle's EdDSA key, planned) | not decided | sign the update manifest | [H-15](#h-15-update-key-compromise-and-rotation) |

## What this does not cover

### H-61: Releases are unsigned

Live since the first release (2026-10-13). Neither `Cogpit.app` nor `cogpit-cli` carries a code
signature that names Cogpit, the app is not notarized, and `SHA256SUMS` is not signed: whoever can
change a release's assets (the GitHub App's key, a maintainer's account) can replace an artifact
and its checksum together, and a user has nothing to check them against. The Developer ID
signature and the notarization come once the owner has the Apple credentials
([D-4](../developer/decisions.md#d-4-packaging-and-updates)); signing the checksums or the
artifacts for Linux and Windows is not designed.

### H-62: Every push to `main` that passes the gates is released

Live since 2026-10-13. `main` has no branch protection and no required review (checked
2026-10-13), agents and maintainers push to it directly, and semantic-release releases every
`fix:` and `feat:` commit once CI is green, so the checks are the only gate between a commit and a
published release. Renovate's automerged dependency updates are `build(deps):` commits and release
nothing of their own (owner, 2026-10-13); they ship with the next `feat:` or `fix:`, unreviewed like
every automerged update.

### H-15: Update key compromise and rotation

Whoever holds the update key can ship code to every Cogpit installation. How the key is stored,
who can use it, and how a compromised key is replaced in installed apps (a second, offline
rotation key; a forced reinstall) is not designed yet.

### H-16: Availability of the release host

Updates depend on the GitHub release host. If it is unreachable, Cogpit keeps running and offers no
update; there is no mirror.

### H-17: Installs owned by a package manager

For Homebrew (and, later, winget, Flathub and Linux repositories) the integrity of an update is
the package manager's, not Cogpit's.

### H-18: Dependencies

The Go modules Cogpit is built from are a supply chain of their own. The team's other repositories
use Renovate and govulncheck; Cogpit adopts both with M0, but no page describes the dependency
policy yet.
