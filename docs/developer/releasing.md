# Releasing

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** major (Go and Fyne packaging; semantic-release as in the team's other repositories; signed updates for the one-click update) · **Target:** Lens Desktop 2026.10.51501

No release pipeline exists yet (2026-10-11). This page is the plan. Platform targets and how they
compare with Lens Desktop: [platform parity](../parity/platform-parity.md#operating-systems-and-packages).

## Release flow (planned, as in the team's other repositories)

The team's Go repositories release with semantic-release on `main` (for example
`valkey-operator`: `.releaserc.json` with the `conventionalcommits` preset and workflows
`build.yml`, `release.yml`, `renovate.yml`). Cogpit follows the same flow:

1. Commits on `main` follow [Conventional Commits](conventions.md#commits-and-pull-requests).
2. `release.yml` runs semantic-release, which derives the next version from the commits since the
   last tag, tags it and writes the release notes.
3. The same workflow builds every artifact for the tag, code-signs what it has signing material
   for, writes `SHA256SUMS.txt`, an SPDX SBOM and the update manifest, and attaches them to the
   GitHub release.
4. The update manifest is signed in a separate step with the update key, which CI cannot reach on
   its own: offline by the owner, or in a protected environment that needs the owner's approval.
   Until it is signed, the release is published but not offered as an update.
5. A `::warning::` in a job summary means a code-signing secret was missing and that artifact is
   unsigned; such an artifact is not offered as an update.

Every binary reports its version, commit and build date (`cogpit --version`, `cogpit-cli
--version`, Help › About), set at build time with `-ldflags -X`.

## What gets built (planned)

| Platform | Artifacts | Built on | Signing |
|---|---|---|---|
| macOS 13+, universal (Apple silicon + Intel) | `Cogpit-<v>-macos-universal.dmg` with the app bundle `Cogpit.app` ([D-4](decisions.md#d-4-packaging-and-updates)); `cogpit-cli-<v>-darwin-universal.tar.gz` | macOS runner: the Fyne app built per architecture with cgo, joined with `lipo`, bundled with `fyne package`, imaged with `hdiutil` | app and disk image signed with Developer ID; disk image notarized, ticket stapled |
| Windows 10+ x64 and arm64 | `Cogpit-<v>-windows-<arch>.msi`, `Cogpit-<v>-windows-<arch>-portable.zip`; `cogpit-cli` zip | native x64 and arm64 runners (cgo) | Authenticode |
| Linux x86_64 and aarch64 | `Cogpit-<v>-linux-<arch>.{AppImage,deb,rpm,tar.xz}`, Flatpak; `cogpit-cli` tar.gz | native runners (cgo, OpenGL and X11/Wayland headers) | checksums, SBOM, signed update manifest |

**The macOS artifact is the app bundle `Cogpit.app` in a disk image**: the standard way to install
a Mac app outside the App Store, without a terminal. Opening the image shows the app next to a
link to Applications; the user drags it across. The disk image is notarized and its ticket stapled,
so it also opens offline without a Gatekeeper prompt. The Homebrew cask (`auto_updates true`) and
Sparkle's updates use the same disk image.

`cogpit-cli` is pure Go (`CGO_ENABLED=0`) and cross-compiles from one runner. The desktop app
needs cgo for Fyne's OpenGL driver, so each platform builds on its own runner. Whether `fyne
package` alone covers MSI, AppImage, deb, rpm and Flatpak, or `packaging/` needs extra tools, is
criterion S7 of the [spike](spike-fyne.md).

## Updates

Cogpit shows available updates and installs them with one click
([D-4](decisions.md#d-4-packaging-and-updates)); on macOS through Sparkle 2. Each release carries
an update manifest, on macOS Sparkle's appcast (version, disk image, its EdDSA signature), signed
with an update key that is kept outside CI's reach (step 4 above); the app verifies the manifest signature with the public key built into it, then the
checksum, then the operating system's code signature, before it replaces anything. Installs owned
by a package manager (Homebrew, winget, Flathub, deb or rpm repositories) hand over to the
manager. Security perspective: [updates](../security/updates.md).

## Secrets (planned)

| Secret | For | Without it |
|---|---|---|
| Apple Developer ID certificate and password | signing the app bundle | unsigned app, Gatekeeper warns; not offered as an update |
| App Store Connect API key | notarization | not notarized |
| Windows code-signing certificate or signing service credentials | Authenticode | SmartScreen warns; not offered as an update |
| Update signing key | the update manifest | no update offered for that release |

**Human:** none of this material exists yet; the owner provides it before beta
([G14](../parity/gaps.md#g14-packaging-signing-and-updates-macos)).
