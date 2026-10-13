# Releasing

> **Last reviewed:** 2026-10-13 · **Last updated:** 2026-10-13 · **Change:** major (CI split into parallel jobs of at most ten minutes, the UI's race tests sharded; semantic-release live, D-16 revised; the release pipeline attaches `cogpit-cli` and the unsigned app) · **Target:** Lens Desktop 2026.10.51501

Every push to `main` and every pull request runs the checks, each a job of its own with a make
target that runs the same on a laptop. Since 2026-10-13 ([D-16](decisions.md#d-16-the-first-real-release))
semantic-release is live: once every check on `main` is green it creates the tag and the GitHub
release, and the release pipeline builds the artifacts from that tag and attaches them
([release flow](#release-flow)). The artifacts are **unsigned** ([updates](../security/updates.md#what-a-release-holds-today)).
Platform targets and how they compare with Lens Desktop:
[platform parity](../parity/platform-parity.md#operating-systems-and-packages).

## CI

Three workflows in [`.github/workflows/`](../../.github/workflows/), laid out as in the team's
other repositories (`valkey-operator`):

| Workflow | Triggers | What it does |
|---|---|---|
| [`release.yml`](../../.github/workflows/release.yml) "Test and Release" | push to `main`, pull request to `main`, manual | the checks below, each a job; on `main`, Semantic Release once all of them passed. The README's build badge shows this workflow. |
| [`build.yml`](../../.github/workflows/build.yml) "Release Build" | a release published (`release: types: [published]`) | builds the artifacts from the release's tag and attaches them ([the release pipeline](#the-release-pipeline)) |
| [`renovate.yml`](../../.github/workflows/renovate.yml) | daily at 02:00 Berlin time, manual | dependency updates ([Renovate](#renovate)) |

**Budget: no job needs more than ten minutes** (owner, 2026-10-13). The UI's race tests take about
18 minutes in one process on a Linux runner, so they run in eight shards. Every job's timeout is
15 minutes: a run after a `go.sum` change starts without caches and saves them, which takes longer
once; the budget is for the runs in between.

**Concurrency:** a newer push cancels the running run of the same branch, on `main` too:
semantic-release releases every commit since the last tag, so a cancelled run loses nothing.

**Runners:** the Linux jobs run on the organisation's self-hosted runners (`runs-on: self-hosted`),
as the team's other repositories do: ephemeral pods on Ubuntu 24.04 with 16 CPUs of a shared
machine and Docker, but without `make`, a C compiler or `SHELL`. The macOS job and the Windows
compile run on GitHub's hosted runners (`macos-latest`, Apple silicon; `windows-latest`), which cost
nothing for a public repository; a Fyne app with cgo and AppKit does not cross-compile from Linux.

**Setup:** every Go job starts with [`.github/actions/go-setup`](../../.github/actions/go-setup/action.yml):
Go from `go.mod`, the build cache under a key per kind of build (`lint`, `unit`, `race-ui`,
`race-rest`, `kind`, `macos`, …) and `go.sum`, Fyne's Linux build dependencies with
`build-essential`, `SHELL=/bin/bash` where the pod exports none, and the modules from the module
proxy. A key is saved once per `go.sum` (about two minutes from the self-hosted runners, 11 to
14 MB/s up), so the next runs start with the dependencies compiled. The modules are not cached:
the self-hosted runners restore a cache at 6 to 9 MB/s, slower than `go mod download`.
**Every test runs on every run** (`GOFLAGS=-count=1`): the restored build cache also holds Go's
cached test results, and a race shard once "passed" in four seconds without running a test.

**Tokens:** every job's `GITHUB_TOKEN` is read-only (`contents: read`) except the release
pipeline's last job (`contents: write`, to attach the artifacts). Semantic Release and Renovate work
with an installation token of the organisation's GitHub App (`APP_CLIENT_ID`, `APP_PRIVATE_KEY`,
organisation secrets visible to this repository; [keys](../security/updates.md#keys-and-who-holds-them)).

| Job | Runner | Make target | What it proves | Minutes, run 38439307913 (caches new) |
|---|---|---|---|---:|
| Code Linting | self-hosted | `make lint` | golangci-lint with the team's linters and `depguard`'s layering ([architecture §3](architecture.md#3-dependency-layering-enforced)) | 2.4 |
| Parity | self-hosted | `make parity` | the parity checklists' rows and the numbers the progress pages report ([`make parity`](../parity/README.md#make-parity)) | 3.6 |
| GoSec Security Scan | self-hosted | `make gosec` | golangci-lint's gosec on the shipped code (tests, `internal/testkit` and `hack` left out); every accepted finding is excluded in [`.golangci.yml`](../../.golangci.yml) with its reason, a real one is fixed or an H-gap in [`docs/security`](../security/README.md) | 8.3 |
| Vulnerability Check | self-hosted | `make vuln` | govulncheck: no known vulnerability in a function Cogpit's code reaches | 7.0 |
| Cyclomatic Complexity | self-hosted | `make cyclo` | gocyclo: no function above `CYCLO_THRESHOLD`; the 20 most complex in the job summary ([below](#cyclomatic-complexity)) | 1.1 |
| Unit Tests | self-hosted | `make test-unit-coverage` | every package's tests without the race detector, with a coverage profile | 9.2 |
| Race Tests (UI n/8) | self-hosted, eight jobs | `make test-race-ui RACE_SHARD=n RACE_SHARDS=8` | one shard of `internal/ui/fyneui`'s tests under the race detector ([sharding](#the-uis-race-tests-in-shards)) | 5.2–12.2 |
| Race Tests (other packages) | self-hosted | `make test-race-rest` | every other package under the race detector, and the patched terminal widget's tests (`make test-fork`) | 7.5 |
| Integration Tests (kind) | self-hosted | `make kind-up`, `make test-kind` | the `TestKind*` tests against a real API server and kubelet (kind v0.33.0, Kubernetes 1.37.0, the seed of [`hack/kind`](../../hack/kind/README.md) and the Metrics Server); the tests compile while the cluster starts | 13.8 |
| Integration Tests (kind drain) | self-hosted | `make kind-drain-up`, `make test-kind-drain` | cordon, a blocked drain, a simulated crash and the finish on a three-node cluster of its own | 5.7 |
| macOS | `macos-latest` | `make test-darwin`, `make lint LINT_PKGS=…`, `make app` | the tests only macOS can run (the title bar's AppKit harness, the locale's CoreFoundation read, the tests that expect macOS's shortcut glyphs), the lint of the packages with darwin-only files, and the app bundle | 5.0 |
| Compile (windows) | `windows-latest` | `go build`, `go vet`, `go test -run '^$'` | every package and test compiles and links on Windows with cgo (MinGW-w64 from MSYS2) and Fyne; it runs no test | 11.6 |
| Combined Coverage Report | self-hosted, after the test jobs | `make coverage-report` | the unit, race and cluster profiles merged: the job summary, and the badge Semantic Release commits ([coverage](#coverage)) | 1.2 |
| Semantic Release | self-hosted, `main` only, after every job | `npx semantic-release` | the tag, the GitHub release with its notes, and the release commit with the badge ([release flow](#release-flow)) | 0.8 (the dry run) |

Linux needs no compile job: the unit and race jobs build and link every package and test there with
cgo. The cluster jobs name their kind cluster after the run (`cogpit-<run id>`), so two runs on one
Docker host cannot meet; `KUBECONFIG` points at the file `make kind-up` writes, and nothing touches
`~/.kube`.

**Reading a run:** `gh run list --limit 5`, then `gh run view <id>` for the jobs, `gh run view <id>
--log-failed` for what failed, and `gh run view <id> --json jobs` for each job's start and end. The
job summaries hold the coverage per package and the complexity report.

### The UI's race tests in shards

[`hack/shard`](../../hack/shard/main.go) lists the top-level tests, examples and fuzz targets of
`internal/ui/fyneui` from the test files `go list -tags race` reports, assigns each name to a shard
by an FNV-1a hash of the name, and prints one shard's `-run` pattern; `make test-race-ui` runs it.
A test added later lands in a shard without a change to the workflow, and a name keeps its shard as
long as the number of shards stays. Eight shards from measurements (2026-10-13, the package's 548
tests under the race detector, 1,236 s locally on an M1 Ultra, 1,077 s on a 4-CPU Linux runner):
the hash puts 111 to 179 s of tests into a shard, the longest single test is
`TestRenderPrometheusMetrics` at 46 s. To see the split: `go run ./hack/shard -pkg
./internal/ui/fyneui -tags race -of 8 -summary`, with `-durations <go test -json output>` for each
shard's time. To change the number of shards, change the matrix and `RACE_SHARDS` in
`release.yml`'s `race-ui` job together.

### Coverage

Every test job writes a profile to `.cache/coverage` and uploads it (`profile-*`): `unit.out`
(every package without the race detector, each package counting its own tests), `race-ui-<n>.out`
and `race-rest.out` (the same under the race detector), `integration.out` and
`integration-drain.out` (the cluster tests with `-coverpkg=./...`, so they count for every package
they reach). The Combined Coverage Report job merges them with [`hack/coverage`](../../hack/coverage/main.go)
(`make coverage-report`): a block counts once, as covered when any profile covered it. It writes the
total and a table per package to the job summary and
[`.github/badges/coverage.json`](../../.github/badges/coverage.json), the [shields.io endpoint
badge](https://shields.io/badges/endpoint-badge), coloured as in the team's other repositories
(≥ 90 % brightgreen, ≥ 80 % green, ≥ 70 % yellow, ≥ 60 % orange, else red). Semantic Release
commits the badge with the release (`@semantic-release/git`), so the README's number moves with
releases. **Measured:** 83.6 % on 2026-10-13 (run 38439307913: 37,635 of 45,002 statements; the unit profile alone gave 82.9 %).

### Cyclomatic complexity

`make cyclo` runs gocyclo v0.6.0 over `cmd`, `internal`, `hack` and `third_party`, tests and
generated code left out, and fails above `CYCLO_THRESHOLD`. The threshold is **56**, the highest
complexity `main` had when the check came (2026-10-13: `(*shell).applyUIState` in
[`shell.go`](../../internal/ui/fyneui/shell.go)), so no function grows past it. The team's other
repositories use 15; 121 functions are above that today. Lowering the threshold is the owner's
call.

### Running a job locally

Each job's make target runs the same here (fast local checks first: [AGENTS.md §5](../../AGENTS.md#5-before-you-finish-a-task)):

```sh
make lint parity gosec vuln cyclo
make test-unit-coverage
make test-race-ui RACE_SHARD=3 RACE_SHARDS=8   # one shard; make test-unit-race runs everything
make test-race-rest
make kind-up && make test-kind                 # the cluster tests; make kind-down after
make test-darwin                               # macOS only
make coverage-report                           # merges what is in .cache/coverage
```

Tests run with `KUBECONFIG=/dev/null` and without `KUBERNETES_SERVICE_HOST` and
`KUBERNETES_SERVICE_PORT` (`TEST_ENV` in the [Makefile](../../Makefile)): inside a pod, as on CI's
self-hosted runners, client-go would otherwise take the pod's namespace as a context's default and
could reach the pod's own cluster.

### Renovate

[`renovate.yml`](../../.github/workflows/renovate.yml) runs self-hosted Renovate with an
installation token of the team's GitHub App (`guided-traffic-automation`), from the organization
secrets `APP_CLIENT_ID` and `APP_PRIVATE_KEY`; both are visible to this repository (2026-10-11), and
the Semantic Release job's token from the same App proves it is installed here (the dry run of
run 38439307913, 2026-10-13: "Allowed to push to the Git repository", next version 0.1.0). The configuration is
[`.github/renovate.json`](../../.github/renovate.json), the team's with what Cogpit lacks taken
out (container images, Helm charts, Valkey images) and one group added (Fyne):

- Minor, patch and digest updates of Go modules, the Go version, GitHub Actions and the tools the
  Makefile pins merge on their own once CI passes; majors wait for review (Actions excepted, as in
  the team's repositories).
- Groups: the Go version with `golang.org/x/*`; `k8s.io/*` and `sigs.k8s.io/*`; Fyne
  (`fyne.io/*`, `github.com/fyne-io/*`).
- The Makefile's tool pins (`FYNE_TOOLS`, `GOLANGCI_LINT_VERSION`, `GOVULNCHECK_VERSION`,
  `GOCYCLO_VERSION`, `METRICS_SERVER_VERSION`) carry a
  `# renovate:` comment the custom manager reads; the `go` line of `go.mod` has its own manager,
  and the workflows follow it through `go-version-file`.
- Dependency updates are committed as `fix:`, so since 2026-10-13 every automerged update becomes
  a patch release ([H-62](../security/updates.md#h-62-every-push-to-main-that-passes-the-gates-is-released));
  Actions updates as `chore:`, which release nothing.
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

The team's Go repositories release with semantic-release on `main`; Cogpit has the same
[`.releaserc.json`](../../.releaserc.json) (the `conventionalcommits` preset, `@semantic-release/git`
committing the coverage badge with the release) and the same pinned tooling
([`package.json`](../../package.json), [`package-lock.json`](../../package-lock.json)):

1. Commits on `main` follow [Conventional Commits](conventions.md#commits-and-pull-requests):
   `feat:` makes a minor release, `fix:` and `perf:` a patch release, every other type none.
2. Once every job of a push to `main` is green, the Semantic Release job creates a token of the
   organisation's GitHub App (`actions/create-github-app-token@v3`, `permission-contents: write`),
   installs the tooling from `package-lock.json` with verified registry signatures (`npm audit
   signatures`), puts the combined coverage badge in place and runs `npx semantic-release`: it
   derives the version from the commits since the last tag, writes the release notes, commits the
   badge as `chore(release): <version> [skip ci]` to `main`, tags that commit `v<version>` and
   publishes the GitHub release. `[skip ci]` keeps the release commit from starting another run.
3. The release's `published` event starts [the release pipeline](#the-release-pipeline), which
   builds the artifacts from the tag. It starts because the App's token created the release: a
   release made with `GITHUB_TOKEN` starts no workflow.

**v0 until the owner decides on 1.0.** Releases continue from the owner's anchor tag `v0.0.0`
(2026-10-11). A breaking change makes a **minor** release, not 1.0.0 (`releaseRules: [{"breaking":
true, "release": "minor"}]` in the commit analyzer; the other rules are its defaults). The rule is
needed already: commit `f3ad000` has a body line starting with `BREAKING CHANGE:`, which the
analyzer reads as a breaking change, and without the rule the first release would have been 1.0.0
(the dry run of 2026-10-13 said so; with it, 0.1.0). Commits still carry no `!` and no `BREAKING
CHANGE:` footer ([conventions](conventions.md#commits-and-pull-requests)).

**After a release, `main` has moved:** the release commit is on `main`, so the next local push
needs `git pull --rebase` first.


## The release pipeline

[`build.yml`](../../.github/workflows/build.yml) "Release Build" runs on `release: published`,
checks out the release's tag and builds with `VERSION` set to the tag without its `v`:

| Job | Runner | Make target | Builds |
|---|---|---|---|
| cogpit-cli (linux, darwin, windows) | self-hosted | `make dist-cli` | `cogpit-cli_<v>_<os>_<arch>.tar.gz` for linux and darwin, `.zip` for windows, each on amd64 and arm64: pure Go (`CGO_ENABLED=0`), `-trimpath`, `-s -w`, the version set with `-X main.version`, with `LICENSE` and `README.md`. The job checks that the Linux binary's `--version` says the release's version. |
| Cogpit.app (macOS arm64) | `macos-latest` | `make dist-app` | `Cogpit_<v>_darwin_arm64.zip`: `make app` (`fyne package`, the version in `Info.plist` and the binary, an ad-hoc signature) zipped with `ditto`. The job checks the bundle's version, its signature seal and its architecture. |
| Attach to the release | self-hosted, after both | `make dist-checksums` | `SHA256SUMS` over the archives; attaches everything to the release (`softprops/action-gh-release@v3`) and appends a note to the release text: what the downloads are, that they are unsigned, how macOS lets the user start the app anyway, and that desktop packages for Linux and Windows come with M11 |

`make dist-cli`, `make dist-app VERSION=0.0.99` and `make dist-checksums` run the same locally into
`dist/` (gitignored). To check a download: `sha256sum -c SHA256SUMS --ignore-missing` (on macOS
`shasum -a 256 -c`), then `cogpit-cli --version`.

**Not built yet:** the Developer ID signature and notarization (they need the owner's Apple
credentials, as secrets of a GitHub environment `release` that only tag workflows may use; until
then the app is unsigned and the release page says so, never a failed release), the disk image, the
universal (Intel) app, the Homebrew cask and Sparkle's update feed
([D-4](decisions.md#d-4-packaging-and-updates)); desktop packages for Linux and Windows (M11); an
SBOM.

## What gets built

| Platform | Today (2026-10-13) | Planned | Signing |
|---|---|---|---|
| macOS 13+ | `Cogpit_<v>_darwin_arm64.zip` (the app bundle, Apple silicon); `cogpit-cli` for amd64 and arm64 | `Cogpit-<v>-macos-universal.dmg` with `Cogpit.app` ([D-4](decisions.md#d-4-packaging-and-updates)), built per architecture with cgo and joined with `lipo`, imaged with `hdiutil` | today none (ad-hoc only); planned: app and disk image signed with Developer ID, the disk image notarized, its ticket stapled |
| Windows 10+ x64 and arm64 | `cogpit-cli` zip | `Cogpit-<v>-windows-<arch>.msi`, a portable zip (M11), on native runners (cgo) | planned: Authenticode |
| Linux x86_64 and aarch64 | `cogpit-cli` tar.gz | AppImage, deb, rpm, tar.xz, Flatpak (M11), on native runners (cgo, OpenGL and X11/Wayland headers) | planned: checksums, SBOM, signed update manifest |

**The macOS artifact is to become the app bundle `Cogpit.app` in a disk image**: the standard way to
install a Mac app outside the App Store, without a terminal. Opening the image shows the app next to
a link to Applications; the user drags it across. The disk image is notarized and its ticket
stapled, so it also opens offline without a Gatekeeper prompt. The Homebrew cask (`auto_updates
true`) and Sparkle's updates use the same disk image.

`cogpit-cli` is pure Go (`CGO_ENABLED=0`) and cross-compiles from one runner. The desktop app needs
cgo for Fyne's OpenGL driver, so each platform builds on its own runner. Whether `fyne package`
alone covers MSI, AppImage, deb, rpm and Flatpak, or `packaging/` needs extra tools, is criterion S7
of the [spike](spike-fyne.md).

## Updates

Cogpit shows available updates and installs them with one click
([D-4](decisions.md#d-4-packaging-and-updates)); on macOS through Sparkle 2. Each release carries
an update manifest, on macOS Sparkle's appcast (version, disk image, its EdDSA signature), signed
with an update key that is kept outside CI's reach (offline by the owner, or in a protected
environment that needs the owner's approval; until it is signed, a release is not offered as an
update); the app verifies the manifest signature with the public key built into it, then the
checksum, then the operating system's code signature, before it replaces anything. Installs owned
by a package manager (Homebrew, winget, Flathub, deb or rpm repositories) hand over to the
manager. Security perspective: [updates](../security/updates.md).

## Secrets

| Secret | For | Where | Without it |
|---|---|---|---|
| `APP_CLIENT_ID`, `APP_PRIVATE_KEY` (the organisation's GitHub App) | Semantic Release's tag, release and release commit; Renovate | organisation secrets, exist | no release (the job fails) |
| `MACOS_SIGN_P12` (the Developer ID Application certificate with its key, base64), `MACOS_SIGN_P12_PASSWORD` | signing the app bundle and the disk image | planned: GitHub environment `release` (deployment rule: tags `v*`) | unsigned app, and the release page says so |
| `APPLE_TEAM_ID` (a variable) | the signing identity | planned: environment `release` | as above |
| `APPLE_API_KEY_P8` (an App Store Connect team API key, role Developer, base64), `APPLE_API_KEY_ID`, `APPLE_API_ISSUER_ID` | notarization with `notarytool` | planned: environment `release` | not notarized |
| Windows code-signing certificate or signing service credentials | Authenticode | planned (M11) | SmartScreen warns |
| Update signing key | the update manifest | not decided | no update offered for that release |

Who holds which key and what follows if one leaks:
[updates › keys](../security/updates.md#keys-and-who-holds-them).
**Human:** the Apple and Windows material does not exist yet; the owner provides it
([G14](../parity/gaps.md#g14-packaging-signing-and-updates-macos)).
