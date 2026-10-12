# Releasing

> **Last reviewed:** 2026-10-12 · **Last updated:** 2026-10-12 · **Change:** minor (the first real release decided: dry runs until the alpha, D-16) · **Target:** Lens Desktop 2026.10.51501

CI exists as workflow files since 2026-10-11; they have not run on GitHub yet, so no job is known
to be green there. Releases do not exist: semantic-release runs as a **dry run** until the owner
cuts the first release ([below](#the-first-real-release)). Artifacts, signing and updates further
down are the plan. Platform targets and how they compare with Lens Desktop:
[platform parity](../parity/platform-parity.md#operating-systems-and-packages).

## CI

Three workflows in [`.github/workflows/`](../../.github/workflows/), laid out as in the team's
other repositories (`valkey-operator`: `release.yml` is the pipeline of every push and pull
request and ends in semantic-release; Renovate runs from `renovate.yml`):

| Workflow | Triggers | What it does |
|---|---|---|
| [`release.yml`](../../.github/workflows/release.yml) "Test and Release" | push to `main`, pull request to `main`, manual | calls `build.yml`; on `main` it also commits the coverage badge and runs the semantic-release dry run. The README's build badge shows this workflow. |
| [`build.yml`](../../.github/workflows/build.yml) "Build" | called by `release.yml` (`workflow_call`), manual | the checks below |
| [`renovate.yml`](../../.github/workflows/renovate.yml) | daily at 02:00 Berlin time, manual | dependency updates ([Renovate](#renovate)) |

`build.yml` has no push or pull request trigger of its own, so each push runs the checks once. A
pull request run that a newer push supersedes is cancelled; runs on `main` queue, so every push
to `main` is checked. Go comes from `go.mod` (`actions/setup-go` with `go-version-file`, which
caches the module and build caches). Every job's `GITHUB_TOKEN` is read-only (`contents: read`)
except the Coverage Badge job's; Renovate works with the team's GitHub App token. Every job has a
timeout. Actions are pinned to the tags the team's other repositories use
(`actions/checkout@v7`, `actions/setup-go@v7`, `actions/upload-artifact@v7`,
`actions/download-artifact@v8`, `actions/setup-node@v7`, `msys2/setup-msys2@v2`,
`helm/kind-action@v1.15.1`, `renovatebot/github-action@v46.3.8`,
`actions/create-github-app-token@v3`); Renovate moves them.

| Job | Runner | Runs | What it proves |
|---|---|---|---|
| macOS: lint, race tests, build | `macos-latest` | `make lint`, `make test-unit-coverage`, `make build`, `make coverage-badge` | the lint with the `depguard` layering; every test that needs no cluster (unit, engine, MCP, UI offscreen, integration against the fake API server) passes under the race detector; `cogpit` builds with cgo and Fyne, `cogpit-cli` without cgo. The platform Cogpit ships first. |
| Compile (linux) | `ubuntu-latest` | apt: `gcc libgl1-mesa-dev xorg-dev libxkbcommon-dev libwayland-dev wayland-protocols` (Fyne's documented packages plus the Wayland headers GLFW 3.4 compiles against); `go build ./...`, `go vet ./...`, `go test -run '^$' ./...` | every package and every test compiles and links on Linux with cgo and Fyne's native dependencies. It runs no test. |
| Compile (windows) | `windows-latest` | MinGW-w64 gcc from MSYS2 on `PATH`, `CGO_ENABLED=1`; the same three commands | the same on Windows, the terminal's Windows stub ([`localpty_windows.go`](../../internal/terminal/localpty_windows.go)) included. It runs no test. |
| Cluster Tests (kind) | `ubuntu-latest` | kind v0.33.0 and kubectl v1.37.1 (`helm/kind-action`, install only); `make kind-up`, a wait for the seed's `web`, `api` and `toolbox`, `make test-kind`, `make kind-down` | views, logs, exec and the terminal's `kubectl` against a real API server and kubelet (Kubernetes 1.37.0, kind v0.33.0's default node image). `make test-kind` runs every package's tests with tag `kind` and the race detector, so the other tests also run on Linux here. The job's `KUBECONFIG` is `.cache/kind/kubeconfig`, the file `make kind-up` writes; nothing in it touches `~/.kube`. |
| Vulnerability Check | `ubuntu-latest` | `make vuln` (govulncheck v1.9.0) | no known vulnerability in a function Cogpit's code reaches |
| Coverage Badge | `ubuntu-latest`, `main` only | commits `.github/badges/coverage.json` when it changed | the README's coverage badge follows `main` ([Coverage](#coverage)) |
| Semantic Release (dry run) | `ubuntu-latest`, `main` only | `npm ci`, `npm audit signatures`, `npx semantic-release --dry-run` | the next version and its release notes, in the job log; the release tooling installs from [`package-lock.json`](../../package-lock.json) with verified registry signatures ([the dry run](#the-dry-run)) |

The Makefile targets the macOS job runs are the ones a contributor runs before a pull request
([testing](testing.md#before-a-pull-request)); `make test-unit-coverage` is `make test-unit-race`
with a coverage profile, the same packages and the same race detector.

**Verified locally on 2026-10-11** (macOS 26.6.2, Apple M1 Ultra): `make lint`, `make
test-unit-race`, `make test-unit-coverage` and `make build` pass; actionlint 1.7.12 with
shellcheck finds nothing in `build.yml` and `release.yml` (in `renovate.yml` it flags
`client-id`, which `actions/create-github-app-token@v3` accepts and its bundled metadata predates);
Renovate 44.151.2's config validator accepts [`.github/renovate.json`](../../.github/renovate.json) with `--strict` (2026-10-12);
the dry run passes with the committed configuration against a stand-in for the GitHub API.
**Not verified, because it only runs on GitHub:** the Windows and Linux cgo builds, the kind job,
the dry run with the real `GITHUB_TOKEN`, the badge commit. `make vuln` fails today: govulncheck
v1.9.0 reports vulnerabilities that Cogpit's code reaches in `golang.org/x/net` v0.57.0 (fixed in
v0.60.0) and `golang.org/x/image` v0.24.0 (fixed in v0.43.0), both indirect; the fix is a
`go.mod` change.

### Coverage

`make test-unit-coverage` runs `KUBECONFIG=/dev/null go test -race -covermode=atomic
-coverprofile=.cache/coverage/unit.out ./...`. The badge's number is the `total:` line of `go
tool cover -func` over that profile: the statements each package's own tests cover, over the
statements of every package in the module. Packages without tests count at 0 %
(`cmd/cogpit`, `cmd/cogpit-cli`, `hack/icon`, `internal/testkit`); there is no `-coverpkg`, so the
tests in `internal/testkit/integration` add nothing to the packages they exercise; the `kind`
tests do not count. **Measured: 84.5 %** on 2026-10-11 (macOS, commit `59b42f2`; the CI change adds
no Go code). CI computes it on macOS too, in the macOS job.

`make coverage-badge` turns the profile into the [shields.io endpoint
JSON](https://shields.io/badges/endpoint-badge) in
[`.github/badges/coverage.json`](../../.github/badges/coverage.json), coloured as in the team's other
repositories: ≥ 90 % brightgreen, ≥ 80 % green, ≥ 70 % yellow, ≥ 60 % orange, else red. The macOS
job uploads it as the artifact `coverage-badge` (the profile as `unit-coverage`) and writes
coverage per package to its job summary.

In the team's other repositories the release commit carries the badge (`@semantic-release/git`
with the badge as its asset). While Cogpit's releases are dry runs, the **Coverage Badge** job in
`release.yml` does that on `main`: when the number changed, it commits the file alone as
`chore(ci): coverage badge <n>% [skip ci]` and pushes it to `main`, rebasing up to three times if
`main` moved. It is the one job with a writing token (`contents: write`); it runs no release
tooling, checks that its commit touches nothing but the badge, and pushes only `main`. A push made
with `GITHUB_TOKEN` starts no workflow, so it cannot loop. The token's scope would allow a tag; a
tag ruleset (a repository setting) would make that impossible. Side effect: after a push that moves
the number, `main` gains a bot commit, and the next local push to `main` needs a `git pull
--rebase` first.

### Renovate

[`renovate.yml`](../../.github/workflows/renovate.yml) runs self-hosted Renovate with an
installation token of the team's GitHub App (`guided-traffic-automation`), from the organization
secrets `APP_CLIENT_ID` and `APP_PRIVATE_KEY`; both are visible to this repository (2026-10-11).
Whether the app is installed on this repository is not verified. The configuration is
[`.github/renovate.json`](../../.github/renovate.json), the team's with what Cogpit lacks taken
out (container images, Helm charts, Valkey images) and one group added (Fyne):

- Minor, patch and digest updates of Go modules, the Go version, GitHub Actions and the tools the
  Makefile pins merge on their own once CI passes; majors wait for review (Actions excepted, as in
  the team's repositories).
- Groups: the Go version with `golang.org/x/*`; `k8s.io/*` and `sigs.k8s.io/*`; Fyne
  (`fyne.io/*`, `github.com/fyne-io/*`).
- The Makefile's tool pins (`FYNE_TOOLS`, `GOLANGCI_LINT_VERSION`, `GOVULNCHECK_VERSION`) carry a
  `# renovate:` comment the custom manager reads; the `go` line of `go.mod` has its own manager,
  and the workflows follow it through `go-version-file`.
- Dependency updates are committed as `fix:` (a patch release once releases are real), Actions
  updates as `chore:`.
- `third_party/**` is not scanned (2026-10-12): the patched terminal widget keeps upstream's
  `go.mod` and workflows, which Cogpit's build does not use (the root `go.mod` decides every version
  through MVS, and GitHub runs only the root's workflows). The widget itself is tracked through
  the `github.com/fyne-io/terminal` line of the root `go.mod`, which never merges on its own: the
  `replace` makes that bump a no-op, so its pull request is the signal to move the copy onto the new
  upstream commit ([PATCHES.md](../../third_party/fyne-terminal/PATCHES.md)).
- Held back, each with the reason in its rule: `k8s.io/kube-openapi`, `structured-merge-diff` and
  `sigs.k8s.io/kustomize/*` follow the `k8s.io/*` release (kustomize v0.21.3 needs a kube-openapi
  on `structured-merge-diff/v7`, which apimachinery v0.37.x does not compile against);
  `github.com/godbus/dbus/v5` v5.2.3 does not compile on macOS and Windows.

Automerge needs every check green, so one flaky test holds back every pull request: on 2026-10-12,
44 were open, 3 of them broken by the update itself (the three held back above) and the rest by
tests that assumed an order of asynchronous results, which CI's slower runners do not keep.

Renovate's pull requests change `go.mod` and `go.sum` on their own branches; agents working in
parallel rebase onto them like onto any other commit on `main`. Renovate runs `go mod tidy` after
each Go update, which drops a requirement that no package imports yet.

## Release flow

The team's Go repositories release with semantic-release on `main`: `.releaserc.json` with the
`conventionalcommits` preset, and `@semantic-release/git` committing the coverage badge with the
release. Cogpit has the same [`.releaserc.json`](../../.releaserc.json) and the same pinned tooling
([`package.json`](../../package.json), [`package-lock.json`](../../package-lock.json)):

1. Commits on `main` follow [Conventional Commits](conventions.md#commits-and-pull-requests).
2. After the checks pass on `main`, `release.yml` runs semantic-release, which derives the next
   version from the commits since the last tag and writes the release notes. **Today this step is
   a dry run**: it logs the version and the notes and stops ([the dry run](#the-dry-run)).
3. *Planned:* a workflow on `release: published` (as the team's other repositories build their
   images) builds every artifact for the tag, code-signs what it has signing material for, writes
   `SHA256SUMS.txt`, an SPDX SBOM and the update manifest, and attaches them to the GitHub
   release. In `valkey-operator` that workflow is called `build.yml`; in Cogpit `build.yml` is the
   checks, so the artifact workflow gets a name of its own.
4. *Planned:* the update manifest is signed in a separate step with the update key, which CI
   cannot reach on its own: offline by the owner, or in a protected environment that needs the
   owner's approval. Until it is signed, the release is published but not offered as an update.
5. *Planned:* a `::warning::` in a job summary means a code-signing secret was missing and that
   artifact is unsigned; such an artifact is not offered as an update.

*Planned:* every binary reports its version, commit and build date (`cogpit --version`,
`cogpit-cli --version`, Help › About), set at build time with `-ldflags -X`.

### The dry run

semantic-release on a repository without tags cuts **1.0.0**: run locally on a clone of `main`
(commit `59b42f2`) with the committed configuration on 2026-10-11, it reported "There is no previous release, the next release
version is 1.0.0". A 1.0 tag and GitHub release for a pre-alpha is a promise nobody can take back
quietly. The owner therefore created the tag and GitHub release **v0.0.0** (2026-10-11) as the
anchor: semantic-release now continues from it (`feat:` → 0.1.0, `fix:` → 0.0.1), and 1.0.0 comes
only from a breaking change, which is cut when Cogpit is usable (owner). Until then **no commit
carries `!` or a `BREAKING CHANGE:` footer** ([conventions](conventions.md#commits-and-pull-requests)).
The job "Semantic Release (dry run)" keeps three guards, each enough on its own:

1. `npx semantic-release --dry-run`: no tag, no GitHub release, no release commit.
2. The job's token is read-only (`contents: read`): GitHub refuses a pushed tag and a created
   release.
3. Every `git push` from the job's checkout goes to an empty bare repository in the runner's temp
   directory (`url.<sink>.pushInsteadOf https://github.com/`).

The third guard is also what lets the dry run work with a read-only token: semantic-release checks
that it may push (`git push --dry-run` to the repository URL from `package.json`) before it does
anything else, in a dry run too, and a read-only token fails that check. Fetching tags and history
still goes to GitHub.

### The first real release

Decided (owner, 2026-10-12, [D-16](decisions.md#d-16-the-first-real-release)): releases stay dry
runs until the alpha; the alpha's release is the first real one, together with the job that builds
the unsigned macOS app as a download (part of the alpha's work). To cut it:

1. **The first version** continues from the owner's anchor tag `v0.0.0` (2026-10-11): the next
   `feat:` makes 0.1.0, a `fix:` 0.0.1; a breaking change makes 1.0.0, which waits until Cogpit is
   usable (owner).
2. **Flip the job** in [`release.yml`](../../.github/workflows/release.yml), as the team's other
   repositories run it: remove `--dry-run`; remove the step "Redirect git pushes into a throwaway
   repository"; create a GitHub App token (`actions/create-github-app-token@v3` with
   `permission-contents: write`) and use it for the checkout and as `GITHUB_TOKEN`, with the job's
   `permissions` raised to `contents: write`. A release created with the default `GITHUB_TOKEN`
   would start no `release: published` workflow.
3. **Remove the Coverage Badge job.** The release commit carries the badge from then on, as
   `@semantic-release/git` is already configured to do.
4. The artifact workflow (step 3 of the [release flow](#release-flow)) is
   [G14](../parity/gaps.md#g14-packaging-signing-and-updates-macos); without it a release has
   notes and a tag but no downloads.

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
