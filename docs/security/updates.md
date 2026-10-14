# Updates

The update channel: how Cogpit learns about a new version and replaces itself, one of the two
places where Cogpit downloads code that then runs. Since 2026-10-14 the macOS app carries Sparkle 2
and the release pipeline signs and publishes its appcast
([D-4](../developer/decisions.md#d-4-packaging-and-updates),
[releasing › the appcast](../developer/releasing.md#the-appcast)); the update key does not exist
yet, so no release can be offered and no copy updates itself until the owner has made it
([setting up Sparkle](../developer/releasing.md#setting-up-sparkle)). `cogpit-cli`, Windows and
Linux have no updater. The other place, built 2026-10-13, is the
`kubectl` a terminal offers for its cluster: downloaded only on the user's click and verified against
the Kubernetes release's own sigstore signature, not Cogpit's update key
([trust boundaries](trust-boundaries.md#boundaries-and-required-checks),
[H-35](trust-boundaries.md#h-35-kubectls-signature-is-checked-without-sigstores-transparency-log)).

## Chain of trust

```text
release pipeline ─► Cogpit-<v>-macos-arm64.dmg, Developer ID-signed, notarized, stapled
   "Appcast (draft)", no secret ─► the feed from Pages (only if the update key signed it) + this release's item
   "Appcast (sign)", environment sparkle, after the owner's approval
        ─► EdDSA signature over the disk image, then over the feed (Sparkle's sign_update)
   "Appcast (deploy)" ─► checked with the public key ─► https://guided-traffic.github.io/cogpit/appcast.xml
                                   │
Cogpit.app (Sparkle 2) ─► fetch the feed over HTTPS once a day, or on Help › Check for Updates…
       ─► the feed's EdDSA signature with SUPublicEDKey from the app's Info.plist
       ─► an item newer than the running CFBundleVersion, for this macOS? ─► Sparkle's window, the user's click
       ─► download the disk image from the release ─► its EdDSA signature, before anything is extracted
       ─► Apple's code signature of the new app (the running app's Developer ID team)
       ─► replace the app and relaunch
```

The chain is Sparkle 2's ([D-4](../developer/decisions.md#d-4-packaging-and-updates); its
[documentation](https://sparkle-project.org/documentation/#segue-for-security-concerns)), set up
in the app's `Info.plist` by [`hack/macos/sparkle.sh`](../../hack/macos/sparkle.sh). The app's side
is [`internal/update`](../../internal/update/); the pipeline's is
[`build.yml`](../../.github/workflows/build.yml), [`hack/macos/appcast.sh`](../../hack/macos/appcast.sh)
and [`hack/appcast`](../../hack/appcast/main.go).

| Check | Where | Prevents |
|---|---|---|
| The feed's EdDSA signature with the key in the app (`SURequireSignedFeed`, Sparkle 2.9) | every copy, on every check | a feed changed on or by the host: forged release notes or links, a critical or informational mark, an item pointing elsewhere ([H-91](#h-91-after-20-days-of-unverifiable-feeds-sparkle-presents-items-again) for when it fails for long) |
| The disk image's EdDSA signature, before extraction (`SUVerifyUpdateBeforeExtraction`) | every copy, per update | a swapped download; nothing of an unsigned archive is unpacked |
| Apple's code signature of the new app against the running app's | every copy, per update | an app that is not Cogpit's team's |
| Only a newer `CFBundleVersion` is offered; `sparkle:minimumSystemVersion` | every copy | a downgrade to a version with known flaws; an update the Mac cannot run |
| The update key behind the owner's approval, the environment's rules checked before signing | "Appcast (sign)" | a run the owner did not approve signing a feed ([H-92](#h-92-an-environment-sparkle-without-rules-hands-the-update-key-to-any-run)) |
| The history taken from Pages only when the update key signed it | "Appcast (draft)" | a feed altered on Pages being signed again with the next release |
| The signed feed and the disk image checked with the public key before Pages gets them | "Appcast (deploy)" | a feed every copy would refuse (a secret that does not match the app's key) |
| HTTPS | the feed and the download | eavesdroppers learning which version a user runs (not integrity: the signatures provide that) |
| Package-manager installs hand over to the manager | planned | Cogpit overwriting files a package manager owns |

## What a release holds today

Since 2026-10-13 ([D-16](../developer/decisions.md#d-16-the-first-real-release)) semantic-release
creates a tag and a GitHub release once every gate on `main` is green, and the release pipeline
([`build.yml`](../../.github/workflows/build.yml)) builds the artifacts from that tag and attaches
them ([releasing › the release pipeline](../developer/releasing.md#the-release-pipeline)):

| Artifact | Built by | What vouches for it today |
|---|---|---|
| `Cogpit-<v>-macos-arm64.dmg`, the app bundle in a disk image (from the first release after v0.4.0 on; no release has it yet) | `make dist-app` on GitHub's `macos-latest`, signed and imaged by `make dist-macos` ([`hack/macos/release.sh`](../../hack/macos/release.sh)) in the job "Sign and notarize (macOS)" | the app and the disk image signed with the Developer ID "Developer ID Application: Hans Fischer (6L57P2HXN6)", the hardened runtime and Apple's timestamp; the disk image notarized by Apple, its ticket stapled, so Gatekeeper accepts it offline as `Notarized Developer ID`; its SHA-256 in `SHA256SUMS` |
| `cogpit-cli_<v>_darwin_<arch>.tar.gz`, two architectures | `make dist-cli` on a self-hosted Linux runner, signed in the same job | the binary signed like the app and notarized; it cannot carry its ticket, so Gatekeeper looks it up online when a quarantined copy first runs; its SHA-256 in `SHA256SUMS` |
| `cogpit-cli_<v>_<os>_<arch>.tar.gz` for Linux, `.zip` for Windows, four platforms | `make dist-cli` on a self-hosted Linux runner | its SHA-256 in `SHA256SUMS`; no code signature |
| `Cogpit_<v>_darwin_arm64.zip`, the app bundle (v0.4.0 and earlier, and any build without the signing secrets) | `make dist-app` on GitHub's `macos-latest` | its SHA-256 in `SHA256SUMS`; an ad-hoc signature (`codesign --sign -`), which seals the bundle but names nobody: **no Developer ID, not notarized**, so Gatekeeper blocks the first start until the user allows it |
| `SHA256SUMS` | the release pipeline's last job | nothing: it is not signed, so it shows that a download matches what CI attached to the release, not who attached it |
| The release's item in `https://guided-traffic.github.io/cogpit/appcast.xml` (from the first release built with the update key; none yet) | the three appcast jobs after "Attach to the release" ([releasing › the appcast](../developer/releasing.md#the-appcast)) | the update key's EdDSA signature over the feed and over the disk image it names, made after the owner approved the run; checked with the public key before Pages gets it |

**What the Developer ID signature and the notarization vouch for:** that the code was signed with
the key of team 6L57P2HXN6's certificate and has not changed since, and that Apple's automated
check found no known malware in it when it was submitted. They do not say which commit it was
built from, and they reach as far as the key: whoever runs a workflow on a `v*` tag in this
repository can have that key sign what the workflow built
([H-62](#h-62-every-push-to-main-that-passes-the-gates-is-released)). The release page says what was
signed, naming the certificate the signature carries, or that the build was unsigned.

There is no update key yet, so no appcast is signed and no copy checks for an update: the app
says "this build has no update key" under Preferences › App › Check for updates automatically and
greys Help › Check for Updates…. Until the owner has made the key, a user learns of a new version
from the release page, and the Developer ID signature is what Gatekeeper checks when the new copy
first starts. Copies installed before the first release built with the key never update themselves
(no updater, or none with a key): one manual download moves them onto the chain above.

**What vouches for an update once the key exists:** the update key's EdDSA signatures over the
feed and over the disk image, computed in a job the owner approved, checked by Sparkle in every
copy; Apple's Developer ID signature and notarization of the app inside, checked by Sparkle against
the running app's and by Gatekeeper. **What does not:** the host. GitHub Pages serves the feed and
GitHub's release storage the disk image; either can withhold an update or serve an older signed
feed ([H-93](#h-93-the-feeds-host-can-withhold-updates)), and neither can make a copy install
anything the key did not sign.

### Keys and who holds them

| Key | Where it lives | What it can do | If it leaks |
|---|---|---|---|
| The organisation's GitHub App private key (`APP_CLIENT_ID`, `APP_PRIVATE_KEY`) | organisation secrets of `guided-traffic`, visible to this repository; used by the Semantic Release job (token with `contents: write`) and by Renovate | push to `main`, create tags and releases; a release it creates starts the release pipeline, which builds whatever the tag points at | rotate the App's private key in the App's settings and replace the organisation secret; delete the releases and tags it made. Secrets are not handed to workflows of pull requests from forks |
| The release pipeline's `GITHUB_TOKEN` | per job; `contents: write` only in its last job, to attach the artifacts | upload assets to and edit the text of this repository's releases | expires with the job |
| Developer ID Application certificate, App Store Connect team API key (role Developer) | secrets of the GitHub environment `release` (created 2026-10-13; the material in it since 2026-10-14), which only tag workflows may use (deployment rule: tags `v*`), so that no pull request workflow can read them. One job reads them, "Sign and notarize (macOS)": it builds nothing and runs no Go, imports the certificate into a temporary keychain (key not extractable, usable by `codesign` only) and the API key into an owner-only file, and deletes both when it ends ([releasing › signing and notarization](../developer/releasing.md#signing-and-notarization)) | sign and notarize code as team 6L57P2HXN6: whatever is signed and notarized with the two passes Gatekeeper on every Mac until Apple revokes the certificate | revoke the certificate at Apple, and the key in App Store Connect; releases notarized before stay valid (the owner's note, 2026-10-13; not checked against Apple's documentation) |
| The update key (Sparkle's EdDSA key; not made yet) | decided (owner, 2026-10-13): `SPARKLE_ED_PRIVATE_KEY`, a secret of the environment `sparkle` with the owner as required reviewer and tags `v*` only, so CI signs a release's feed only after the owner approved that run; made by the owner with Sparkle's `generate_keys`, which keeps it in his login keychain ([setting up Sparkle](../developer/releasing.md#setting-up-sparkle)). One job reads it, "Appcast (sign)": it refuses while the environment lacks its rules, hands the key to Sparkle's `sign_update` on its standard input only and runs no Go and no dependency's code. Its public half is the repository variable `SPARKLE_PUBLIC_ED_KEY`, built into every copy as `SUPublicEDKey` | sign appcasts and disk images that every copy with this key accepts | [H-15](#h-15-update-key-compromise-and-rotation) |

## What this does not cover

### H-61: Releases are unsigned

Live since the first release (2026-10-13); narrowed 2026-10-14. v0.4.0 and earlier carry no code
signature that names Cogpit. Since 2026-10-14 the release pipeline signs the macOS app, its disk
image and `cogpit-cli` for macOS with the Developer ID and has them notarized
([releasing › signing and notarization](../developer/releasing.md#signing-and-notarization));
no tag run has done it yet, so this holds for every published release until the first one after
v0.4.0. What stays live after it: `cogpit-cli` for Linux and Windows is not signed, and
`SHA256SUMS` is not signed, so whoever can change a release's assets (the GitHub App's key, a
maintainer's account) can replace one of those archives and its checksum together, and a user has
nothing to check them against. A swapped macOS download would need the Developer ID key as well.
Signing the checksums or the artifacts for Linux and Windows is not designed.

### H-62: Every push to `main` that passes the gates is released

Live since 2026-10-13. `main` has no branch protection and no required review (checked
2026-10-13), agents and maintainers push to it directly, and semantic-release releases every
`fix:` and `feat:` commit once CI is green, so the checks are the only gate between a commit and a
published release. Since 2026-10-14 that release's macOS downloads are signed with the owner's
Developer ID and notarized, so a commit that passes the gates ships as code Gatekeeper accepts
without a warning. The environment `release` admits any tag `v*`: whoever may create a release in
this repository can also have an arbitrary commit, tagged `v<anything>`, built, signed and
notarized; the environment has no required reviewer. Renovate's automerged dependency updates are `build(deps):` commits and release
nothing of their own (owner, 2026-10-13); they ship with the next `feat:` or `fix:`, unreviewed like
every automerged update.

### H-15: Update key compromise and rotation

Dormant until the key exists. Whoever holds the update key and can serve the feed (Pages) can offer
every copy an update Sparkle accepts. Sparkle's documentation says a release may change either the
Apple code signing certificate or the EdDSA key, but not both, so the key alone lets an update
carry another team's Developer ID ([rotating signing keys](https://sparkle-project.org/documentation/#rotating-signing-keys);
not tried). Where it lives is decided (owner, 2026-10-13: an environment secret that CI may use only
after the owner approved the run), which leaves a workflow the owner approves able to sign whatever
it built ([H-92](#h-92-an-environment-sparkle-without-rules-hands-the-update-key-to-any-run)).
Replacing a leaked key: a release whose app carries the new public key, offered in a feed and a
disk image still signed with the old one; with `SUVerifyUpdateBeforeExtraction` a change of the
EdDSA key needs a Developer ID-signed disk image, which Cogpit's is. A lost key: no feed can be
signed again; after 20 days Sparkle shows the items of an unsigned feed
([H-91](#h-91-after-20-days-of-unverifiable-feeds-sparkle-presents-items-again)). No second, offline
rotation key exists.

### H-16: Availability of the release host

Updates depend on two GitHub hosts: Pages for the feed (`guided-traffic.github.io`) and the release
storage for the disk image. If either is unreachable, Cogpit keeps running and offers no update; a
scheduled check fails silently, Help › Check for Updates… shows Sparkle's error. There is no
mirror.

### H-92: An environment `sparkle` without rules hands the update key to any run

Dormant today: the environment `sparkle` holds no secret. GitHub creates an environment the moment a
workflow run names it, without a reviewer and without a deployment rule; the first release run
after 2026-10-14 does so ("Appcast (sign)"). Live from the moment the owner stores
`SPARKLE_ED_PRIVATE_KEY` in it before the reviewer and the tag rule: then any workflow run in this
repository that names `environment: sparkle`, on any branch (agents and maintainers push to `main`
and branches directly, [H-62](#h-62-every-push-to-main-that-passes-the-gates-is-released)), gets the
key without anyone's approval. Cogpit's own job refuses to sign until it reads a required reviewer
and exactly the tag rule `v*` from the API, but a workflow written to take the key does not ask.
The owner's steps put the rules first ([setting up Sparkle](../developer/releasing.md#setting-up-sparkle)).
What stays live with the rules: the owner approves a run on a tag `v*`, and the job does with the
key what the workflow at that tag says; whoever may create such a tag (the GitHub App's key, a
maintainer) can bring a run of a changed workflow to the owner's approval, and the approval page
names the run and its tag, not what changed. The feed also signs the release's text as it stood
when the job ran: whoever can edit it before then (the same actors) has their text and links shown
in Sparkle's window. Administrators may bypass the reviewer while "Allow administrators to bypass"
is on (GitHub's default; `release` has it on).

### H-93: The feed's host can withhold updates

Live once a signed feed is on Pages. The signed feed has no expiry, and every feed ever deployed
is public. Per case: the host serves nothing or an error, and copies stay on their version without
a notice ([H-16](#h-16-availability-of-the-release-host)); the host serves an older signed feed (a
freeze), and copies older than its newest item are offered at most that item, a version with flaws
fixed since, while newer copies see nothing; a downgrade is not possible, since Sparkle offers only
a `CFBundleVersion` newer than the running one. Who can change what Pages serves: a workflow run
with `pages: write` that the environment `github-pages` admits (by default the default branch, and
the tag rule the owner adds), so whoever may push a workflow to `main` (H-62). No expiry or
freshness check is designed.

### H-91: After 20 days of unverifiable feeds, Sparkle presents items again

Dormant until the key exists; then live by design. Sparkle's `SUSignedFeedFailureExpirationInterval`
(default 20 days, Cogpit keeps it) lets a copy that has not validated a feed for that long present
the items of an unverified feed, without their release notes and links, as a failsafe for a lost
key ([Sparkle's security settings](https://sparkle-project.org/documentation/customization/#security-settings)).
Such an item still needs a disk image the update key signed, or, for a change of the key, a
Developer ID-signed disk image. Per case: an attacker who can serve the feed (H-93) and get a disk
image signed with this team's Developer ID (any run on a tag `v*` in the environment `release`,
which has no reviewer, H-62) could, after 20 days of failing feeds, have copies offer an update
that changes the update key; the copies' logs show the failures, the user still clicks Install.
Setting the interval to 0 closes this and turns a lost key into one manual download per copy; it
is the owner's choice (Cogpit's default: Sparkle's 20 days, as Sparkle documents the failsafe;
not tried).

### H-17: Installs owned by a package manager

For Homebrew (and, later, winget, Flathub and Linux repositories) the integrity of an update is
the package manager's, not Cogpit's.

### H-18: Dependencies

The Go modules Cogpit is built from are a supply chain of their own. The team's other repositories
use Renovate and govulncheck; Cogpit adopts both with M0, but no page describes the dependency
policy yet.
