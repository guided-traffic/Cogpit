# Updates

The update channel: how Cogpit learns about a new version and replaces itself, one of the two
places where Cogpit downloads code that then runs. No code exists yet (2026-10-11); the design is
[D-4](../developer/decisions.md#d-4-packaging-and-updates) and
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

## What this does not cover

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
