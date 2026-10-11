# Security policy

## Reporting a vulnerability

Please do not open a public issue with exploit details, credentials, cluster names or crash dumps.

GitHub private vulnerability reporting is **not enabled** on this repository yet. Until it is,
open a public issue titled "Security contact request" that contains no details, and a maintainer
will set up a private channel with you. Once private reporting is enabled, use the
**Report a vulnerability** button on the repository's Security tab instead.

A useful report names the Cogpit version or commit, the operating system and architecture, the
affected component, the impact and the access an attacker needs, and exact reproduction steps.
Remove tokens, client keys, kubeconfig contents, cluster endpoints and other identifying data from
anything you send.

## Supported versions

No version has been released. Once releases exist, only the latest release receives security fixes.

| Version | Supported |
|---|---|
| none released | – |

## Known limitations

The security architecture, its trust boundaries and its open gaps (`H-<n>`) are documented in
[`docs/security/`](docs/security/README.md). A gap listed there is known and need not be
reported again, but a way to exploit it is still welcome.
