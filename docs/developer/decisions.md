# Decisions

> **Last reviewed:** 2026-10-12 · **Last updated:** 2026-10-16 · **Change:** minor (D-4: the appcast's deploy lets a newer release win over an older item its draft never saw, the owner's correction of F18 after v0.19.0's refused deploy, 2026-10-16); minor (D-58: the selection bar's greyed Delete names the engine's reason in its tooltip, "Delete — connection interrupted", the owner's audit decision of 2026-10-16 (c), built the same day with the build's choices for the owner to confirm; its index row); minor (D-59: the copy icon over a Secret's editor, the owner's audit decision of 2026-10-16 (c), built the same day, every copy through the clipboard rule for Secret values; D-25 says so; their index rows); minor (D-55 and D-59: keys added and removed in a Config Map's and a Secret's drawer, the owner's audit decision of 2026-10-16 (N152), built the same day with the lead's decisions for the owner to confirm; a save that removes a key destructive, the review its confirmation; their index rows); minor (D-10 (6): a chat connection's CA file replaces the system's roots for that connection, as kubectl's certificate-authority, the owner's decision in the audit of 2026-10-16, N141, built the same day); minor (D-57: every cluster and kubeconfig command under the menu path `Cluster/`, the inconsistency it noted resolved, the owner's decision in the audit of 2026-10-16, N121); minor (the owner's audit of the night decisions of 2026-10-15/16 (2026-10-16 afternoon): the lead's decisions in D-10, D-11, D-15, D-43, D-55, D-57, D-58, D-59, D-62, D-63, D-64 and D-65 confirmed as built where he chose the built option, the rows the audit did not ask left open; changed by him and not built yet: the copy from a Secret's editor, D-59, and the selection bar's greyed Delete, D-58; to be changed, how not said yet: the welcome page's button, D-57; his four other changes (the checked rows after `ui.navigate`, the Cluster menu's paths, D-10's CA file, D-59's keys) left to the changes that build them; their index rows, the stale duplicate rows of D-43, D-62 and D-63 removed); minor (D-61 withdrawn by the owner 2026-10-16 and removed: Cogpit installs nothing into a cluster for its own features; the stack, its commands and N183 to N192 gone, H-105 closed, the reading improvements kept; its index row); minor (D-69 stage 3 built 2026-10-16, the two settings: a cluster that is not shown holds its credential plugin while its setting "Renew the login while the cluster is not shown" is off, its login then lapsing (`loginLapsed`, not red) until it is shown or logged in again; Preferences › App › Restore the last session, off by default, connecting the clusters of the last quit at start, the shown one first; D-41 and D-42 changed by the owner's decision of 2026-10-16, D-58's background renewal; the lead's decisions for the owner to confirm); minor (D-69 stage 2 built 2026-10-16, the window: switching without reconnecting at every entry point, the marks of item 5 on every dot, each cluster blinking for itself, the dock per cluster as engine state, Disconnect per cluster in its menus, Diagnostics per cluster, a failed connect shows the cluster shown before again, `ConnectedCluster.LoginLapsed` for stage 3; the lead's decisions for the owner to confirm; its index row); minor (D-69 stage 1 built 2026-10-16, the engine: a pool of backends by context, one shown, `cluster.connect` of a connected context only shows it, the left cluster's views warm for 5 minutes and then cold, `cluster.disconnect {context?}`, every command on its own cluster (a reference's, a dock tab's, an optional `context`), the connection watch and the write gate per cluster, `UIState.Connections`; the lead's decisions for the owner to confirm; its index row); minor (D-69 added: several clusters connected at once, the owner's interview of 2026-10-16, not built); minor (D-58 S3: the light Running green #177f4a, the owner's decision 2026-10-16, 3.1:1 on a selected row); minor (D-68, the owner's F17 of 2026-10-16: one short name for a cluster everywhere, the part after the last "@"; the user Cogpit is logged in as in the navigator's root row from a SelfSubjectReview at every connect, the kubeconfig fallback marked, the last known user while not connected, the groups in a tooltip; agents read the username and the groups in `ui_state`; built, the lead's decisions for the owner to confirm; its index row); minor (D-70 added: the disk image's window, the owner's visual review and pipeline choice of 2026-10-16, built); minor (D-66: the connection log, decided by the owner 2026-10-16 and built, the lead's choices for the owner to confirm, H-112 opened, its index row); minor (D-58's S3 amended by the owner, F13 2026-10-16: the connection as a dot, never a word, Connected in the Running green, blinking while (re)connecting with a 3 s minimum, a click opening the connection log, D-66; the lead's decisions for the owner to confirm; D-22 and D-23 follow; its index row); minor (D-67: the demo cluster compiled out of releases by the tag `nodemo`, the owner's decision of 2026-10-16, built); minor (D-4: the appcast's deploy refuses a feed that drops a served item, F18, owner 2026-10-16); minor (F11, owner 2026-10-16: D-4's scheduled update check every hour instead of Sparkle's day; D-54's Update button names the version, "Update to v0.16.0", and says "Update" where the bar has no room for it; its index rows); minor (D-4: the first update through the feed, 2026-10-16); minor (D-36 checked against a live Argo CD 3.5.5 controller on a real cluster, 2026-10-16: the sync windows' check, the project's namespace and the operation's source brought to Argo CD's, the lead's decisions for the owner to confirm; its index row); minor (D-63: Drain built as a background job 2026-10-16, night run: a step per pod with the PodDisruptionBudget's wait, Cancel, a stopped drain kept in the journal as `stopped` and offered as after a crash, an MCP client that gives up no longer stops it; the lead's decisions for the owner to confirm; D-11 and D-58 follow, H-111 opened, its index row); minor (D-65 discovery cached on disk: the lead's decisions of the night run of 2026-10-16 for the owner to confirm, built, measured on a real cluster, its index row; D-13 says the answer holds no object); minor (D-62: checks against a real cluster, the lead's decisions for the owner to confirm, 2026-10-16); minor (D-15: the scorecard built 2026-10-16 in the night run, `make scorecard` and docs/parity/scorecard.md, the lead's choices for the owner to confirm; its index row); minor (D-64: envtest, a real API server in CI without Docker, built 2026-10-16 in the night run with the lead's decisions for the owner to confirm: controller-runtime's envtest v0.25.2, the binaries pinned by SHA-512, SIGSTOP of the API server for D-58's standby case, a job of its own); minor (D-64: envtest, a real API server in CI without Docker, built 2026-10-16 in the night run with the lead's decisions for the owner to confirm: controller-runtime's envtest v0.25.2, the binaries pinned by SHA-512, SIGSTOP of the API server for D-58's standby case, a job of its own); minor (D-61: Cogpit Metrics, the bundled metrics stack of Lens Metrics, the lead's decision for the owner to confirm, built and tried on kind and a real cluster; D-43: a node's network and filesystem from node-exporter, the lead's choices for the owner to confirm); minor (D-63 built 2026-10-16, night run: long commands as background jobs, an action on a selection of more than ten objects with progress in the dock and Cancel for every origin, the lead's decisions for the owner to confirm, with its index row); minor (D-58 standby and lost connections: the owner's decisions S1–S8 of 2026-10-15 evening, the lead's defaults, built 2026-10-15/16 with the build's decisions for the owner to confirm, its index row); minor (D-15: parity-checklist.md generated by hack/parity, built 2026-10-16); minor (D-59: a Secret's values edited in the drawer, the window's only, decided by the owner 2026-10-15 and built the same night; D-55: the owner's audit answers for a Config Map's drawer built); minor (D-55: the owner's audit answers of 2026-10-15 for a Config Map's drawer built: Save rebases by itself when no edited key changed, unsaved values kept as disk drafts; a Secret's values in the drawer decided); minor (D-57: an insecure context's warning icon on its section row; the selection bar in the item count's place); minor (D-10: the owner's audit answers of 2026-10-15 for the chat recorded and built in the night run: send all tools per connection; the warning for `http://` to another machine and a CA file per connection, item 10 revised; F9, a sent message edited and sent again, H-100; the index row); minor (D-10: the owner's audit answers of 2026-10-15 for the chat recorded and built in the night run: send all tools per connection; the warning for `http://` to another machine and a CA file per connection, item 10 revised; the index row); minor (D-10: the owner's audit answers of 2026-10-15 for the chat recorded and built in the night run: send all tools per connection; the index row); minor (D-57 built: choosing a cluster in the navigator's section only, the context picker, "Switch context" and the scope dropdown gone; D-42, D-47 and D-51 follow); minor (D-9: verifying the credential plugins against EKS, GKE and AKS out of scope until long after parity, the owner's decision of 2026-10-15, with its index row; the index brought to the code: D-10 built (M12), the missing rows of D-14 to D-32 added; D-1: S7's re-check half done since CI; D-2: the round's answers built, the token kept in the configuration directory; D-4: Sparkle no longer "not built"; D-44: the built-in agent built; D-48: the releases' Developer ID signature; D-16: the bundled `cogpit-cli` in signed releases since v0.7.0, a quarantined copy not tried; the round's intro says what the night run built); minor (D-4: Sparkle's update key set up by the owner on 2026-10-15, in v0.11.0 to v0.13.0, no signed appcast yet; A10 decided further, the App the tag ruleset's only bypass, the settings not made; confirmed by the owner on 2026-10-15: D-47 as a whole, D-10 item 12's repaired key file and no keychain migration, D-51's ↵ on an agent's palette text; D-51's dropdown bank moot; the index rows of D-2, D-33, D-34, D-36, D-41 and D-44 no longer say "not built", checked against the code); minor (D-24 revised: the glowing C helm as the logo everywhere, every earlier version kept and drawable)

Decisions that shape the code, each with its options, the reasons, and its status. A decision
moves from **open** to **decided** only by the owner; a recommendation is a proposal, not an
outcome. Ecosystem facts were observed on 2026-10-11 from the Go module proxy, crates.io and the
projects' repositories; re-check them before relying on a version number.

| Id | Decision | Status |
|---|---|---|
| [D-1](#d-1-ui-toolkit) | UI toolkit | decided: Fyne, after the spike gate; re-checks of S7 (CI, M0) and S5 (Windows and Linux, M11) can reopen it per platform |
| [D-2](#d-2-agent-interface-scope) | What agents may do over MCP | decided: everything, the UI included; destructive actions confirmed by the user; 2026-10-14 evening (owner): more agents' calls always wait for the user (every RBAC kind, Helm install and upgrade, removing a bank with tiles) or become confirmed writes (`cluster.setPrometheus`, a proxy), notifications' read state GUI-only; built 2026-10-14 night |
| [D-3](#d-3-helm-and-kubectl-binaries) | Where `helm` and `kubectl` come from | decided: Helm in-process; `kubectl` from `PATH`, a verified matching download on one click; 2026-10-14 evening (owner): the `PATH` hand-over for bash and fish, a setting that enforces Cogpit's kubectl (both built 2026-10-14), the signature with Rekor ([D-50](#d-50-kubectls-signature-check), built 2026-10-15) |
| [D-4](#d-4-packaging-and-updates) | Packaging, signing and updates | decided for macOS: the app in a DMG, Homebrew cask, Sparkle updates; Windows and Linux with M11. Built: since 2026-10-14 the app for Apple silicon in a signed, notarized DMG with `cogpit-cli` inside it, the only download besides `SHA256SUMS` ([D-16](#d-16-the-first-real-release) revised; v0.6.0 to v0.8.0 carry it, v0.5.0 and v0.9.0 to v0.12.0 got no files; before, the unsigned app zip and `cogpit-cli` archives for six platforms); since 2026-10-14 Sparkle 2 in the app and the signed appcast's jobs; the update key set up by the owner on 2026-10-15 and in the builds since v0.11.0; the signed feed served since 2026-10-16 and the first update through it that day (the owner's copy, 0.15.0 to 0.16.0); 2026-10-14 evening (owner): the failure rule, the 20-day failsafe and who may change the automatic checks confirmed, a tag ruleset to set up (2026-10-15, owner: the semantic-release App its only bypass; not set up); scheduled checks every hour (owner, 2026-10-16, F11; built) |
| [D-5](#d-5-extensions) | Extensions | decided: built into Cogpit's own code; no external extensions for now |
| [D-6](#d-6-kubernetes-libraries) | Kubernetes libraries | decided: client-go, kubectl libraries, Helm SDK |
| [D-7](#d-7-language-and-platforms) | Language and platforms | decided: Go; a macOS app first, Windows and Linux later |
| [D-8](#d-8-licence) | Licence | decided: Apache-2.0 |
| [D-9](#d-9-scope-against-lenss-paid-and-vendor-features) | Scope against Lens's paid and vendor features | decided: Flux CD and Argo CD in the beta (built 2026-10-13 and 2026-10-14), other paid features after it, vendor features never; verifying the credential plugins against EKS, GKE and AKS out of scope until long after parity, no longer an alpha-gate or M1 item (owner, 2026-10-15) |
| [D-10](#d-10-built-in-agent) | Built-in agent and LLM interface | decided: a chat column at the window's right edge, OpenAI-compatible endpoints, several connections, a private-connection mark, kept scrubbed conversations, limits per turn (owner, 2026-10-14 late evening); the API keys in an owner-only file, no keychain (owner, 2026-10-15); a key file others could access repaired with a log warning, no migration of keychain keys (confirmed, owner, 2026-10-15); built: the engine side 2026-10-14 late evening, the chat column, Preferences › Agent and the private connection in the night of 2026-10-14/15, the key file 2026-10-15; the owner's audit answers of 2026-10-15 for the chat (send all tools per connection, a warning for `http://` to another machine and a CA file per connection, a sent message edited and sent again, F9): all three built 2026-10-15 (night run); the CA file replacing the system's roots for its connection since the owner's decision in the audit of 2026-10-16 (N141); M12 stays open for the live runs against LiteLLM and Open WebUI ([roadmap](../parity/roadmap.md#milestones)) |
| [D-11](#d-11-crash-stance) | Crash stance | decided: crash-tolerant UI, crash-safe cluster operations; Helm install (owner, 2026-10-14 evening): nothing offered after an interrupted install, built 2026-10-14 (the journal still records installs and the banner offers Show release: the lead's decisions for the owner to confirm) |
| [D-12](#d-12-parity-sources) | Parity sources | decided: current Lens Desktop only; no OpenLens or Freelens |
| [D-13](#d-13-cache) | Cache | decided: the UI reads only from a watch-fed cache, held in memory only; Secrets in full (revised 2026-10-12, built), Helm's releases from that watch or their own `owner=helm` watch |
| [D-14](#d-14-terminal-widget) | Terminal widget | decided (owner, 2026-10-12): the patched copy of `fyne-io/terminal` in `third_party/fyne-terminal`, its patches offered upstream, the widget fuzzed alone; built 2026-10-12; 2026-10-14 evening: its four shell tests under the race detector in CI (A116), built 2026-10-15; the two newer patches offered upstream at the end (A86) |
| [D-15](#d-15-progress-generators) | Progress generators | decided (owner, 2026-10-12): `make parity` first, the checklist page and the scorecard later; `make parity` built 2026-10-12; the checklist page and `make scorecard` built 2026-10-16 (night run; the scorecard's main choices confirmed by the owner, audit of 2026-10-16, the rest and the checklist page's for the owner to confirm); the scorecard in CI waits for a fixed runner |
| [D-16](#d-16-the-first-real-release) | The first real release | decided (owner, 2026-10-12): a dry run until the alpha; revised 2026-10-13 (releases live: semantic-release on every green `main`, built) and 2026-10-14 (only the complete product, `cogpit-cli` inside the app, built 2026-10-14) |
| [D-17](#d-17-token-files-named-by-added-kubeconfigs) | Token files named by added kubeconfigs | decided (owner, 2026-10-12): a token file outside the kubeconfig's folder only once the user allowed it, as a credential plugin; built 2026-10-12 |
| [D-18](#d-18-secret-annotations-seen-by-agents) | Secret annotations seen by agents | decided (owner, 2026-10-12): agents see them in full but kubectl's `last-applied-configuration`; the risk documented (H-23), nothing to build |
| [D-19](#d-19-editing-and-creating-secrets-from-yaml) | Editing and creating Secrets from YAML | decided (owner, 2026-10-12): in the GUI only; built 2026-10-12 |
| [D-20](#d-20-a-port-forward-that-cannot-connect) | A port forward that cannot connect | decided (owner, 2026-10-12): Failed at once on an error that waiting does not cure, retried on one that can pass; built 2026-10-12 |
| [D-21](#d-21-pasting-a-kubeconfig) | Pasting a kubeconfig | decided (owner, 2026-10-12): no kubeconfigs from pasted text; built the same day (the paste removed) |
| [D-22](#d-22-colour-theme) | Colour theme | decided (owner, 2026-10-12): Solarized Dark and Light after VS Code's themes; built 2026-10-12; 2026-10-14 evening: Sync with computer the default (A94, built 2026-10-14), a free cluster colour (A93, built 2026-10-15) |
| [D-23](#d-23-a-link-to-the-owner-in-the-status-bar) | A link to the owner in the status bar | decided (owner, 2026-10-12); built 2026-10-12 |
| [D-24](#d-24-logo) | Logo | decided (owner, 2026-10-12): the C helm; built 2026-10-12; revised (owner, 2026-10-15): the glowing C helm everywhere, every version kept drawable; built 2026-10-15 |
| [D-25](#d-25-copying-values) | Copying values | decided (owner, 2026-10-12); built 2026-10-12 and 2026-10-13; 2026-10-14 evening: the copy icon over a value's end in the lists and the drawer (A69, A71), built 2026-10-15; 2026-10-16: a copy icon over a Secret's editor in the drawer (D-59), built |
| [D-26](#d-26-binding-a-credential-plugins-approval-to-its-program) | Binding a credential plugin's approval to its program | decided (owner, 2026-10-12): the program's content where a folder's writer can replace it, else its path; built 2026-10-12; 2026-10-14 evening: Cogpit's own proxy for the cluster bound too (A23), built 2026-10-14 |
| [D-27](#d-27-an-agents-helm-values-in-the-confirmation) | An agent's Helm values in the confirmation | decided (owner, 2026-10-12): shown in the dialog before the user answers; built 2026-10-12 |
| [D-28](#d-28-the-default-kubeconfig-as-sources-as-lens-has-it) | The default kubeconfig as sources, as Lens has it | decided (owner, 2026-10-12); built 2026-10-12; 2026-10-14 evening: Remove from Cogpit on a sync folder's context (A97), built 2026-10-15 |
| [D-29](#d-29-helm-chart-repositories-from-helms-repositoriesyaml) | Helm chart repositories from Helm's `repositories.yaml` | decided (owner, 2026-10-13): read, never written, credentials included; built 2026-10-13 and 2026-10-14; 2026-10-14 evening: http and insecure origins marked (A14, built 2026-10-15), dotted repository names (A36, built 2026-10-14), the lead's details for the owner to confirm |
| [D-30](#d-30-certificates-in-secrets-decoded) | Certificates in Secrets, decoded | decided (owner, 2026-10-13); built 2026-10-13; 2026-10-14 evening: combined PEMs and encrypted keys (A99, A100), built 2026-10-14; the kind of CA (R1), built 2026-10-14 night as the lead's design for the owner to confirm |
| [D-31](#d-31-every-timestamp-in-the-local-time-zone-and-format) | Every timestamp in the local time zone and format | decided (owner, 2026-10-13); built 2026-10-13; the format as a setting (the owner's "later") not built |
| [D-32](#d-32-every-byte-amount-in-binary-units-with-one-decimal) | Every byte amount in binary units with one decimal | decided (owner, 2026-10-13); built 2026-10-13 |
| [D-33](#d-33-the-flux-actions) | The Flux actions | decided (owner, 2026-10-14): Reconcile, Reconcile with Source, Reset, Suspend and Resume as the flux CLI patches them, mutating, on objects and selections; Force an agent's parameter; built 2026-10-13 and 2026-10-14; 2026-10-14 evening: Reconcile with Source waits for the source (built 2026-10-15, the night run, A91), the confirmation names the controller's identity (built 2026-10-14 night, A34) |
| [D-34](#d-34-the-flux-dashboard) | The Flux dashboard | decided (owner, 2026-10-14): five states after kstatus, counts per kind and the objects that are not ready (built 2026-10-13); a state opens its kind's view filtered, through a status filter of every view (option a), built 2026-10-14; the filter's rules confirmed by the owner (2026-10-14 evening); the button in the accent colour while a filter is active, built 2026-10-14 night (A80) |
| [D-35](#d-35-what-argo-cds-repository-and-cluster-secrets-show) | What Argo CD's repository and cluster Secrets show | decided (owner, 2026-10-14): an allowlist of non-credential keys for the window and agents (option b), built 2026-10-13; the labelled Secrets kept whole in the cache, confirmed 2026-10-14 evening |
| [D-36](#d-36-how-argo-cds-sync-writes-the-operation) | How Argo CD's Sync writes the operation | decided (owner, 2026-10-14): what `argocd app sync` writes, bound to the UID and resourceVersion (option b), built 2026-10-13; 2026-10-14 evening: the AppProject's sync windows respected, built 2026-10-14 night (A33); checked against a live Argo CD 3.5.5 controller 2026-10-16, the windows' check, the project's namespace and the operation's source brought to Argo CD's (the lead's decisions for the owner to confirm) |
| [D-37](#d-37-the-node-shell) | The node shell | decided (owner, 2026-10-14): Lens's privileged pod with nsenter (option b), built 2026-10-13; the default image the newest supported Alpine release pinned by digest and kept current by Renovate, the maximum duration a setting (default 2 hours), both built 2026-10-14; its bounds, 5 minutes to 24 hours, confirmed 2026-10-14 evening |
| [D-38](#d-38-who-sets-the-node-shells-image) | Who sets the node shell's image | decided (owner, 2026-10-14): the user only, agents may reset it (option b), the maximum duration too; built 2026-10-13 and 2026-10-14 |
| [D-39](#d-39-pod-metrics-for-a-user-who-may-list-pods-in-some-namespaces-only) | Pod metrics for a user who may list pods in some namespaces only | decided (owner, 2026-10-14): one list per namespace the Pods view reads (option b); a namespace's refusal affects that namespace only; built 2026-10-13 and 2026-10-14; the Overview's Pods history and a refused namespace's drawer history confirmed as built, 2026-10-14 evening |
| [D-40](#d-40-a-pods-table-in-the-workloads-drawers) | A Pods table in the workloads' drawers | decided (owner, 2026-10-14): the node drawer's table in the drawers of the six workload kinds (option b), without its Namespace column; built 2026-10-13 and 2026-10-14 |
| [D-41](#d-41-what-survives-a-restart) | What survives a restart: the tabs and the tables' layout | decided (owner, 2026-10-14): Cogpit's own `workspace.json`, restored without connecting anything; built 2026-10-13; 2026-10-14 evening: the navigator's collapsed source groups kept too, built 2026-10-15 (the night run, A63); **changed by the owner's decision of 2026-10-16 (D-69 item 8), built 2026-10-16:** the tabs come back only while Preferences › App › Restore the last session is on (off by default), and then the clusters connected at quit connect again; the window's arrangement comes back either way |
| [D-42](#d-42-the-start-screen) | The start screen: the welcome page, no context picker by itself | decided (owner, 2026-10-14): no picker at start (option b); built 2026-10-13; the picker gone since D-57; since 2026-10-16 (D-69 item 8) the start shows the last run's tabs only with the restore preference on |
| [D-42](#d-42-the-start-screen) | The start screen: the welcome page, no context picker by itself | decided (owner, 2026-10-14): no picker at start (option b); built 2026-10-13 |
| [D-43](#d-43-where-the-nodes-disk-comes-from) | Where the nodes' disk comes from | decided (owner, 2026-10-14): node-exporter's root filesystem through Prometheus, never the kubelet's nodes/proxy (option b), built 2026-10-14; 2026-10-14 evening: the nodes' CPU and memory history from node-exporter where present (A43), built 2026-10-14 night with the lead's choices for the owner to confirm; a node's network and filesystem the same way, built 2026-10-16 night, the lead's choices confirmed by the owner (audit of 2026-10-16) |
| [D-44](#d-44-what-agents-read-of-the-docks-logs-and-terminal-tabs) | What agents read of the dock's logs and terminal tabs | decided (owner, 2026-10-14): logs freely, a terminal tab only once the user turned on its switch (option b), built 2026-10-13; 2026-10-14 evening: the switch ends 1 hour after it was turned on at the latest, the filter's minimum 10 bytes, a chart's README framed as third-party text, built 2026-10-14 night (A3, A5, A31) |
| [D-46](#d-46-chart-icons-from-the-hosts-that-serve-them) | Chart icons from the hosts that serve them | decided (owner, 2026-10-14): fetched when shown, https only, no credentials, bounded and checked, with Preferences › Load chart icons on by default (option b); built 2026-10-14 |
| [D-47](#d-47-deep-links-cogpit) | Deep links (`cogpit://`) | decided (owner): a link that names another cluster than the connected one asks once before switching (2026-10-14 evening, built 2026-10-14); the rest the lead's decision of 2026-10-14, confirmed (owner, 2026-10-15): navigation only, through the `ui.*` commands with origin `deeplink`; a write unreachable by construction; never a context whose login waits for approval (option a); the window raised, a toast and an audit line per link; an ask before every connect; built 2026-10-14 |
| [D-48](#d-48-start-at-login) | Start at login | decided (owner, 2026-10-14 evening, as built): the macOS app's own login item (SMAppService), read from macOS, off by default, only the user turns it on (option a); built 2026-10-14 |
| [D-49](#d-49-the-external-mcp-servers-switch-honoured-by-cogpit-cli-mcp) | The external MCP server's switch, honoured by `cogpit-cli mcp` | decided (owner, 2026-10-14 evening): persisted, and with MCP off the bridge refuses to serve, headless too; built 2026-10-14 |
| [D-50](#d-50-kubectls-signature-check) | kubectl's signature check | decided (owner, 2026-10-14 evening): sigstore-go with Rekor's transparency-log proof instead of the standard library; built 2026-10-15 (night run), the lead's decisions for the owner to confirm listed |
| [D-51](#d-51-the-navigators-and-the-command-palettes-state) | The navigator's and the command palette's state | decided (owner, 2026-10-14 evening): the navigator's search text and keyboard selection and the palette's typed text are engine state; built 2026-10-15, with the lead's night decisions: an agent's palette text never running a change on ↵ confirmed (owner, 2026-10-15 evening, as built); the fields, the row's form and the bounds not asked, standing as built; one per window and what `workspace.json` keeps for the owner to confirm; the dropdown's bank moot (owner, 2026-10-15 evening: the scope dropdown goes; gone since D-57) |
| [D-52](#d-52-several-windows) | Several windows | postponed (owner, 2026-10-15): designed 2026-10-15; the owner chose the model (a) as the direction and answers the rest in a separate, larger interview once the rest is done (the owner's requirement, 2026-10-14 evening: a hotbar tile dragged out of the window opens another window with that cluster; the lead's proposal: one session, a window's state and a connection per window, seven questions in order); not built |
| [D-53](#d-53-the-mcp-details-and-a-token-kept-until-the-user-regenerates-it) | The MCP details, and a token kept until the user regenerates it | decided (owner, 2026-10-15, F6): the top bar's MCP button gone, a gear in the status bar's MCP item for the details and Regenerate token (with a warning first), the token and the port kept across starts; built 2026-10-15, the lead's decisions for the owner to confirm listed |
| [D-54](#d-54-the-layout-toggles-and-the-update-button) | The layout toggles and the Update button | decided (owner, 2026-10-15, F7): three toggles at the top bar's right end for the navigator (⌘B), the dock (⌘J) and the chat column (⌘I) as engine state, kept in `workspace.json`; an Update button while Sparkle holds an update, "Update to v<version>" since 2026-10-16 (F11); built 2026-10-15, the lead's decisions for the owner to confirm listed |
| [D-55](#d-55-a-config-maps-values-edited-in-the-drawer) | A Config Map's values edited in the drawer | decided (owner, 2026-10-15, his finding F3): the key above its value, a Config Map's in an editor, names without spaces wrap at - . _ /, values saved in the drawer through the existing write path, a Secret's layout too; built 2026-10-15 with the lead's decisions for the owner to confirm; the owner's audit answers of 2026-10-15: Save rebases by itself when no edited key changed (shown to the user), unsaved values kept as disk drafts, built the same night, the lead's decisions confirmed by the owner (audit of 2026-10-16); a Secret's values in the drawer, the window's only: [D-59](#d-59-a-secrets-values-edited-in-the-drawer); the owner's audit decision of 2026-10-16: keys added and removed in the drawer too (a removal destructive, the review its confirmation), built the same day with the lead's decisions for the owner to confirm |
| [D-56](#d-56-the-terminals-scrollback) | The terminal's scrollback | decided (owner, 2026-10-15): no line limit, a compact scrollback bounded at 256 MiB, the view and a draggable bar, reflow, the wheel to full-screen programs, agents by stable line numbers; built 2026-10-15 with the lead's decisions for the owner to confirm; fixes and this model to be offered to #147 |
| [D-57](#d-57-choosing-a-cluster) | Choosing a cluster | decided (owner, 2026-10-15 evening): one list, the navigator's section; the context picker, the top bar's "Switch context" and the header's scope dropdown gone, ⌘P and Connect to Cluster… show the section with its search focused, Escape on its empty search goes back, a "+" adds kubeconfigs; built 2026-10-15 (night run); the lead's decisions confirmed by the owner (audit of 2026-10-16) but the registry's menu path, which he changed (every cluster and kubeconfig command under the menu path `Cluster/`, built the same day, N121), and the welcome page's button, which he wants changed (how, not said yet) |
| [D-58](#d-58-standby-and-lost-connections) | Standby and lost connections | decided (owner, 2026-10-15 evening, S1–S8): reproduce the hang, detect by wake, heartbeat and broken watch, a banner (S3 amended 2026-10-16, F13: the dot alone, blinking green while reconnecting), writes disabled, reconnect with backoff without end, "Login expired" after one automatic renewal, streams by their kind, operations as after a crash; built 2026-10-15/16 (the hang not reproduced offscreen; a waiting credential plugin found to stall every request), the build's decisions confirmed by the owner (audit of 2026-10-16), the lead's defaults for the owner to confirm; the selection bar's greyed Delete names the reason in its tooltip (the owner, audit of 2026-10-16), built the same day |
| [D-59](#d-59-a-secrets-values-edited-in-the-drawer) | A Secret's values edited in the drawer | decided (owner, 2026-10-15, his audit answers): editable after Show as a Config Map's, in the window only (GUI-only, no path an agent or the CLI reaches), a masked diff and a dry run, never on disk; built 2026-10-15 (night run); the lead's decisions confirmed by the owner (audit of 2026-10-16) but adding and removing keys and the copy from the editor, which he changed: keys added and removed in the drawer too, as a Config Map's (D-55), the window's only, built the same day; the copy an icon over the editor, built the same day; H-103 opened |
| [D-61](#d-61-cogpit-metrics-the-bundled-metrics-stack) | Cogpit Metrics: the bundled metrics stack (Lens Metrics) | withdrawn by the owner (2026-10-16) and removed the same day: Cogpit installs nothing into a cluster for its own features, it reads what the cluster runs; the night's stack (built 2026-10-16, in v0.16.0 and v0.17.0) is gone, H-105 closed; its improvements to reading an installed Prometheus stay (D-43, architecture §9) |
| [D-62](#d-62-checks-against-a-real-cluster) | Checks against a real cluster | the lead's decisions (night run, 2026-10-16), confirmed by the owner (audit of 2026-10-16): the `TestReal*` tests in the kind-tagged `real_kind_test.go`, never in CI, acting only on their own namespaces and on what a variable names; P11 counts reads of objects; an OCI HelmRepository without a status Ready; events and owners link custom kinds through the definitions; a failed drain says what it left; confirmations name kinds, not kind keys; built |
| [D-63](#d-63-long-commands-as-background-jobs) | Long commands as background jobs | built 2026-10-16 (night run), the lead's decisions confirmed by the owner (audit of 2026-10-16) but six details it did not ask, named in the section: an action on a selection of more than ten objects runs as a job with progress in the dock and Cancel (`job.cancel`, any origin), ten and fewer within the call; the bound of 500 kept; Drain built as a job the same night (a step per pod, the budget's wait in it, Cancel leaves the drain `stopped` in its journal, offered as after a crash; nothing uncordoned by itself) |
| [D-64](#d-64-envtest-a-real-api-server-in-ci-without-docker) | envtest, a real API server in CI without Docker | the lead's decision (2026-10-16, G1), confirmed by the owner (audit of 2026-10-16) but four details it did not ask, named in the section: controller-runtime's envtest, etcd and kube-apiserver 1.37.0 pinned by SHA-512, the API server stopped with SIGSTOP for D-58's standby case; built 2026-10-16 (night run), the CI job not observed yet |
| [D-69](#d-69-several-clusters-connected-at-once) | Several clusters connected at once | decided by the owner (2026-10-16 interview, F16): several connections open, one shown per window, switching without reconnecting; a left cluster warm 5 minutes, then slim; no limit; background logins renewed by default, configurable per cluster, a lapsed one not red; filled, blinking, ring and red marks; agents with a context parameter; the dock per cluster; restoring the last session a preference, off by default. Stage 1 (the engine) built 2026-10-16 with the lead's decisions for the owner to confirm: the pool, a connect of a connected context only shows it, cold after the warm period, every command on its own cluster, a `context` parameter, per-cluster connection watch and write gate, `UIState.Connections`. Stage 2 (the window) built 2026-10-16 with the lead's decisions for the owner to confirm: switching without reconnecting at every entry point, the marks on every dot (filled, blinking, green ring, grey ring, red), each cluster blinking for itself, the dock per cluster in the engine, Disconnect per cluster in its menus, Diagnostics per cluster, a failed connect shows the cluster before again. Stage 3 (the settings of items 4 and 8) built 2026-10-16 with the lead's decisions for the owner to confirm: the credential plugin of a cluster not shown held in client-go's authenticator, `loginLapsed`, renewed when shown or by Log in again; the restore of the last session, the shown cluster first, the others one after another in the background. |
| [D-70](#d-70-the-disk-images-window) | The disk image's window | decided by the owner (2026-10-16) in a visual review: the plate background, an arrow, Cogpit's own helm wheel faint behind the Applications folder, names on grey pills for both Finder themes; the official Kubernetes logo not used (the Linux Foundation's guidelines); dmgbuild makes a template in the app's job, the signing job fills it with Apple's tools (the owner's choice); built 2026-10-16, checked locally |
| [D-66](#d-66-the-connection-log) | The connection log | decided (owner, 2026-10-16): the Connection Updates dialog extended, opened from every status dot of a cluster, live, with Copy; one owner-only file per context on disk, 7 days and 1 MiB; every step of a connect, reconnect, login and disconnect with its time and duration, a failure's error picture; a credential plugin's standard error on a failed login only, scrubbed, cut to 4 KiB; built 2026-10-16 with the lead's choices for the owner to confirm (the order, the folder, the capture through client-go's field, the scrubber's shapes); H-112 opened |
| [D-65](#d-65-discovery-cached-on-disk) | Discovery cached on disk | the lead's decision (night run, 2026-10-16), built the same night, confirmed by the owner (audit of 2026-10-16) but three details it did not ask: Cogpit's own owner-only file per server and identity under `<user cache dir>/Cogpit/discovery`, for the same server version, 6 hours once the definitions' digest confirmed it, else 10 minutes; measured on a real cluster |
| [D-67](#d-67-the-demo-cluster-stays-out-of-releases) | The demo cluster stays out of releases | decided by the owner (2026-10-16): compiled out of release builds by a tag only the release pipeline sets; built 2026-10-16 with the tag `nodemo` (`DIST_TAGS`), checked by `make dist-app`, a Go test and CI's lint job (`make check-nodemo`); about 0.3 MB less per binary; the build's choices for the owner to confirm |
| [D-68](#d-68-one-short-name-for-a-cluster-and-the-user-cogpit-is-logged-in-as) | One short name for a cluster, and the user Cogpit is logged in as | decided by the owner (2026-10-16, F17): the part of a context's name after the last "@" everywhere the window names it (a display name still wins, the full name in the tooltips), the username a SelfSubjectReview confirms at every connect in the navigator's root row with the groups in a box's tooltip, the kubeconfig's user entry marked where the API server does not confirm it, the last known user dimmed while not connected, agents read the username and the groups; built 2026-10-16, the lead's decisions for the owner to confirm |

## Decision round 2026-10-14 evening

On 2026-10-14 in the evening the owner answered a list of open questions kept locally (`plan/`,
not in the repository): 117 answers (A1–A117, each to a question #n of that list), two findings
in a real window (F1, F2) and three new requirements (R1–R3). "To build" means decided and not
built; "as built" means the owner confirmed what Cogpit does. Each line names where the answer is
recorded: a decision record on this page, a security gap (`H-n`) in [docs/security](../security/README.md),
[architecture](architecture.md), or the [parity pages](../parity/README.md) for details of the
window. Nothing of it was built in the round. The night run of 2026-10-14/15 and the day after built
every answer that asked for a build but two: R3 (designed in [D-52](#d-52-several-windows),
postponed by the owner) and A10 (a GitHub ruleset, not set up); A86's upstream offer waits for the
end by the owner's own answer. Each line says its state.

<details>
<summary>The answers, one line each (A1–A117, F1, F2, R1–R3 and the notes)</summary>

| Answer | Asked | What the owner decided | Recorded in |
|---|---|---|---|
| A1 | H-27 | Every RBAC object an agent creates or changes (Roles, ClusterRoles and their bindings) waits for the user, also where agent writes are allowed; built 2026-10-14 | [D-2](#d-2-agent-interface-scope), [H-27](../security/privilege-footprint.md#h-27-a-binding-is-judged-by-its-roles-name-and-the-cache) |
| A2 | Helm | An agent's Helm install or upgrade always waits for the user, whatever the chart holds; built 2026-10-14 | [D-2](#d-2-agent-interface-scope) |
| A3 | H-83 | A terminal tab's agent-read switch ends at the latest 1 hour after it was turned on, whatever the reads; the tab says so; one click turns it on again; built 2026-10-14 | [D-44](#d-44-what-agents-read-of-the-docks-logs-and-terminal-tabs), [H-83](../security/agents-and-mcp.md#h-83-an-allowed-terminal-tab-gives-agents-everything-it-shows-from-then-on) |
| A4 | H-82 | Values under naming keys (user, host, port, database, namespace, …) are not replaced; as built | [D-44](#d-44-what-agents-read-of-the-docks-logs-and-terminal-tabs), [H-82](../security/agents-and-mcp.md#h-82-secret-values-printed-into-logs-reach-agents) |
| A5 | H-82 | The filter's minimum 10 bytes instead of 8; built 2026-10-14 | [D-44](#d-44-what-agents-read-of-the-docks-logs-and-terminal-tabs), [H-82](../security/agents-and-mcp.md#h-82-secret-values-printed-into-logs-reach-agents) |
| A6 | H-40 | An allowance of credentials over http does not expire; bound to name, URL and user; as built | [D-29](#d-29-helm-chart-repositories-from-helms-repositoriesyaml), [H-40](../security/credentials.md#h-40-credentials-over-plain-http) |
| A7 | D-47, H-89 | A `cogpit://` link that names another cluster than the connected one asks once, naming the context, before switching; within the connected cluster it stays immediate; built 2026-10-14 | [D-47](#d-47-deep-links-cogpit), [H-89](../security/trust-boundaries.md#h-89-a-link-switches-the-cluster-and-the-view-under-the-user) |
| A8 | D-48 | Start at login off by default, only the user turns it on; as built | [D-48](#d-48-start-at-login) |
| – | D-4 | Agent REL's "the disk image replaces the zip" and "`cogpit-cli` signed and notarized, not stapled": moot, only the disk image ships, the CLI inside `Cogpit.app` | [D-4](#d-4-packaging-and-updates), [D-16](#d-16-the-first-real-release) |
| A9 | D-4 | A failed signature or notarization fails the job; no files until it is re-run; Apple gets 45 minutes; as built | [D-4](#d-4-packaging-and-updates) |
| A10 | D-4 | A repository ruleset: only the semantic-release GitHub App (bypass) and the owner create `v*` tags; the environment `release`'s admin bypass off; GitHub settings, to set with the owner's OK. Decided further (owner, 2026-10-15 afternoon): the App the ruleset's only bypass, nobody tags `v*` by hand, an emergency tag by an admin switching the ruleset off for the moment; not set up (verified 2026-10-15: no ruleset, `release`'s admin bypass on) | [D-4](#d-4-packaging-and-updates) |
| A11 | H-91 | Sparkle's 20-day failsafe kept (setting it to 0 declined); as built | [D-4](#d-4-packaging-and-updates), [H-91](../security/updates.md#h-91-after-20-days-of-unverifiable-feeds-sparkle-presents-items-again) |
| A12 | D-4 | Agents and the CLI may turn the automatic update checks on, never off; the user both ways; as built. SPARKLE's technical defaults not asked: they stand as built | [D-4](#d-4-packaging-and-updates), [D-2](#d-2-agent-interface-scope) |
| A13 | #3, H-42 | A repository that appears in Helm's file while Cogpit runs is downloaded at once; as built | [D-29](#d-29-helm-chart-repositories-from-helms-repositoriesyaml), [H-42](../security/trust-boundaries.md#h-42-a-repository-another-program-writes-into-helms-file-is-used-without-asking) |
| A14 | #4, H-41 | An http or insecure-TLS origin marked in the Charts view (the repository cell), the Install tab and the install and upgrade confirmations; built 2026-10-15 (the window's own install confirmed in its preview, its upgrade in the Upgrade tab: the lead's decisions for the owner to confirm, see D-29) | [D-29](#d-29-helm-chart-repositories-from-helms-repositoriesyaml), [H-41](../security/trust-boundaries.md#h-41-repositories-whose-content-nothing-authenticates) |
| A15 | #31, H-28 | The MCP off switch persisted and honoured by `cogpit-cli mcp`: off, the bridge refuses to serve, headless too, with a clear message to the agent; built 2026-10-14 | [D-49](#d-49-the-external-mcp-servers-switch-honoured-by-cogpit-cli-mcp), [D-23](#d-23-a-link-to-the-owner-in-the-status-bar), [H-28](../security/agents-and-mcp.md#h-28-the-status-bars-switch-does-not-reach-a-headless-bridge) |
| A16 | #40, H-26 | `cluster.setPrometheus` by an agent a write with confirmation (mutating); the window as today; built 2026-10-14 (always asks: N8) | [D-2](#d-2-agent-interface-scope), [H-26](../security/privilege-footprint.md#h-26-prometheus-is-found-by-name) |
| A17 | #45, D-11 | Nothing offered after an interrupted Helm install; Cogpit shows the release's state; the user cleans up in the Helm Releases view; built 2026-10-14 (the offer removed, Show release in its place) | [D-11](#d-11-crash-stance) |
| A18 | #81, H-32 | Marking notifications read and clearing them GUI-only; agents read the list; built 2026-10-14 | [D-2](#d-2-agent-interface-scope), [H-32](../security/agents-and-mcp.md#h-32-an-agent-can-clear-the-users-notifications) |
| A19 | #82, H-31 | Window-only notifications reach agents as text, through the H-82 filter and with inline Helm values removed, instead of withheld; built 2026-10-14 | [D-2](#d-2-agent-interface-scope), [H-31](../security/privilege-footprint.md#h-31-an-agents-helm-values-can-copy-what-lookup-reads), [architecture §4.3](architecture.md#43-ui-state) |
| A20 | #87 | Agents may set a proxy, as a write with confirmation, the password redacted in the confirmation, the audit log and the results; clearing as today; built 2026-10-14 (always asks: a night decision) | [D-2](#d-2-agent-interface-scope), [architecture §4.3](architecture.md#43-ui-state) |
| A21 | #89, H-33 | Proxy passwords stay in the 0600 settings files; as built | [H-33](../security/credentials.md#h-33-proxy-passwords-are-plaintext-in-cogpits-settings) |
| A22 | #90, H-33 | No warning for an `http://` proxy with a password; as built | [H-33](../security/credentials.md#h-33-proxy-passwords-are-plaintext-in-cogpits-settings) |
| A23 | #91, H-34 | A credential plugin's approval shows and binds Cogpit's own proxy for the cluster too; a changed proxy asks again; built 2026-10-14 (bound as shown, the password as •••: a night decision) | [D-26](#d-26-binding-a-credential-plugins-approval-to-its-program), [H-34](../security/trust-boundaries.md#h-34-the-plugin-approval-shows-the-kubeconfigs-proxy-not-cogpits) |
| A24 | #93 | The terminal's right-click copy and paste preference GUI-only; as built | [D-14](#d-14-terminal-widget) |
| A25 | #97, H-35 | kubectl's signature checked with sigstore-go, the Rekor transparency-log proof included; a record of its own; built 2026-10-15 | [D-50](#d-50-kubectls-signature-check), [H-35](../security/trust-boundaries.md#h-35-kubectls-signature-check-rests-on-an-embedded-trusted-root-and-rekor-v1s-search) |
| A26 | #98, H-36 | The kubectl `PATH` hand-over for bash (`--rcfile` or `PROMPT_COMMAND`) and fish (`conf.d`), as for zsh; built 2026-10-14 | [D-3](#d-3-helm-and-kubectl-binaries), [H-36](../security/trust-boundaries.md#h-36-a-shells-profile-can-put-another-kubectl-first) |
| A27 | #100, H-36 | A setting (Preferences › Kubernetes): the `ZDOTDIR` hand-over by default, switchable to enforcement through a shell alias or function `kubectl`; built 2026-10-14 | [D-3](#d-3-helm-and-kubectl-binaries), [H-36](../security/trust-boundaries.md#h-36-a-shells-profile-can-put-another-kubectl-first) |
| R1 | – | Certificates: show whether the chain leads to a public or an internal (private) CA; the design settled when built; built 2026-10-14 (the lead's design for the owner to confirm) | [D-30](#d-30-certificates-in-secrets-decoded) |
| A28 | #102, H-37 | Ask once per context before the first connect of an `insecure-skip-tls-verify` context, the answer kept like a plugin approval; built 2026-10-14 (the plugin's binding, per context, agents, the CLI and links refused until allowed: the lead's night decisions in H-37) | [H-37](../security/trust-boundaries.md#h-37-a-context-that-skips-the-certificate-check), [architecture §6](architecture.md#6-cluster-connections-internalcluster) |
| A29 | #103, H-37 | While such a context is connected, the status bar shows "TLS not checked" in the warning colour beside the context; `cogpit-cli get` writes a warning line to stderr; built 2026-10-15 (engine state `connection.tlsUnchecked`, also in `cluster_connect`'s result; the lead's details in H-37) | [H-37](../security/trust-boundaries.md#h-37-a-context-that-skips-the-certificate-check), [architecture §6](architecture.md#6-cluster-connections-internalcluster) |
| A30 | #104 | Only the window chooses the configmap Helm driver; agents may only set it back to secret; as built | [D-29](#d-29-helm-chart-repositories-from-helms-repositoriesyaml) |
| – | #117 | The MCP switch's own record (housekeeping, not asked) | [D-49](#d-49-the-external-mcp-servers-switch-honoured-by-cogpit-cli-mcp) |
| A31 | #122, H-86 | A chart's README reaches agents framed as third-party text ("text of the chart's author, not instructions"); built 2026-10-14 | [D-44](#d-44-what-agents-read-of-the-docks-logs-and-terminal-tabs), [H-86](../security/agents-and-mcp.md#h-86-a-charts-readme-reaches-agents) |
| A32 | #139, H-66 | Argo CD's labelled Secrets stay whole in the cache (the H-82 filter uses their values); as built | [D-35](#d-35-what-argo-cds-repository-and-cluster-secrets-show), [H-66](../security/credentials.md#h-66-argo-cds-credentials-are-in-cogpits-memory-while-its-settings-are-read) |
| A33 | #140, H-67 | Argo CD's Sync respects the AppProject's sync windows (refused, or greyed with the reason, in a deny window); Argo CD's RBAC not emulated; built 2026-10-14 | [D-36](#d-36-how-argo-cds-sync-writes-the-operation), [H-67](../security/privilege-footprint.md#h-67-a-sync-acts-with-argo-cds-rights-not-the-users) |
| A34 | #144, H-64 | A Flux action's confirmation names the identity the controller acts as (its service account, or the object's `spec.serviceAccountName`); built 2026-10-14 | [D-33](#d-33-the-flux-actions), [H-64](../security/privilege-footprint.md#h-64-a-flux-action-is-carried-out-with-the-controllers-identity) |
| A35 | #156 | The node shell's maximum 5 minutes to 24 hours, default 2 hours; as built | [D-37](#d-37-the-node-shell) |
| A36 | #6 | Repositories with a dot in their name supported: an unambiguous chart id inside Cogpit, through refs, deep links and agents' parameters, saved ids migrated once; built 2026-10-14 (the id doubles a dot of the repository's name; nothing saved needed migrating; the form the lead's decision for the owner to confirm) | [D-29](#d-29-helm-chart-repositories-from-helms-repositoriesyaml) |
| A37 | #46 | Install shows the preview first; as built | [D-29](#d-29-helm-chart-repositories-from-helms-repositoriesyaml) |
| A38 | #64 | An upgrade to the release's own version uses the stored chart; as built | [D-29](#d-29-helm-chart-repositories-from-helms-repositoriesyaml) |
| A39 | #105 | A release name in both drivers: one release, the one deployed last, with a drawer note "also stored in the `<other>` driver (older)"; built 2026-10-14 (largely built before, the wording finished, see D-29) | [D-29](#d-29-helm-chart-repositories-from-helms-repositoriesyaml) |
| A40 | #106 | Helm's sql driver out of scope; as built | [D-29](#d-29-helm-chart-repositories-from-helms-repositoriesyaml) |
| A41 | #38 | The network and filesystem series asked for only while their chart shows; as built | [architecture §9](architecture.md#9-metrics-internalmetrics) |
| A42 | #39 | A failed metrics provider retried after 1 minute; as built | [architecture §9](architecture.md#9-metrics-internalmetrics) |
| A43 | #41 | Node CPU and memory history from node-exporter when present (kube-prometheus-stack), cAdvisor otherwise; built 2026-10-14 night | [D-43](#d-43-where-the-nodes-disk-comes-from), [architecture §9](architecture.md#9-metrics-internalmetrics) |
| F1 | – | The owner's finding in a real window: no node history is shown; take the history from Prometheus where it is available; built 2026-10-14 night (A43; the cause in architecture §9) | [D-43](#d-43-where-the-nodes-disk-comes-from), [architecture §9](architecture.md#9-metrics-internalmetrics) |
| A44 | #42 | A provider's status from its last answers, no probe; as built | [architecture §9](architecture.md#9-metrics-internalmetrics) |
| A45 | #54 | Chart ranges up to 30 days; as built | [architecture §9](architecture.md#9-metrics-internalmetrics) |
| A46 | #114 | Charts of one object whose ranges lie far apart get separate Prometheus queries, each at its own step; built 2026-10-14 night (the lead's choices for the owner to confirm in architecture §9) | [architecture §9](architecture.md#9-metrics-internalmetrics) |
| A47 | #120 | The cluster Overview's Pods history counts the whole cluster, for a namespace-limited user too; as built | [D-39](#d-39-pod-metrics-for-a-user-who-may-list-pods-in-some-namespaces-only) |
| A48 | #162 | A pod drawer in a refused namespace still shows Prometheus's history; as built | [D-39](#d-39-pod-metrics-for-a-user-who-may-list-pods-in-some-namespaces-only) |
| A49 | #11 | `UIState.Context` forgotten when its cluster is removed and no connection holds it; as built | [architecture §6](architecture.md#6-cluster-connections-internalcluster) |
| A50 | #12 | A file that leaves a sync folder closes its tabs, with a notice; as built | [architecture §6](architecture.md#6-cluster-connections-internalcluster) |
| A51 | #27 | The navigator's search text and keyboard selection engine state, visible and settable for agents; built 2026-10-15 (`UIState.Navigator`, `ui.setNavigatorSearch`, `ui.setNavigatorRow`; the fields' form a night decision) | [D-51](#d-51-the-navigators-and-the-command-palettes-state), [architecture §4.3](architecture.md#43-ui-state) |
| – | FLAKE | Noted from reading only: a reopened log stream names pod and container, not the UID, so a StatefulSet pod recreated under its name while the cache lags more than 500 ms may show lines twice; not reproduced, to check in a later run | this table |
| A52 | #29 | The navigator's scope dropdown lists the hotbar's and the connected clusters only, every other context through the picker (⌘P); built 2026-10-15 (the shown bank's tiles, a night decision; All Clusters… ⌘P at its end); the bank moot since the owner decided on 2026-10-15 evening that the scope dropdown goes, gone since (D-57) | [D-51](#d-51-the-navigators-and-the-command-palettes-state) |
| – | #86 | Capping the scope dropdown: moot by A52, the dropdown gone since D-57 | [D-51](#d-51-the-navigators-and-the-command-palettes-state) |
| A53 | #30 | The navigator's focus ring on keyboard navigation only, not after a mouse click; built 2026-10-15 | [D-51](#d-51-the-navigators-and-the-command-palettes-state) |
| A54 | #34 | The command palette's typed text engine state, as A51; built 2026-10-15 (`UIState.PaletteText`, `ui.palette.setText`; on an agent's text ↵ runs no change until the user picked the entry, a night decision, confirmed by the owner on 2026-10-15 evening) | [D-51](#d-51-the-navigators-and-the-command-palettes-state), [architecture §4.3](architecture.md#43-ui-state) |
| A55 | #35 | ⌘F and ⌘⇧T as main-menu items with their shortcuts shown; the keys stay window shortcuts (the editor keeps its ⌘F); built 2026-10-15 (View › Search, View › New Terminal) | [architecture §4.2](architecture.md#42-the-command-registry), [parity](../parity/README.md) |
| A56 | #36 | GLFW's Window menu is there (verified by the owner in a real window); resolved | [parity](../parity/README.md) |
| A57 | #37 | "New Bank…" and "New Hotbar Bank"; as built | [parity](../parity/README.md) |
| A58 | #61 | On close, the nearest tab of the same cluster; as built | [parity](../parity/README.md) |
| A59 | #73 | No onboarding wizard (not a target), the welcome page covers it; as built | [parity](../parity/README.md) |
| A60 | #74 | One About entry, in the app menu (Cogpit › About Cogpit), showing Cogpit's own dialog; the Help entry goes; built 2026-10-14 (GLFW's item pointed at Cogpit's dialog; on Windows and Linux, without an app menu, About stays at the end of Help) | [parity](../parity/README.md) |
| F2 | – | The owner's finding: the About dialog shows version 0.0.1 and is laid out confusingly; clean it up (where the version comes from: a local `make app` build, `CFBundleShortVersionString` against `main.version`; a dev build should say so); built 2026-10-14: the dialog read Fyne's default version 0.0.1 in every build, now the binary's own ([D-16](#d-16-the-first-real-release), the version a build reports) | [parity](../parity/README.md) |
| A61 | #75 | Disconnect closes the cluster's tabs too; built 2026-10-15 | [architecture §4.3](architecture.md#43-ui-state), [parity](../parity/README.md) |
| A62 | #83, #84 | Every connect drills into the cluster, a tab click and the connected cluster's hotbar tile too; as built | [D-51](#d-51-the-navigators-and-the-command-palettes-state) |
| A63 | #85 | The Local Kubeconfigs section's source groups collapse (click, arrow keys), kept across restarts; built 2026-10-15 (`ui.setNavigatorGroup`, `workspace.json`) | [D-41](#d-41-what-survives-a-restart), [D-51](#d-51-the-navigators-and-the-command-palettes-state) |
| A64 | #109 | F1–F12 work as the hotbar in plain search fields too; the terminal and the editor keep their F-keys; built 2026-10-15 | [architecture §4.3](architecture.md#43-ui-state), [parity](../parity/README.md) |
| A65 | #110 | An agent removing a hotbar bank with tiles waits for the user; built 2026-10-14 | [D-2](#d-2-agent-interface-scope) |
| A66 | #113 | The "+" below the tiles; as built | [parity](../parity/README.md) |
| R2 | – | Hotbar tiles reordered by drag and drop; built 2026-10-15 (`hotbar.moveCluster`; the line in the gap, the tile last when dropped on the "+", a slipped click opens: the lead's decisions for the owner to confirm) | [parity](../parity/README.md), [D-52](#d-52-several-windows) |
| R3 | – | Dragging a hotbar tile out of the window opens another Cogpit window with that cluster; designed 2026-10-15; postponed by the owner on 2026-10-15 until the rest is done (the model (a) as the direction, the rest in a separate interview) | [D-52](#d-52-several-windows) |
| A67 | #133 | The welcome page's "This session" heading becomes "Recent"; built 2026-10-15 | [parity](../parity/README.md) |
| A68 | #8 | A list copy shows the checkmark and the toast; as built | [D-25](#d-25-copying-values) |
| A69 | #10 | The lists' copy icon lies over the cell's end; the text is not shortened for it; built 2026-10-15: the cell keeps its text, the icon's plate covers its end | [D-25](#d-25-copying-values) |
| A70 | #14, #15 | The copy icon's exclusion list as built, custom resource columns included; as built | [D-25](#d-25-copying-values) |
| A71 | #21 | The drawer's Forward… right after the value as before; the copy icon over the value's end on hover; built 2026-10-15: a value with buttons keeps no room for its icon, the text is not shortened | [D-25](#d-25-copying-values) |
| A72 | #76 | Delete of a selection at most 500 objects; as built | [architecture §4.2](architecture.md#42-the-command-registry) |
| A73 | #77 | A plain click on a row opens its drawer and keeps the checked rows checked; built 2026-10-15: `ui.select`'s mode `open`, the drawer's row highlighted with its checkbox as it was | [architecture §4.3](architecture.md#43-ui-state), [parity](../parity/README.md) |
| A74 | #78 | Restart on a selection (workloads); Scale stays single-object; built 2026-10-14 (`workload.restartMany`) | [architecture §4.2](architecture.md#42-the-command-registry), [parity](../parity/README.md) |
| A75 | #80 | The name column's minimum 140 points; as built | [parity](../parity/README.md) |
| A76 | #116 | Links and chips not drag-selectable, the copy icon covers them; as built | [D-25](#d-25-copying-values) |
| A77 | #157 | A row's states from the kind's state columns, the tokens as fallback; as built | [D-34](#d-34-the-flux-dashboard) |
| A78 | #158 | AND between state columns, OR within one; as built | [D-34](#d-34-the-flux-dashboard) |
| A79 | #159 | The status filter belongs to its view and tab; as built | [D-34](#d-34-the-flux-dashboard) |
| A80 | #160 | The status filter's button in the accent colour while a filter is active; built 2026-10-14 | [D-34](#d-34-the-flux-dashboard) |
| A81 | #49 | An Application's logs in one merged tab, each line `<workload>/<pod>/<container>`, a filter per source; built 2026-10-15 for the Applications view's applications, and the same tab for Argo CD Applications and Flux Kustomizations (the lead's decisions for the owner to confirm in architecture §4.4) | [parity](../parity/README.md), [architecture §4.4](architecture.md#44-tables-details-editors-and-streams) |
| A82 | #66 | Older log lines only through the row or `logs.loadOlder`; as built | [parity](../parity/README.md) |
| A83 | #68 | Init containers listed apart; as built | [parity](../parity/README.md) |
| A84 | #69 | A logs tab's Download saves what is shown, the active filter applied; without a filter everything; built 2026-10-14 | [parity](../parity/README.md) |
| A85 | #94 | Ctrl+C always to the shell on Windows and Linux; as built | [D-14](#d-14-terminal-widget) |
| A86 | #96 | The two newer terminal fork patches offered upstream later, at the end; the app first | [D-14](#d-14-terminal-widget) |
| A87 | #99 | A verified, matching kubectl downloaded earlier goes first without asking; as built | [D-3](#d-3-helm-and-kubectl-binaries) |
| – | #169 | The logs' Jump to time field cut at 1440 points: a defect, not a decision; to check in a render and fix; checked 2026-10-14 (its placeholder was cut after "10m" in a 1440×900 render, in either format) and fixed: the field as wide as its placeholder needs | [parity](../parity/README.md) |
| A88 | #32 | `cogpit-cli get` prints Cogpit's view of a list, one object's manifest redacted; as built | [architecture › Current state](architecture.md#current-state-2026-10-16) |
| – | #33 | D-2's "cogpit mcp": a doc fix, no owner's call; D-2 reads `cogpit-cli mcp` already | [D-2](#d-2-agent-interface-scope) |
| A89 | #130 | GUI-only commands listed for agents, described and refused; as built | [D-2](#d-2-agent-interface-scope) |
| A90 | #125 | Without the right to list the definitions, Flux and Argo CD recognised from API discovery with their curated profiles; generic custom resources stay invisible; built 2026-10-14 (every curated profile, the Vertical Pod Autoscalers' too, and Flux's HelmCharts left out, having none: the lead's decisions for the owner to confirm in architecture §5) | [architecture §5](architecture.md#5-cache) |
| A91 | #161 | Reconcile with Source waits like the flux CLI: a background job until the source is ready, with a timeout, then the object; progress in the dock; built 2026-10-15 (the lead's decisions for the owner to confirm in D-33) | [D-33](#d-33-the-flux-actions) |
| A92 | #56 | The cluster's terminal namespace beats the filter's; as built | [architecture §4.3](architecture.md#43-ui-state) |
| A93 | #58 | A cluster's accent colour: Solarized's eight plus a free colour picker; built 2026-10-15 | [D-22](#d-22-colour-theme) |
| A94 | #59 | The default theme becomes Sync (follow macOS); built 2026-10-14 | [D-22](#d-22-colour-theme) |
| A95 | #115 | The light theme's accent a darkened cyan #1F7F78; as built | [D-22](#d-22-colour-theme) |
| A96 | #7, #13 | Single-target reads read the target's namespace; as built | [architecture §5](architecture.md#5-cache) |
| A97 | #71 | Remove from Cogpit on a sync-folder context offers to unlink the whole folder, the confirmation naming every context it takes; built 2026-10-15 | [D-28](#d-28-the-default-kubeconfig-as-sources-as-lens-has-it), [architecture §6](architecture.md#6-cluster-connections-internalcluster) |
| A98 | #124 | The Overviews wait for the access reviews like the dashboards, no refused cluster-wide read first; built 2026-10-14 | [architecture §5](architecture.md#5-cache) |
| A99 | #19 | A combined PEM (certificate and key in one value) gets the key check too; built 2026-10-14 | [D-30](#d-30-certificates-in-secrets-decoded) |
| A100 | #20 | An encrypted private key in the warning colour with "encrypted key: TLS/Ingress cannot use it"; built 2026-10-14 | [D-30](#d-30-certificates-in-secrets-decoded) |
| A101 | #23 | A grant's confirmation says "Grants these rights immediately, cluster-wide" or "in `<ns>`"; "This cannot be undone." only where something is lost; built 2026-10-14 | [D-2](#d-2-agent-interface-scope), [parity](../parity/README.md) |
| A102 | #24 | After a create the new object's drawer opens and the list scrolls to its row; built 2026-10-14 (the user's creates of one object; a night decision) | [parity](../parity/README.md) |
| A103 | #48 | An Application's Violations are its workloads in warning or error; as built | [parity](../parity/README.md) |
| A104 | #50 | A VPA's mode "Auto" when unset; as built | [parity](../parity/README.md) |
| A105 | #51 | The pod drawer shows its Resource Claims' state, watching the namespace's claims while it is open; built 2026-10-15 | [architecture §4.4](architecture.md#44-tables-details-editors-and-streams), [parity](../parity/README.md) |
| A106 | #52 | Connect opens Pods; as built | [parity](../parity/README.md) |
| A107 | #55 | The Issues rule as built | [parity](../parity/README.md) |
| A108 | #95 | A node device shows the claim that holds it, linked; built 2026-10-15 | [architecture §4.4](architecture.md#44-tables-details-editors-and-streams), [parity](../parity/README.md) |
| A109 | #136 | No Pods table in Replication Controllers' drawers; as built | [D-40](#d-40-a-pods-table-in-the-workloads-drawers) |
| – | #168 | Whether the CLI archive ships the README: moot, no CLI archives any more | [D-16](#d-16-the-first-real-release) |
| A110 | #164 | README images through `<picture>`: light renders for a light GitHub theme, dark for dark; built 2026-10-15: every showcase picture rendered in both themes, `<name>_dark.png` and `<name>_light.png`, the dark one the `<img>` fallback; the theme section's separate light picture dropped | [the README's pictures](#the-readmes-pictures-a110-a111-a112-a114) |
| A111 | #165 | A README banner (logo, name, one sentence), light and dark through `<picture>`; built 2026-10-15: a render of the showcase (`banner_dark.png`, `banner_light.png`) under the H1 `# Cogpit`, in place of the right-floated icon | [the README's pictures](#the-readmes-pictures-a110-a111-a112-a114) |
| A112 | #166 | README images optimised losslessly (oxipng, zopfli); built 2026-10-15: `make showcase` renders them and runs oxipng 10.2.1 (pinned by SHA-256 in `.cache/oxipng`) with `-o max --fast --zopfli --strip safe`; the 30 pictures 14,212,774 bytes as rendered, 10,560,330 after (−25.7 %, every pixel unchanged) | [the README's pictures](#the-readmes-pictures-a110-a111-a112-a114) |
| A113 | #167 | The welcome page links the README for what Cogpit does and `docs/developer/commands.md` for the commands; the README stays the showcase; no further Markdown files in the repository root; built 2026-10-15 (two links to the repository on GitHub, in the browser) | [parity](../parity/README.md) |
| A114 | #170 | One neutral demo context name in every showcase image; built 2026-10-15: `demo` connected in every picture (the Flux and Argo CD ones had `kind-flux` and `argo`), the other clusters of the many-clusters picture invented (`kind-cogpit` became `kind-local`) | [the README's pictures](#the-readmes-pictures-a110-a111-a112-a114) |
| A115 | #119 | node-exporter and kube-state-metrics on in the `hack/kind` Prometheus values; kind tests cover node and pod history from Prometheus; built 2026-10-14 night | this table (tests), with F1 |
| A116 | #153 | The terminal fork's four shell tests under `-race` in CI; built 2026-10-15 (seven races in `RunLocalShell`, its directory check and `ExitCode` patched in the fork, `TestRunLocalShell_MainLoop`; the patches upstream candidates not offered, the lead's decision for the owner to confirm) | [D-14](#d-14-terminal-widget) |
| A117 | #163 | Renders for the Deployment and StatefulSet drawers only; as built | [parity](../parity/README.md) |

</details>

### The README's pictures (A110, A111, A112, A114)

Built 2026-10-15 ([`showcase_test.go`](../../internal/ui/fyneui/showcase_test.go),
[`banner_showcase_test.go`](../../internal/ui/fyneui/banner_showcase_test.go), `make showcase`,
[`hack/showcase/oxipng.sh`](../../hack/showcase/oxipng.sh); [testing](testing.md#tiers)). The
lead's decisions for the owner to confirm, each with the options weighed:

- **File names** (A110): `<name>_dark.png` and `<name>_light.png` side by side in `docs/images`,
  rather than a directory per theme; the existing pictures renamed to `_dark` (the old
  `hero_light.png` keeps its name). One directory keeps each pair together in a listing.
- **The `<img>` fallback** (A110), what a viewer that ignores `<source>` shows: the dark render,
  rather than the light one of GitHub's default theme, because it is the look the README had and
  Cogpit's palette at its most recognisable.
- **The theme section** (A110): its picture "the same window in the light theme" dropped, its text
  kept with a line that the page's pictures follow the reader's theme. Weighed against it: dropping
  the section (the theme is a feature worth naming), and a fixed picture with both themes side by
  side (one more picture to load for what every other picture now shows).
- **The banner** (A111): a PNG the showcase renders in the app's own font on each theme's
  background, rather than an SVG, whose text would fall back to the reader's fonts. The logo is the
  welcome page's drawing without the app icon's plate, which would vanish on the dark background
  (since 2026-10-15 the tile with its plate, as everywhere: [D-24](#d-24-logo) revised);
  the sentence is the pitch's first clause, "A fast, native Kubernetes IDE for your desktop.". The
  H1 `# Cogpit` stays as text for GitHub's outline and the banner follows it, instead of the
  right-floated icon; weighed against it: the banner alone (no H1 in the outline) and the banner
  inside the H1 (an outline entry and an anchor without text).
- **oxipng** (A112): `-o max --fast --zopfli --strip safe`. zopfli takes about two minutes of one
  core per picture and saves 2.2 % over oxipng without it (measured on the hero, 2026-10-15); it is
  kept because the owner asked for it, and `--fast` gives the same size for a quarter of the CPU
  time. Never `--alpha`, which alters pixels. The release binaries for macOS on Apple silicon and
  Linux on x86-64 are pinned by version and SHA-256 and updated by hand, without a Renovate
  manager as Sparkle has: a tool that only touches the README's pictures.
- **The context name** (A114): `demo`, the context of the fixtures and the name of the demo
  cluster a reader can open in Cogpit (⌘P), rather than an invented production-like name. The Flux
  and Argo CD pictures show it too, as "Kubernetes cluster" rather than "Synthetic demo cluster":
  the built-in demo cluster runs neither. The many-clusters picture keeps four invented clusters
  (`prod-eu-central`, `staging-us-east`, `dev-us-west` and `kind-local`, formerly `kind-cogpit`,
  Cogpit's own kind test cluster): no name of a person, a company or a real cluster.

## D-1: UI toolkit

**Decided (owner, 2026-10-11): Fyne** (`fyne.io/fyne/v2` v2.8.1, BSD-3-Clause), after the spike
gate. The spike's plan and gate criteria are in [`spike-fyne.md`](spike-fyne.md); it ran on
2026-10-11 as the local MVP and its [results](spike-fyne.md#results) are recorded there: S1 passes
with a reservation (the start-up stall, tracked in G15), S2–S6 pass (S5 on a patched terminal
widget), S7 is partial, S8 is recorded.

**Re-checks** (owner, 2026-10-11), because the gate ran on macOS only:

- **S7** when M0's CI lands: the app cross-builds and packages for Windows and Linux on CI runners.
- **S5** at the start of M11: local shell, pod exec, resize, copy and paste on Linux (X11 and
  Wayland) and Windows.

**S7 since CI landed (2026-10-13; [releasing › CI](releasing.md#ci)):** half done. On every push the
macOS job packages the app (`make app`) and the Windows job compiles, vets and links every package
and test with cgo and Fyne (`go build ./...`, `go test -run '^$' ./...` in "Compile (windows)");
no job packages the app for Windows or Linux, so the re-check of packaging there stays open until
M11's packages ([releasing › What gets built](releasing.md#what-gets-built)).

A re-check that fails without a workaround reopens D-1 for that platform only; macOS stays on
Fyne. Accessibility (S8: the table and the drawer announce nothing to a screen reader) is a known
gap of Fyne 2.8 and the one criterion where egui, with AccessKit, would be ahead; it does not gate.

**If a re-check fails**, the engine stays in Go and the UI for that platform moves to a separate
Rust process with egui (option D of the 2026-10-11 discussion). That move needs no engine rewrite because the
engine/UI seam is a serializable protocol from the start: the CLI and the MCP server need the same
seam ([architecture §4](architecture.md#4-the-ui-seam-the-most-important-interface)).

Options considered on 2026-10-11:

| Option | State observed | Why not chosen |
|---|---|---|
| Fyne v2.8.1 | stable, 28.7k stars; tables with header rows; terminal widget `fyne-io/terminal` for Linux, macOS, Windows; accessibility experimental (`-tags accessibility`) | chosen; gaps measured by the spike |
| Gio v0.10.3, Cogent Core v0.3.45 | pre-1.0 | too low-level (Gio) or too young (Cogent Core) |
| GTK4 through gotk4 | used by Seabird (a Go Kubernetes IDE for GNOME, no push since 2025-08) | weak on macOS and Windows |
| Wails v2.16 (v3 in beta) | Go backend, web frontend in a webview | Lens's own web stack, where Freelens already is; a second language (TypeScript); three browser engines to test |
| Rust with egui | best native toolkit of the survey, AccessKit by default | the team writes Go (12 Go repositories, 7 of them Kubernetes operators; no Rust); kept as the fallback UI process |

## D-2: Agent interface scope

**Decided (owner, 2026-10-11):** the whole tool is controllable over MCP. An agent can

- read the UI state: which view is open, which namespace and filter apply, which objects are
  selected, which object the detail drawer shows;
- operate the UI: open a view, set a filter, select an object, open its drawer, open a dock tab;
- find objects ("the Service with ClusterIP 1.2.3.4") and navigate the UI to them;
- run cluster actions: every engine command, with the same verbs and danger levels as the GUI.

**Safety rules** (engine-enforced, so they bind the GUI, the CLI, external agents and the
built-in agent alike):

- Read commands and UI operations run without asking.
- `Mutating` commands from an agent show what will happen in the GUI and run after the user
  accepts, unless the user has allowed agent writes for that cluster context.
- A role binding or cluster role binding an agent creates or changes always waits for the user's
  acceptance, also where agent writes are allowed, whatever its role; the user's own command in
  the window is unchanged (owner, 2026-10-14;
  [H-27](../security/privilege-footprint.md#h-27-a-binding-is-judged-by-its-roles-name-and-the-cache)).
- `Destructive` commands from an agent always need the user's confirmation in the GUI, naming the
  objects, the namespace and the context.
- Every agent-initiated command is recorded in an audit log the user can read.

**The decision round of 2026-10-14 evening** ([the answers](#decision-round-2026-10-14-evening)).
All of it built 2026-10-14, the night after the round, each with its evidence below:

- **Decided (owner, 2026-10-14 evening, A1):** every RBAC object an agent creates or changes waits
  for the user's acceptance, also where agent writes are allowed: Roles and ClusterRoles as well as
  RoleBindings and ClusterRoleBindings. Built 2026-10-14: `AgentsAsk` on `role.create`,
  `clusterRole.create` and `roleBinding.create` always, and on `object.create` and `object.apply`
  whose text or Edit tab holds any of the four kinds (`engine.RBACKind`, `TextHoldsRBAC`,
  `EditorHoldsRBAC`; `TestCreateDialogsAgentsNeedAcceptance`, `TestAgentRoleBindingsAlwaysAsk`,
  `TestTextHoldsRBAC`).
- **Decided (owner, 2026-10-14 evening, A2):** an agent's Helm install or upgrade always waits for
  the user, whatever the chart holds, not only a chart that renders RBAC. Built 2026-10-14:
  `AgentsAsk` on `helm.chart.install` and `helm.release.upgrade` (`TestHelmInstallAgents`,
  `TestHelmUpgradeAgents`).
- **Decided (owner, 2026-10-14 evening, A16,
  [H-26](../security/privilege-footprint.md#h-26-prometheus-is-found-by-name)):**
  `cluster.setPrometheus` from an agent is a write the user confirms (mutating); the window's use
  stays as it is. Built 2026-10-14: the UI command declares `AgentsAsk`, so an agent's call always
  waits for the user, whatever the allowance of agent writes (the mechanism's night decision below),
  the question naming the service (`TestAgentSetPrometheusAsks`).
- **Decided (owner, 2026-10-14 evening, A18,
  [H-32](../security/agents-and-mcp.md#h-32-an-agent-can-clear-the-users-notifications)):** marking
  notifications read and clearing them is GUI-only; agents read the list. Built 2026-10-14:
  `notifications.markRead` and `notifications.clear` are GUI-only (`ErrNotificationsGUIOnly`,
  `TestNotificationCommands`).
- **Decided (owner, 2026-10-14 evening, A19,
  [H-31](../security/privilege-footprint.md#h-31-an-agents-helm-values-can-copy-what-lookup-reads)):**
  the notifications that stay in the window today (the window's own, and engine toasts emitted
  `WindowOnly`) reach agents as text, through the filter of the cached Secrets' values
  ([D-44](#d-44-what-agents-read-of-the-docks-logs-and-terminal-tabs)) and with inline Helm values
  removed, instead of `model.Withheld`. Built 2026-10-14 (`Session.EmitWindowToast`,
  `withoutInlineValues`, `TestWindowOnlyNotificationsReachAgents`,
  `TestHelmErrorToastReachesAgentsFiltered`). The follow-up (6 of the round: accept the wider H-31,
  or keep Helm's error toasts withheld) is the lead's night decision for the owner to confirm:
  ★ built as decided, the residual H-31 names stays documented (options: keep Helm action errors
  withheld as an exception to A19). Note: the Helm package already removes the release's
  user-supplied values from its errors (`<value withheld>`); the removal of inline values is a
  second line behind it.
- **Decided (owner, 2026-10-14 evening, A20):** agents may set a proxy, as a write the user
  confirms, the password redacted in the confirmation, the audit log and the results; clearing
  stays as it is. Built 2026-10-14: an agent's proxy, in a cluster's settings and in the
  Preferences, always waits for the user, the confirmation saying where the traffic goes with the
  password as `•••`; the CLI is still refused (`ErrProxyGUIOnly`: it has no window to confirm in)
  (`TestClusterProxyCommands`, `TestNonClusterProxyCommands`). **The lead's night decisions for the
  owner to confirm** (the follow-ups 1, 2 and 8 of the round): the allowance of agent writes does
  not skip it (options: an ordinary mutating write the allowance skips, ★ always ask as A1 and A2:
  a proxy reads the bearer token wherever TLS is not checked); both proxies, a cluster's and the
  Preferences' `httpProxy` (options: the cluster's only, ★ both: the Preferences' steers which chart
  archives and kubectl Cogpit receives); the Preferences' proxy, which belongs to no context, asks
  always (it has no allowance that could skip it).
- **Decided (owner, 2026-10-14 evening, A65):** an agent removing a hotbar bank that holds tiles
  waits for the user, as the window asks the user today. Built 2026-10-14: `hotbar.removeBank`
  declares `AgentsAsk` for a bank with tiles, the confirmation naming them.
- **The mechanism for A16, A20 and A65 (the lead's night decision for the owner to confirm):** their
  commands are UI commands, and `AgentsAsk` took mutating commands only. Options: raise the
  agent's call to mutating (as `Escalate` raises Scale to zero; then the allowance of agent writes
  would skip the question, and the MCP tool would lose its read-only hint for every call);
  ★ let UI commands declare `AgentsAsk`, which makes an agent's call wait for the user whatever the
  allowance (`NeedsConfirmation`'s UI case; `TestNeedsConfirmationAgentsAsk`,
  `TestAgentsAskDispatch`). Why: these three change Cogpit's own settings, not the cluster, so the
  allowance of agent writes for a context was never given for them.
- **Decided (owner, 2026-10-14 evening, A101):** a grant's confirmation (a role binding, a cluster
  role binding) says "Grants these rights immediately, cluster-wide" or "in `<namespace>`", and
  "This cannot be undone." stands only where something is lost (delete, drain, uninstall …). Built
  2026-10-14: the grant's `Detail` begins with the owner's words; `Spec.Undoable` (and
  `ConfirmRequest.Undoable`) drops the line for the destructive commands where nothing is lost.
  **The lead's night decision for the owner to confirm**, which ones: the two grants
  (`clusterRoleBinding.create`, an escalated `roleBinding.create`: deleting the binding takes the
  grant back), `helm.release.rollback` (the release's history keeps the revision) and the node
  shell (nothing is lost); delete, delete of a selection, evict, drain, Helm uninstall, restart (the
  pods' state goes), scale to zero and an Argo CD sync with prune keep it.
- **Confirmed (owner, 2026-10-14 evening):** GUI-only commands stay listed for agents, described
  as GUI-only and refused (A89, as built); agents and the CLI may turn the automatic update checks
  on, never off (A12, [D-4](#d-4-packaging-and-updates)). The transport below names the bridge
  `cogpit-cli mcp` already, the doc fix #33 asked for.

**Transport:** MCP Streamable HTTP on `127.0.0.1` with a random bearer token, written owner-only
to `<user config dir>/Cogpit/mcp-token` and, since 2026-10-15, kept until the user regenerates it
([D-53](#d-53-the-mcp-details-and-a-token-kept-until-the-user-regenerates-it)); a `cogpit-cli mcp`
stdio bridge for agents that only speak stdio.
SDK: `github.com/modelcontextprotocol/go-sdk` v1.8.0 (MCP specification 2026-07-28). Lens
Desktop's own MCP server is read-only and paid ([Lens MCP server](https://docs.lenshq.io/k8slens/mcp-server/));
Cogpit's is the difference. Security perspective: [agents and MCP](../security/agents-and-mcp.md).

**The command the window copies (the lead's default, 2026-10-14, for the owner to confirm; built
by agent CLIBUNDLE).** Help › Copy MCP command copies a `claude mcp add`
command (the top bar's MCP button did too until 2026-10-15; since F6 the MCP details show it beside
the HTTP one, [D-53](#d-53-the-mcp-details-and-a-token-kept-until-the-user-regenerates-it)). Since `Cogpit.app` carries `cogpit-cli` ([D-16](#d-16-the-first-real-release)), an app
running from an installed bundle that holds it copies the stdio registration of that `cogpit-cli`,
`claude mcp remove --scope user cogpit >/dev/null 2>&1; claude mcp add --scope user cogpit --
/Applications/Cogpit.app/Contents/MacOS/cogpit-cli mcp` # example: run once, it follows every
restart and update of the app, since the bridge reads the endpoint and token files itself. A build
without the bundled CLI (`bin/cogpit`, `go run`), and an app running from its disk image or
translocated (a path that is gone once it quits), copy the HTTP command with the port and the
token, which since F6 stay until the user regenerates the token (D-53). The path is the running executable's own (`os.Executable`, links resolved, the
`cogpit-cli` beside it in `Contents/MacOS`), never a guessed `/Applications` path, quoted for a
POSIX shell when it needs it ([`internal/mcp/register.go`](../../internal/mcp/register.go),
[`cmd/cogpit/bundledcli.go`](../../cmd/cogpit/bundledcli.go)). The command is offered only while
the external MCP server runs, as before: with it off, the bridge would serve an engine of its own
rather than the window. Options weighed: (a) the HTTP command only (no change, but the
registration breaks at every restart, and the token sits in Claude Code's configuration); (b) the
stdio registration when the bundle has the CLI, else HTTP (chosen: one command, made once, no
token in Claude Code's configuration); (c) both in the dialog (two commands to explain for one
job).

## D-3: Helm and kubectl binaries

**Helm, decided by D-6:** Helm runs in-process through its Go SDK (`helm.sh/helm/v4` v4.3.0, or v3
v3.22.0 for clusters whose releases need it). No `helm` binary is bundled, downloaded or required.

**kubectl in terminals, decided (owner, 2026-10-11):** a terminal bound to a cluster is where users
type `kubectl` themselves, so it needs a binary that suits the cluster. `kubectl` is supported only
within one minor version, older or newer, of the API server
([version skew policy](https://kubernetes.io/releases/version-skew-policy/#kubectl)).

- The `kubectl` on the user's `PATH` is the default; Cogpit does not override a working setup.
- When `PATH` has no `kubectl`, or its minor version is more than one away from the cluster's, the
  terminal offers to download the matching version with one click. The download comes from
  `dl.k8s.io` (or a configured mirror), is verified against the published `.sha256` and `.sig`
  files, goes into Cogpit's data directory, and is put first on `PATH` in Cogpit's terminals only.
- Preferences as in Lens: download on or off, mirror, directory for binaries, path to `kubectl`.

Options weighed: `PATH` only (fails without `kubectl` or across version skew), always downloading
like Lens (about 60 MB per minor version, overrides deliberate setups), bundling one (60 MB larger
app, still skewed for other clusters, every `kubectl` fix forces a Cogpit release).

**Built 2026-10-13** ([`internal/kubectl`](../../internal/kubectl/), [`engine/kubectl.go`](../../internal/engine/kubectl.go),
[architecture §10](architecture.md#10-terminal-internalterminal)). Where it differs from the text
above, or fills what it leaves open (Cogpit's defaults, not the owner's decisions):

- The version compared is the API server's at connect against the Preferences' path to `kubectl`,
  else the first `kubectl` on Cogpit's own `PATH` (its `version --client`, read once per file); a
  login shell may compute another `PATH`.
- A release downloaded earlier that suits the cluster is put first without asking again; the
  terminal's line says so. The offer is the server's own major.minor.patch (an EKS
  `v1.36.1-eks-…` gets 1.36.1). **Confirmed (owner, 2026-10-14 evening, A87):** a verified,
  matching release downloaded earlier goes first without asking, as built.
- The download is GUI-only (`kubectl.download` refuses agents and the CLI), from
  `https://dl.k8s.io/release` or an https mirror base; the files go into
  `<user config dir>/Cogpit/kubectl/<version>/kubectl` unless the Preferences name another
  directory. A running terminal keeps its `PATH`; the terminals opened afterwards use the release.
- The `.sig` check was sigstore's keyless signature verified with the standard library until
  2026-10-15, without sigstore's transparency log. **Decided (owner, 2026-10-14 evening, A25):**
  sigstore-go with the Rekor transparency-log proof instead, a record of its own:
  [D-50](#d-50-kubectls-signature-check). **Built 2026-10-15:** each download asks
  `rekor.sigstore.dev` for the signature's entry and sigstore-go checks it, the certificate and the
  signature against the trusted root Cogpit embeds; no answer from the log refuses the download
  ([H-35](../security/trust-boundaries.md#h-35-kubectls-signature-check-rests-on-an-embedded-trusted-root-and-rekor-v1s-search)).
- "First on `PATH`" is a link in the terminal's own directory, first in its environment; for zsh
  also after the login profile ran (a `ZDOTDIR` of Cogpit's that hands over to the user's), since
  macOS's `path_helper` moves inherited entries behind `/etc/paths`. Other shells keep only the
  environment's order ([H-36](../security/trust-boundaries.md#h-36-a-shells-profile-can-put-another-kubectl-first)).
  **Decided (owner, 2026-10-14 evening):** the same hand-over for bash (`--rcfile` or
  `PROMPT_COMMAND`) and fish (`conf.d`) (A26); and a setting under Preferences › Kubernetes (kubectl
  for terminals) whose default is the hand-over as today and which can switch to enforcement, a
  shell alias or function `kubectl` that always calls Cogpit's kubectl, in zsh, bash and fish
  (A27). Built 2026-10-14 ([`terminal/kubectl.go`](../../internal/terminal/kubectl.go),
  `Preferences.KubectlEnforce`; `TestBashKubectlFirstAfterTheProfile`, `TestZshKubectlEnforced`,
  `TestFishKubectlFirstAfterTheConfig`). **The lead's night decisions for the owner to confirm:**
  bash is started with `--init-file` (the way VS Code's terminal starts it), not as a login shell,
  its rc reading the files a login bash reads and then putting kubectl first (options: `--rcfile`,
  which a login bash ignores; `PROMPT_COMMAND`, which a profile can overwrite) — bash then reads no
  `~/.bash_logout` and `shopt login_shell` says off; fish gets `-C`, which it runs after its
  configuration (options: a file in `conf.d`, which runs before `config.fish`); enforcement is a
  function (an alias of that name removed first), not an alias; only the user sets the switch, in
  the window (the round's follow-up 4: options: agents too).

## D-4: Packaging and updates

**Decided (owner, 2026-10-11):** Cogpit shows when an update is available and installs it with one
click. Lens Desktop does the same for its exe, dmg and AppImage builds
([install](https://docs.lenshq.io/k8slens/getting-started/install-lens/)).

**Release artifact, decided (owner, 2026-10-11):** the macOS standard way, so that it works for
users who never open a terminal. The app bundle `Cogpit.app` (universal, signed) ships in a disk
image `Cogpit-<v>-macos-universal.dmg` with the usual window to drag the app into Applications; the
disk image is signed, notarized and its ticket stapled. A Homebrew cask, for users who prefer the
shell, installs from the same disk image. Sparkle updates from the same disk image too.

What one click can mean depends on how Cogpit was installed:

| Installed as | One-click update | Status |
|---|---|---|
| macOS app installed from the disk image | download, verify signature and checksum, replace the app bundle, relaunch | decided: Sparkle 2 (below) |
| Windows MSI or portable zip | download, verify, run the MSI upgrade or replace the binary, relaunch | to design |
| Linux AppImage or tar.gz | download, verify, replace the file, relaunch | to design |
| Homebrew, winget, Flathub, deb or rpm repository | the package manager owns the files; the button runs or shows the manager's command | to design |

**macOS updater, decided (owner, 2026-10-11): Sparkle 2** (2.10.0, 2026-09-13, MIT). It verifies updates with
EdDSA signatures and Apple's code signature, installs atomically, offers delta updates, handles
`/Applications` permissions, quarantine and relaunch, signs "the published update archive (dmg,
zip, etc)" and can be instantiated from toolkits other than Cocoa
([Sparkle documentation](https://sparkle-project.org/documentation/)); it needs macOS 12 or later.
The signed update manifest of these documents is then Sparkle's EdDSA-signed appcast. The Homebrew
cask declares `auto_updates true`, which tells Homebrew that the app updates itself
([Cask Cookbook](https://docs.brew.sh/Cask-Cookbook)). Alternatives weighed: an updater written in
Go (all of Sparkle's hard cases as own security-critical code), `go-selfupdate` v1.6.0 (built for
single executables in archives, so the bundle swap would still be own code), Homebrew only (no
one-click update for users who installed from the disk image). Windows and Linux are decided with M11. `fyne.io/tools`
v1.7.3 (`fyne package`) builds the app bundle. Updates are signed with a key held outside CI's reach and verified before
anything is replaced. **Status:** decided for macOS (artifact, cask, Sparkle); Windows and Linux
are open until M11.
**Built (2026-10-13), the first step:** every release up to v0.4.0 carries `Cogpit_<v>_darwin_arm64.zip` (the app
bundle from `make dist-app`: `fyne package`, ad-hoc signed, Apple silicon only) and `cogpit-cli`
archives for Linux, macOS and Windows, with `SHA256SUMS`, built by the release pipeline
([releasing](releasing.md#the-release-pipeline)). The owner's Apple credentials
exist since 2026-10-14 in the `release` environment that only the tag pipeline may use
([releasing › secrets](releasing.md#secrets)).
**Built (2026-10-14), the second step: a signed, notarized disk image**
([releasing › signing and notarization](releasing.md#signing-and-notarization)). The release
pipeline signs the app with the Developer ID (hardened runtime, Apple's timestamp, no
entitlements), puts it into `Cogpit-<v>-macos-arm64.dmg` (a folder with the app and a link to
Applications; the universal image stays a later step), signs the disk image, has it notarized and
staples its ticket; `cogpit-cli` for macOS is signed and notarized too. The release page says so
and names the certificate. No tag run has done it yet: the first release after v0.4.0 is the
proof (since then, verified 2026-10-15: v0.6.0, v0.7.0 and v0.8.0 carry the signed, notarized
disk image; v0.5.0 and v0.9.0 to v0.12.0 have no files). Not built yet: the universal binary and
the Homebrew cask. Sparkle followed the disk image (built 2026-10-14, below), its EdDSA key set up
by the owner on 2026-10-15. The decisions in this step:

- **A failure fails the release's macOS job (the lead, 2026-10-14; confirmed by the owner the same
  evening, A9, as built).**
  With the secrets present, a failed signature, notarization or check fails the job "Sign and
  notarize (macOS)", and the release gets no files until the failed jobs are re-run; Apple gets 45
  minutes, then the job names the submission and the owner re-runs it once Apple has answered
  ([when it fails](releasing.md#when-signing-or-notarization-fails)). Since 2026-10-15 (after
  Apple kept each submission of v0.9.0 to v0.12.0 in progress beyond the 45 minutes) the wait is a job of its own, "Notarize and staple (macOS)", up to 330 minutes for the
  one submission "Sign and submit (macOS)" made, and a re-run waits for the same one. Without any of the secrets
  (a fork), the unsigned zip as before; with some of them, a failure. Options weighed: the
  unsigned zip when signing fails (a release whose files are not what a signed release promises),
  the CLI archives attached without the macOS downloads (a release that looks finished without
  its main download).
- **Signing in a job of its own (agent REL, 2026-10-14, for the owner to confirm):** the only job
  in the environment `release`; it takes the app and the archives as artifacts and runs no Go and
  no dependency's code, so nothing but Apple's tools and the script share a job with the
  identity. Option weighed: signing where the app is built (one hosted runner less, about a
  minute), where `fyne`, the module downloads and the cgo build would run beside the key.
- **The disk image replaces the zip when signed (agent REL, 2026-10-14):** one macOS download, the
  one the cask and Sparkle are to use. Options weighed: the signed app zipped beside it (two
  downloads to explain, of no use to Sparkle's disk-image updates), the unsigned zip beside it (an
  unsigned copy next to the signed one). Moot (owner, 2026-10-14 evening): a release carries only
  the disk image ([D-16](#d-16-the-first-real-release)).
- **`cogpit-cli` for macOS signed and notarized, not stapled (agent REL, 2026-10-14):** the same
  identity in the same job, submitted together with the disk image. A bare binary cannot carry a
  ticket, so Gatekeeper looks it up online when a quarantined copy first runs. Options weighed:
  unsigned as before (blocked when downloaded with a browser), signed but not notarized (still
  blocked: Gatekeeper wants a ticket). **Superseded the same day** by D-16's revision (below);
  moot, the owner said the same evening.
- **Only the disk image is submitted (agent REL, 2026-10-14),** the app inside it. Option weighed:
  notarizing and stapling the app before imaging it (a second submission to wait for before the
  disk image), so that the copy in Applications carries its own ticket; whether its first start
  without a network needs that is not verified, the first signed release is where to try it.

**Revised (owner, 2026-10-14): `cogpit-cli` inside the app, no archives**
([D-16](#d-16-the-first-real-release)). A release carries the disk image and `SHA256SUMS` only;
`cogpit-cli` sits in `Cogpit.app/Contents/MacOS`, signed inside-out with the app (identifier
`io.guided-traffic.cogpit.cli`), so the disk image's one submission covers it, its stapled ticket
carries the whole app, and Sparkle's update replaces it with the bundle. The separate signing and
submission of a bare `cogpit-cli` (the bullet above) and the archives for Linux and Windows are
gone; those systems get Cogpit with their desktop app (M11). Signed releases with the bundled
`cogpit-cli` exist since v0.7.0 (v0.7.0 and v0.8.0, checked by download on 2026-10-14: signed with
the Developer ID as `io.guided-traffic.cogpit.cli`, notarized, it runs and reports its version).
Not tried: whether it starts without a prompt when run from a terminal in a quarantined copy before
the app was ever opened
([releasing › signing and notarization](releasing.md#signing-and-notarization)).

**Decided (owner, 2026-10-13):** the GitHub
environment `release` exists (deployment rule: tags `v*`; the Apple material in it since 2026-10-14) for the Apple
credentials; Sparkle's EdDSA key will be a secret of an environment with the owner as required
reviewer, so CI signs a release's update feed only after the owner approved the run (options
weighed: the key offline with the owner, signing by hand per release; deciding later).
**Decided (owner, 2026-10-14 evening, A10):** a repository ruleset lets only the semantic-release
GitHub App (as a bypass) and the owner create `v*` tags, and the environment `release`'s admin
bypass is off. **Decided further (owner, 2026-10-15 afternoon):** the ruleset's only bypass is the
semantic-release GitHub App (`guided-traffic-automation`, app id 5070048, the releases' author);
nobody tags `v*` by hand, and an emergency tag means an admin switches the ruleset off for the
moment (the organisation's audit log records it). The environment `sparkle` already has no admin
bypass; `github-pages` keeps its bypass (the feed is signed). Options weighed: the App plus a team
holding only the owner (a team to keep, a path nobody uses); the App plus the Repository admin
role (the bot's token alone could start a signed release). Not verified: whether rulesets accept a
single user as a bypass actor (why "the owner" of the earlier wording is not expressible
directly). Not done yet (verified 2026-10-15 ~19:00 UTC with `gh`: no ruleset exists, and the
environment `release` still has admin bypass on): these are GitHub settings, an outward change,
made only with the owner's explicit OK. For whoever makes them: a PUT on the environment `release`
must carry its `deployment_branch_policy` `{protected_branches: false, custom_branch_policies:
true}`, or the tag policy `v*` is lost.
**Decided (owner, 2026-10-14): Sparkle's feed on GitHub Pages**,
`https://guided-traffic.github.io/cogpit/appcast.xml`, the `SUFeedURL` every installed copy keeps.
Options: (a) GitHub Pages, one appcast with the releases' history and Sparkle's channels, so a
Beta channel later needs no other URL, at the cost of Pages (source: GitHub Actions) and a deploy
job; (b) a release asset under `releases/latest/download/appcast.xml`, no setup, but "latest"
leaves prereleases out, so a Beta channel would need a second URL that installed copies do not
know; (c) a rolling release with a fixed tag whose asset each version replaces, channels without
Pages, but a pseudo-release in the releases list for good. The owner chose (a): the URL is built
into every copy and cannot move without stranding the copies that have it. The appcast is signed
in a job of the environment `sparkle` (the owner its required reviewer, the private EdDSA key its
secret `SPARKLE_ED_PRIVATE_KEY`, generated by the owner with Sparkle's `generate_keys`, never seen
by an agent), after the disk image is stapled, and deployed to Pages from there.
**Built (2026-10-14), the third step: Sparkle 2** (agent SPARKLE, the owner's go-ahead for M10
part 2; [releasing › the appcast](releasing.md#the-appcast), [updates](../security/updates.md)).
Sparkle 2.10.0, the newest stable 2.x on 2026-10-14 (MIT; it needs macOS 12, Cogpit 13), pinned in
the [`Makefile`](../../Makefile) by version and the archive's SHA-256 (`make sparkle`, Renovate
proposes new versions and never merges them on its own). `make app` puts `Sparkle.framework` into
`Cogpit.app/Contents/Frameworks` and writes `SUFeedURL`, `SUPublicEDKey` (from
`SPARKLE_PUBLIC_ED_KEY`, empty by default), `SUEnableAutomaticChecks`, `SUScheduledCheckInterval`
(since 2026-10-16, below), `SUVerifyUpdateBeforeExtraction`, `SURequireSignedFeed` and
`CFBundleVersion` into its `Info.plist` ([`hack/macos/sparkle.sh`](../../hack/macos/sparkle.sh)). `CFBundleVersion` is the
release's version since this step: `fyne package` wrote 1 into every build, so Sparkle would never
have seen a release as newer. The binary does not link Sparkle: [`internal/update`](../../internal/update/)
loads the framework at run time from the running bundle (cgo, darwin only) and makes
`SPUStandardUpdaterController` without starting it; the updater starts once the window's event loop
runs. Without the framework, the key, an installed bundle or macOS there is no updater, and Help ›
Check for Updates… is greyed with the reason ("this build has no update key", "not installed from
the disk image", "not installed yet", "not macOS"); `cogpit-cli` has none. Preferences › App: Check
for updates automatically (on by default, as Lens) and Update channel (Stable; the row says Beta
comes with prereleases); `app.checkForUpdates` checks now with Sparkle's window. The release
pipeline signs Sparkle's parts inside-out with the Developer ID, and three jobs draft, sign
(environment `sparkle`) and deploy (GitHub Pages) the appcast. **Checked locally** (2026-10-14):
the bundle built with the framework and a throwaway key, signed ad-hoc by `make dist-macos
MACOS_SIGN=adhoc` (four Mach-O files, each with the hardened runtime, `codesign --verify --strict
--deep` on the app and in the disk image); the framework loaded and the controller made in a
harness process of its own for each bundle layout (`TestSparkleLoadsFromTheBundle`); the three
appcast jobs run with stand-ins for `gh` and the network and a throwaway Ed25519 key: Sparkle's
`sign_update` signed the disk image and the feed, `hack/appcast verify` accepted them, a second
release built on the signed feed, a tampered feed, an environment without its rules and a
prerelease (channel beta) behaved as below. **Not checked, and only a signed release can show it:**
an installed copy finding, downloading, verifying and installing an update, its relaunch through
GLFW's terminate handling, Apple's acceptance of Sparkle's parts signed with the Developer ID,
Pages serving the feed. **What remained (human; done 2026-10-15, below):** the owner's steps in
[releasing › setting up Sparkle](releasing.md#setting-up-sparkle), in order (the environment
`sparkle` with its reviewer and tag rule, Pages and the `github-pages` tag rule, the key, the
secret, the variable); then the first release built with the variable is the first that can
update, and the release after it is the first update. Copies installed before (v0.4.0's zip, and
the first disk images, built without the key) never update themselves: one manual download.
**Set up (2026-10-15; verified with `gh` ~19:00 UTC):** the environment `sparkle` (the owner as
required reviewer, the tag rule `v*` only, admin bypass off) and Pages from GitHub Actions, steps 1
and 2 by the lead with the owner's go-ahead; the key generated by the owner, the secret
`SPARKLE_ED_PRIVATE_KEY` in `sparkle` (created 07:00 UTC), the repository variable
`SPARKLE_PUBLIC_ED_KEY` set. v0.11.0, v0.12.0 and v0.13.0 (18:05 UTC) are built with the key (the
app job's log: "Sparkle bundled with the update key"). No signed appcast is deployed yet
(`https://guided-traffic.github.io/cogpit/appcast.xml` answers 404): v0.9.0 to v0.12.0 have no
files, since Apple kept each notarization in progress beyond the old job's 45 minutes, and
v0.13.0's notarization was in progress, its three appcast jobs to run after "Attach to the release"
and wait for the owner's approval in `sparkle`. So the first release that passes notarization with
the key starts the signed feed, and the release after it is the first real update; until then no
installed copy updates itself, and copies of v0.8.0 and earlier have no key and never will.
**The deploy keeps the served feed's items (owner, 2026-10-16, F18; built the same day):** asked
whether the Pages deploy should refuse an older feed after v0.15.0's appcast, approved after
v0.16.0's, had put the feed back to 0.15.0, the owner chose the guard (the lead's recommendation)
over the documented rule alone: "Appcast (deploy to Pages)" fetches the served feed and refuses a
feed that would drop one of its items, unless its channel's newest were kept
([releasing › the appcast](releasing.md#the-appcast)). Merging in the deploy job was not an option:
the feed is signed, and the deploy job holds no key. **Corrected by the owner the same day (built
2026-10-16):** v0.19.0's deploy was refused although the appcasts were approved in release order,
because its draft ran (16:01 UTC) before v0.18.0's appcast was deployed (16:34 UTC); "v0.19.0 is the
younger one". The guard now refuses only a feed that takes back a newer release Pages serves (a
channel's newest served item newer than the new feed's newest of that channel, or the channel
gone); an older served item the newer release's draft never saw is left out and named in a warning,
since Sparkle offers each copy the newest item it may install. What the left-out item costs: its
place in the feed's history, unless the release's draft is re-run; no copy is offered less while
every release has the same `LSMinimumSystemVersion` (`MACOS_MIN`, 13.0), since an older item is only
a fallback for a system the newest does not support.
**The first update (2026-10-16):** Pages serves the signed feed with 0.16.0 and 0.15.0 since ~06:20
UTC; the owner's MacBook, 0.15.0 from its disk image, found 0.16.0 at its launch check (after
`defaults delete io.guided-traffic.cogpit SULastCheckTime`), showed the Update button, and on its
click Sparkle installed 0.16.0 and started Cogpit again (a person's observation, not a test).
The decisions of this step, each the lead's or agent SPARKLE's for the owner to confirm:

- **The appcast itself is signed (agent SPARKLE).** Sparkle 2.9 and later signs the feed
  (`SURequireSignedFeed`, which needs `SUVerifyUpdateBeforeExtraction`): whoever can change what
  Pages serves cannot change what Sparkle's window shows (release notes, links, a critical or
  informational mark) or point an item at another download. Cost: losing the key stops updates
  until a release rotates it through a Developer ID-signed disk image, and after 20 days without a
  valid feed signature Sparkle falls back to the items without their notes and links. Option
  weighed: the enclosures' signatures alone (Sparkle's default), where the feed itself is
  unsigned. **Confirmed (owner, 2026-10-14 evening, A11,
  [H-91](../security/updates.md#h-91-after-20-days-of-unverifiable-feeds-sparkle-presents-items-again)):**
  the 20-day failsafe stays; the recommendation to set it to 0 was declined.
- **Cogpit's preference drives the automatic checks (the lead's design).** `SUEnableAutomaticChecks`
  YES in `Info.plist`, so Sparkle never asks at the second start, and the engine sets
  `automaticallyChecksForUpdates` at start when it differs and on every change. Option weighed:
  Sparkle's own prompt and setting (two places for one setting, and agents could not see it).
- **Who may change the automatic checks (the lead's default):** the user turns them on and off;
  agents and the CLI may turn them on, not off, since off leaves the copy without the next
  release's fixes (the reverse of Load chart icons). Options weighed: free for agents (an agent
  could quietly stop security updates); the user's only both ways. **Confirmed (owner, 2026-10-14
  evening, A12):** as built.
- **Any origin runs `app.checkForUpdates` (agent SPARKLE):** it changes nothing and Sparkle asks
  before it installs. Option weighed: GUI-only (an agent asked to look for an update could not).
- **The Beta channel (agent SPARKLE):** refused for every origin until Cogpit publishes
  prereleases; once offered, the user's only (a prerelease is less tested code). A prerelease's
  `CFBundleVersion` (0.6.0-beta.1 is not Apple's format) needs a numeric scheme then: open.
- **No XPC services in the bundle (agent SPARKLE):** they are for sandboxed apps, and Sparkle's
  documentation allows removing them otherwise; nothing in the bundle then carries an entitlement.
  Option weighed: keeping and signing them (`Downloader.xpc` with its entitlements kept), code that
  never runs in Cogpit, signed and notarized anyway.
- **No entitlements for Sparkle's parts (agent SPARKLE):** signed as Sparkle's documentation's
  commands sign them, which drops the `com.apple.application-identifier` of `Autoupdate`'s ad-hoc
  signature. The first update (2026-10-16, the owner's copy from 0.15.0 to 0.16.0) installed
  through `Autoupdate` signed so: it needs nothing it had (a person's observation).
- **Three appcast jobs (agent SPARKLE):** the draft (Go, no secret), the signature (the key and
  Sparkle's `sign_update` only), the check with the public key and the deploy. Option weighed: one
  job, where Go, its toolchain and `hack/appcast` would run beside the key.
- **The history comes from the feed on Pages, only when the key signed it (agent SPARKLE):** an
  altered feed is never signed again. Options weighed: rebuilding the history from the releases
  (every past disk image downloaded and signed again per release), the newest item alone (a copy
  the newest release does not support would get nothing). Each channel keeps its 10 newest items
  (Cogpit's default).
- **Release notes embedded as Markdown (agent SPARKLE):** the release's text up to the pipeline's
  download note, at most 32 KiB, with a link to the release; signed with the feed. Options weighed:
  `sparkle:releaseNotesLink` to the release's page (a signed feed needs a signature over the linked
  file, and GitHub's page changes), no notes.
- **Help › Check for Updates… before About Cogpit, its label with the short reason when greyed
  (agent SPARKLE):** a native menu item has no tooltip. Lens's own place for it is not captured.

Not asked in the decision round of 2026-10-14 evening, so standing as built (A12's note): the signed
feed, the preference driving Sparkle, any origin checking, Beta refused until prereleases, no XPC
services, no entitlements on Sparkle's parts, the three appcast jobs, the history from the signed
feed (10 per channel), the embedded notes, the "not installed yet" reason, the menu item's place.

**Built (2026-10-15, the owner's finding F7, [D-54](#d-54-the-layout-toggles-and-the-update-button)):
the Update button.** Sparkle's standard user driver gets a delegate (`sparkle_darwin.m`) that takes
over showing the updates scheduled checks find (Sparkle's gentle reminders): Sparkle shows no window
by itself; the delegate reports the update with its version until the update session ends, the
engine keeps it as `UIState.Update`, and the top bar shows an Update button whose click runs
`app.checkForUpdates`, which brings Sparkle's window with that update up. A check the user starts
shows Sparkle's window as before. Without the framework or the key there is no updater, so no
button. Checked with the real Sparkle 2.10.0 in a harness against a local feed
(`TestSparkleReportsAScheduledUpdate`); not checked until a keyed release is in the signed feed:
an installed copy finding one.

**Decided and built (owner, 2026-10-16, F11): the scheduled check every hour.** `make app` writes
`SUScheduledCheckInterval` 3600 into the bundle's `Info.plist`
([`hack/macos/sparkle.sh`](../../hack/macos/sparkle.sh), checked by
`TestSparkleScriptWritesTheSettings`); without it Sparkle's default applied, a day. An hour is the
least Sparkle 2.10.0 uses: `SPUUpdater` raises a shorter interval to `minimumUpdateCheckInterval`,
3600 seconds in a release build of Sparkle (`Sparkle/SPUUpdaterSettings.m`). Cogpit never sets
Sparkle's `updateCheckInterval`, which would keep a value in the user's defaults over the
`Info.plist`'s. Cost: one fetch of the signed appcast from GitHub Pages per hour for each running
copy with the automatic checks on, and none of the cluster. A launch checks at once when the last
check is older than the hour. A copy takes the interval with the release that brings it.

**Human:** Windows code-signing material (the Apple Developer ID exists since 2026-10-14); the
owner's approval of the first keyed release's appcast jobs in `sparkle`; the tag ruleset and
`release`'s admin bypass (A10, with the owner's explicit OK). Sparkle's update key and the
environment `sparkle`'s rules are set up since 2026-10-15
([releasing › setting up Sparkle](releasing.md#setting-up-sparkle)).

## D-5: Extensions

**Decided (owner, 2026-10-11): extensions are built into Cogpit's own code.** What Lens ships as
an extension (a view for a custom resource, an integration such as Flux CD or Argo CD) becomes
part of the project: kind profiles, commands and views in Cogpit's packages, reviewed and released
with Cogpit. There is no mechanism for external extensions for now; adding one later needs a new
decision.

Lens returned to extensions in 2026.10.51501 with npm packages that need a paid plan to install
([Lens extensions](https://docs.lenshq.io/k8slens/extensions/)); Cogpit has no JavaScript runtime,
so those cannot run in any case. Lens's extension features (installing from the file system,
auto-update, the extension registry, safe mode) are therefore not targets.

Options weighed: kind profiles as data files (no third-party code, but a format to maintain),
WebAssembly plug-ins with wazero (sandboxed custom logic, a large API and review effort), external
programs as plug-ins (no sandbox), Lens-compatible JavaScript extensions (impossible without a
JavaScript runtime).

## D-6: Kubernetes libraries

**Decided (owner, 2026-10-11), with D-7:**

| Need | Module | Version observed |
|---|---|---|
| API access, informers, exec, port-forward | `k8s.io/client-go` | v0.37.1 (2026-09-23; Kubernetes 1.37) |
| kubectl behaviour (drain, rollout, describe) | `k8s.io/kubectl` | v0.37.1 |
| Metrics API | `k8s.io/metrics` | v0.37.1 |
| Helm | `helm.sh/helm/v4` | v4.3.0 (2026-09-09) |

These are the libraries `kubectl` and `helm` are built from, so their behaviour (credential
plugins, drain semantics, Helm release handling) is the reference behaviour rather than a
reimplementation of it. This replaces the earlier choice of kube-rs, whose typed API reached only
Kubernetes 1.36 in a release on 2026-10-11.

## D-7: Language and platforms

**Decided (owner, 2026-10-11): Go**, replacing the earlier Rust-only decision. Reasons: the
Kubernetes ecosystem (D-6) is Go, and the team writes Go. Costs accepted: weaker native UI
toolkits than Rust (D-1 has the gate and the fallback), and fewer crash guarantees from the
compiler (D-11).

**Platforms (owner, 2026-10-11):** Cogpit is a desktop client application, installed on the user's
machine. **macOS comes first**: alpha and beta are macOS releases. **Windows and Linux** (Ubuntu
first) follow after the macOS beta ([roadmap M11](../parity/roadmap.md#milestones)). Until then the
code stays portable: platform services (PTY, keychain, paths, file dialogs) sit behind interfaces,
and CI compiles for Windows and Linux from M0 so that portability breaks are found when they are
cheap. Cogpit is never a web UI and is never deployed into a cluster; "Helm" in these documents is
Lens's feature for managing Helm releases in the user's clusters.

## D-8: Licence

**Decided:** Apache-2.0, as in [`LICENSE`](../../LICENSE) since the initial commit.

## D-9: Scope against Lens's paid and vendor features

**Context.** Lens Desktop gates twelve features behind paid plans
([premium features](https://docs.lenshq.io/k8slens/premium-features/)) and has vendor features
(Lens ID activation, subscriptions, telemetry, surveys, Teamwork's relay through Mirantis's
servers).

**Decided (owner, 2026-10-11):**

- **Flux CD and Argo CD views are in the beta.** The owner's organization keeps GitOps
  repositories for both (`k8s-flux-base`, `k8s-base-argo`), so these views are daily work there,
  not a paid extra. Their resources already appear as generic custom-resource views; the beta adds
  the curation: kind profiles with columns and status, the two dashboards, Argo CD's Sync and
  Refresh ([G23](../parity/gaps.md#g23-flux-cd-and-argo-cd-views)).
- **The other paid features come after the beta** and count toward the full target only: the EKS,
  AKS and GKE account integrations, Security Center, cost monitoring
  ([G18](../parity/gaps.md#g18-paid-lens-desktop-features)). They need cloud accounts and data
  sources and are not part of the mainstream audience's daily work.
- **Cheap ones are built anyway:** the Hotbar (with the interface, G13) and the editor's diff (with
  the editor, G9).
- **Never:** Lens ID activation, subscriptions, telemetry, surveys.
- **Teamwork** needs a hosted relay and stays out of scope unless the owner decides to run one.
- Lens's paid MCP server and Ask AI are superseded by D-2 and D-10.

**Decided (owner, 2026-10-15): verifying the credential plugins against EKS, GKE and AKS is out of
scope until long after parity.** It was an item of the path to alpha (item 7 of the
[roadmap](../parity/roadmap.md)'s current focus) and part of M1's definition of done: connecting
through `aws eks get-token` or `aws-iam-authenticator`, `gke-gcloud-auth-plugin` and `kubelogin`
against real clusters of the three clouds, which needs accounts there
([G2](../parity/gaps.md#g2-cluster-connection-and-authentication),
[platform parity](../parity/platform-parity.md#cluster-authentication-and-connectivity)). Cogpit
runs these plugins through client-go's exec support as it runs any other, once the user allowed
them (D-26); what is out of scope is checking them end to end on the clouds, not the plugins. The
rows stay `missing` in the platform checklist and count toward the full target, as the account
integrations above do. Recorded here because this entry holds Cogpit's scope against Lens; the
alpha gate itself lives in the roadmap, which no D-entry decides.

Options weighed: all paid features after the beta (GitOps views late although used daily), all
paid features in the beta (150–320 h more before it, with cloud accounts and data sources), none
(Cogpit behind Lens Pro, GitOps included).

## D-10: Built-in agent

**Decided (owner, 2026-10-11):** a chat panel runs beside the views, like the chat beside the
editor in VS Code. Its agent answers questions about what the user is looking at and acts for the
user, through the same MCP tools external agents use (D-2): it reads the current view and
selection, navigates the UI ("show me the Service with ClusterIP 1.2.3.4"), and runs cluster
actions under D-2's confirmation rules. Lens Desktop's "Ask AI" only launches an external AI
command-line tool with the cluster's context
([Ask AI](https://docs.lenshq.io/k8slens/ask-ai/)); a built-in agent that operates the UI is
Cogpit's difference.

**LLM interface:** the OpenAI Chat Completions API with tool calling and streaming, the interface
both targets offer: the LiteLLM proxy ("OpenAI-Compatible": `/chat/completions`, `/models`,
[LiteLLM proxy](https://docs.litellm.ai/docs/proxy/user_keys)) and Open WebUI (`POST
/api/chat/completions`, `GET /api/models`, bearer API key,
[Open WebUI API](https://github.com/open-webui/docs/blob/main/docs/reference/api-endpoints.md)).
Settings: base URL, API key (kept in the operating system's keychain; in an owner-only file instead
since the owner's decision of 2026-10-15, item 12 below), model. Go client candidate:
`github.com/openai/openai-go/v3` v3.76.0.

**The owner's requirements, restated (2026-10-12, "so that they are not lost"):** a chat inside
Cogpit like the chat of Claude Code for VS Code, switched on and off by the user; connected to an
OpenAI-compatible API such as Open WebUI or LiteLLM, so that local models can be used; its agent
gets Cogpit's tools (the same as the MCP tools) to operate the UI and the cluster. The external
MCP server (the loopback endpoint other apps on the computer use) can be switched off while the
chat keeps working: the chat's agent does not go through that server but dispatches the
registry's commands in-process (`Session.Dispatch` with origin `BuiltInAgent`), so the status
bar's MCP switch (built 2026-10-12, [D-23](#d-23-a-link-to-the-owner-in-the-status-bar) and
[architecture §7](architecture.md#7-mcp-and-the-built-in-agent)) affects external agents only.
The owner confirmed the timing below on 2026-10-12 (the chat comes later).

**Timing:** after the alpha (milestone M10); pulled forward by the owner (2026-10-14: "dass ich endlich meinen Agent Chat haben will"), built from 2026-10-14 late evening. **Open details:** what cluster data may be sent to
the LLM endpoint (Secret values never, by default; decided 2026-10-14 late evening, below); whether
Open WebUI passes client-defined tools through to every model (its documentation says it forwards
them, checked 2026-10-14 below; not run against a real Open WebUI).

**The owner's requirements of 2026-10-14 late evening (binding):** endpoints LM Studio (local),
LiteLLM and Open WebUI, each through its OpenAI-compatible Chat Completions API with tool calling;
several LLM connections (name, kind, base URL, API key in the keychain (an owner-only file since
the owner's decision of 2026-10-15, item 12), model, limits), one picked
per conversation, one the default; an optional connection test that checks and shows each layer on
its own (TCP, Layer 7 with TLS, Auth, tool use) and never sends cluster data; the chat's tools
always on, whatever the external MCP server's switch says (the switch, A15 and
[D-49](#d-49-the-external-mcp-servers-switch-honoured-by-cogpit-cli-mcp), governs the endpoint
other clients use); a kill switch (a Stop button, Esc and ⌘.) that cancels the model's stream and a
running tool call, and a guard against a local model looping on the same tool call. Local tests run
against the owner's LM Studio (an environment variable names the endpoint and the model; skipped
without it, never failing CI); CI gets a LiteLLM virtual key as a GitHub secret.

**Decided (owner, 2026-10-14 late evening, one question at a time):**

- **Where the chat sits:** a column of its own at the window's right edge, the full height beside
  the drawer, the view and the dock; ⌘I (Ctrl+I) shows and hides it (`chat.toggle`, UI state
  `UIState.Chat`); its width is dragged and kept like the sidebar's and the drawer's. One chat per
  window (Cogpit connects one cluster at a time); a conversation is bound to the context it began
  on. Options weighed: a dock tab (the cheapest, but the dock shows one tab at a time, so the chat
  would vanish whenever its agent opens a logs or terminal tab), a column above the dock only.
  Design direction (owner): "wie die VSCode Claude integration … kein 100%iger Clone, aber nimm das
  als Richtung" — a pinned card for the user's message with its attached context, the answer as
  Markdown on the panel's background, compact tool-call rows, an input box with a toolbar, the
  connection and model below it.
- **What the chat sends (H-13), and a private connection:** the owner's stance: the goal is local
  models wherever possible, whose risk for transmitted values differs; Cogpit's own chat will
  usually run one. Every connection carries the user's mark **"private connection (a local model or
  a private cloud)"**, off by default. Without it the chat follows exactly the external agents' rules
  (no extra question per cluster: the chat is not stricter than an external agent). With it the
  chat **reads like the user**: what the window shows reaches the model — Secret values (in
  `object_get`, the drawer, unscrubbed logs), Helm's user-supplied values and the Upgrade and
  Install editors' text, Flux's credential fields, a failed Helm action's error text; terminal,
  shell and attach tabs still need the tab's own switch (H-83). The write rules are the same in
  every case: D-2, H-27 (agents always ask for role bindings), A1, A2, and Secret create and edit
  stay GUI-only. Built as `Invocation.ReadsLikeUser`, honoured only with origin `builtInAgent`
  (the engine's side built 2026-10-14 night, M12 (c), below). What
  each turn sent is shown in the chat. Options weighed: ask once per cluster and connection (the
  lead's first proposal, stricter than the external agents: declined), always on and only shown, a
  list of contexts per connection; for the private mark also terminals without their switch, and
  Secret values only on a question per Secret.
- **Conversations across restarts:** kept, scrubbed: an owner-only file (0600) in Cogpit's
  configuration directory, the last 20 conversations per context, deleted one by one or all at once;
  what is stored passes the H-82 scrubber first (tool results and answer text: the values of the
  cached Secrets replaced), as D-19 keeps Secret values out of drafts; a kept conversation goes on
  with its scrubbed history. Options weighed: gone at quit; kept unscrubbed.
- **Limits per turn** (adjustable per connection): the turn stops at the third identical tool call
  (the same tool with the same canonical arguments) with a note; at most 25 tool calls, 300,000
  tokens (prompt and answer over all of the turn's requests, as the endpoint reports them, else
  estimated) and 10 minutes. Options weighed: strict (the second repetition, 10 calls, 100,000
  tokens, 3 minutes), loose (the fifth, 50 calls, no token limit, 20 minutes).

**Built 2026-10-14 late evening, the engine side (M12 (a), [`internal/agent`](../../internal/agent/),
[architecture §7](architecture.md#7-mcp-and-the-built-in-agent)):** the connections with their keys
in the macOS keychain (in an owner-only file since 2026-10-15, item 12), the client, the connection test, the turn loop over the registry's commands
with the limits and the kill switch, and the conversations kept scrubbed, installed by the desktop
binary (`Session.SetChatAgent`); the chat column in the window is built apart. Tested against a
fake endpoint ([`testkit/fakellm`](../../internal/testkit/fakellm/fakellm.go)) and, locally, against
the owner's LM Studio with `qwen/qwen3.8-27b` over the demo cluster (`TestLiveEndpoint`,
2026-10-14: the four layers of the connection test passed, the tool layer in 3.2 s with the model
loaded and in 43.8 s on the first run; "How many pods run in namespace demo-identity?" took one
`resources_list` call and two requests, 63,232 bytes and 16,123 tokens as LM Studio reported them,
61 s, and the answer was right).

**The endpoints, checked against their documentation on 2026-10-14** (the kind sets the paths; every
kind takes the OpenAI Chat Completions request and a `Bearer` key):

| Kind | Default base URL | Chat path | Models path | Verified | Not verified |
|---|---|---|---|---|---|
| LM Studio | `http://localhost:1234/v1` | `<base>/chat/completions` | `<base>/models` | the base URL, both paths, tool calls streamed as `delta.tool_calls` with an `index` ([OpenAI compatibility](https://lmstudio.ai/docs/developer/openai-compat), [tool use](https://lmstudio.ai/docs/developer/openai-compat/tools)); against the owner's LM Studio: `{"data":[{"id"}]}`, no key needed, thinking sent as `reasoning_content` deltas, usage reported with `stream_options.include_usage`, a tool whose `parameters` lacks `properties` refused with HTTP 400 (the chat adds `"properties":{}`) | – |
| LiteLLM | `http://localhost:4000` | `<base>/chat/completions` (also under `/v1`) | `<base>/models` (also `/v1/models`) | both paths with and without `/v1`, the key as `Bearer`, `{"data":[{"id"}]}` ([user keys](https://docs.litellm.ai/docs/proxy/user_keys), [model discovery](https://docs.litellm.ai/docs/proxy/model_discovery)) | whether tool calls and usage pass through for every backing model (the model's and the provider's); not run against a real proxy (CI's LiteLLM key is not there yet) |
| Open WebUI | `http://localhost:3000` | `<base>/api/chat/completions` | `<base>/api/models` | both paths, the key as `Bearer`, and that caller-provided `tools` are forwarded to the model instead of Open WebUI's own tools ([API endpoints](https://github.com/open-webui/docs/blob/main/docs/reference/api-endpoints.md)): the open detail above is answered from the documentation | the models list's shape (a third-party guide shows `data[].id`; the client takes `id` or `name`, wrapped in `data` or `models` or a bare array) and streaming in OpenAI's SSE format; not run against a real Open WebUI |
| Other OpenAI-compatible | `https://api.openai.com/v1` | `<base>/chat/completions` | `<base>/models` | OpenAI's own shape | each server's quirks |

**The lead's decisions for the owner to confirm** (night run of 2026-10-14, nobody to ask; each with
the options weighed):

1. **The client.** Plain `net/http` with the protocol subset the chat uses (chosen), or
   `github.com/openai/openai-go/v3` (v3.78.0 on the module proxy, published 2026-10-13). Why: the
   subset is small (one POST with SSE, one GET); the servers' quirks need the raw events anyway
   (`reasoning_content` and `reasoning`, `<think>` in the content, an index missing from a tool-call
   delta, a JSON body where a stream was asked for, an error object inside the stream, Open WebUI's
   `/api` paths); no `go.mod` change and a smaller govulncheck surface; the SDK moves fast (a
   release a day). Revisit if Responses API features become a target.
2. **The tools a request carries.** A core set of 20 registry commands as native tools (the reads,
   the UI commands that show the user something, and Scale, Restart and Delete) and two meta-tools,
   `search_commands` (finds any other command by words, returns its id, label, danger, description
   and schema) and `run_command` (runs it by id, through the same `Dispatch`) (chosen); or every
   command as a tool, as the MCP server offers them; or only the meta-tools. Measured 2026-10-14
   (`TestToolsetSize`): every command 183 tools, 219,945 bytes, about 55,000 tokens (169 tools and
   202,445 bytes without the excluded ones); the core set 22 tools, 29,443 bytes, about 7,400 tokens.
   Every command would fill a local 27B model's context before the first message. Left out of both:
   GUI-only commands (refused to every agent anyway) and `chat.toggle`. Since 2026-10-15 a
   connection may send every command as a tool instead (the owner's audit answer (4), below).
3. **A tool result's bound.** 32 KiB a result, about 8,000 tokens, the model told where it was cut
   and how to ask for less (chosen); or 8 KiB (too small for one object's YAML with its events), or
   64 KiB.
4. **Thinking.** Dropped: never shown as the answer, never kept, never sent back (chosen); or a
   collapsed item in the conversation (needs an item kind the contract does not have, and would put
   long reasoning on disk).
5. **What each user message carries.** A compact `ui_state` before the text: context, connection
   state, demo, view, namespace filter, search text, the selection's count (with *attach selection*
   its objects, at most 50), the drawer's object, the shown dock tab (chosen); or the whole
   `UIState` (thousands of tokens of tabs, dock, preferences and settings each turn), or nothing.
6. **The window switched clusters during a turn.** The turn stops before its next tool call with a
   note (chosen); or it goes on against the cluster now connected (would read and write another
   cluster than the conversation's), or reads only.
7. **What the conversations file keeps.** Each piece scrubbed as it becomes final during the turn
   (the user's message and the title, each answer, each tool call's arguments, result and error,
   the history the model is sent), so that a cluster switched afterwards does not matter; a piece
   that cannot be scrubbed (the Secrets not listed within 10 s, the cluster no longer connected) is
   kept as `<withheld: …>` (fail closed) (chosen); or scrubbing at save time, or not keeping a
   conversation whose scrub failed. Conversations without a message are not kept, nor, since
   2026-10-15, one in which a turn ran over a private connection (item 20); a conversation
   whose connection was deleted takes the default one on its next turn (the contract's).
8. **The connection test's details.** TCP 5 s, HTTP 10 s (a `GET` of the models path *without* the
   key: any status passes), Auth 15 s (the models list *with* the key; 404 says to check the base URL
   and the kind; a configured model not listed is noted, not failed), Tools 120 s (a streamed request
   offering `get_time` with `tool_choice` left to the model) (chosen; the first tool call of an
   unloaded 27B model took 43.8 s).
9. **`stream_options.include_usage`** sent with every chat request; an endpoint that answers 400
   naming it gets the request once more without, remembered for the connection until it changes or
   Cogpit quits (chosen); or never sent (every turn's tokens estimated).
10. **Where the key goes.** No redirects are followed, so the key reaches the configured endpoint
    only; proxies from the environment as Go's default; a plain `http://` base URL with a key is
    allowed and the connection test's HTTP layer says the key travels unencrypted (chosen; refusing
    it would block a LiteLLM on the local network; recorded as
    [H-95](../security/credentials.md#h-95-an-llm-api-key-over-plain-http)). **Revised by the owner
    (2026-10-15, his audit answer (6), below):** still allowed, and warned about in the warning
    colour wherever it is set up when the host is another machine; an `https://` server with a
    private CA gets a CA file per connection.
11. **Errors.** An endpoint's 401 or 403 is reported without its message (OpenAI's quotes a part of
    the key); other errors carry at most 200 characters of the server's message with the key
    removed; never the request body.
12. **The keychain.** The login keychain through the Security framework, a generic password per
    connection, service `io.guided-traffic.cogpit.llm`, account the connection's id, label "Cogpit
    LLM connection" (chosen); or the data-protection keychain (needs a keychain-access-groups
    entitlement; Cogpit is signed with none, [`hack/macos/release.sh`](../../hack/macos/release.sh)). Elsewhere a key is refused ("no keychain on this
    platform yet", M11). **Replaced (owner, 2026-10-15):** no keychain at all, "übertrieben, erzeugt
    Reibung": every self-built binary carries another code signature, so macOS asked "cogpit wants
    to access key 'Cogpit LLM connection' in your keychain" with the login password again and
    again. The key goes into an owner-only file in Cogpit's configuration directory, as the MCP
    token and the proxy passwords ([H-33](../security/credentials.md#h-33-proxy-passwords-are-plaintext-in-cogpits-settings))
    do. **Built 2026-10-15:** `llm-keys.json` beside `llm-connections.json`, the connection's id to
    its key, through the engine's owner-only writer (0600 in 0700, replaced atomically, symlinks
    refused), read and written anew on every call, on every platform, so Windows and Linux keep
    keys too; the keychain code (cgo, the Security framework) removed
    ([`agent/keystore.go`](../../internal/agent/keystore.go)). What it costs: the key is plain text
    protected by the file's mode, as kubeconfig tokens and proxy passwords are
    ([credentials › H-99](../security/credentials.md#h-99-llm-api-keys-are-plaintext-in-cogpits-settings-folder)). The lead's decisions, each with the
    options weighed (the first two confirmed by the owner on 2026-10-15 afternoon, the third not
    asked):
    - **A key file other users could access** (group or other bits set, a copy restored without
      its mode): made owner-only again before its keys are used, the log naming the file and its
      mode, never a key, and advising to replace keys that others may have read (chosen); or
      refused, its keys unused until the user enters them again (the MCP token's rule; but Cogpit
      makes a new token itself, while an API key entered again is the same key, so refusing adds
      friction and revokes nothing); or used as it is. Why: only a new key at the provider undoes
      a leak, which the user alone can make; and while Cogpit's writer kept the directory 0700,
      no other user could reach the file through it. **Confirmed (owner, 2026-10-15 afternoon):**
      repaired to 0600 and its keys used, with a Warn line in `cogpit.log` and on stderr advising
      to replace the keys, no notice in the window; chosen over the lead's recommendation in the
      question, the repair plus a bell notification so that the advice to rotate reaches the user.
      Dormant: Cogpit always writes 0600.
    - **A key the keychain kept: no migration.** Never read, not even once to move it: reading it
      is the prompt the owner wants gone. Such a connection, whose file still says a key was
      given, shows "no API key", and a turn ends with "the connection's API key is missing: enter
      it again in the chat's settings", the keychain no longer named. The chat's model menu (which
      saves with the key kept) leaves it so; the settings form saved with the key field empty
      gives it up ("Empty: no key", as the form's hint says). The item stays in the login keychain
      for the user to delete in Keychain Access ([usage](usage.md)); Cogpit leaves it alone.
      **Confirmed (owner, 2026-10-15 afternoon):** such a key is entered once again, the old item
      left in the keychain. Options weighed: migrating once at the first start (one keychain prompt
      per connection, the cgo Security code kept for macOS only); an "Import from Keychain" button.
    - **A key file that does not decode:** a read fails without quoting the file (a JSON error can
      quote its bytes, which hold keys); the next key saved replaces it. Not asked in the owner's
      round of 2026-10-15: it stands as the lead's decision.

    Tests: `TestKeyFileKeepsTheKey`, `TestKeyFromTheKeychainIsAskedForAgain`,
    `TestKeyFileAccessibleByOthersIsRepaired`, `TestFileKeyStoreRefuses`, `TestNoKeychainCode`,
    `TestAgentPreferencesKeyHint`.
13. **Window updates while an answer streams.** `EventChat` at most every 75 ms per conversation,
    at once on every state change (chosen; the contract's range was 50–100 ms).
14. **A connection without a model.** It may be saved without one, so that the settings form can
    list the endpoint's models for it (`Models` takes a saved connection's id); a turn refuses to
    start without a model (chosen); or a model required on save (the form could not offer the list
    before the first save).

**The window's chat, built 2026-10-14 night (M12 (b), [architecture §7](architecture.md#7-mcp-and-the-built-in-agent)):
the lead's decisions for the owner to confirm** (a night run: each question weighed, one option
built):

- **A settings page and the column:** the column stays beside the page (the page replaces the work
  area only). Options: hide the column while a page is open. Why: the chat is the user's
  conversation, not a view; a page opened by the agent must not take it away.
- **⌘I:** shows and hides the column as `chat.toggle` declares; opened by the user (⌘I, View ›
  Agent Chat) the input takes the focus, opened by an agent it does not. Options: VS Code's "focus
  or unfocus" (⌘I focusing an open column instead of hiding it). Why: one meaning for the user and
  for agents (the command's description says show or hide), and an agent cannot steal the focus.
- **⌘. and Escape:** ⌘. is View › Stop Agent and stops the shown conversation's turn from any focus,
  the column hidden or not; Escape stops it only while the input has the focus (otherwise Escape
  leaves the input, and the window's Escape keeps closing the palette and the drawer; the picker it
  closed too went with D-57).
  Options: ⌘. only while the chat has the focus. Why: a kill switch has to work when the user is
  looking elsewhere, which is when a looping model does damage.
- **The pinned message:** built: the user's message whose top has scrolled out stays at the column's
  top, at most two lines and its chip, on a band of the column's background; a click scrolls back to
  it. Options: no pinning (Fyne has no sticky headers). Why: it was feasible with a layer over the
  scroll, and it is what the design direction shows.
- **Tables in answers:** a grid of their own with thin borders, a bold header on the header
  surface, the columns as wide as their text up to a fair share of the width (narrow columns keep
  theirs, the wide ones share the rest and wrap), a column keeping its longest word whole
  ("CrashLoopBackOff") where the others have room to give. Options: monospace text. Why: it reads
  like the screenshot's tables and stays inside a 300-point column.
- **What was sent:** "Sent 4 requests · 47.1 KiB · 12,840 tokens · to <host>" under each message, a
  lock for a private connection; a click opens the exact request body in a read-only dialog with
  Copy (at most 2,000 lines drawn). Options: a read-only dock tab. Why: a dock tab is UI state agents
  read (`ui_state`), and what the chat sends stays in the window.
- **The chip of the view and selection:** names the view and the drawer's object, else the one
  selected object or how many; its × drops it for the next messages until the view or the selection
  changes, "+" brings it back. Without a selection it names the view alone and the message still
  attaches (the agent then adds nothing beyond `ui_state`). Options: no chip without a selection;
  a dropped chip staying dropped. Why: VS Code's file chip comes back for the next file.
- **Conversations:** the column shows the connected cluster's newest kept conversation when it
  opens or the cluster changes; "New conversation" shows an empty one that the first message
  creates (`NewConversation`), so no empty conversation is kept. Without a cluster the last one stays
  readable, its input disabled; a conversation about another cluster says so and asks for a new one.
  Deleting one in the history asks nothing (it is that row's ×); Clear all asks.
- **The model under the input:** the connection pill's menu switches the conversation's connection
  (`SetConversationConnection`) and offers its connection's models; picking one changes the
  connection's model (`SaveConnection`, the key kept), for every conversation that uses it. Options:
  a model per conversation. Why: the contract has none (a gap named for the lead).
- **Preferences › Agent:** the last section of Preferences (`SectionAgent`, `agent`); the kind
  prefills LM Studio's `http://localhost:1234/v1` where the field is empty or holds another kind's
  prefill, the others show a hint; the model list is read for saved connections only (`Models` takes
  a connection's id: a gap named for the lead), any name may be typed; empty limits take the
  defaults, shown in the fields; the test runs with the form's values, unsaved, at most 30 s, and
  ends when the form closes. "Make Default" on a row, and "use for new conversations" in the form
  (saved, then `SetDefaultConnection`).
- **The answer's reasoning and the thinking line:** an answer that streams without text yet (the
  agent drops a reasoning model's reasoning deltas, which a local 27B model may send for 30–60 s)
  shows a muted "Thinking…" line whose dots move every half second and which, after 3 s, says how
  long the model has thought ("Thinking.. · 12 s"), so the column does not look stuck; it goes with
  the answer's first text. Options: a spinner icon, nothing (an empty item). Why: the contract has
  no item for reasoning, and a line that moves and counts is the cheapest proof of life (one timer
  through the main goroutine while it shows). The window also cuts `<think>…</think>` and
  `<thinking>…</thinking>` from an answer's text (the agent keeps reasoning out of `ChatItem.Text`;
  this is the window's second line). A Copy button under a finished
  answer copies its Markdown without the reasoning. Links open in the browser for http and https
  only; images are their alt text, never loaded.

**Built 2026-10-14 night, the private connection in the engine (M12 (c),
[`engine/readslikeuser.go`](../../internal/engine/readslikeuser.go)):** `Dispatch` honours
`Invocation.ReadsLikeUser` only with origin `builtInAgent`, carries it in the command's context
(`engine.ReadsLikeUser`, `engine.ReadsAsAgent`, `engine.PrivateTurn`), skips the H-82 filter for the
turn's results but frames a chart's README, and marks the turn's audit lines `readsLikeUser`; every
read path that withheld from agents by origin asks it, no write rule does. The sites, the rules
left and the tests are in [agents and MCP › H-94](../security/agents-and-mcp.md#h-94-a-private-connection-gives-the-model-what-the-window-shows);
what a private turn's kept conversation holds that the scrubber does not know is
[H-96](../security/agents-and-mcp.md#h-96-a-private-turns-reads-are-kept-where-the-scrubber-does-not-know-them)
(decided 2026-10-15 and built the same day: a private connection's conversations are not kept,
item 20 below).
The tool descriptions that say agents read a value withheld now say that the private connection
reads it as the window shows it, which put the core set at 22 tools and 29,746 bytes (measured
2026-10-14 after the change, `TestToolsetSize`; decision 2 above has the earlier measure).

**The lead's decisions for the owner to confirm** (night run of 2026-10-14, M12 (c); each with the
options weighed):

15. **A Secret's Edit or Create tab the user opened.** The private turn reads its text
    (`editor_get`, `editor_restoreDraft`) (chosen: it is what the window shows the user, and the
    owner's rule names "editor text"); or `<withheld>` as for every agent. Opening such a tab,
    setting, diffing, dry-running, applying or reverting its text and creating a Secret stay refused
    either way.
16. **A Secret in `object_get`.** For the private turn its Data section holds each value as the
    drawer's Show reveals it (text cut at 4 KiB and 40 lines, the cut said; binary by size) and its
    YAML the values as an Edit tab holds them (base64), a Helm release record in a ConfigMap its
    YAML likewise (chosen: the owner named `object_get` and the drawer, and rejected a question per
    Secret); or the Data section only (the YAML stays redacted), or the YAML only (a model reads
    base64 badly), or neither (the drawer as it looks before a click). `secret_reveal` stays a window
    action.
17. **The window's own `object_get` of a Secret** keeps the drawer's masked values (chosen: values
    reach the window's drawer model only through the user's Show; no caller needs them there); or
    the values for the GUI too.
18. **A chart install's preview for the private turn** renders against the cluster as the window's
    (`--dry-run=server`, `lookup` included), its error in full and kept out of the audit log
    (chosen: the offline rendering exists only to keep what `lookup` reads from agents, which the
    private turn may read); or offline as every agent's.
19. **An exec plugin's arguments** (`kubeconfig_execPlugin`, `kubeconfig_pendingApprovals`) stay
    withheld from the private turn (chosen: they can carry the user's own kubeconfig credential,
    which is not the cluster data the owner's rule names, and AGENTS.md's rule 9 names exec-plugin
    material); or shown as the window's approval dialog shows them.
20. **A private turn's kept conversation** keeps what the H-82 scrubber does not know (Helm's
    values, Flux's inline credentials, values under 10 bytes; H-96), as the owner decided the
    scrubber; recorded as a gap, not changed (options for the owner: keep no tool result of a private
    turn, or scrub its pieces also with the values the turn read). **Decided (owner, 2026-10-15):** a
    conversation over a private connection is not kept at all: memory only, gone when Cogpit quits.
    Chosen over the lead's recommendation (its tool results withheld, the rest kept), keeping it as
    built, and scrubbing with the values read. **Built 2026-10-15** with this rule (written at the
    `conversation` type in [`agent/conversation.go`](../../internal/agent/conversation.go)): a
    conversation becomes private when a turn over a connection marked private starts in it
    (`ChatConversationInfo.Private`; the user item's `ChatSent.Private` records the mark its turn ran
    with), and from that turn's start the conversations file is written without it; it stays
    private when it switches to a connection not marked private or the mark is taken off its
    connection, because its history holds what the private turn read and the next turn sends it; a
    conversation whose connection is private but which has no turn yet is not private (nothing to
    keep). A file of v0.8.0 to v0.10.0 loses every conversation with a user item marked private when
    it is loaded, and is rewritten at once through the owner-only writer. The window says it: a hint
    under Preferences › Agent's private mark, "not saved" in the chat's history (the lead's choice:
    the history otherwise reads as the list of kept conversations). Tests:
    `TestPrivateConversationIsNotKept`, `TestPrivateTurnUnkeepsTheConversationForGood`,
    `TestLoadDropsPrivateConversations`, `TestAgentPreferencesPrivateHint`,
    `TestChatHistoryMarksUnsaved`; what remains is in
    [H-96](../security/agents-and-mcp.md#h-96-a-private-turns-reads-are-kept-where-the-scrubber-does-not-know-them).

**The owner's audit answers of 2026-10-15, the chat's part** (decided by the owner on 2026-10-15
after using the chat in a real window; F8, selectable text in the chat, postponed by him; the
chat stays no command and no MCP tool, N3):

- **(4) Send all tools, per connection.** Off by default: a request carries the core set (the 20
  core tools, `search_commands` and `run_command`, decision 2). On: every command the chat may run
  as a tool of its own, for large cloud models. The same rules for every tool (D-2, the
  confirmations, the H-82 filter); a private connection reads as before. **Built 2026-10-15 (night
  run)**: `ChatConnection.AllTools`, kept in `llm-connections.json`; the core tools first in their
  order, then the rest in the registry's ([`agent/tools.go`](../../internal/agent/tools.go)
  `defsFor`), the system prompt's rule about the meta-tools replaced by "every command you may run
  is one of the tools"; measured (`TestToolsetSize`, 2026-10-15): 179 tools, 223,961 bytes, about
  56,000 tokens a request, against the core set's 22 tools, 30,447 bytes, about 7,600. Preferences
  › Agent: "Send all tools" with what it costs, the connection's row says "all tools". The lead's
  decision, **confirmed (owner, audit of 2026-10-16)**: with all tools on, no meta-tools. The options: (a) every
  command and no meta-tools (chosen: nothing is left to find, and the prompt's rule would point at
  a search the model does not need); (b) every command plus the two meta-tools (redundant, two tools
  more). Tests: `TestAllToolsSendsEveryCommand` (a command outside the core set read through the
  agents' filter, one that asks agents waiting for the user), `TestToolsetSize`,
  `TestAgentPreferencesAllTools`.
- **(6) A connection over `http://` to another machine** (localhost, 127.0.0.1 and ::1 exempt): a
  warning in the warning colour in Preferences › Agent and in the connection test, saying that not
  only the key but the conversation and every Secret or other secret the agent reads go
  unencrypted over the network; and a CA file per connection for an `https://` server with a
  private CA, its path kept with the connection, the file read when connecting, its errors named;
  for that connection the file replaces the system's roots (macOS: its trust store), as a
  kubeconfig's `certificate-authority` does for kubectl (**the owner's decision in the audit of
  2026-10-16**, N141, built the same day; the night run had added the file to the system's roots,
  the lead's decision for the owner to confirm). **Built 2026-10-15 (night run):**
  `engine.ChatPlainHTTPRemote` and `ChatPlainHTTPWarning`; Preferences › Agent shows the warning
  under the base URL while it is typed and marks such a connection "plain http" in the list; the
  connection test's HTTP layer passes with the warning (`ChatTestStep.Warning`: the warning icon
  and colour, the text under the layers). `ChatConnection.CAFile` (an absolute path, not secret,
  in `llm-connections.json`), [`agent/cafile.go`](../../internal/agent/cafile.go): the file is
  read whenever a request is sent (a turn, the test, the models list; at most 1 MiB of a regular
  file, its `CERTIFICATE` blocks only), its certificates the pool of `x509.NewCertPool()` alone
  (since 2026-10-16; the night run's `x509.SystemCertPool()` with the file appended), the client
  kept per connection and file digest. **Checked in Go 1.27.2's source** (`crypto/x509`
  `cert_pool.go`, `verify.go` `Certificate.Verify`): only a pool marked as the system's is handed
  to macOS's platform verifier (its trust store and the user's trust settings); a pool from
  `x509.NewCertPool()` is not, so Go's own verifier checks the chain against the file's
  certificates alone and no public CA vouches for the endpoint. The night run's pool kept the
  system's mark, so macOS was asked first and every public CA trusted beside the file. `SSL_CERT_FILE`
  and `SSL_CERT_DIR`, which since Go 1.27 replace the platform verifier for the system's roots,
  do not reach a connection with a CA file
  ([trust boundaries › H-104](../security/trust-boundaries.md#h-104-a-connections-ca-file-replaces-the-systems-roots-checked-by-gos-verifier)).
  An untrusted certificate fails the HTTP and TLS layer and a turn with "the endpoint's certificate
  is not trusted (…): for a server with a private CA, name its CA file in the connection's
  settings"; with a CA file set, "… this connection trusts only its CA file <path>, not the
  system's certificate authorities; for a server with a public certificate, clear the CA file in
  the connection's settings" (a server that moved to a public certificate); a CA file that does
  not exist, is not a file, holds no certificate or is larger than 1 MiB fails them with the file
  named, never its content. The lead's decisions, each with the options weighed, **confirmed
  (owner, audit of 2026-10-16)** as built, except that a CA file now replaces the system's roots
  for its connection (the owner's choice, N141):
  - **What is exempt from the warning:** the name `localhost` and every loopback address
    (127.0.0.0/8, ::1) (chosen: none of them leaves the computer); or exactly the three the owner
    named (127.0.0.2 would warn though it never leaves the computer); or also `*.localhost` names
    (not exempt: whether they resolve to loopback is the resolver's, not guaranteed).
  - **Where it warns:** under the base URL as the user types it, the list's "plain http" mark, and
    the connection test's HTTP layer, which passes with the warning (chosen: the owner named
    Preferences › Agent and the test; the mark keeps it visible after the form closes); or also a
    line in the chat per turn (noise at every message for a set-up the user chose), or a question on
    save (friction the owner did not ask for).
  - **A CA file with an `http://` base URL:** refused on save and by the test ("a CA file is for an
    https base URL") (chosen: it would do nothing, and silence would suggest a protection); or kept
    and ignored.
  - **The path:** absolute, the form expanding `~/`, with a Choose… button (the file dialog, `.pem`,
    `.crt`, `.cer`) (chosen); or relative to the configuration folder (where the user does not keep
    certificates).
  - **When it is read:** whenever a request is sent, the client kept while the file's content is
    the same (chosen: the owner's "read when connecting", and a renewed CA counts at the next
    request without a restart); or once at save (a renewed CA would need the form again).
  Tests: `TestCAFileTrustsAPrivateCA`, `TestCAFileErrorsAreNamed`, `TestCAFileReplacesTheSystemRoots`,
  `TestCAFileRefusesAPublicServer`, `TestPlainHTTPToAnotherMachineIsWarned`, `TestChatPlainHTTPRemote`,
  `TestAgentPreferencesPlainHTTPWarning`, `TestAgentPreferencesCAFile`,
  `TestAgentPreferencesTestWarns`; renders `preferences_agent_plainhttp_*.png`,
  `preferences_agent_*.png`.
- **F9: a sent message edited and sent again**, ChatGPT's form (the owner's choice): every user's
  message can be edited and sent again; everything after it is dropped from the window, the model's
  history and the kept conversation (the audit log keeps everything); a failed or stopped turn is
  retried the same way; the chip and `ui_state` come from the new send; the model learns nothing of
  the cut part (no note to the model, no warning when editing). **Built 2026-10-15 (night run):**
  `ChatAgent.Resend(conversation, item, message)`; the agent checks everything Send checks and
  that the item is a user's message before it changes anything (a refused resend leaves the
  conversation as it was), then cuts the items, their kept copies, the history from the user
  message of the same rank and the request bodies (`conversation.cut`), rewrites the conversations
  file at once, and starts the turn. The window: a pencil at the right end of the line under each
  of the user's messages opens it in an editor in the card's place (the entry outlined as the
  focused input, Cancel and Send under it); Enter or Send resends, Shift+Enter is a new line,
  Escape or Cancel leaves it; a refused resend keeps the editor and its text with the reason. What
  it leaves open, the cut part's writes leaving the model's view, is
  [H-100](../security/agents-and-mcp.md#h-100-a-cut-parts-tool-calls-leave-the-models-view). The
  lead's decisions, each with the options weighed, **confirmed (owner, audit of 2026-10-16)**, all seven as built:
  - **Editing while a turn runs:** the pencils dimmed with "Stop the running turn to edit and send
    again"; an editor opened before the turn started keeps its text, its Send disabled until the
    turn ends (chosen: one way to stop a turn, Stop, and no turn cut mid-way by an edit); or
    editing stops the turn and sends (one click, but a stop the user did not ask for, a tool call
    cut mid-way); or the edit sent once the turn ends (a queued send the user may have forgotten).
  - **Where editing starts:** a pencil always there, dimmed while it cannot be used, at the end of
    the "Sent …" line (chosen: discoverable and costs no height); or on hover only (ChatGPT's; Fyne
    gives no cheap hover per card, and it hides the feature); or a context menu.
  - **The keys:** as the input's, Enter sends, Shift+Enter a new line, Escape cancels, ⌘. stops
    (chosen); or a Send button only.
  - **When the input could not send** (no cluster, another cluster's conversation, a send under
    way): the pencil dimmed with the input's reason (chosen); or the editor opens and refuses at
    Send.
  - **The first message edited:** its title goes with it and the new text names the conversation
    (chosen: the title is the first message); or the old title kept.
  - **A private conversation:** stays memory-only after its private turn is cut (chosen: H-96's
    "for good", nothing private can reach the file through a cut); or kept again once no private
    turn remains (more state to reason about for a rare case).
  - **The file:** rewritten at once at the resend, without the cut part (chosen: the file never
    holds what the window no longer shows); or at the turn's end.
  Tests: `TestResendCutsTheConversation`, `TestResendRetriesAFailedTurn`,
  `TestResendTakesTheWindowOfNow`, `TestResendRefusals`, `TestResendKeepsAPrivateConversationPrivate`,
  `TestChatEditAndResend`, `TestChatEditCancel`, `TestChatEditWhileATurnRuns`,
  `TestChatEditFailureKeepsTheText`; renders `chat_edit_*.png`.
- **The live run of the night** (`TestLiveEndpoint` against the owner's LM Studio on another
  machine of his network, `qwen/qwen3.8-27b`, twice, before and after F9): every layer of the
  connection test passed, the HTTP layer with the plain-http warning, the tool layer in 34.5 s and
  12.3 s; the question about a namespace's pods ended both times at the loop guard (the model asked
  `resources_list` with the same arguments a third time; 3 and 4 requests, 99 KiB and 131 KiB), so
  no answer. The night's work leaves what a request carries without "send all tools" as it was (the
  core set 30,447 bytes before and after it, the same system prompt); on 2026-10-14 (29,746 bytes)
  one call answered: taken as the model's behaviour, recorded, not chased.

## D-11: Crash stance

**Decided (owner, 2026-10-11):** a Cogpit crash is a bug, but not a catastrophe. The cluster's
state lives in the cluster and survives; PhotoCraft's "never crash, it loses the user's work"
priority does not carry over as is. Consequences:

- **The UI is crash-tolerant.** Panics are fixed like other bugs; a crash is not a release blocker
  on its own. Goroutines that run cluster I/O still recover from panics at their boundary and
  report an error, because one panicking watch must not close every open cluster.
- **Cluster operations are crash-safe.** Some operations are several API calls in a row and leave
  the cluster between states if the process dies halfway:
  - Helm install, upgrade and rollback run in-process (D-3) and leave a release in
    `pending-install`, `pending-upgrade` or `pending-rollback`, which blocks the next Helm operation
    until it is resolved;
  - drain cordons the node, then evicts pods one by one, and leaves a cordoned, half-drained node;
  - an interactive shell in a pod ends with the connection, and the processes it started may end
    with it.

  Each such operation either runs in a worker process that survives the UI, or detects the
  half-finished state on the next start and offers to finish or revert it. Which of the two, per
  operation, is designed with the operation. Decided with the operations: drain (2026-10-11) and
  Helm rollback and uninstall (2026-10-11, third run) detect it: a journal in the user's config
  directory (`drains.json`, `helm.json`, mode 0600) records the operation before it starts, and the
  next connect of that context offers to finish (run it again) or dismiss it. Helm upgrade
  (decided and built 2026-10-12, the same road): the journal records the upgrade (never its
  values), and one interrupted with the release `pending-upgrade` is offered as a rollback to the
  revision it started from, since the Helm SDK's upgrade refuses a pending release and its
  rollback does not. The journals are locked between Cogpit processes, each entry naming its
  owning process (H-22). Helm install is open; built 2026-10-13 on the same road as the lead's
  default until the owner decides: the journal records the install, never its values, and one
  interrupted with the release `pending-install` or failed is offered as an uninstall.
  **Decided (owner, 2026-10-14 evening, A17):** nothing is offered after an interrupted install;
  Cogpit shows the release's state only, and the user cleans up in the Helm Releases view. **Built
  2026-10-14:** the Helm banner's row of an install says what happened and the release's state
  (pending-install, failed, at its revision) with Show release and Dismiss, no Uninstall; a failed
  install's error and `helm.pendingOperations`'s description point to the Helm Releases view, never
  to running the install again; rollback, upgrade and uninstall entries are unchanged. Settled at
  the build, the lead's decisions for the owner to confirm: (1) the journal still records an
  install (the options: record it as before, or record nothing once nothing is offered; recording
  keeps a second Cogpit process from starting an action on the release while the install runs,
  H-22, and is what tells the user after a crash that an install was left, while recording nothing
  would leave only the Helm Releases view's status); (2) the row offers Show release, which opens
  the release in the Helm Releases view (`ui.navigate`): a way to where the owner said the user
  cleans up, not an action on the release; a release that no longer exists gets Dismiss only.
  **An interrupted connection is handled the same way** (owner, 2026-10-15 evening, [D-58](#d-58-standby-and-lost-connections)
  S8; built 2026-10-15/16): a drain or a Helm install, upgrade or rollback running when the
  connection is interrupted stops there (a drain's job stops, [D-63](#d-63-long-commands-as-background-jobs);
  a Helm action's context ends, `Session.OperationContext`), its entry stays, and once the
  connection is back the banner offers Finish or Dismiss against the cluster as it is then; nothing
  continues by itself. **A drain stopped while Cogpit runs is handled the same way** (2026-10-16,
  night run, D-63, the lead's decision, confirmed by the owner in the audit of 2026-10-16): a drain runs as a background job,
  and when the job is stopped (Cancel, its tab closed, another cluster, the interruption, Cogpit
  closing) the journal keeps the drain as `stopped` with why, which the banner and
  `node.pendingDrains` offer as an interrupted one (Finish drain, Uncordon, Dismiss); nothing is
  undone or uncordoned by itself. The journal is written and cleared exactly as before.
- **Unsaved editor content** is kept in a local draft so that a crash does not lose it. **Built
  2026-10-13 (night run)**, with defaults of that run that the owner has not decided: an Edit or
  Create resource tab's unsaved text is written 2 seconds after it stops changing (at least every
  10 seconds while it changes) to `<user config dir>/Cogpit/drafts/<id>.json` (0600, atomic,
  naming its process as the journals do, H-22), removed when the text is saved again or the tab
  is closed (closing discards it, without asking), and written on a quit; the next connect of its
  context offers it (Restore, Discard; `editor.drafts`). A tab that closes without the user, its
  context having left Cogpit (its kubeconfig source removed, its file gone), keeps its draft,
  written at once (2026-10-14, Cogpit's default, `Drafts.keep`). Never a Secret's text and never Helm
  values (D-19). [Architecture §4.4](architecture.md#44-tables-details-editors-and-streams),
  [credentials](../security/credentials.md#where-they-live-inside-cogpit).
- **Crash reports (built 2026-10-12, after a crash in the owner's smoke test left nothing to read):**
  the Go runtime writes a crash's text into `<user config dir>/Cogpit/crash.log` as well (mode 0600,
  `runtime/debug.SetCrashOutput`), which a clean exit removes; the next start moves a report left
  behind into `crashes/` (the newest ten kept) and says where it is. Nothing is uploaded
  ([credentials](../security/credentials.md)). The same day it showed that every quit on macOS
  crashed in the title bar's cleanup (fixed, [G16](../parity/gaps.md#g16-stability-and-the-user-bug-backlog)).

## D-12: Parity sources

**Decided (owner, 2026-10-11):** the target is the current Lens Desktop. Rows in the parity
checklists come from Lens Desktop's documentation and release notes, and, where those are silent,
from a capture of the running Lens Desktop. OpenLens is outdated (its source was retired in January
2024) and is not a source, and neither is its fork Freelens. No code, text or asset from Lens
Desktop, OpenLens or Freelens is used.

## D-13: Cache

**Decided (owner, 2026-10-11):** Cogpit caches intensively so that the UI is fast and the API
server carries as little load as possible while the view stays current. Every change of cluster
state is written into a cache, and the UI consumes only the cache. The cache is kept current by
watches; what cannot be watched is refreshed as often as needed and no more. The design is
[architecture §5](architecture.md#5-cache): one shared, reference-counted informer per (cluster,
kind, namespace scope), warm for a grace period after its last consumer, no resync, bookmarks,
streaming initial lists, metadata-only informers where enough, `managedFields` dropped, discovery
and access reviews cached, polling only while visible, request counters to prove the load.

**Persistence across restarts, decided (owner, 2026-10-11): in memory only.** Writing the cache to
disk would show the last known state immediately at start-up; the options weighed:

| Option | For | Against |
|---|---|---|
| In memory only (recommended) | No stale state shown as current; no cluster data at rest | Every start lists again |
| Persist all kinds | Fastest start | Secrets and other sensitive objects at rest; stale state until the first relist |
| Persist non-sensitive kinds, mark as stale until relisted | Fast start for the common views | Complexity; the line between sensitive and not is the user's, not Cogpit's |

**Decided:** in memory only. Revisit only if the measured cold start (budget P7) is too slow, and
then with this table as the starting point. Since 2026-10-16 the discovery answer (which groups and
resources the server serves, no object) is kept on disk for the next connection
([D-65](#d-65-discovery-cached-on-disk)); the objects stay in memory only.

**Secrets in the cache, decided (owner, 2026-10-11): metadata only, plus Helm's release Secrets.**
Listing or watching Secrets returns their values ([H-8](../security/privilege-footprint.md#h-8-secret-data-arrives-with-the-list)),
and the Secrets view and Helm both need Secrets. The options weighed:

| Option | For | Against |
|---|---|---|
| (a) A full watch on Secrets | The Keys column as in Lens; Helm reads its releases from the same cache | Every Secret value of the watched namespaces in Cogpit's memory, masked only in the UI |
| **(b) Metadata only for the list, a GET when a Secret's drawer opens, a watch on `owner=helm` for Helm (decided)** | Only Helm's release payloads, which Helm needs anyway, are held as data; one Secret's data is fetched when the user opens it | The Keys column cannot be filled from metadata: a documented difference from Lens |
| (c) No cache for Secrets, a GET whenever needed | Nothing held | Breaks cache-first: the list costs a request per view |

**Built (2026-10-11, third run)** as decided below, with one difference the decision did not
foresee: the Type column cannot be filled from metadata either (`type` is a top-level field), so
Keys and Type both stay empty. The server-side Table format (`as=Table`, `includeObject=Metadata`)
would carry the type and the number of keys without values, at the cost of an informer that holds
table rows and a Keys column that counts instead of naming; whether to use it is open for the
owner. A second finding: the `kubectl.kubernetes.io/last-applied-configuration` annotation repeats
a Secret's values in its metadata; the cache drops it from every Secret before the store.

**Revised (owner, 2026-10-12): (a), a full watch on Secrets.** The Secrets view will list and
watch Secrets in full, so that its Keys and Type columns are filled as in Lens. This revokes the
choice of (b) below, which is built (2026-10-11) and stays until (a) is: the full scope replaces the
metadata-only one, the drawer reads the opened Secret from the cache instead of its own GET, and
Helm's releases can come from the same informer while it runs (the `owner=helm` scope stays for when
it does not, so that the Releases view alone never holds every Secret). Every value of the watched
namespaces is then in Cogpit's memory while the scope runs, its warm period included; the UI keeps
values masked until the user reveals one, and agents keep getting them redacted
([H-8](../security/privilege-footprint.md#h-8-secret-data-arrives-with-the-list)). Whether Secrets
can then be edited from YAML is a separate decision, not made.

**Built (2026-10-12, fourth run)** as revised: the Secrets view lists and watches every Secret in
full (Keys and Type filled), the drawer reads the opened Secret from the cache (the GET is gone),
values stay masked and `secret.reveal` GUI-only, and Helm's releases read from the full watch while
it runs and from their own `owner=helm` watch otherwise, through the covering scopes of
[architecture §5](architecture.md#scopes-local-and-virtual-kinds). Edit and Create from YAML stay
refused for Secrets.

**Decided before (owner, 2026-10-11, revised above):** (b). The Secrets view lists through a metadata-only informer
(`PartialObjectMetadata`); the drawer's GET of one Secret is the direct read architecture §5
allows for data the cache does not hold, and its values stay masked until the user reveals a
field; Helm's releases come from a label-selected informer (`owner=helm`) whose Secrets are held in
full. RBAC cannot narrow `list secrets` by label, so the verbs are unchanged.

## D-14: Terminal widget

**Decided (owner, 2026-10-12):** keep the patched copy of `fyne-io/terminal` in
`third_party/fyne-terminal`, offer its patches upstream, and fuzz the widget on its own, without
Cogpit's escape filter in front of it ([H-19](../security/trust-boundaries.md#h-19-the-terminal-widget-trusts-its-input)).
The options weighed:

| Option | For | Against |
|---|---|---|
| **(a) Keep the fork, offer the patches upstream, fuzz the widget alone (decided)** | Closes the largest open risk, inputs nobody has found yet, for a few hours; the patches are small and general | The fork stays to maintain until upstream merges equivalent fixes |
| (b) Keep the fork as it is | Nothing to do | Unknown crashing inputs stay untested; the fork drifts from upstream |
| (c) Cogpit's own terminal widget on a maintained VT emulator | Full control over hostile input, colours from the theme | 20–40 h; replaces tested code; budget P5 to prove again |
| (d) Back to upstream without the patches | – | The known crashes come back |

Upstream's head asks in its `go.mod` for an unreleased Fyne (v2.8.2), but its code builds and passes
its tests on Fyne 2.8.1: Cogpit's copy moved onto it on 2026-10-12 and keeps its `go.mod` at 2.8.1
([PATCHES.md](../../third_party/fyne-terminal/PATCHES.md)). Returning to upstream waits for the
patches to be merged; (c) is the step to take only if fuzzing shows the
parser cannot be kept. Two inputs still made the patched widget alone panic when this was decided
(a CSI whose final character is not ASCII, a set mode without `?`; only the filter caught them):
they were the first patches to add.

**Built (2026-10-12):** both patched, and the widget fuzzed alone (`FuzzHandleOutput` in the copy,
`FuzzRawWidget` in `termview`); the fuzzing and the review that came with it found and fixed a
panic with debug output on, five sequences whose counts made the widget loop for as long as the
count, strings parsed in quadratic time, an APC that never ended, and data races between its two
goroutines ([PATCHES.md](../../third_party/fyne-terminal/PATCHES.md)). Its tests and fuzz seeds run
in `make test-unit-race` and CI (`make test-fork`). The patch series is offered upstream since 2026-10-12 (owner's OK): pull requests [#155](https://github.com/fyne-io/terminal/pull/155) (crashes), [#156](https://github.com/fyne-io/terminal/pull/156) (hangs), [#157](https://github.com/fyne-io/terminal/pull/157) (OSC 7) and [#158](https://github.com/fyne-io/terminal/pull/158) (races and the fuzz test), 20 commits, each fix with a test that failed before it. The fork stays until upstream has equivalent fixes. Nothing found so far calls for option (c).

**Decided how (owner, 2026-10-12):** the lead prepares the upstream series as three or four grouped
pull requests (crashes, hangs and performance, races, OSC 7), each fix one commit with a test in
upstream's style; the owner looks at the series once; after the owner's OK the lead opens the pull
requests under the owner's GitHub account. Nothing is published before that OK. The options
weighed: the lead opens them after the owner's OK (decided), the owner opens them, issues with
reproducing inputs instead of pull requests, or keeping the fork for good (which would reverse
the decision above).

**Added (2026-10-13, night run; candidates, not offered):** two patches for terminal input, each
with a test in the copy, marked `Cogpit patch:`: what the right mouse button does is the
application's choice (`SetRightClickAction`: paste the last selection, upstream's behaviour and
the default; nothing; or copy the selection, else paste the clipboard), which Cogpit's terminal
preference needs; and ⌃Ins and ⇧Ins copy to and paste from the clipboard off macOS instead of the
widget's own selection buffer. A variable `goos` lets the tests on one system cover the keys of
the others ([PATCHES.md](../../third_party/fyne-terminal/PATCHES.md)). They are offered upstream
like the others once the owner looks at them.

**The decision round of 2026-10-14 evening (owner):**

- **Later (A86):** the two newer patches are offered upstream, but at the end; the app comes first.
- **Confirmed (A24, as built):** what the right mouse button does in a terminal stays the user's
  preference alone (`ErrRightClickGUIOnly` refuses agents and the CLI).
- **Confirmed (A85, as built):** on Windows and Linux Ctrl+C always goes to the shell;
  Ctrl+Shift+C and Ctrl+Ins copy.
- **Decided (A116), built 2026-10-15:** the fork's four shell tests run under the race detector in
  CI: `make test-fork`, which the job Race Tests (other packages) runs first, has no `-skip` any
  more ([`Makefile`](../../Makefile)). They start the widget's `RunLocalShell`, which Cogpit does not
  use. Under `-race` they showed one race: the time of the last key, which
  `TestRunLocalShell_MissingStartDir` wrote while an earlier test's directory check read it. A new
  test that runs a local shell on a stand-in for the desktop driver's main goroutine
  (`TestRunLocalShell_MainLoop`; Fyne's test driver runs `fyne.Do` on the calling goroutine, which
  hides the rest) showed six more: the size polled and the directory, the input and the
  pseudo-terminal written on the caller's goroutine, the directory written by the directory check
  on its ticker's goroutine, and `ExitCode` read while `cmd.Wait` wrote the shell's state. All seven are patched in the fork, each field kept on
  one goroutine as the patches of 2026-10-12 do, the last key's time an atomic and `ExitCode` under
  the widget's lock; the known race of `content` needed no code
  ([PATCHES.md](../../third_party/fyne-terminal/PATCHES.md#data-races-h-19)). Measured: the fork's
  tests passed 20 runs under `-race` with no race reported on macOS (the shell tests alone also at
  `-cpu 1,2`, 10 runs each) and on Linux (`golang:1.27.2` container, the whole module); against the
  code before the patches the new test reports all seven.
- **The lead's decision for the owner to confirm (night run 2026-10-15):** the new race patches
  are upstream candidates not offered yet, to be offered with the two input patches at the end
  (A86). The options: (a) with the input patches at the end (chosen: A86 puts the app first and
  groups the later patches, and the owner looks at a series once before anything is published);
  (b) now, as commits added to [#158](https://github.com/fyne-io/terminal/pull/158), the open
  races pull request (changes a series the owner already approved, without his look); (c) kept as
  Cogpit's own (`RunLocalShell` is upstream's code, Cogpit gains nothing from carrying them alone).
- **The lead's decision for the owner to confirm (night run 2026-10-15):** the fork's tests run
  under the race detector in the Linux job only, as before. The options: (a) the Linux job Race
  Tests (other packages) only (chosen: it meets A116, and adds no time to the macOS job); (b) also
  `make test-fork` in the macOS job (`make test-darwin`), which would cover the fork's macOS-only
  directory lookup (`term_darwin.go`) and a macOS pseudo-terminal under `-race` in CI, about
  10 seconds more there (estimated: the module's tests take 8 s under `-race` on the M1 Ultra).
  Locally on macOS they run either way.

## D-15: Progress generators

**Decided (owner, 2026-10-12):** `make parity` is built first, as a counter and checker of the
hand-written checklists that runs in CI: it counts the views, actions and interface rows the
progress pages report, and fails on a row that appears twice, a row marked partial or done without
evidence, and a command id in a row that the registry does not know. The generated
`parity-checklist.md` and the scorecard (`make scorecard`) come later, the scorecard once
performance budgets are measured regularly on a fixed runner. The options weighed: both generators
as planned (6–14 h, a scorecard with hardly any data yet), the checker alone (3–6 h, decided),
counting by hand for good (0 h), after the alpha (0 h now). Why the checker first: merging the
third run's eight branches put the same parity rows in twice, twice, and the counts needed
throwaway scripts whose denominator (121 or 123 free interface rows) could not be reproduced.

**Built (2026-10-12):** [`hack/parity`](../../hack/parity/main.go), run by `make parity` as a CI
step after the lint; it also refuses links to missing files and unknown test names, and compares
its counts with the numbers ROADMAP and the progress pages report. The 121 holds: the 123
counted the two paid keyboard shortcuts, whose table had no Plan column
([rules](../parity/README.md#make-parity)).

**Built (2026-10-16, night run; the lead's choices for the owner to confirm):** the generated
[`parity-checklist.md`](../parity/parity-checklist.md) from [`hack/parity`](../../hack/parity/checklist.go):
every row of the four checklists with its status (the first word) and its section, linked, and the
counts. A plain `go run ./hack/parity` writes it when it differs; `make parity` passes `-check`, under
which a page that no longer matches the checklists is a problem (CI). Options weighed: written only
on a flag and checked in CI (every checklist edit needs the flag remembered); generated in CI and
never committed (the page would not be on GitHub to read).

**Built (2026-10-16, night run; the lead's choices, those on how the tests report, where the
limits live, P8 and a run by hand rather than in CI confirmed by the owner in the audit of 2026-10-16,
the table's other rows for the owner to confirm):** the scorecard,
[`scorecard.md`](../parity/scorecard.md), from `make scorecard` ([`hack/scorecard`](../../hack/scorecard/main.go)):
the budget tests and benchmarks of [testing › Performance budgets](testing.md#performance-budgets-proposed-not-measured)
run without the race detector; per budget its proposed target, every measured value with its limit,
pass or fail, the date and the machine, and the budgets nothing measures with the reason. The tests
record through [`testkit/scorecard`](../../internal/testkit/scorecard/scorecard.go) (`Record`, one call
per measurement, in `TestBudgetFirstRows`, `TestBudgetIdleViewAndWarmReopen`,
`TestShowcaseServiceByClusterIP`, `TestBudgetNamespaceLimitedUser`, `TestNodesListDiskBudget`,
`TestClusterBudget`, `TestBudgetAccessibleNamespaces`, `TestBudgetAgentReadTakesTheSecrets` and
`TestLogsP5`); five benchmarks of the engine and the editor are read from their `ns/op`. The first
run (2026-10-16, Apple M1 Ultra, 20 CPUs): 13 budgets measured, every one within its limits (32
measurements, 5 of them context); P3, P6 and P7 not measured. Still waiting, as decided: the run in
CI on a fixed runner, once budgets are measured there regularly.

| Question | Options weighed | Chosen | Why |
|---|---|---|---|
| How the tests report | (a) the generator parses their `t.Logf` lines; (b) one JSON line per measurement through a helper, written when `COGPIT_SCORECARD` names a directory | (b), `scorecard.Record` in a leaf package of `internal/testkit` | Log lines change their wording with every edit; the helper imports nothing of Cogpit's, so the window's tests may call it (depguard keeps the cache, which `testkit` imports, out of `internal/ui`) |
| Where the numbers of the limits live | parsed from testing.md's prose; in each call | in each call, testing.md's target printed beside them | The targets are prose ("1 list per 60 s", "no frame over 16 ms"); the number is what each test can judge |
| Benchmarks | a call in each benchmark; their `ns/op` parsed | parsed | Benchmarks print a machine-readable line already; they stay as they are |
| The race detector | runs with and without; without | without; a line from a race build fails the run | Budgets are judged without it ([testing](testing.md#performance-budgets-proposed-not-measured)); the tests' own bounds stay wide for it |
| A listed test that records nothing | ignored; the run fails | the run fails, naming the test | A renamed or skipped test would drop its budget from the page silently |
| P8 | the keystroke with a full software paint of the window (25 ms); the editor widget's work alone (0.015 ms) | the widget's work against 16 ms, the paint as context | The software painter rasterises every text object on every capture, which a GPU frame does not (the benchmark's own note); the real window's frames are G15's |
| The cluster tier's counts | part of a run (needs kind); left out | left out, said on the page | A run needs nothing but Go; testing.md keeps the kind numbers |
| In CI | a job; by hand | by hand, the page committed | D-15: once budgets are measured regularly on a fixed runner; the shared runners would measure their own load |
| The machine on the page | the host name; GOOS/GOARCH, CPU model and count, OS version, Go | the latter | No host name |

## D-16: The first real release

**Revised (owner, 2026-10-14): only the complete product, the CLI inside the app.** "Ich widerrufe
die Entscheidung cogpit-cli in alle Releases aufzunehmen. Es soll nur noch das komplette
Software-Produkt ausgeliefert werden. Aktuell nur macOS als dmg, das passt für mich. Die CLI muss
in Cogpit.app inkludiert werden." So a release carries the disk image
`Cogpit-<v>-macos-arm64.dmg` and `SHA256SUMS` over it, nothing else; without the Apple secrets (a
fork) the unsigned app zip in its place, the note saying so. The `cogpit-cli` archives for six
platforms (2026-10-13, below) are revoked: releases up to v0.6.0 carry them where their pipeline
finished (v0.5.0 has no files; v0.6.0's run was in progress when this was written), later
releases do not, and Linux and Windows get Cogpit with their desktop app (M11). `cogpit-cli` comes inside `Cogpit.app`,
built with the app's version and architecture by `make app`, signed inside-out with the app,
covered by the disk image's notarization and replaced with the app by Sparkle's updates
([releasing](releasing.md#the-release-pipeline), [updates](../security/updates.md)). Built by
agent CLIBUNDLE on 2026-10-14 with these decisions, each for the owner to confirm:

- **Where in the bundle: `Contents/MacOS/cogpit-cli`** (agent CLIBUNDLE). Apple's "Placing
  Content in a Bundle" lists "help app, helper tool" under `Contents/MacOS` and `Contents/Helpers`.
  Chosen: `Contents/MacOS`, beside the app's own executable, so the app finds it as the file next
  to its running executable without knowing the bundle's layout, and the path to type is
  `/Applications/Cogpit.app/Contents/MacOS/cogpit-cli`. Option weighed: `Contents/Helpers`, which
  keeps `Contents/MacOS` for the main executable alone, at the cost of a second code location the
  signing script and the app would both have to know. Either is sealed and notarized the same way.
- **Its code-signing identifier `io.guided-traffic.cogpit.cli`** (agent CLIBUNDLE), under the
  app's `io.guided-traffic.cogpit` as a part of it. Option weighed: `io.guided-traffic.cogpit-cli`,
  the standalone binary's identifier until now (v0.6.0's macOS archives carry it if that release's
  signing run succeeds), which reads as a product of its own. Nothing in Cogpit keys on it; it is
  part of the binary's designated requirement.
- **The bundled binary is `make build`'s `bin/cogpit-cli`** (agent CLIBUNDLE): the same
  `GO_LDFLAGS` as the app's binary (in a release `-s -w -X …/internal/buildinfo.version=<v>`), so the same version,
  `CGO_ENABLED=0`, the bundle's architecture (checked with `lipo`), `minos 13.0` (checked with
  `otool`, Go's linker writes it). Option weighed: building it again for the bundle with
  `-trimpath`, as `make dist-cli` did, which compiles every package a second time in a release job
  without a build cache, while the app's own binary carries no `-trimpath` either.
- **`make dist-cli` removed** (agent CLIBUNDLE): it built only release archives, and keeping it
  would have kept a platform list, a target and documentation for files no release carries, and
  `dist-checksums` would have summed its archives. A `cogpit-cli` for another system is
  `CGO_ENABLED=0 GOOS=<os> GOARCH=<arch> go build ./cmd/cogpit-cli`; `make build` builds
  `bin/cogpit-cli` as before.
- **The release job runs the bundled CLI, the signing job does not** (agent CLIBUNDLE): "Cogpit.app
  (macOS arm64)" checks `cogpit-cli version` against the release's version, its architecture and
  its identifier; "Sign and submit (macOS)" (until 2026-10-15 "Sign and notarize (macOS)") signs it without running it, so no code of Cogpit's or
  its dependencies' runs beside the identity.
- **The app offers the stdio registration of its own `cogpit-cli`** (the lead's default):
  [D-2](#d-2-agent-interface-scope).
- **The version a build reports (2026-10-14, the owner's finding F2):** one variable,
  [`internal/buildinfo`](../../internal/buildinfo/buildinfo.go)'s `version`, which the release
  build sets for both binaries (`DIST_LDFLAGS`, `-X
  github.com/guided-traffic/cogpit/internal/buildinfo.version=<v>`; `TestMakefileSetsTheVersion`
  checks the path, because the linker ignores a `-X` for a symbol that does not exist), read by
  About Cogpit, `cogpit-cli version`, the MCP server and the stdio bridge. Without it a release tag
  Go recorded as the module's version counts (`go install …@v0.7.0`); every other build reports
  `dev` and the commit Go recorded (`vcs.revision`, `vcs.modified`), nothing asked at run time.
  `make dist-app` refuses to run without `VERSION=`, so no local build claims release 0.0.1. The
  bundle's own versions stay `VERSION` (`CFBundleShortVersionString`, and `CFBundleVersion` for
  Sparkle; 0.0.1 for a local `make app`, which has no update key, so Sparkle never compares it).
  Options weighed: `main.version` in each binary handed down through options (two defaults to keep
  alike; the UI package may not import `internal/app`, where the shared wording would live);
  `git describe` through the linker (`v0.7.0-5-g0691fb3` tells the distance from the tag, but only
  `make` builds would carry it, and Go records the commit in every `go build` anyway); Fyne's
  `SetMetadata` from the build (the app only, `cogpit-cli` has no Fyne).

**Revised (owner, 2026-10-13): releases are live now.** After every gate on `main` is green,
semantic-release creates the tag and the GitHub release with the organisation's GitHub App token,
and the release pipeline (`build.yml`, `on: release: published`) builds and attaches the artifacts:
`cogpit-cli` for six platforms, the unsigned `Cogpit.app` for Apple silicon, `SHA256SUMS`
([releasing](releasing.md#release-flow)). The first releases are minor releases under v0, so the
owner can judge whether the pipeline runs end to end; a breaking change makes a minor release, not
1.0.0, until the owner decides on 1.0 (`releaseRules` in [`.releaserc.json`](../../.releaserc.json),
needed at once: commit `f3ad000` has a body line starting with `BREAKING CHANGE:`, which made the
dry run's next version 1.0.0). Signing and notarization follow now that the Apple credentials exist
(2026-10-14, [D-4](#d-4-packaging-and-updates)). **Dependency updates (owner, 2026-10-13):** Renovate commits
them as `build(deps):`, so they make no release of their own and ship with the next `feat:` or
`fix:`; the options weighed: `fix:` and a patch release per update (the team's convention), or
`fix:` only for security updates.

**Superseded (owner, 2026-10-12):** semantic-release stays a dry run until the alpha. The alpha's
release is the first real one, cut together with the job that builds the unsigned macOS app as a
download, so that the first version (0.1.0) has something to install; that job is part of the
alpha's work ([releasing](releasing.md#release-flow),
[G14](../parity/gaps.md#g14-packaging-signing-and-updates-macos)). The options weighed: the dry run
until the alpha (decided), releasing now as the team's other repositories do (a release on every
push to `main`, without downloads), releasing with a minimal artifact job now, or releasing by hand
once per run.

## D-17: Token files named by added kubeconfigs

**Decided (owner, 2026-10-12):** a context of an added or synced kubeconfig whose user names
a `tokenFile` outside the kubeconfig's own folder connects only after the user allowed it, in the
same GUI dialog and with the same approval store as a credential plugin
([H-24](../security/trust-boundaries.md#h-24-a-dropped-kubeconfig-can-send-a-local-file-as-its-token),
[H-1](../security/trust-boundaries.md#h-1-kubeconfig-files-added-by-sync-folders)): the dialog
names the file and the server its content goes to, the approval is bound to the path (and, as for
plugins, to the server and its trust), and only the GUI can give it. A token file in the
kubeconfig's own folder comes from the same source as the kubeconfig and needs no approval; the
default kubeconfig keeps kubectl's behaviour. **Built 2026-10-12** (`cluster.BindPluginFiles`,
`checkTokenFilePolicy`, [`kubeconfig/exec.go`](../../internal/kubeconfig/exec.go)); what remains is
in H-24. The options weighed:

| Option | For | Against |
|---|---|---|
| **(a) Ask, as for plugins (decided)** | Closes the gap without breaking a legitimate setup; one mechanism for everything a kubeconfig does on the user's computer; 2–4 h | One more dialog for setups that use token files |
| (b) Refuse for synced files only | No dialog | A legitimate synced setup breaks without recourse; added kubeconfigs stay open |
| (c) Refuse every file reference of an added kubeconfig | Strictest | Breaks the many kubeconfigs that name certificate files, which send nothing that matters (a client key is never sent; a client certificate only in the TLS handshake) |
| (d) Document only | Nothing to do | The gap stays live from the moment a file lands in a synced folder |

## D-18: Secret annotations seen by agents

**Decided (owner, 2026-10-12):** agents keep seeing a Secret's annotations in full (except
kubectl's `last-applied-configuration`, which the cache drops); the risk that a tool copies a
Secret's manifest, values included, into an annotation is documented and accepted
([H-23](../security/agents-and-mcp.md#h-23-secret-annotations-other-than-kubectls-reach-agents)).
Only `object_get` shows annotations to agents; lists, CSV and search do not. The options weighed:

| Option | For | Against |
|---|---|---|
| (a) Withhold every Secret annotation's value from agents | Safe without knowing the tools; 1–2 h | Agents lose harmless values (a certificate's issuer) |
| (b) An allowlist of harmless annotations | More useful to agents | A list to maintain; 2–3 h |
| (c) A denylist of known manifest copies | Keeps everything else | Never complete: an unknown tool slips through |
| **(d) Document only (decided)** | Agents see what the user sees in the drawer; nothing to build | A Secret deployed by a tool that copies its manifest into an annotation reaches agents with its values |

## D-19: Editing and creating Secrets from YAML

**Decided (owner, 2026-10-12):** Secrets can be edited and created from YAML in the GUI only. An Edit
or Create resource tab that holds a Secret shows the user its values (base64 in `data`, plain text
in `stringData`) and works like any other editor tab (diff, server-side dry run, the
resourceVersion precondition); agents and the CLI can neither open such a tab, nor read its text
(`<withheld>`), nor apply or create a Secret; values never reach the audit log, an error or a
toast. The same pattern as `secret.reveal` and the Helm Upgrade tab. Lens's Secret editor is not
captured (inferred). **Built 2026-10-12** ([`secretedit.go`](../../internal/engine/secretedit.go)).
Since 2026-10-13 (the drafts of D-11, a default of the night run that follows from this decision):
a Secret's tab, and any tab whose text holds a Secret's manifest or may be one being typed, has no
draft on disk ([`drafts.go`](../../internal/engine/drafts.go), `TestNoDraftsForSecretsOrHelmValues`).
Since 2026-10-13 (night build, G9) the same holds for Lens's Create Secret dialog: `secret.create`
is an MCP tool like every command but refuses every origin but the GUI before anything is asked or
sent, and the audit line of a refused call names the Secret, its type and its keys, never a value
([`secretcreate.go`](../../internal/engine/secretcreate.go), `TestSecretCreateIsGUIOnly`). Cogpit's
default, not the owner's decision: the dialog sends the values base64-encoded in `data`, not in
`stringData` (options: (a) `data`, (b) `stringData`; (a) because what is sent is what the API server
stores and returns, as `kubectl create secret` sends it, and the scrubbing of an API message knows
each value in both forms). Since 2026-10-15 (the owner's audit answer) a Secret's values are edited in its
drawer too, after Show, under the same rule: the window's only ([D-59](#d-59-a-secrets-values-edited-in-the-drawer)). The options weighed:

| Option | For | Against |
|---|---|---|
| **(a) GUI only (decided)** | Lens's Edit for the user; values never reach an agent | Agents cannot change a Secret |
| (b) Agents too, writing values they bring without reading existing ones | Agents can rotate credentials | Keeping withheld values on save is complex; an agent can overwrite credentials |
| (c) Metadata only (labels, annotations) | Little risk | Half of Lens's Edit |
| (d) Keep refusing | Nothing to build | Users need kubectl to change a Secret |

## D-20: A port forward that cannot connect

**Decided (owner, 2026-10-12):** a forward whose connection to the pod fails with an error that
does not go away by waiting (forbidden, unauthorized, the namespace gone) turns Failed at once,
releases its local port and names the cause; an error that can pass (the network, a pod still
starting, a server error) is retried with the backoff as today. Today a forward that first waited
for a pod and then could not connect stays Reconnecting and retries for ever
([`portforward_run.go`](../../internal/engine/portforward_run.go)). Lens's behaviour is not
captured. **Built 2026-10-12** ([`cluster/portforward_errors.go`](../../internal/cluster/portforward_errors.go),
`TestPortForwardFailsAtOncePermanentErrors`, `TestDialErrorsAreClassified`). The permanent errors,
as the API server answers the portforward request: **404 Not Found** (the pod does not exist, or
its namespace is gone), **403 Forbidden** (no `create` on `pods/portforward`) and **401
Unauthorized** (credentials rejected after client-go's refresh). client-go's SPDY dialer returns
those as a plain error that keeps the message and loses the status, so Cogpit records the status of
the dialer's last response (`DialRefusedError`). Retried as before: a network error, a dial
timeout, 5xx, 429, and 400 (the answer while a pod is not running yet), and an error without a
status (a proxy resetting the connection). A refused connection to the pod's port is not a dial
error and stays a reconnect (the application may be starting); a pod or namespace that vanishes
from the cache while the forward waits keeps it Waiting, since the cache cannot tell a deleted pod
from one a controller recreates. The options weighed:

| Option | For | Against |
|---|---|---|
| **(a) Permanent errors fail at once (decided)** | A clear state, no pointless requests, the port freed; passing errors still bridged | Which errors count as permanent is a list to keep right |
| (b) Fail after N attempts that never connected | Simple | A slow pod start kills the forward |
| (c) Retry for ever with a distinct "Failing" state | Never gives up | The load and the occupied port stay |
| (d) Leave it | Nothing to build | "Reconnecting: forbidden" for ever |

## D-21: Pasting a kubeconfig

**Decided (owner, 2026-10-12), built the same day:** Cogpit does not add kubeconfigs from pasted
text. Lens's "Add Kubeconfigs" menu offers "Paste from clipboard" beside syncing files and
folders (captured 2026-10-12); Cogpit offers files and sync folders only. The paste dialog, the
`kubeconfig.paste` command and the owner-only store of pasted kubeconfigs built earlier on
2026-10-12 are removed, with their tests and the documentation that described them. What happens
to what an earlier build wrote: Cogpit no longer lists or reads a `pasted` source and leaves its
file under `<user config dir>/Cogpit/kubeconfigs/` alone, because it is the user's file and
deleting user files silently is not acceptable; the entry leaves `kubeconfigs.json` with its next
change ([architecture §6](architecture.md#kubeconfig-sources-built-2026-10-12)). The options weighed: keeping
Cogpit's paste dialog without copying Lens's, removing it (decided), or reading the clipboard
directly as Lens's menu item does.

## D-22: Colour theme

**Decided (owner, 2026-10-12):** Cogpit's dark theme takes the colours of VS Code's built-in
**Solarized Dark**, the theme the owner works in, and its light theme those of **Solarized Light**;
`--theme dark|light|system` stays. The colours are copied into Cogpit's theme tokens once, not read
from VS Code at run time (which would tie Cogpit to VS Code's files and to themes whose colours
need not suit Cogpit's status tokens). Source: VS Code's `theme-solarized-dark` and
`theme-solarized-light` (MIT; the palette is Ethan Schoonover's Solarized, MIT), credited in
`ATTRIBUTION.md`. **Built 2026-10-12** in [`theme.go`](../../internal/ui/fyneui/theme.go) (each
value carries a comment naming its VS Code key) and, for the terminal's sixteen colours, in
[`termview.go`](../../internal/ui/fyneui/termview/termview.go) through a patch to the terminal
widget ([PATCHES.md](../../third_party/fyne-terminal/PATCHES.md)). Where Solarized's own contrast
is too low for text, the build departs from the table, always in the direction of legibility, and
`TestPaletteContrast` holds the floors: the light foreground is `sideBarTitle.foreground`
(#586E75) because `editor.foreground` (#657B83) is 4.1:1 on the light view; the light accent,
buttons and focus are a darkened cyan (#1F7F78, links #1B6761) instead of the light file's tan
`focusBorder` and olive `button.background`, to match the cyan of the logo and carry text; status
tokens are the accents lightened (dark) or darkened (light) until they read at 3:1 on the views,
the navigator, the hover row and the header, and at 2.5:1 on a selected row; the text of a selected
row is base2 (dark) or base02 (light), a new token, because the foreground is 2.4:1 on
#005A6F; the dark tooltip and toast are #003847 and the light ones #EEE8D5 where VS Code's hover
widget colours (#004052, #CCC4B0) are 3.6:1 and 3.1:1 under the text; the light dock strip is
#DDD6C1. The two dark terminal colours black and bright black (#073642, #002B36) are VS Code's,
and bright black is invisible on the dark view (the session-ended notice therefore uses bright
yellow, #657B83). The mapping read from Solarized Dark (2026-10-12; Solarized Light's file gives
the light values the same way):

| Cogpit surface | VS Code key | Solarized Dark |
|---|---|---|
| Views and editors | `editor.background` | #002B36 |
| Navigator, drawer, status bar | `sideBar.background`, `statusBar.background` | #00212B |
| Header and title bar | `activityBar.background`, `titleBar.activeBackground` | #003847, #002C39 |
| Tabs, inactive and active | `tab.inactiveBackground`, `tab.activeBackground`, `tab.activeForeground` | #004052, #002B37, #D6DBDB |
| Text | `editor.foreground`, `sideBarTitle.foreground` | #839496, #93A1A1 |
| Accent, focus, buttons | `focusBorder`, `button.background` | #2AA198 (cyan) |
| Selected and hovered row | `list.activeSelectionBackground`, `list.hoverBackground` | #005A6F, #004454 |
| Inputs | `input.background` | #003847 |
| Status tokens | the Solarized accents | green #859900, yellow #B58900, orange #CB4B16, red #DC322F, blue #268BD2, magenta #D33682, violet #6C71C4 |
| Terminal | `terminal.ansi*` | the theme's sixteen colours |

**Amended 2026-10-13 (G13, Cogpit's default, not the owner's decision):** the theme is also a
preference (Preferences › App › Theme: Sync with computer, Dark, Light, Lens's three), switched at
once and kept in `preferences.json`; `--theme` stays and overrides it for one run; Dark stays the
default until the user picks another. A cluster's icon colour (Cluster settings › General) is one
of Solarized's eight accents, in each variant's legible shade (`identityColors` in
[`theme.go`](../../internal/ui/fyneui/theme.go), 3:1 held by `TestIdentityColorsContrast`).

**The decision round of 2026-10-14 evening (owner):**

- **Decided (A94):** the default theme becomes Sync with computer (follow macOS). Built
  2026-10-14: `DefaultPreferences` starts `ThemeSystem`
  ([`preferences.go`](../../internal/engine/preferences.go)); a stored choice stays as it is.
- **Decided (A93):** a cluster's accent colour is one of Solarized's eight or any colour from a
  colour picker. Built 2026-10-15: Cluster settings › General shows a ninth swatch for a picked
  colour and a palette button after the eight (the "…" menu's "Any Colour…" too), which opens
  Fyne's colour picker; the setting stores `#rrggbb` (`engine.IsCustomIconColor`), so agents set
  it through `cluster.settings.set` as they set the eight. A night decision for the owner to
  confirm: a picked colour keeps its hue and saturation, and the window moves its lightness (up in
  the dark theme, down in the light one) just far enough to read at 3:1 on every surface an
  identity is drawn on, as the eight are held to (`legibleShade` in
  [`identitycolor.go`](../../internal/ui/fyneui/identitycolor.go), `TestCustomIdentityColorContrast`).
  Weighed against it: drawing the colour as picked and warning when it fails the contrast in a
  theme (the user's exact colour, but initials that may not read in one of the themes), and
  refusing such a colour (the picker would reject most pale and dark colours).
- **Confirmed (A95, as built):** the light theme's accent stays the darkened cyan #1F7F78.

**Amended 2026-10-16 (the owner's F13, [D-58](#d-58-standby-and-lost-connections)):** a cluster's
connection dot does not use the status green: Connected and a (re)connect are the Running
containers' green (`colorContainerRunning`, #3dd68c dark, #177f4a light since 2026-10-16, #1a8f53 before), Login expired and
Connection failed the status red, Not connected the unknown grey; no amber.

## D-23: A link to the owner in the status bar

**Decided (owner, 2026-10-12):** the status bar shows, always (before connecting and on the demo
cluster too), "Need a Kubernetes Expert? Visit: hans-fischer.com"; a click opens
https://hans-fischer.com in the user's default browser. Nothing is sent or tracked; only the
click opens it. When the bar is too narrow, the link gives way first, so that the context and the
connection's dot stay readable. It is a GUI link, not an engine command: a documented exception
to "everything is a command", so that no agent can open the page. **Built 2026-10-12** in
[`bars.go`](../../internal/ui/fyneui/bars.go) (`statusLayout`): the link is a status bar item with a
divider; the bar's minimum width is the status items' alone, and the link takes the room left,
is shortened with an ellipsis and, below the width of "Need a…", hidden with its divider. It opens
through `openBrowser`, the function the port-forward view uses.

**Moved (owner, 2026-10-12):** the link sits at the right end of the bar, directly left of the MCP
item (and ends the bar under `--no-mcp`, without a divider). The other items stay at the left end
with their widths; the link still gives way first.

**The MCP item is a switch (owner, 2026-10-12).** A click on the status bar's MCP item switches the
*external* MCP server, the loopback HTTP endpoint other programs on this computer reach with the
bearer token, off and on. Off: the endpoint stops listening, every connection and every pending
confirmation of an agent ends (its command does not run), the endpoint file is removed (until
2026-10-15 the token file too), the item shows
a muted, struck pill with the tooltip "External MCP server (for other apps on this computer): off —
click to start", and Help › Copy MCP command is disabled (the top bar's MCP button, disabled with it,
is gone since F6), so no stale command is offered. On: the server starts again with the token and
the port it kept (since 2026-10-15, F6, [D-53](#d-53-the-mcp-details-and-a-token-kept-until-the-user-regenerates-it);
until then with a new token and port) and the tooltip names the URL. A gear beside the pill opens
the MCP details (D-53); a click on the pill still switches. Like the link
it is a GUI control and not an engine command or an MCP tool: an agent cannot switch the server on
(`TestMCPSwitchIsNoCommand`). Nothing else is coupled to it: the engine and an in-process agent
(D-10, origin `BuiltInAgent`) keep working while the external server is off. The user's switch is
one line in the audit log (origin `gui`, command `gui.mcpServer`, `{"enabled":true|false}`).
**Persistence (built as recommended; the owner may revise):** the choice is stored in Fyne's
preferences (`mcp.enabled`, written by a click that worked) and read at the next launch; no stored
choice means on, as before. `--no-mcp` still wins (no server, no item), and `--mcp-addr` only sets
the address; there is no environment variable. A failed start at launch shows the item off with the
reason in its tooltip and leaves the stored choice alone. **Decided (owner, 2026-10-14 evening,
A15):** the switch stays persisted and `cogpit-cli mcp` honours it: with MCP off the bridge refuses
to serve, headless too, with a clear message to the agent. A record of its own:
[D-49](#d-49-the-external-mcp-servers-switch-honoured-by-cogpit-cli-mcp), built 2026-10-14 (the
choice kept in a file beside the token file, where `cogpit-cli` reads it).

## D-24: Logo

**Decided (owner, 2026-10-12):** Cogpit's logo is the "C helm": an open C (its opening to the right,
75°, the C drawn heavy) around a ship's wheel with eight spokes, an inner ring and a hexagon hub; a
needle fills the opening where the spokes stop; bar handles stand outside the C, and a thin outer
ring closes the wheel. Cyan and Solarized base1 on Solarized Dark's base03, in a macOS squircle
plate for the app icon, and without the plate as the 20-pixel mark in the header bar, drawn with
heavier lines there. It replaces the ring with a dot, which the owner found too plain. Chosen in an
interactive lab among four directions (an instrument dial, a cog with a visor, the C helm, a radar)
and twelve variations of the C helm; the lab's settings were
`{"concept":"helm","accent":"cyan","second":"cyan","plate":"base03","shape":"squircle","stroke":2,"spokes":8,"gap":75,"handles":1,"rotate":0,"weight":2.8,"ring":1,"outer":1,"pointer":1,"hexhub":1,"bars":1}`.
The exact geometry is the reference: [`assets/logo/helm.svg`](assets/logo/helm.svg) (the app
icon) and [`assets/logo/helm-mark-dark.svg`](assets/logo/helm-mark-dark.svg),
[`assets/logo/helm-mark-light.svg`](assets/logo/helm-mark-light.svg) (the mark on dark and light bars).
**Built 2026-10-12:** both are drawn from the reference's geometry and settings by one rasteriser,
[`internal/ui/logo`](../../internal/ui/logo/logo.go) (pure Go, no Fyne, so that the icon generator
and the header share it; its test checks the stroke widths against the three SVGs). The app icon
(`make icon` → `hack/icon` → `packaging/icon.png`) carries its own plate with transparent corners
and the margin of macOS icons (100 of 1024 pixels, the radius `fyne package` would mask), so `make app`
passes `--use-raw-icon`; without it `fyne package` shrinks and masks an icon a second time. The
header's mark ([`bars.go`](../../internal/ui/fyneui/bars.go), `newLogo`) is drawn at the pixel size the
canvas shows it at, 20 units (`sizeLogo`), in the heavy weights, with the theme's `logo` tokens
(base1 and cyan on the dark theme, base01 and cyan on the light one, as the reference's two marks).

**Revised (owner, 2026-10-15):** the app icon got lost in the macOS Dock: a nearly black,
unsaturated tile among bright ones, its spokes 0.9 points thin at a 64-point Dock (the C's contrast
against the plate, 5.6:1, was not the cause). In a second icon lab (a claude.ai artifact, kept; it
sets the C's colour and weight, its glow, core and halo, the wheel's colour, weight and glow, the
handles' and outer ring's colour, the plate's gradient and rim, with a Dock preview against D-24)
the owner chose a blue C with a white core, glowing over the wheel, which lies under the glow,
white handles and outer ring, on a petrol plate darkening downwards:
`{"concept":"helm","base":"D-24","cColor":"#4fb6ff","cWeight":7.3,"handleColor":"#eef6ff","glow":1,"glowRadius":4.5,"core":1,"halo":0.35,"accent":"#bfe9ff","spokeWeight":0.8,"accentGlow":0,"outer":1,"plateTop":"#03293a","plateBottom":"#00141d","rim":0.16}`.
That tile is now the logo everywhere: the app icon, the header (20 points), the welcome page and
About, the README's banner and the repository's social preview, the same in both themes (owner,
2026-10-15; weighed against it: the plate-less mark in the new colours, whose pale lines need darker
inks on the light theme first; the tile on the welcome page only; no change inside the app).
**Every version is kept** (owner, 2026-10-15): `Versions` in
[`internal/ui/logo`](../../internal/ui/logo/versions.go) holds the MVP's cog around a gauge
(`gauge`, 2026-10-11), D-24's C helm (`helm`, 2026-10-12, drawn as it was then: its test matches
the icon it shipped as pixel for pixel) and the glowing C helm (`helm-glow`, 2026-10-15); `Current`
is the one Cogpit shows, and `go run ./hack/icon -version <name>` draws any of them. Their pictures
and references are in [`assets/logo/`](assets/logo/): `<name>.png` (the app icon as it shipped),
D-24's `helm.svg` and its two marks, and the lab's SVG and export of the new one,
`helm-glow.svg` and `helm-glow.json`, which its tests hold the drawing to. The glow is a Gaussian
blur in the rasteriser, as the lab's `feGaussianBlur`; Chrome's render of the lab's SVG and the Go
drawing differ by 1/255 at the median (measured 2026-10-15). The social preview
(`docs/images/social_preview_dark.png`, rendered by `make showcase`) is set by hand under the
repository's Settings › General › Social preview: GitHub's API cannot set it (its GraphQL schema has
no such mutation, checked 2026-10-15). How to change the logo or bring an earlier one back:
[extending › Changing the logo](extending.md#changing-the-logo).

## D-25: Copying values

**Decided (owner, 2026-10-12):** text in Cogpit can be selected and copied, above all in the detail
drawer, where everything is selectable (values, labels, annotations, events, notes, table cells).
Every value in the drawer gets a copy icon, shown on hover, that puts it on the clipboard and says
so in a toast; a value made of several entries (labels, annotations, ports, images) copies the one
entry, in YAML syntax (`app.kubernetes.io/name: grafana`). A Secret's value can be copied only once
revealed, in the GUI, and the clipboard is cleared 60 seconds later if it still holds that value.
Agents get no copy command (they read values through `object_get` already). This is Cogpit's own
approach; how Lens Desktop copies is not captured and not needed.

**Added (owner, 2026-10-12, H-25):** on macOS a copied Secret value is marked for clipboard
managers to skip (nspasteboard.org's concealed and transient types) and kept to this Mac (no
Universal Clipboard); the 60 seconds and the clearing on close stay. Options weighed: marking it
(decided), leaving it as built, a shorter time, no copying of Secret values at all. **Built 2026-10-12**
([`selection.go`](../../internal/ui/fyneui/selection.go), [`copy.go`](../../internal/ui/fyneui/copy.go),
[`secretreveal.go`](../../internal/ui/fyneui/secretreveal.go)):

- The drawer's text lines (names, values, table cells, section titles, events' reason and
  source) are `selLine`s: a drag selects, a double click selects a word, ⌘C or Ctrl+C copies, all
  inside one line; Tab does not stop at them (Fyne's focus chain skips a "disabled" object that has
  `SelectedText`, and a click still focuses it). An event's message is a selectable `widget.Label`.
  A revealed Secret value is a `secretText`: monospace, wrapped at any character, selectable over
  its lines. Not drag-selectable, because they are tappable or drawn as shapes: links, chips and
  status pills (they have the copy icon).
- One copy icon (`copyButton`), which the drawer moves to the value the pointer is over. It sits
  right after the value's text, or on the value's right end when the value is wider; on a chip or
  a pill it sits right after the chip, outside it and vertically centred (a chip with a neighbour
  in its row keeps a gap for it), so it covers no chip text. When the drawer rebuilds under a
  stationary pointer (a watch event), the icon is shown again at the value that now covers the
  place of the old one, without a new `MouseIn` (`keepCopy`); it goes when the pointer moves
  away. A tap puts the value on the clipboard and shows a toast
  ("Copied to the clipboard"). Values of several entries copy the entry under the pointer: a folded
  list has no icon while folded; an entry of the form `key=value` (labels, annotations, selectors,
  environment variables) is copied as `key: value`, any other entry as a YAML scalar, quoted only
  where YAML needs it (`yamlEntry`, through sigs.k8s.io/yaml). Single values are copied as shown.
- **Reach and confirmation (owner, 2026-10-12):** every chip owns the room for its icon (the
  `hoverWrap` gutter), the last of a row and one alone on its line too: the icon sits after the
  chip, and the pointer's way from the chip to it stays inside the chip's hover area. Before, only
  a chip with a right neighbour had that room, so the icon of a label chip hid (its `MouseOut`)
  before the pointer reached it, while the icon of a Pod IP, whose area spans the whole value
  cell, could be reached. After a copy the icon is a checkmark in the OK colour for 1.5 s
  (`copyConfirmTime`, the shell's `copyConfirm`), then the copy icon again while the pointer is on
  the value; a Secret's value confirms the same way. `TestCopyIconReachableOnLabelChips`,
  `TestCopyIconReachableOnWrappedChips`, `TestCopyIconTurnsIntoACheckmark`.
- **Right after the value (owner, 2026-10-13, in a real window):** "The copy button at Not After
  has wandered to the far right again": a TLS Secret's Not After is a status pill, and the icon of
  a pill sat at the drawer's right edge. A pill's wrapper (`hoverWrap`) spans the whole value cell,
  and the icon was placed after the wrapper instead of after what it draws; an event's message did
  the same. **Built 2026-10-13:** the icon follows what a value draws (`drawnWidth`: a pill or a
  chip as wide as it asks, a message to the end of its longest line), a padding after it, for
  every kind of value: a line, a link, a revealed Secret value, a pill, a chip (labels,
  annotations, a list's entries), an event's message, a line of a value of several lines (one icon
  per line, as before), a cell of a drawer table. Clamped to the value, never over another one:
  a line or a link whose text leaves no room sits at the value's end and its text is shortened
  with "…" before the icon while it is there (`iconRoom`; the text comes back when the icon
  goes, and stays short while the pointer is on the icon; the lists no longer do this since
  A69); a pill or a chip that
  reaches the details' right edge keeps the icon at that edge; a message that wraps keeps it at
  its top right. A value with action buttons (a container port with Forward…) owned the room for
  its icon after its text, so the icon sat before the buttons, which moved right by the icon's
  width, as a chip owns its gutter (Cogpit's default, not the owner's decision); replaced by A71
  on 2026-10-15 (below): the buttons right after the value, the icon over the value's end.
  `TestDrawerCopyIconAfterThePill`, `TestDrawerCopyIconAfterTheText`,
  `TestDrawerCopyIconAfterTheMessage`, `TestDrawerCopyIconInAnActionRow`; renders
  `TestRenderDrawerCopyIconAfterTheValue` (PNG [Not After](../../internal/ui/fyneui/testdata/drawer_copy_notafter_dark.png),
  [a long fingerprint](../../internal/ui/fyneui/testdata/drawer_copy_long_light.png)), and the pod's
  status pill in [`drawer_copy_pill_dark.png`](../../internal/ui/fyneui/testdata/drawer_copy_pill_dark.png).
  Not tried in a real window again.
- A masked Secret value has no icon; a revealed one has, and copies the value alone (not the note
  that a long value was cut). A copy of a Secret value, by the icon or by ⌘C on its selection, goes
  through `shell.copyText(…, secret=true)`: 60 seconds later (`secretClipboardTTL`; the shell's
  `secretClipTTL`, shortened in tests) the clipboard is emptied if it still holds that value; a
  later copy of anything cancels the clearing, and closing the window clears at once
  (`w.SetOnClosed`). Since 2026-10-16 a Secret's value in an editor of its drawer
  ([D-59](#d-59-a-secrets-values-edited-in-the-drawer)) has a copy icon of its own in the key's
  line, shown with the editor, that copies the whole value the same way, as ⌘C and ⌘X in the
  editor copy a selection (`TestSecretEditorCopyIcon`).
- It is no command and no tool: `TestCopyHasNoCommand`. The one copy command, `object.copyName`
  (the drawer's first icon), is older and stays.

**Added (owner, 2026-10-13):** names in the lists are copyable too (Pods, Deployments and every
other view): a list's cells get the drawer's copy icon on hover, at least the Name column's, with
the same checkmark after a copy. **Built 2026-10-13** ([`tablecopy.go`](../../internal/ui/fyneui/tablecopy.go),
the icon generalised in [`copy.go`](../../internal/ui/fyneui/copy.go)): one icon per list, the
drawer's `copyButton`, in a layer above the table and moved to the cell under the pointer (a list
of 10,000 rows gets no widget per cell); the table hands it the cell when its row band moves
(`setHoverCell`), and a pointer move shows it only while the pointer is inside that cell, not in
the gaps or over the header. A scroll or new column widths hide it until the pointer moves; a
refresh keeps it on its cell, which copies the text it shows when clicked; a click checks first
that the icon still lies on its cell. The copy goes through `shell.copyText` as a plain copy, with
the drawer's checkmark for 1.5 s and its toast ("Copied to the clipboard"); no command, no audit
line.

**Changed (owner, 2026-10-13, after seeing it):** not every cell: "a name, an IP, a node name, a
message in the Events is great. But no Age, no CPU, no Memory, no Restarts." It works by a
blacklist of the columns where copying makes no sense, so that a custom resource's columns need no
marking: every column not on it keeps the icon, custom resource columns included. And the icon
belongs right after the text, not at the end of a wide column. **Built 2026-10-13**: the blacklist
is one place, `noCopyTypes` and `noCopyColumns` with `copyColumn` in
[`tablecopy.go`](../../internal/ui/fyneui/tablecopy.go) (the UI's alone: no command or tool
copies). The owner set the principle and the four examples; the lists below are Cogpit's reading
of it for every built-in kind (the lead's proposal, not the owner's decision): identifiers and free
text keep the icon; numbers, times, durations, counts, ratios, booleans and short state words do not.

| No icon | Columns |
|---|---|
| By type: `number` | Restarts, Count, Replicas, a workload's Pods (`3/3`), Ready, Desired, Current, Completions, Active, Size, Capacity, Taints, Value, Webhooks, Revision, Local Port, Pod Port and the other counts; a custom resource's `integer` and `number` printer columns |
| By type: `age`, `until` | Age, Last Seen, Last Schedule, Updated (Helm releases), Next Execution; a custom resource's `date` printer columns |
| By type: `metric`, `warning` | CPU, Memory; the warning triangle |
| By id, short state words | `status` (Pods, Services, PVCs, PVs, Namespaces, HPAs, Port Forwards, Helm releases, VPAs, Applications), `conditions` (Nodes, Deployments, Jobs), `state` (Resource Claims), `qos` (Pods), `mode` (VPAs), `phase` |
| By id, booleans | `suspend` (Cron Jobs), `default` (Ingress Classes), `globalDefault` (Priority Classes), `privileged` (Pod Security Policies) |
| By id, counts, ratios and durations in text | `minAvailable`, `maxUnavailable` (Pod Disruption Budgets), `metrics` (HPAs: current against target), and `ready`, `desired`, `current`, `upToDate`, `available`, `completions`, `duration` |
| By id, policy and mode words | `failurePolicy`, `reinvocationPolicy` (admission policies), `reclaimPolicy`, `volumeBindingMode` (Storage Classes), `validationActions` (validating admission policy bindings), `policyTypes` (Network Policies), `scope` (Definitions, Ingress Classes), `protocol` (Port Forwards) |

The ids apply in every view, so a custom resource's printer column whose name reads as one of them
in lowerCamelCase ("Ready", "Status", "Phase") has none either; `phase` and the text ids in the
counts row are there for those, since the built-in kinds hold such columns as numbers. A custom
resource's printer columns map to the types by their declared type
([`kinds/customresources.go`](../../internal/model/kinds/customresources.go), `compilePrinterColumn`):
`integer` and `number` to `number`, `date` to `age`, `string` and `boolean` to text, so a
`boolean` column keeps the icon unless its name is on the list (the table cannot tell it from a
string column). Every other column keeps the icon: Name, Namespace, Node, Controlled By, the IPs
(Cluster IP, External IP, Endpoints, Load Balancers), Ports, Labels, Selector, Node Selector, Keys,
Type and Address Type, Kind, API Group, Group, Resource, Version, Roles, Rules, Message, Involved
Object, Source, Schedule, Time Zone, Storage Class, Claim, Provisioner, Handler, Holder, Controller,
Driver, Pool, Devices and Device Classes of a claim, Reserved For, Bindings, Policy, Volumes, a
claim's Pods (their names), Chart, App Version, Repository, Description, Target, Managed By, App
Name. Placeholder cells (empty, `-`, `–`, `—`, `<none>`) still have none.

Cogpit's defaults besides, the lead's proposals and not the owner's decision:

- The icon sits a padding (4 points) after the cell's whole text, measured with the table's text
  style, centred on the row. When the text leaves no room for it, it sits at the cell's right end;
  until 2026-10-15 the text was then shortened with "…" before it while it was there, since A69
  (below) the icon lies over the cell's end and the text keeps its width. Always inside the cell,
  never over the next column. A watch event that changes the text moves the icon to the new end.
- A click on it copies the cell's full text, not the shortened one, and neither selects the row,
  toggles its checkbox nor opens the drawer: the icon is a widget above the table, so the click
  never reaches the cell. A right-click on it opens the row's actions menu, as anywhere else on
  the row.
- A Secret's list holds no values: its columns are Name, Namespace, Labels, Keys (the keys' names),
  Type and Age ([`kinds/secrets.go`](../../internal/model/kinds/secrets.go),
  `TestSecretDetailShowsNoValue`), so no cell of it can put a value on the clipboard; Age has no
  icon (`TestSecretsListCopiesNoValue`).

Tests: `TestListCopyIconByColumn` (Pods with CPU and Memory values, Services, Events, Nodes and a
custom resource with an integer, a string, a "Ready" string and a date printer column),
`TestCopyColumns`, `TestListCopyIconAfterTheText`, `TestListCopyIconOnHover`,
`TestListCopyIconPlaceholders`, `TestListCopyIconCopiesTheFullText`,
`TestListCopyIconFollowsTheCells`, `TestSecretsListCopiesNoValue`, `TestListCopyIconHasNoCommand`;
renders `TestRenderListCopyIcon` (the Nodes list: PNG
[hover](../../internal/ui/fyneui/testdata/list_copy_hover_dark.png),
[after the copy](../../internal/ui/fyneui/testdata/list_copy_done_light.png)). Not tried in a real
window: the order in which the GLFW driver sends `MouseOut` to the table and `MouseIn` to the icon
was read in Fyne 2.8.1's source and replayed with Fyne's test driver.

**The decision round of 2026-10-14 evening (owner):**

- **Decided (A69):** in the lists the copy icon lies over the cell's end, and the text is not
  shortened for it. **Built 2026-10-15** ([`tablecopy.go`](../../internal/ui/fyneui/tablecopy.go),
  `place`): when the text leaves no room after it, the icon lies over the right end of the cell's
  visible part, and the cell keeps its own text, cut off with "…" only where the cell itself ends
  (its tooltip shows the whole text, as without the icon); the icon's opaque plate covers what lies
  under it. The cells' room for the icon is gone (`tableCell.room`), so placing the icon never
  re-lays out a cell. `TestListCopyIconOnHover` (the same text with and without the icon),
  `checkIconInCell`; renders `TestRenderListCopyIconOverTheEnd` (PNG
  [dark](../../internal/ui/fyneui/testdata/list_copy_over_end_dark.png),
  [light](../../internal/ui/fyneui/testdata/list_copy_over_end_light.png)), the Events tooltip
  ([`tooltip_dark.png`](../../internal/ui/fyneui/testdata/tooltip_dark.png)). The drawer's lines and
  links keep shortening their text before the icon (`iconRoom`, above): A69 names the lists, A71
  only the values with buttons (the lead's reading, for the owner to confirm).
- **Decided (A71):** in the drawer, a value with action buttons (a container port's Forward…)
  keeps its buttons right after the value, as before 2026-10-13, and the copy icon appears over the
  value's end on hover, as A69 in the lists. **Built 2026-10-15** ([`drawer.go`](../../internal/ui/fyneui/drawer.go)
  `itemValue`, [`copy.go`](../../internal/ui/fyneui/copy.go) `showCopy`): a value with buttons keeps
  no room for its icon, so the buttons follow it a padding after its text; its copy hooks are
  marked `over`, and the icon lies over the value's right end (its right edge a padding after the
  text, so it covers the text's end by its width less a padding: "TCP" of `http: 8080/TCP`)
  without shortening the text, also while the
  pointer is on the icon. A value that is a link without buttons (a forward's URL) keeps the room
  after its text and the icon there, as before (the lead's reading: A71 names the rows with
  buttons). `TestDrawerCopyIconInAnActionRow`, `TestDrawerCopyIconAfterALinkWithoutButtons`;
  renders `TestRenderDrawerCopyIconOverThePort` (PNG
  [dark](../../internal/ui/fyneui/testdata/drawer_copy_port_dark.png),
  [light](../../internal/ui/fyneui/testdata/drawer_copy_port_light.png)),
  [`pod_drawer_ports_dark.png`](../../internal/ui/fyneui/testdata/pod_drawer_ports_dark.png).
- **Confirmed (A70, as built):** the columns without an icon as the tables above list them, the
  lead's reading of the owner's principle, custom resource columns included.
- **Confirmed (A68, as built):** a copy from a list shows the checkmark and the toast.
- **Confirmed (A76, as built):** links, chips and pills are not drag-selectable; their copy icon
  covers them.

## D-26: Binding a credential plugin's approval to its program

**Decided (owner, 2026-10-12, from R-5 of the fourth run's review):** the approval of a kubeconfig's
credential plugin (exec) also binds the content (SHA-256) of the program it runs when that program
lies inside the kubeconfig's own folder or the sync folder it came from, where a writer of the
synced folder (H-1's threat) can replace it; replacing it asks again. A program found elsewhere
(on `PATH`, `/usr/local/bin`: kubelogin, aws, gke-gcloud-auth-plugin) stays bound by its path, so
an update of the tool does not ask again; a writer of a `PATH` directory controls the user's
account anyway. Built with D-17 by the fifth run's agent D. The options weighed:

| Option | For | Against |
|---|---|---|
| **(a) Hash programs in the folder only (decided)** | Closes H-1's own threat; no prompt after a tool update | A program on `PATH` stays bound by path |
| (b) Hash every program | Binds every program | Asks again after every update of a plugin (brew, gcloud components) |
| (c) Document only | Nothing to build | The swap through the synced folder stays open |

**Decided (owner, 2026-10-14 evening, A23,
[H-34](../security/trust-boundaries.md#h-34-the-plugin-approval-shows-the-kubeconfigs-proxy-not-cogpits)):**
the approval also shows and binds the proxy Cogpit itself sets for the cluster (Cluster Settings ›
Proxy), not only the kubeconfig's `proxy-url`; changing that proxy asks for the approval again.
**Built 2026-10-14** ([architecture §6](architecture.md#kubeconfig-sources-built-2026-10-12)):
`ExecPlugin.SetClusterProxy` records the context's proxy from Cogpit's cluster settings in every login
the sources describe (`KubeconfigSources.Files`, bound to the connector's settings) and in the one the
connect-time policy is asked about (`proxyBoundPolicy`, from the same read of the setting that the
connection uses); the fingerprint binds it, and the dialog shows it in the warning colour, the
kubeconfig's own `proxy-url` beside it as "not used" (`TestPluginApprovalBindsTheClusterProxy`,
`TestFingerprintBindsCogpitsProxy`, `TestApprovalDialogShowsCogpitsProxy`). Changing the proxy in
Cluster Settings sends `EventContexts`, so the lists show the context as changed at once. **The lead's
night decisions for the owner to confirm:**

- *What is bound:* ★ the proxy as it is shown, its password as `•••` (`RedactProxyURL`): another
  scheme, user, host or port asks again, a new password alone does not, and `exec-approvals.json`
  holds no hash over the password. Options: the URL as typed, password included (a rotated proxy
  password would ask for every plugin of the cluster again); host and port only (a changed proxy
  user would not ask).
- *Approvals given before:* ★ the field is left out of the fingerprint when no proxy is set, so the
  approvals of contexts without a proxy of Cogpit's still match after the upgrade; a context that
  has one asks once again (its approval never named it). Option: a new fingerprint version that asks
  every plugin again.

## D-27: An agent's Helm values in the confirmation

**Decided (owner, 2026-10-12, from the remainder of R-2):** when an agent upgrades a release with
values it passes inline (`helm_release_upgrade` with `values`), the confirmation dialog shows those
values, monospace and selectable, in a box that scrolls, before the user answers. The upgrade runs
with exactly that text (it is part of the request's parameters). The text is carried in
`ConfirmRequest.Review`, which is never serialised: no audit line, log or MCP result holds it. An
upgrade from a Helm Upgrade tab shows no values in the dialog: the tab is what the user reads, and
the upgrade is bound to its text revision (R-2). Built 2026-10-12 by the lead
(`TestConfirmationShowsTheReviewText`, `TestHelmUpgradeAgents`). The options weighed:

| Option | For | Against |
|---|---|---|
| **(a) Show the values in the dialog (decided)** | One call for agents; the user reads exactly what is applied | A long text needs a scrolling box |
| (b) Refuse inline values for agents | Only the tab path, already bound | Agents need two calls; the user must open the tab to read |
| (c) Leave it, document it | Nothing to build | The user accepts values they did not see |

## D-28: The default kubeconfig as sources, as Lens has it

**Decided (owner, 2026-10-12, from the fifth run's agent C2):** every context comes from a source
file read on its own; nothing is merged any more. `~/.kube/config` is an ordinary file of the
default sync folder `~/.kube`, so removing that folder hides its contexts too, as in Lens. The
files named in `$KUBECONFIG` appear at start as additional default sources, each read alone and
removable (for the session), so a developer's or `make run`'s `KUBECONFIG` keeps working.
Consequence the owner accepted: a credential plugin in `config` asks for the user's approval once,
like a plugin in any other file (H-1). **Built 2026-10-12** (scope C3,
[architecture §6](architecture.md#kubeconfig-sources-built-2026-10-12)): `engine.WithDefaultFolder`,
`engine.WithSessionFiles`, `cluster.NamedKubeconfigFiles`; a connector with sources merges nothing.
Details the decision left open, decided in the build: a bare context name (`--context`, an agent's
`cluster.connect`) resolves to the one file that defines it and is refused, naming the ids, when
several do; a `$KUBECONFIG` file lying directly in `~/.kube` is the folder's, so removing the folder
hides it too; a session file removed in the window returns at the next start with its approvals kept;
an existing `kubeconfigs.json` is read as it is (no migration, nothing deleted or written), so a user
who had removed `~/.kube` loses `config`'s contexts now. The options weighed: `config` as kubectl
reads it, merged with `$KUBECONFIG` and not removable (as built by C2); as Lens (decided); and
for `$KUBECONFIG` under it: its files as sources (decided) or ignored, as Lens does.

**Asking once per file and plugin (owner, 2026-10-12, built the same day, scope C4).** A `config`
with ten EKS contexts and the same `aws eks get-token` plugin must not ask ten times. One dialog
per file and identical plugin lists every context of that file that uses it and is not allowed yet,
each with its server and how that server is reached and trusted (the warning lines of review R-11
per context where they apply), and one click allows all of them. "Identical" is
`kubeconfig.ExecPlugin.Identity`: the same command, arguments, environment (names and values), API
version, interactive mode, `provideClusterInfo`, and, after D-26, the same binding of the program
(its resolved path and, inside the folder, its content digest); for token files (D-17) the same token
file. Context, user, server and trust are not part of it. The approvals stay **one per context, each
with its own fingerprint** (server and trust bound, as strict as H-1 had it); only the asking is
bundled. A context added to the file later asks on its own, or with the other new ones. The approve
command stays GUI-only and takes the list of (context, fingerprint) the dialog showed; it checks
each against the file as it is now and refuses the whole list, naming the contexts, if any changed or
left meanwhile (`kubeconfig.pendingApprovals`, `kubeconfig.approveExec`; `TestPendingApprovalsGroupContextsByFileAndPlugin`,
`TestApprovingAGroupApprovesExactlyItsContexts`, `TestApprovalIsRefusedAsAWholeWhenOneContextChanged`).
The options weighed: one dialog per context (as built before; ten clicks and ten dialogs, which
invites approving without reading); one approval for the file's plugin whatever the context (one
click, but an approval that no longer binds the server: a changed server in `config` would not ask);
**one dialog, per-context approvals (decided)**.

**Decided (owner, 2026-10-14 evening, A97):** Remove from Cogpit on a context of a sync folder
offers to unlink the whole folder, its confirmation naming every context that leaves with it.
**Built 2026-10-15:** the default `~/.kube` included; the dialog reads "Remove <cluster> and its
folder from Cogpit?", says the cluster comes from a synced folder and can only leave with it, names
the folder's path, what stays on disk, and every context of the folder's files, and its button
reads "Remove Folder" ([architecture §6](architecture.md#6-cluster-connections-internalcluster)).
The lead's night decision for the owner to confirm: the removal is bound to the contexts the dialog
named (`cluster.remove`'s `contexts`), and a folder whose contexts changed while the dialog was open
is refused with the reason, so that nothing leaves Cogpit the user was not shown. Weighed against
it: unlinking whatever the folder holds at the click (simpler, but a file synced in during the
dialog would leave unseen), and asking again by itself with the new list (a dialog that changes
under the user's pointer).

## D-29: Helm chart repositories from Helm's repositories.yaml

**Decided (owner, 2026-10-13):** Cogpit reads Helm's own `repositories.yaml` regularly and takes
over its changes by itself: a repository added, changed or removed there (by `helm repo add`, by
hand, by another tool) is in Cogpit within 5 seconds. Repositories with credentials are included
when their credentials are available (Helm's file keeps a username and password, a client
certificate and key, a CA file). This replaces the lead's default of the night run (2026-10-13:
a list of Cogpit's own, public https repositories only, Helm's file never read). **Built
2026-10-13** (below).

What the decision leaves to the build, as the lead reads it (proposals, not decided):

- **Which file.** The one the helm CLI uses: `$HELM_REPOSITORY_CONFIG`, else
  `repositories.yaml` in `$HELM_CONFIG_HOME`, else in Helm's default configuration folder
  (`~/Library/Preferences/helm` on macOS, `$XDG_CONFIG_HOME/helm` or `~/.config/helm` on Linux,
  `%APPDATA%\helm` on Windows; Helm's `helmpath` rules, not verified on this machine).
- **How often.** A local file, not the API server: its size and modification time checked every
  2 seconds, as the kubeconfig sync folders are, and read again when they change; a file that does
  not parse keeps the last good list and says why. The 5 seconds are the owner's bound.
- **Reading, never writing (owner, 2026-10-13):** repositories are added with the helm CLI
  (`helm repo add`), not in Cogpit. Cogpit writes nothing into Helm's file; it neither adds nor
  removes a repository (removing would write the file too: the lead's reading of the decision).
  The night run's own list and its `helm.repo.add` and `helm.repo.remove` go when D-29 is built;
  the Repositories dialog then lists Helm's repositories read-only (charts, index age, the last
  error) and can refresh their indexes, which writes nothing into Helm's folders.
- **Credentials.** Read from Helm's file into memory only: never written to Cogpit's files, never
  in logs, errors, toasts, the audit log or an MCP result (agents see a repository's name and URL,
  and whether it has credentials); sent only to the repository's own host, unless Helm's
  `pass_credentials_all` says otherwise for that repository
  ([H-29](../security/credentials.md#h-29-authenticated-helm-repositories)).
- **Indexes.** Downloaded into Cogpit's own cache as today (no writes into Helm's cache), or read
  from Helm's cache when it is current: to decide at the build.

**Built (2026-10-13)** ([`helm/repofile.go`](../../internal/helm/repofile.go),
[`helm/repo.go`](../../internal/helm/repo.go), [`engine/helmrepos.go`](../../internal/engine/helmrepos.go),
[`commands/helmrepos.go`](../../internal/engine/commands/helmrepos.go),
[`fyneui/helmcharts.go`](../../internal/ui/fyneui/helmcharts.go); [architecture §8](architecture.md#8-helm-internalhelm)).
Verified in this repository by tests; on the owner's Mac only the path was resolved, never read:

- **Which file (as proposed).** The helm CLI's own resolution through the Helm SDK
  (`cli.New().RepositoryConfig`: `$HELM_REPOSITORY_CONFIG`, else `$HELM_CONFIG_HOME`, else
  `helmpath`, which also honours `$XDG_CONFIG_HOME`), in one place, `helm.RepositoryConfigPath`,
  called from `cmd/cogpit` only and handed to `engine.NewHelmRepos`, so no test can resolve the
  developer's file (`TestRepositoryConfigPath`, on a temporary `HOME`). On this Mac, without any
  `HELM_*` or `XDG_*` variable set, it resolves to `~/Library/Preferences/helm/repositories.yaml`
  (the path printed on 2026-10-13; the file itself was not opened).
- **How often (as proposed).** Size, modification time and mode looked at every 2 seconds, as the
  kubeconfig sync folders are; a change is read (at most 1 MiB, parsed with the SDK's `repo.File`)
  once the file stayed unchanged for 250 ms and taken only if it is still unchanged after reading,
  because `helm repo add` rewrites the file in place (Cogpit's default). Measured in
  `TestHelmRepoFileChangesWithinFiveSeconds` at the default poll: added and its index downloaded
  after 2.2 s, a changed URL after 2.0 s, removed after 2.0 s. A file that does not parse keeps the
  last good list, and `helm.repo.list` and the dialog say why, naming the line, never the content;
  a missing file is an empty list with the hint "add repositories with helm repo add".
- **Indexes: Cogpit's own cache (the lead's recommendation, decided at the build).**
  `<user cache dir>/Cogpit/helm/indexes/<key>-index.yaml`, owner-only, the key a hash of the
  repository's name and URL: nothing is written into Helm's folders, freshness stays under
  Cogpit's rules (an explicit update, or while the Charts view or a Helm Install tab is open), and
  a URL changed in Helm's file never shows an index fetched for the old one, which Helm's cache,
  named by the repository alone, would until the next `helm repo update`. The cost: a repository
  is downloaded once by helm and once by Cogpit. A repository that appears in the file, or whose
  URL, credentials or TLS settings change, counts as never downloaded and is fetched at once while
  the view or a tab is open (the file's change wakes the poller); its running download is stopped
  when it leaves the file or changes. At start an index an earlier run downloaded for the same
  name and URL is shown until its time-to-live passes.
- **Credentials.** Memory only (`helm.Repository`, whose credentials are unexported and print as
  its name and redacted URL through `%v`, `%#v`, slog and JSON); agents and the dialog see name,
  URL without user, password or query values, `hasCredentials`, `clientCertificate`, `caFile`,
  `insecureSkipTLSVerify`, `passCredentialsAll`, `plainHTTP`. Basic auth goes only to the
  repository's own scheme, host and port, decided per request on every redirect hop and for chart
  URLs the index points elsewhere (Go's own rule would keep it for a subdomain), unless the entry
  says `pass_credentials_all` (`TestCredentialsStayOnTheRepositoryHost`,
  `TestCredentialsNotSentToAChartElsewhere`). A CA file replaces the trust roots for that
  repository, as helm uses it; a client certificate is used when both `certFile` and `keyFile` are
  set, as helm does; the files are read at every download, at most 1 MiB, so a rotated file
  applies, and an error names the path only (`TestClientCertificateRepository`). Errors are
  scrubbed of the password and the user name as a guard (`TestScrubbedErrors`), and the proof
  that nothing leaks runs across the list, the install tab, events, notifications, the audit log
  and the log (`TestHelmRepoCredentialsNeverShown`; on kind `TestKindHelmInstall`).
- **Cogpit's defaults, not the owner's decisions:**
  - plain `http://` repositories are listed and used as helm uses them and marked "http" in the
    dialog (one with credentials only once the user allowed it: the owner's decision of
    2026-10-14, below); a redirect never goes from https to http, and an https
    repository's index never sends a download to http ([H-40](../security/credentials.md#h-40-credentials-over-plain-http),
    [H-41](../security/trust-boundaries.md#h-41-repositories-whose-content-nothing-authenticates));
  - `insecure_skip_tls_verify` is honoured for that repository only and marked "insecure";
  - a user and password in the URL are that repository's basic auth (Go's HTTP client, and so the
    helm CLI, sends them), and are shown nowhere; the query's values are shown as `•••`;
  - a repository whose name Cogpit's chart naming (`<repository>.<chart>`) cannot hold, a dot for
    one (until 2026-10-14: since A36 a dot is used, below), and an OCI URL are listed with a clear
    reason and not used; a second entry of a name
    likewise (helm uses the first); a relative CA, certificate or key path likewise (Cogpit does
    not run in the folder helm ran in); at most 256 repositories are used;
  - the night run's `<user config dir>/Cogpit/helm/repositories.json` and its indexes in
    `<user cache dir>/Cogpit/helm/` are left on disk, unread (no implicit deletes);
  - the Repositories dialog follows the engine's new `EventHelmRepos` while it is open.
- **Commands.** `helm.repo.add` and `helm.repo.remove` are gone (registry, MCP, README, the
  dialog, tests: `TestNoCommandWritesHelmsFile`); `helm.repo.list` returns the file's path, its
  status (`ok`, `missing`, `error`), the reason, a hint and the repositories in the file's order;
  `helm.repo.update` reads the file first, then downloads.

**Decided (owner, 2026-10-14): a repository's credentials over plain http, asked once per
repository** ([H-40](../security/credentials.md#h-40-credentials-over-plain-http)). Cogpit asks
once per repository, keeps the approval like a kubeconfig credential plugin's, bound to the
repository's name, URL and user, and then uses the repository as helm does, the background refresh
included; without the approval no credentials go out over http: the index is not downloaded and the
Charts view says why. This replaces the build's default of 2026-10-13 (used as helm uses it,
credentials included, marked "http"). The options weighed:

| Option | For | Against |
|---|---|---|
| Refuse credentials over http | Nothing in clear, ever | A repository the user runs that way (an internal ChartMuseum) is unusable in Cogpit while helm uses it |
| Warn once, then send | One toast, no question | The first download has sent them already when the user reads it |
| **Ask once per repository (decided)** | Nothing leaves before the user's yes; one question per repository; a changed URL or user asks again | One more step for a repository the user knowingly runs over http |

**Built (2026-10-14)** ([`engine/helmhttp.go`](../../internal/engine/helmhttp.go),
[`commands/helmhttp.go`](../../internal/engine/commands/helmhttp.go),
[`fyneui/helmhttp.go`](../../internal/ui/fyneui/helmhttp.go), [`helm/repo.go`](../../internal/helm/repo.go)):
the helm package refuses a plain http request of a repository with a user and password before any
connection (`helm.ErrCredentialsOverHTTP`) and never adds them to a plain http request, unless the
engine hands it a copy the user allowed; the engine checks the allowance before every index, chart
and README download, leaves a waiting repository out of the background refresh and downloads it at
once when allowed. `helm.repo.list` says `httpCredentials` (`required`, `changed`, `allowed`) and,
while it waits, the reason in `error` and an `httpToken`; `helm.repo.allowHTTPCredentials` (GUI-only:
agents and the CLI are refused whatever they pass, D-2) records the allowance for the token the
window showed, all or none. The window shows a banner row per waiting repository below the Helm
banner while the Charts view or a Helm Install tab is open; Allow… says what crosses the network,
who can read it, what the allowance binds and that no agent can give it. Tests:
`TestPlainHTTPRepository`, `TestHTTPBinding`, `TestHelmCredentialsOverHTTPWaitForTheUser`,
`TestHelmHTTPWithoutCredentialsNeedsNoAllowance`, `TestHelmHTTPUnreadableApprovals`,
`TestAllowHTTPCredentialsIsTheUsers`, `TestHelmHTTPBannerAsksTheUser`. Cogpit's defaults of the
build, not the owner's decisions:

- **Where it is kept:** `<user config dir>/Cogpit/helm-http-approvals.json` (0600, written
  atomically), per repository its name, its URL redacted, a SHA-256 of its name, URL and user name,
  and when; no password, so a rotated password keeps the allowance (the owner's binding); without
  a configuration directory the allowances last for the session. An unreadable file allows nothing
  and is never overwritten.
- **What is asked about:** basic auth only. A client certificate needs TLS and is never presented
  over http, so a plain http repository with only a client certificate needs no allowance.
- **The token** the window hands back is a hash of the binding with a salt of the process: it
  changes with the URL or user name, reveals neither, and means nothing to another process.
- **Where it asks:** a banner in the window while the Charts view or a Helm Install tab is open (the
  times Cogpit would download), not a dialog that opens by itself; the Repositories dialog shows the
  reason in the repository's row.

**The decision round of 2026-10-14 evening (owner):**

- **Confirmed (A6, as built):** an allowance of credentials over http does not expire; it stays
  bound to the repository's name, URL and user.
- **Confirmed (A13, as built,
  [H-42](../security/trust-boundaries.md#h-42-a-repository-another-program-writes-into-helms-file-is-used-without-asking)):**
  a repository that appears in Helm's file while Cogpit runs is downloaded at once, while the
  Charts view or an Install tab is open.
- **Decided (A14, [H-41](../security/trust-boundaries.md#h-41-repositories-whose-content-nothing-authenticates)):**
  an http or insecure-TLS origin is marked in the Charts view (the repository's cell), the Helm
  Install tab and the install and upgrade confirmations. Before, only the Repositories dialog
  marked it ("http", "insecure"). **Built 2026-10-15** ([architecture §8](architecture.md#8-helm-internalhelm),
  `helm.Repository.Origin`, `kinds.HelmOriginMark`, `kinds.HelmOriginWarning`): the chart object's
  `spec.origin` and its Repository cell "lab (http)", "charts.edge (insecure TLS)", the mark in the
  warning colour; the Install tab's and the Upgrade tab's `origin` with a pill and a line of
  warning; `helm.chart.preview`'s `origin` and `warning`; the confirmation of an agent's install or
  upgrade (`ConfirmRequest.Warning`). Agents read each of them. The lead's decisions for the owner
  to confirm:
  - **Where the window's own install and upgrade are confirmed.** The window's install has no
    engine confirmation (a mutating command of the user's own window); its confirmation is the
    preview dialog (A37), which shows the warning above what the install creates. The window's
    upgrade has neither: its tab is what the user reviews, so the tab carries the mark beside the
    chart and, while Save downloads from such a repository, the warning under it. No new dialog
    was added.
  - **The words.** "http" and "insecure TLS" (the dialog's "insecure" became "insecure TLS" too,
    one word for the same thing everywhere), and one sentence for both places a reason shows:
    "Repository lab is served over plain http: nothing authenticates the chart, and anyone on the
    network path can change it." (for insecure TLS: "…'s server certificate is not checked
    (insecure_skip_tls_verify in Helm's file): …"). Plain http wins over the entry's TLS setting,
    which does not apply to it.
  - **The drawer too.** A marked chart's drawer has an Origin item with the sentence, in the
    warning colour: the Charts view's row says the mark, its drawer why.
  - **What the user read is what runs.** An agent's install or upgrade is refused when its warning
    would read differently when it runs (Helm's file changed while the confirmation was open), and
    so is a download whose origin is not the one the warning was made from; a window's upgrade from
    a tab whose repository's origin changed meanwhile marks the tab anew and refuses that save, so
    the next save is made on what the tab shows. An Install tab keeps the origin its archive came
    with, which cannot change.
  - **The Repository column** is 210 points wide (it took the default width before), so that a
    name and its mark fit; Lens's width is not captured.
  - **No refusal and no question:** a mark, as decided; installing from such a repository is as
    before.
- **Decided (A36):** repositories with a dot in their name are supported: Cogpit's chart id gets a
  form that tells the repository from the chart without doubt (Helm itself writes `repo/chart`),
  carried through refs, deep links and agents' parameters, and the ids saved before are migrated
  once. Purely internal, no change upstream. **Built 2026-10-14** ([`model/kinds/helmcharts.go`](../../internal/model/kinds/helmcharts.go)
  `HelmChartName`, `SplitHelmChartName`; [`helm/repofile.go`](../../internal/helm/repofile.go)):
  a repository's name may hold dots (1 to 63 letters, digits, `.`, `-` or `_`, the first a letter
  or a digit; another name is still listed as not used, helm refusing only a `/`). The id's form,
  the lead's decision for the owner to confirm:

  | Option | For | Against |
  |---|---|---|
  | `<repository>/<chart>`, Helm's own notation | Familiar; what the Install tab, the journal and the confirmations already write | `/` separates a cache key's namespace from its name (`ObjectRef.Key`) and a reference's printed parts, and a `cogpit://` link's `name` takes no `/`: every id changes, with every place that holds or parses one |
  | `<repository>:<chart>` | Readable; a character Cogpit's repository names leave out | Every id changes; a `:` in an object name is unusual |
  | **`<repository>.<chart>`, every dot of the repository's name doubled (chosen)** | Every id that exists is what it was (no repository with a dot was used before), so nothing saved changes; an id keeps an object name's characters, which links and keys take; the first single dot separates, because a chart's name starts with a letter or a digit | A dotted repository's id reads oddly (`charts..example.nginx`); agents take ids from `resources_list`, they do not build them |

  **The migration "once": nothing to migrate.** Searched 2026-10-14 for every file Cogpit writes:
  `workspace.json` keeps a tab's view, filters, sort and history of views, never an object's name;
  drafts are an Edit or Create tab's, never a chart's (charts cannot be edited); the Helm journal
  names an install's chart as `<repository>/<chart>`, text for display, never parsed; the hotbar
  holds contexts; chat conversations hold transcripts, never read back as references; the index,
  README and icon caches are keyed by a hash of the repository's name and URL, not by the id. And
  the id of every repository Cogpit could use before keeps its form. So an id saved before reads as
  it did, without migration code (`TestHelmChartName`, `FuzzHelmChartName`,
  `TestHelmRepositoryWithADot`: the object, the drawer, `ui.navigate`, `object_get`, a
  `cogpit://` link's name, the Install tab and an install from it; an id read the old way names no
  chart).

Helm beyond the repositories, answered in the same round and kept here:

- **Confirmed (A37, as built):** Install shows the preview first; its Install runs
  `helm.chart.install`.
- **Confirmed (A38, as built):** an upgrade to the release's own version uses the stored chart,
  nothing downloaded.
- **Decided (A39):** a release name that both storage drivers hold in a namespace shows as one
  release, the one deployed last, with a drawer note "also stored in the `<other>` driver (older)".
  Largely built before: one release was shown, and the drawer's Storage already said, in the
  warning colour, that the other driver holds a release of the name with its count of revisions,
  not shown (`storageItem` in [`helm/detail.go`](../../internal/helm/detail.go), read 2026-10-14).
  The question's row said the other's revisions are hidden and missed that note. **Built
  2026-10-14:** the note in the owner's wording, "ConfigMaps; also stored in the Secrets driver
  (older)" (the release object's new `hiddenOlder` says the other was deployed before the one
  shown; `TestDeriveBothDriversNamesTheOlder`). Cogpit's wording, the lead's decision for the
  owner to confirm: when both were deployed at the same moment (the tie that goes to Secrets,
  Helm's default) the note says "(deployed at the same time; the Secrets' one is shown, Helm's
  default)" instead of "(older)", which would not be true; the count of the other's revisions is
  no longer in the note (the owner's wording has none) and stays in the object
  (`hiddenRevisions`).
- **Confirmed (A40, as built):** Helm's sql driver is out of scope.
- **Confirmed (A30, as built):** only the window chooses the configmap driver for new installs;
  agents and the CLI may only set it back to Secrets.

## D-30: Certificates in Secrets, decoded

**Decided (owner, 2026-10-13):** "For Secrets of type certificate I want to see all the
certificate's information in the drawer, clearly decoded into separate fields; this information is
available to an agent too." Lens Desktop has no such section: this is beyond Lens, and nothing of
it is taken or inferred from Lens. It is a deliberate exception to the rule that a Secret's values
never reach an agent ([agents and MCP](../security/agents-and-mcp.md#rules)): a certificate is
public material, which every TLS client is sent anyway; the PEM text, the values and every private
key stay withheld. **Built 2026-10-13** ([`engine/secretcerts.go`](../../internal/engine/secretcerts.go),
[`model/certs.go`](../../internal/model/certs.go); [architecture §5](architecture.md#5-cache)).

**The owner's answers to the build's questions (2026-10-13):**

1. **Agents read the certificates, the built-in agent's LLM endpoint included** ("Absolutely
   not!" to withholding them): an agent may read every certificate, above all in its decoded form,
   and never a key. What was open as H-45 (a certificate's names, internal host names and e-mail
   addresses among them, reaching agents and the LLM endpoint) is decided: it is the rule, not a
   gap, and H-45 left [agents and MCP](../security/agents-and-mcp.md#rules) (its number is not
   reused).
2. **Check whether `tls.crt` matches `tls.key`** ("Yes, that would be fantastic!"): built the same
   day, below. For it Cogpit now parses a private key in memory, the one place it does
   ([credentials](../security/credentials.md#secret-objects)).
3. **Decode DER (binary) certificates too** ("Yes!"): built the same day, below.

What the build chose, Cogpit's defaults (the lead's, not the owner's decision):

- **Which keys.** Every key whose value holds a PEM `CERTIFICATE` block or DER certificates, in a
  Secret of any type (`kubernetes.io/tls`, Opaque, a service account token's `ca.crt`,
  cert-manager's), `tls.crt` and `ca.crt` first, then the other keys sorted. A `kubernetes.io/tls`
  Secret's `tls.crt` that holds none says "no certificate in this value". Only certificates are
  decoded (Go's `crypto/x509`): in PEM, the parser hands `pem.Decode` nothing but the text from a
  `CERTIFICATE` block's BEGIN line to its END line, and a block with another BEGIN line before its
  END is invalid, so a `PRIVATE KEY` block beside a certificate is never decoded; a block that does
  not decode or parse says "not a valid certificate" without quoting it. A key named like a private
  key (`tls.key`, `*.key`, `*key.pem` such as `server-key.pem` and Let's Encrypt's `privkey.pem`,
  `*key.der`) is never searched for certificates.
- **DER** (the owner's answer 3). A value with no PEM certificate is read as DER, one ASN.1
  element after the other, when its first element is shaped like a certificate (a SEQUENCE whose
  first element is a SEQUENCE; a private key in PKCS #1, PKCS #8 or SEC 1 starts with an INTEGER,
  so a DER key is never handed to the certificate parser) and parses as one; a later element that
  is none is "not a valid certificate". Any other binary value holds no certificate and shows
  nothing. Agents read `der: true` on such a key.
- **Key match** (the owner's answer 2). A key with certificates pairs with a private key where
  the pairing is unambiguous: `<base>.crt`, `.pem`, `.cer` or `.der` with whichever one of
  `<base>.key` and `<base>-key.pem` the Secret has (`tls.crt` with `tls.key`, `ca.crt` with
  `ca.key`, `server.pem` with `server-key.pem`; none when it has both); otherwise, when the Secret
  has exactly one key with certificates and exactly one key named like a private key, those two
  (`cert.pem` with `privkey.pem`); Let's Encrypt's four files pair with nothing. The private key
  (at most 64 KiB; PEM or DER, PKCS #1, PKCS #8 or SEC 1; RSA, ECDSA, Ed25519, X25519) is parsed in
  memory, its public half compared with the value's first certificate's public key, and dropped:
  the parsed key lives in one function call ([`model/keymatch.go`](../../internal/model/keymatch.go))
  and is never returned, stored or cached; the decoded value and the PEM block are zeroed, an
  Ed25519 key too (an RSA or ECDSA key's numbers wait for the garbage collector: Go cannot wipe
  them). A group's "Checks" section shows "Key Matches": "Yes (tls.key)" (OK), "No (tls.key holds
  another key)" (error), "Key not readable (tls.key: encrypted)" (unknown; also "unsupported key
  type" for an OpenSSH key or an algorithm Go does not know, "not readable", "too large"). Agents
  read `pairedKey` and `keyMatches`: true, false or the reason. No parser error leaves the check,
  so nothing quotes the key.
- **Chain** (Cogpit's default, which the lead left optional; built because it is cheap). In the same Checks section,
  "Chain In Order": whether each certificate of the value is signed by the next
  (`CheckSignatureFrom`: the next must be a CA whose key verifies the signature; no system roots,
  no revocation), "No, certificate 1 is not signed by certificate 2" an error. Checked for a value
  that pairs with a private key or whose first certificate is no CA (a leaf and its chain), not
  for a CA bundle of several roots; not for one certificate, a value with an invalid entry, or a
  SHA-1 signature Go no longer verifies. Agents read `chainInOrder` and `chainBreak`.
- **Bounds.** A value over 1 MiB is not decoded (a key named like a certificate, `*.crt`, `*.pem`,
  `*.cer`, `*.der`, says so); at most 16 certificates per key and 64 per Secret are decoded, the
  rest counted ("4 more certificates are not decoded"); a private key over 64 KiB is not read
  ("too large"); each text taken from a certificate is cut at 1 KiB, each list of names at 100
  entries, and control characters and bidirectional overrides in it are shown as `\x0a`, `\u202e`
  (a name with a line break would read as two names).
- **Fields,** each its own drawer item, empty ones left out: Subject (the common name) and Subject
  DN (the whole name, RFC 2253), Issuer and Issuer DN, Serial Number (hex, colon-separated), Not
  Before, Not After, DNS Names, IP Addresses, URIs and E-mail Addresses (one per line, a list of
  more than 10 folded behind its count), Public Key (`RSA 2048`, `ECDSA P-256`, `Ed25519`),
  Signature Algorithm, CA (`No`, `Yes, path length 0`, `Yes, no path length limit`), Key Usage,
  Extended Key Usage, Self-Signed (the issuer is the subject and the certificate's own key verifies
  its signature), SHA-256 Fingerprint, Subject Key ID, Authority Key ID.
- **Validity.** Times in the Preferences' time zone and the user's format, as Created is (RFC 3339
  until [D-31](#d-31-every-timestamp-in-the-local-time-zone-and-format)); Not After adds the
  time left ("in 90 days", "in 5 hours") or "expired 3 days ago" and is a status pill: OK with
  more than 30 days left, warning at 30 days or fewer, error when expired or not yet valid; Not
  Before is an error pill with "(not yet valid, in 5 days)" while it is.
- **Layout.** After the Data section, one group per key, "Certificates · tls.crt", modelled on the
  drawer's Containers: the Checks section first when there is a check, then one section per
  certificate in the value's order (a chain's leaf, then its
  intermediates and CA), titled by the subject's common name (else the subject), numbered in a
  value with several ("1 · shop.example.com"). Every field has the drawer's copy icon (D-25), which
  copies the value as shown (a name of a list in YAML syntax, as every list entry). The list views
  are not changed.
- **Agents.** `object_get` carries the same sections and a `certificates` list: per key `key`,
  `der`, `certificates`, `omitted`, `notDecoded`, `pairedKey`, `keyMatches`, `chainInOrder` and
  `chainBreak`; per certificate the same fields, the times as RFC 3339
  in UTC, `expiresInDays` (whole days left, rounded down, negative once expired), `expired`,
  `notYetValid` and `status`; an invalid block is `{"invalid": true, "error": "not a valid
  certificate"}`. Never the PEM text, a value of any key or a private key
  (`TestObjectGetDecodesCertificatesForAgents`). Nothing new in the audit log: reading is a read.
- **Not done.** Cogpit does not verify a chain against a trust store or for revocation, and an OK
  pill on Not After means only "within its validity period". An encrypted private key is not
  decrypted (Cogpit has no passphrase): it shows in the warning colour, "Encrypted key: TLS and
  Ingress cannot use it" (A100, built 2026-10-14). A private key inside a value with the
  certificates (a combined PEM file) is compared with the value's own first certificate, "Yes (in
  this value)" (A99, built 2026-10-14; `TestSecretCombinedPEMKeyCheck`). A value wider than the drawer (the SHA-256 fingerprint at the
  default width) is shortened with "…"; its copy icon copies it whole.

Tests: `TestParseCertificatesChain`, `TestParseCertificatesOnlyCertificateBlocks`,
`TestParseCertificatesBounded`, `TestCertificateEvaluate`, `TestCertificateJSON`,
`TestCertificateTextsAreSafe`, `TestSecretCertificateSections`,
`TestSecretCertificateStatusesAndBounds`, `TestSecretCertificateEdges`,
`TestObjectGetDecodesCertificatesForAgents`, `TestSecretDrawerShowsCertificates`; the owner's
answers: `TestMatchPrivateKeyPairs`, `TestMatchPrivateKeyReasons`, `TestKeyMatchJSON`,
`TestChainInOrder`, `TestDecodeDERCertificates`, `TestSecretKeyMatchesCertificate` (matching RSA,
ECDSA and Ed25519 pairs, a mismatched pair, an encrypted key, garbage; the key's PEM, base64 and
DER appear nowhere in the drawer model), `TestSecretCertificatePairingAndChain`,
`TestSecretDERCertificates` (a DER `ca.der` and `tls.crt`, DER garbage, a DER private key under a
certificate's name: no fields, no leak); `TestObjectGetDecodesCertificatesForAgents` also checks
that no form of either private key reaches the result, the audit log, an event or the log; on kind
`TestKindSecretCertificates` (a `kubernetes.io/tls` Secret created through the API, read by an
agent's `object_get`, its key matching and its chain in order); renders
`TestRenderSecretCertificatesDark` and `TestRenderSecretCertificatesLight` (PNG
[dark](../../internal/ui/fyneui/testdata/secret_certs_dark.png),
[light](../../internal/ui/fyneui/testdata/secret_certs_light.png)).

**The decision round of 2026-10-14 evening (owner):**

- **Decided (A99):** a combined PEM, certificate and private key in one value, gets the
  key-matches-certificate check too. Built 2026-10-14 (see "Not done" above).
- **Decided (A100):** an encrypted private key is shown in the warning colour with "encrypted key:
  TLS/Ingress cannot use it". Built 2026-10-14: "Encrypted key: TLS and Ingress cannot use it
  (tls.key)", status warning.
- **New requirement (R1):** besides the key check and the chain, show whether the chain leads to a
  public CA or an internal (private) one, so that the kind of certificate is clear at a glance.
  Built 2026-10-14 night as **the lead's design, for the owner to confirm**
  ([`model/certanchor.go`](../../internal/model/certanchor.go); `TestCertAnchor`,
  `TestSecretChainAnchor`): three answers in the Checks section's "Chain Leads To" and for agents
  (`anchor`: kind and root) — "public" chains to a root of Mozilla's NSS program (the bundle of
  `golang.org/x/crypto/x509roots/fallback/bundle`, a new module, embedded and the same on every
  computer; only the bundle package is imported, never `fallback`, which would change Go's
  fallback roots for the whole process), "a CA this Mac trusts, not a public one" (the computer's
  trust store, `x509.SystemCertPool`, where a company's root counts), "internal or self-signed"
  (neither; named by the top certificate's issuer). Options weighed: the Mac's trust store alone
  (a corporate root would read as public), the public bundle alone (no answer for a corporate
  root). Judged for a checked chain only (a leaf, or a value with its key; a CA bundle gets none),
  at the middle of the first certificate's validity, so that an expired certificate still names its
  anchor; no revocation. The bundle moves with the module's version (Renovate).

## D-31: Every timestamp in the local time zone and format

**Decided (owner, 2026-10-13):** "Timestamps always in the local time zone, summer and winter time
respected, in the format that fits the location. Later it may be configurable in the Settings, but
the default is the local time zone and the local format. For me that would be 10.09.2026 17:41:01.
This applies to every timestamp shown." (Seen in the drawer: a certificate's Not Before read
`2026-09-10T17:41:01+02:00`.) **Built 2026-10-13** ([`internal/locale`](../../internal/locale/),
[`model/timefmt.go`](../../internal/model/timefmt.go); [architecture §3](architecture.md#3-dependency-layering-enforced),
[§4.4](architecture.md#44-tables-details-editors-and-streams)). **Not built:** the format as a
setting (the owner's "later"); the code takes a layout everywhere, so that a preference can replace
the computer's.

What the build chose, Cogpit's defaults (the lead's and the build's, not the owner's decision):

- **The format is the operating system's region, not `$LANG`.** On macOS Cogpit asks
  CoreFoundation for the current locale's short date style and medium time style
  (`CFLocaleCopyCurrent`, `CFDateFormatterCreate`): the region in System Settings, a user's own
  formats and the 24-hour switch included, and a Finder-started app has no `$LANG` anyway. A
  two-digit year is written with four digits; date and time are joined by one space. On the
  owner's Mac (region `en_DE`, the shell's `LANG=en_US.UTF-8`) macOS answers `dd.MM.yy` and
  `HH:mm:ss`, written `10.09.2026 17:41:01` (verified 2026-10-13, `TestSystemFormatOnMacOS` logs
  it). Linux reads the POSIX locale (`LC_ALL`, else `LC_TIME`, else `LANG`) against a table of
  common locales' CLDR patterns (42 territories and 23 languages, taken from macOS's ICU on
  2026-10-13), by territory, else by language; C, POSIX, an unknown locale or none: ISO 8601,
  `2026-09-10 17:41:01`. **Windows reads the environment too, for now**: its region settings are
  not read, so a Windows session without `LANG` gets ISO 8601. `cogpit-cli` reads the environment
  on every platform, as command-line tools do.
- **Conversion** ([`locale.GoLayout`](../../internal/locale/icu.go)). Unicode's patterns (UTS #35)
  become Go layouts: day, month (number, short or full name), year, weekday, 12- and 24-hour
  hours, minutes, seconds, fractions, AM/PM, zones, quoted literals; narrow and non-breaking
  spaces become spaces. Approximations: Go writes 24-hour hours with two digits (Spain's `H:mm:ss`
  reads `07:41:01`, not `7:41:01`) and names and AM/PM in English. A pattern with a field Go cannot
  write (an era, a quarter, a week, the day period `B`), literal text Go would read as a field, or
  more than 128 bytes falls back to ISO 8601 for its part; so do a date without day, month and
  year and a time without hours and minutes.
- **One formatter.** The binaries look the layout up once at start and hand it to the engine
  (`engine.Options.TimeLayout`); the engine writes every time the user reads in it and in the
  Preferences' time zone (`Session.TimeFormat`), and the window writes with the same layout
  (`UIState.TimeFormat`) and zone (`shell.timeFormat`). Summer and winter time come from the zone
  database: Berlin is UTC+2 in September and UTC+1 in December
  (`TestTimeFormatZoneAndSummerTime`). Derived forms: the time without seconds and the day
  without the year (charts' axes), and with milliseconds (log lines, several of which share a
  second).
- **Where.** The drawer: Created and every absolute time a kind shows (a container's start and end,
  a job's start and completion, a cron job's schedules, leases, autoscalers, volumes, events' first
  and last seen, resource claims, custom resources' date columns, a Helm chart's Created and its
  versions' dates, a Helm release's Updated, a certificate's Not Before and Not After). A kind's
  drawer writes a time as a mark that the engine replaces when it builds the drawer
  (`model.MarkTime`, `TimeFormat.WriteSections`). An event's Last Seen in the Events list (a
  column type `time`, written when the cell is drawn, sorted by time). The logs' timestamps, their
  download, and the logs' time field, which reads the same format (and ISO 8601, RFC 3339 and the
  other forms) in the Preferences' zone, where it read UTC before. The notifications and the audit
  log (the time of day today, the date before it otherwise), the drain, Helm and drafts banners,
  the Rollback dialog, the Connection Updates log, the metrics providers' tooltip, the charts'
  axes and hover panels, the crash notice, the CSV export.
- **Kept as they are.** Ages and countdowns ("5m", "160d", "In about 1 hour 57 min"), and what
  Cogpit writes for machines: its files (journals, settings, the crash report's file name), an
  object's YAML, and the structured times of MCP results (certificates' `notBefore` and
  `notAfter`, notifications, audit entries, a logs tab's jump), which stay RFC 3339. A credential
  plugin file's fingerprint note keeps its RFC 3339 time, because the approval compares it.
- **Agents.** What an agent reads as drawn, `view_rows` and `resources_list` cells and
  `object_get`'s sections, is in the user's format; `ui_state` names it (`timeFormat`, Go layouts),
  so that an agent can tell `09.10.` from `10/9` (`TestAgentsReadTimesAsDrawn`).
- **Search.** The list's search reads cells' text, and an event's Last Seen has none now (as an
  age has none): searching for a time no longer finds an event.

Tests: `TestGoLayout`, `TestFromPatterns`, `TestEnvironment`, `TestTableConverts`,
`TestSystemFormatOnMacOS` (macOS with cgo only; no assertion on the developer's region),
`TestTimeFormatZoneAndSummerTime`, `TestDerivedLayouts`, `TestMarksWrittenInTheFormat`,
`TestColumnTime`, `TestCreatedFollowsTheTimezone`, `TestParseLogTimeInTheUsersFormat`,
`TestJumpLogsInTheUsersZone`, `TestAgentsReadTimesAsDrawn`,
`TestTheWindowWritesTimesInTheUsersFormat`, `TestLogTimestampsInTheUsersFormat`; renders
`TestRenderTimesInTheUsersFormat` (PNG [dark](../../internal/ui/fyneui/testdata/logs_times_de_dark.png),
[light](../../internal/ui/fyneui/testdata/logs_times_de_light.png): Germany's format, Berlin's time).
The test fixtures carry no layout, so the other images show the ISO 8601 fallback.

## D-32: Every byte amount in binary units with one decimal

**Decided (owner, 2026-10-13):** "Unacceptable. Bytes only up to 1023; from 1024 bytes KiB, from
1024 KiB MiB, and so on, each with one decimal. This applies to every presentation of bytes,
unless you see it differently: then name me the place where we should not do it and we talk about
it." (Seen in a Node's drawer: Capacity Memory `16360780Ki`, Ephemeral Storage `119842348Ki`,
Allocatable Memory `13856852378`, plain bytes.) **Built 2026-10-13**
([`model/bytes.go`](../../internal/model/bytes.go); [architecture › Current state](architecture.md#current-state-2026-10-16)):
that drawer now reads 15.6 GiB, 114.3 GiB and 12.9 GiB (`TestNodeDrawerBytesInBinaryUnits`).

What the build chose, Cogpit's defaults (the lead's and the build's, not the owner's decision):

- **The form.** A space between number and unit, as IEC 80000-13 writes it ("512 B", "15.6 GiB");
  the units B, KiB, MiB, GiB, TiB, PiB and EiB; whole bytes below 1024 without a decimal. A
  decimal point whatever the region: the UI is English, and unlike D-31's dates the separator does
  not follow the computer's region. The unit is chosen by the number as written, so that rounding
  never shows 1024.0 of a unit: 1 MiB less one byte reads "1.0 MiB", not "1024.0 KiB". An amount
  that need not be whole (a chart's value, a rate averaged over a window) is rounded to whole
  bytes below 1 KiB ("0 B" for 0.25).
- **One formatter** in L0, for the engine, the window and the CLI alike: `model.FormatBytes` (a
  count), `FormatByteAmount` (a float), `FormatByteRate` (bytes per second, ".../s"),
  `FormatByteQuantity` (a Kubernetes quantity of bytes as an object writes it: "16360780Ki",
  "13856852378", "10Gi", "1G", "500M") and `FormatResource` (one entry of a resource list: bytes
  for `memory`, `storage`, `ephemeral-storage` and `hugepages-*`, also as a resource quota names
  them, `requests.memory`, `limits.ephemeral-storage`,
  `<class>.storageclass.storage.k8s.io/requests.storage`; CPU, counts and extended resources as
  written). A quantity in milli-units ("500m", "2000m": almost always a mistake the user should see
  as written), a fraction of a byte ("1.5"), text that is no quantity, and a quantity beyond what
  Kubernetes holds (it clamps "10Ei" to 8 EiB) are shown as written (`TestFormatBytes`,
  `TestFormatByteAmount`, `TestFormatByteQuantity`, `TestByteResource`).
- **Where.** The lists: Persistent Volumes' Capacity, Persistent Volume Claims' Size, the Pods
  list's Memory (and so the node drawer's Pods table, which reuses its cells), the Vertical Pod
  Autoscalers' Memory, Horizontal Pod Autoscalers' Metrics for a memory or ephemeral-storage
  resource metric ("memory: 288.0 MiB/512.0 MiB"). The drawers: a node's Capacity and Allocatable
  (memory, ephemeral storage, huge pages; CPU, pods and devices as written), a container's
  Requests and Limits, a volume's Capacity, a claim's Requested, Limit, Capacity and Allocated, a
  stateful set's volume claim templates, a resource quota's Used and Hard, a limit range's Min,
  Max, Default Request and Default Limit (not its Max Limit/Request Ratio, a ratio), a runtime
  class's Pod Overhead, a vertical pod autoscaler's policies and recommendations, a device's
  capacity named `memory` (resource slices and a node's Devices), a Config Map's binary data
  sizes and its cut notes ("[cut: 4.0 KiB of 8.8 KiB shown; …]", the Helm record note). The
  charts: the drawers' Memory and Filesystem axes and hover panels, the Network and Filesystem
  rates ("48.0 KiB/s"), the cluster Overview's Memory chart and its headline ("41.9 GiB of
  118.0 GiB (35%)"); the axis writes what the hover panel writes, and a byte axis steps by at
  least one byte, since its labels are whole bytes (`TestRateFormatsAndLongRanges`,
  `TestDrawerChartsHoverPerChartAndVariant`; PNG
  [dark](../../internal/ui/fyneui/testdata/chart_hover_memory_dark.png),
  [light](../../internal/ui/fyneui/testdata/chart_hover_memory_light.png)).
  Sizes of data: a Secret's masked values and binary values in the drawer ("•••••••• (38 B)",
  "binary data, 1.2 KiB"; "38 bytes" before), a revealed value's cut note (which wrote 1.5 MiB as
  "1536.0 KiB"), a certificate value too large to decode ("larger than 1.0 MiB"), the kubectl
  download's progress ("22.0 MiB of 54.4 MiB"; decimal "23.1 MB of 57.0 MB" before), the Audit
  Log tab's note on a partial read ("the log's last 4.0 MiB") and on parameters too large to list.
  The messages that state a size: files and downloads against Cogpit's bounds (kubeconfig files,
  Helm's `repositories.yaml`, indexes, chart archives, the kubectl binary and its signature files,
  Prometheus responses, a Helm release payload, private files, the cluster icon "larger than
  1.0 MiB" where it read "1024 KiB"), the editor's "larger than the 4.0 MiB an editor takes", a
  journal that is too large.
- **A size beside its limit is not named** (Cogpit's default): one decimal can make a size just
  over its limit read like the limit ("the binary has 4.0 KiB, more than the 4.0 KiB Cogpit
  accepts" for 4,096 bytes against 4,095), so those messages name the limit only ("the binary is
  larger than the 4.0 KiB Cogpit accepts"), and a download that ended short names the difference
  ("received 10 B less than the 54.4 MiB announced").
- **Sorting.** The sort keys stay bytes (`Cell.Num`), so sorting by size still works; only the text
  changed (`TestPersistentVolumeClaimSizeSorts`).
- **Agents.** What an agent reads as drawn follows the formatter: `view_rows` and
  `resources_list` cells, `object_get`'s sections. Structured numbers stay raw: `metrics_get`'s
  `memoryBytes`, the Overview's chart values, a revealed value's `size`, an object's YAML.
- **Nothing to convert yet:** an `emptyDir`'s `sizeLimit` (the pod drawer lists volumes by type
  and name only), a Helm chart archive's size (shown nowhere; its download bound is converted),
  the logs and the status bar (they show no byte amount).
- **Renders** (offscreen, re-rendered 2026-10-13): the node drawer
  ([dark](../../internal/ui/fyneui/testdata/node_drawer_dark.png), [light](../../internal/ui/fyneui/testdata/node_drawer_light.png)) and with its
  Pods ([dark](../../internal/ui/fyneui/testdata/node_pods_drawer_dark.png), [light](../../internal/ui/fyneui/testdata/node_pods_drawer_light.png)),
  a node's Devices ([dark](../../internal/ui/fyneui/testdata/node_devices_drawer_dark.png), [light](../../internal/ui/fyneui/testdata/node_devices_drawer_light.png)),
  a pod's containers ([dark](../../internal/ui/fyneui/testdata/pod_drawer_ports_dark.png), [light](../../internal/ui/fyneui/testdata/pod_drawer_ports_light.png)),
  the claims list ([dark](../../internal/ui/fyneui/testdata/pvc_list_dark.png), [light](../../internal/ui/fyneui/testdata/pvc_list_light.png)),
  a Secret ([dark](../../internal/ui/fyneui/testdata/secrets_dark.png), [light](../../internal/ui/fyneui/testdata/secrets_light.png)),
  the demo's Pods list ([PNG](../../internal/ui/fyneui/testdata/demo_pods.png)), the cluster Overview
  ([dark](../../internal/ui/fyneui/testdata/cluster_overview_dark.png), [light](../../internal/ui/fyneui/testdata/cluster_overview_light.png)).

**Not converted: the places to talk about** (the owner's "name me the place"; each kept as it is
until the owner decides):

1. **The YAML view and the editors** ([`yamlview`](../../internal/ui/fyneui/yamlview/), the Edit,
   Create and Helm tabs): they show the object as it is, `memory: 128Mi` included.
2. **Decimal suffixes are converted**, the lead's point: `memory: 1G` in a spec reads "953.7 MiB"
   in the drawer and "1G" in the YAML, so the drawer never shows what the user wrote; "500M" reads
   "476.8 MiB", "262144k" "250.0 MiB". Built as the rule says.
3. **Huge pages' names keep their page size**: the node drawer's titles "Hugepages 1Gi" and
   "Hugepages 2Mi" ([`nodes.go`](../../internal/model/kinds/nodes.go) `resourceTitles`), and
   `hugepages-2Mi` wherever a resource is named (a quota's Resource column, a container's
   requests): the resource's name, not an amount. Their amounts are converted ("0 B").
4. **A device's capacity under another name than a byte resource**
   ([`resourceslices.go`](../../internal/model/kinds/resourceslices.go) `deviceCapacity`): a DRA
   driver names its capacities and Kubernetes declares no unit for them; "memory: 40Gi" reads
   "40.0 GiB", "size: 3840Gi" stays as the driver wrote it.
5. **The audit log's records** ([`audit.go`](../../internal/engine/audit.go): "<1234 bytes of
   manifests; not shown>", "<N bytes that are not manifests; not shown>", "<N bytes of Helm
   values; not logged>"): what the file records, with exact counts and in the form older lines
   have; the same note stands for a half-typed Secret in an editor tab an agent reads. The Audit
   Log tab's own note on parameters too large to list is converted.
6. **A credential plugin file's fingerprint note**
   ([`pluginfiles.go`](../../internal/cluster/pluginfiles.go) `hashRegular`: "larger than N bytes
   (N bytes, modified …)"): the approval binds the note in place of the content; another text
   would ask again for every such approval, and the exact size is part of the binding (as D-31
   keeps its RFC 3339 time).
7. **Limits on typed text**, lengths rather than amounts: a shell path and its arguments and the
   terminal's working directory ("one line of at most N bytes",
   [`preferences.go`](../../internal/engine/preferences.go),
   [`clustersettings.go`](../../internal/engine/clustersettings.go)), the proxy URL
   ([`proxy.go`](../../internal/engine/proxy.go)), a list's regular expression
   ([`filteroptions.go`](../../internal/engine/filteroptions.go), 1000 bytes), the find bar
   ([`findbar.go`](../../internal/ui/fyneui/findbar.go)), a logs search
   ([`commands/logs.go`](../../internal/engine/commands/logs.go)), the kubectl mirror
   ([`kubectl/download.go`](../../internal/kubectl/download.go)), a terminal kubeconfig's names
   ([`kubeconfig/terminal.go`](../../internal/kubeconfig/terminal.go)), a date pattern
   ([`locale/icu.go`](../../internal/locale/icu.go)). Bytes stand for characters there; "at most
   4.0 KiB" for a shell path would read as a file size.
8. **Command descriptions and parameter schemas** ([`engine/commands`](../../internal/engine/commands/):
   "at most 1 MiB", "the last 4 MiB", "at most 1000 bytes"), and the README's command table made
   from them: prose for agents, where a limit reads as a person writes it.
9. **The performance harness** ([`perf.go`](../../internal/ui/fyneui/perf.go) "YAML editor,
   5 MB", `cogpit --perf-mode`'s help): a developer's tool, not the product.
10. **Measurements recorded in the docs** ("15.8MiB" on kind in
    [architecture §9](architecture.md#9-metrics-internalmetrics)): records of what was seen, kept
    as written.

Other details of the build, for the owner to know: an amount that need not be whole is rounded to
whole bytes below 1 KiB, so a rate of 0.25 B/s reads "0 B/s" (it read "0.3B/s"); Lens's own form
of a size (with or without a space, "Mi" or "MiB") is not captured, so Cogpit's lists may differ
from Lens's there.

## D-33: The Flux actions

**Context.** D-9 puts Flux CD in the beta; the owner's team works with Flux daily
(`k8s-flux-base`). Lens's Flux section is captured as a sidebar structure only
([ui-parity › Cluster sidebar](../parity/ui-parity.md#cluster-sidebar)); its actions are neither
documented nor captured ([resource-parity › Paid Lens Desktop views](../parity/resource-parity.md#paid-lens-desktop-views)).
Flux's own CLI does three things to an object every day: `flux reconcile`, `flux suspend`,
`flux resume`. **Built 2026-10-13** ([`commands/flux.go`](../../internal/engine/commands/flux.go),
[`cluster/flux.go`](../../internal/cluster/flux.go); `TestFluxActionsPatch`,
`TestFluxActionsApply`, `TestFluxActionsAgentsNeedAcceptance`, `TestFluxActionsOnASelection`,
`TestKindFlux`); Reconcile with Source and Reset **built 2026-10-14**
([`commands/fluxsource.go`](../../internal/engine/commands/fluxsource.go),
[`kinds/flux_withsource.go`](../../internal/model/kinds/flux_withsource.go);
`TestFluxReconcileWithSource`, `TestFluxReset`, `TestFluxReconcileWithSourceAgents`,
`TestFluxReconcileWithSourceOnASelection`, `TestFluxActionsWithSourceInTheMenus`,
`TestFluxSourceOf`, `TestFluxPatch`, `TestFluxHelmReleaseActionsInTheWindow`, `TestKindFlux`; PNGs
[drawer dark](../../internal/ui/fyneui/testdata/flux_helmrelease_drawer_dark.png),
[drawer light](../../internal/ui/fyneui/testdata/flux_helmrelease_drawer_light.png)).

- **Decided (owner, 2026-10-14): which actions.** Options (a) Reconcile only, (b) Reconcile,
  Suspend and Resume, as the flux CLI's everyday commands, (c) (b) plus Reconcile with Source,
  Reset and Force as separate menu entries; the night build had chosen (b). The owner chose (b)
  plus **Reconcile with Source and Reset as the flux CLI does them**; **Force stays as built**, a
  parameter of `flux.reconcile` and `flux.reconcileMany` for Helm releases, for agents only (no
  menu entry).
  - **Reconcile with Source** (`flux.reconcileWithSource`, `flux.reconcileWithSourceMany`;
    Kustomizations and Helm releases), as `flux reconcile kustomization|helmrelease --with-source`
    (flux2's `cmd/flux`, read 2026-10-14): `reconcile.fluxcd.io/requestedAt` on the source first,
    then on the object, each patch bound to its object's UID like Reconcile's. The source: a
    Kustomization's `spec.sourceRef` (a GitRepository, OCIRepository or Bucket, in the
    Kustomization's namespace unless it names one); a Helm release's `spec.chartRef` (an
    OCIRepository, or a HelmChart), else the HelmChart helm-controller makes of `spec.chart`
    (`<namespace>-<name>` of the release, in the namespace of the chart's `sourceRef`). A HelmChart
    is reconciled with its own source before it (a HelmRepository, GitRepository or Bucket), as the
    flux CLI does, and an OCI HelmRepository is skipped, which the flux CLI skips too ("reconciliation
    not supported by the object"). Greyed with the reason on a suspended object, on one whose
    source is of a kind no Flux controller reconciles (an ExternalArtifact), when the cluster does
    not serve the source's kind, and when the access reviews say the user may not patch the source.
    A source shared by several selected objects is patched once.
  - **Reset** (`flux.reset`, `flux.resetMany`; Helm releases), as `flux reconcile helmrelease
    --reset`: `reconcile.fluxcd.io/resetAt` and `reconcile.fluxcd.io/requestedAt` with one token
    (helm-controller's `api/v2`: resetAt takes effect only with requestedAt's value), so that
    helm-controller forgets the release's failure counts and tries again after it ran out of
    retries. Not offered for a suspended release, as Reconcile is not.
- **Cogpit's default (the lead's, 2026-10-14; the owner left it open, so it stands until he asks):
  whether Reconcile with Source waits for the source.** Options (a) as the flux CLI: annotate the source, wait until its
  controller reports it ready (a job with a timeout), then annotate the object; (b) annotate the
  source, then the object at once; chosen (b), because Cogpit's Reconcile does not wait for a
  controller either, and kustomize-controller and helm-controller reconcile an object again by
  themselves when its source's artifact changes (Flux's documentation; inferred, not verified on a
  cluster that runs the controllers), so the object ends on the new revision without a wait that
  would outlive the menu click. **Decided (owner, 2026-10-14 evening, A91): option (a)**, as the
  flux CLI: a background job waits, with a timeout, until the source is ready, then annotates the
  object; its progress shows in the dock. **Built 2026-10-15** ([`commands/fluxsourcejob.go`](../../internal/engine/commands/fluxsourcejob.go),
  [`engine/jobs.go`](../../internal/engine/jobs.go), [`engine/fluxwait.go`](../../internal/engine/fluxwait.go),
  [`fyneui/jobtab.go`](../../internal/ui/fyneui/jobtab.go); `TestFluxReconcileWithSourceWaitsForTheSource`,
  `TestFluxReconcileWithSourceStopsAtASourceNotReady`, `TestFluxReconcileWithSourceTimesOut`,
  `TestFluxReconcileWithSourceStopsWithItsTab`, `TestFluxReconcileWithSourceJobForAgents`,
  `TestFluxReconcileWithSourceJobOnASelection`, `TestWaitFluxReconciled`, `TestJobRunsInItsTab`,
  `TestJobStops`, `TestJobTabFollowsTheEngine`, `TestKindFlux` playing the controllers through the
  status subresource; PNGs [dark](../../internal/ui/fyneui/testdata/job_tab_dark.png),
  [light](../../internal/ui/fyneui/testdata/job_tab_light.png)). The command checks everything as
  before and returns at once with the job's dock tab; the job asks each object of the chain in turn
  (the deepest source first) and waits for its controller through the cache, never polling: done
  when `status.lastHandledReconcileAt` answers the request (Cogpit's token is the time it sent it,
  so a token at or after it counts; a token that is no time counts when it changed, the flux CLI's
  test) and the object is settled (not Reconciling, its generation observed) and Ready; a source
  that ends not ready, times out or fails stops what depends on it, as the flux CLI stops. The
  tab's steps, state and outcome are engine state (`DockTab.Job`), which agents read with
  `ui_state`; a toast says how it ended. A crash ends the job and leaves at most sources asked to
  reconcile whose objects were not asked yet: harmless, the source's controller fetches it again
  and the object's controller reconciles at its interval or when the artifact changes, as if nobody
  had asked; so it keeps no journal (D-11). **The lead's decisions for the owner to confirm (night
  run, 2026-10-15):** (1) **what the job waits for:** options (a) the source alone, then the object
  annotated and the job done, (b) the source, then the object too, as the flux CLI waits for both;
  chosen (b), so that the tab ends with the object's own verdict. (2) **the timeout:** options (a)
  5 minutes per object, the flux CLI's `--timeout` default, (b) one bound for the whole job, (c) a
  parameter; chosen (a), no parameter. (3) **where the progress shows:** options (a) a dock tab per
  job (`DockType` `job`, generic, so that later long-running commands can use it), (b) one shared
  Jobs tab, (c) the drain banner's place; chosen (a): the dock is where the owner asked for it, and
  closing the tab is the way to stop the job (it undoes nothing). (4) **a selection:** one job for
  all objects, their chains side by side, a source two share asked once, one that fails stopping
  only the objects that need it. (5) **agents:** the command returns at once with the job's tab;
  the progress is in `ui_state`, no blocking variant. (6) **Flux's HelmCharts:** a chart has no
  curated profile, so the wait reads its conditions with the same rules as a Flux view
  (`kinds.FluxSourceStatus`).
- **Cogpit's default (the lead's, 2026-10-14; the owner left it open, so it stands until he asks):
  what is checked before the first patch.** Options (a) as the flux CLI: each source checked when its turn comes, so a
  suspended HelmChart stops the action after its Helm repository was annotated; (b) every object
  of the chain read and checked first (present in the cluster, not suspended), nothing patched when
  one fails; chosen (b), because a refusal then leaves nothing half done; a patch that the API
  server refuses in the middle still stops the action, and the error names what was done.
- **Decided (owner, 2026-10-14; the night build's option (a)): what each writes.**
  Options (a) the flux CLI's patches: Reconcile the annotation `reconcile.fluxcd.io/requestedAt`
  set to the time (RFC 3339 with nanoseconds), Force also `reconcile.fluxcd.io/forceAt`, Suspend
  `spec.suspend: true`, Resume `spec.suspend: false` with a reconcile request; (b) server-side
  apply of the same fields; chosen (a), one JSON merge patch per object, because it is exactly
  what the controllers and the CLI agree on, touches no other field, and needs nothing but `patch`.
  Each patch carries `metadata.uid`, as Restart's and Scale's do, so that an object deleted and
  created again under its name is refused (422 "metadata.uid: field is immutable" on Kubernetes
  1.36.1, `TestKindFlux`). Reset adds `reconcile.fluxcd.io/resetAt` with requestedAt's value.
- **Decided (owner, 2026-10-14; the night build's option (a), Reconcile with Source and Reset
  too): their danger.** Options (a)
  mutating, (b) destructive (confirmed for every origin), (c) mutating, Force destructive;
  chosen (a) for all three, Force included, because none deletes or replaces anything the
  controller would not do on its next interval anyway; Suspend stops reconciliation, which Resume
  undoes. So the GUI's user is not asked, agents need the user's acceptance unless agent writes are
  allowed for the context (D-2), and every agent's call is in the audit log.
- **Decided (owner, 2026-10-14 evening, A34,
  [H-64](../security/privilege-footprint.md#h-64-a-flux-action-is-carried-out-with-the-controllers-identity)):**
  a Flux action's confirmation names the identity the controller acts as: the controller's service
  account, or the object's `spec.serviceAccountName`. Built 2026-10-14: the actions' `Detail`
  ([`commands/fluxidentity.go`](../../internal/engine/commands/fluxidentity.go),
  `TestFluxConfirmationNamesTheIdentity`): "Flux carries it out with the rights of …, not yours";
  `spec.serviceAccountName` is read for Kustomizations and Helm releases (the kinds that
  impersonate with it), every other kind names its controller; a controller's enforced default
  service account (`--default-service-account`) is a flag Cogpit does not read, and the sentence
  says so (the lead's night decision for the owner to confirm).
- **Decided with D-33 as a whole (owner, 2026-10-14; the night build's option (b)): where they
  apply.** Options
  (a) every Flux kind, (b) the kinds whose controller reconciles on request, Suspend and Resume
  where the kind has `spec.suspend`; chosen (b): Reconcile on every kind but External Artifacts
  (another controller makes them), and not on an Alert or Provider that reports no status
  (notification v1beta3's static objects, which no controller reconciles); Suspend and Resume on
  every kind but Image Policies and External Artifacts. Reconcile on a suspended object is greyed
  with the reason (the controller would ignore it, as the flux CLI refuses it); Suspend and Resume
  each show where the other does not, as Cron Jobs' do.
- **Decided (owner, 2026-10-14; the night build's option (b), Reconcile with Source and Reset
  too): selections.** Options
  (a) one object at a time, (b) on a selection too, acting on the selected objects the action applies
  to and skipping the others; chosen (b), as Delete of a selection works (`object.deleteMany`),
  because "reconcile everything that failed" from the Flux dashboard's table is the case the
  dashboard exists for. Ids `flux.reconcileMany`, `flux.suspendMany`, `flux.resumeMany`,
  `flux.reconcileWithSourceMany`, `flux.resetMany`.

Cogpit's defaults, not the owner's decision: the labels Reconcile, Suspend and Resume; their place
in the row menu and the drawer's toolbar before Edit and Delete, Reconcile first, then Reconcile
with Source and Reset; their toolbar icons (Restart's, Suspend's and Resume's of the built-in
kinds; Reconcile with Source a download arrow, Reset a replay arrow); the toast "kustomizations…
reconcile requested", which does not wait for the controller (the object's Status column follows
its conditions), Reconcile with Source's toast at its job's end since 2026-10-15 (A91); Reset offered whatever the release's failure counts say, as the flux CLI's
`--reset` is.

## D-34: The Flux dashboard

**Context.** Lens's Flux section starts with a Dashboard (captured as a sidebar entry only); its
contents are not captured or documented. **Built 2026-10-13**
([`engine/fluxdashboard.go`](../../internal/engine/fluxdashboard.go),
[`fyneui/fluxdashboard.go`](../../internal/ui/fyneui/fluxdashboard.go),
[`model/kinds/flux.go`](../../internal/model/kinds/flux.go); `TestFluxDashboardPage`,
`TestFluxDashboardInTheWindow`, `TestFluxStatusTexts`; PNG
[dark](../../internal/ui/fyneui/testdata/flux_dashboard_dark.png)).

- **Decided (owner, 2026-10-14, as built; the night decision of 2026-10-13): what a Flux object's
  state is.** Options (a) the Ready condition alone (True, False, Unknown), (b) kstatus's reading, as
  Flux documents it: suspended first, then a failure (Ready False or Stalled True), then a
  reconciliation in progress (Reconciling True, Ready Unknown, or a generation the controller has
  not observed), then Ready True, else Unknown; chosen (b), five states Ready (green), Not Ready
  (red), Reconciling (amber), Suspended (grey, the stopped token) and Unknown (grey), because (a)
  shows a suspended or freshly changed object as ready. Notification v1beta3's Alerts and Providers
  report no status at all: they count as Ready unless suspended (Flux treats them as active once
  admitted).
- **Decided (owner, 2026-10-14, as built; the night decision of 2026-10-13): what the page shows.**
  Options (a) counts per kind by state only, (b) the counts and a table of every Flux object, (c)
  the counts and a table of the objects that are not ready; chosen (c): a tile per Flux kind the
  cluster serves (a served kind without objects too), and the table Not Ready of every kind, the
  latest change of the Ready condition first, whose selection takes the Flux actions (D-33).
- **Decided (owner, 2026-10-14): a click on a state opens the kind's view filtered to that state,
  through a status filter every view has (option a).** The options were (a) the kind's view
  filtered to that state, through a status filter the views did not have yet, (b) the view with the
  search set to the state's word, (c) the kind's view unfiltered (the night's choice, built
  2026-10-13). The owner chose (a) and asked for the filter in every view, with the Argo CD
  Dashboard's tiles, which used the search box, switched to it too. **Built 2026-10-14**
  ([`model/states.go`](../../internal/model/states.go),
  [`engine/statusfilter.go`](../../internal/engine/statusfilter.go),
  [`commands/statusfilter.go`](../../internal/engine/commands/statusfilter.go),
  [`fyneui/statusfilter.go`](../../internal/ui/fyneui/statusfilter.go); `TestStatesOfColumns`,
  `TestStateMatcher`, `TestKindStates`, `TestStatusFilterSnapshot`, `TestStatusFilterBelongsToTheView`,
  `TestStatusFilterFluxPage`, `TestArgoTilesNameTheStatusFilter`, `TestWorkspaceKeepsTheStatusFilter`,
  `TestStatusFilterCommand`, `TestStatusFilterInTheHeader`, `TestStatusFilterOnAPage`,
  `TestFluxDashboardInTheWindow`, `TestArgoDashboardPanel`; PNGs the list
  [dark](../../internal/ui/fyneui/testdata/status_filter_menu_dark.png),
  [light](../../internal/ui/fyneui/testdata/status_filter_menu_light.png), a filtered Pods view
  [dark](../../internal/ui/fyneui/testdata/status_filter_pods_dark.png),
  [light](../../internal/ui/fyneui/testdata/status_filter_pods_light.png), the Flux dashboard's Not
  Ready of Kustomizations clicked [dark](../../internal/ui/fyneui/testdata/flux_state_filtered_dark.png),
  [light](../../internal/ui/fyneui/testdata/flux_state_filtered_light.png)). What it is: a tab's own
  state, `UIState.States` (`ui.setStatusFilter`; `ui.openView` with `states` opens a view filtered,
  which is what the dashboards' state links do), applied to the view's snapshot like the search, so
  the table, `view.rows`, select all and the CSV export show the same rows; `resources.list` takes
  `states` too; the snapshot offers the states its rows carry with their counts
  (`TableSnapshot.StateChoices`, `view.rows`' `stateChoices`); the window shows it beside the
  namespace filter, a list of checks with each state's colour and count, a × that clears it in one
  click; kept across restarts with the tab's other state ([D-41](#d-41-what-survives-a-restart)).
  Filtering a 10,000-row snapshot costs no request and 2.9–3.7 ms on the development Mac
  (`BenchmarkSnapshot10kStates`, measured 2026-10-14). Lens's views have no status filter: it is
  Cogpit's addition ([beyond Lens › Everyday use](../parity/beyond-lens.md#everyday-use)).
- **Cogpit's default (2026-10-14; the owner left it open, so it stands until he asks): what a row's
  states are.** Options (a) the rows'
  status tokens only, by name (OK, Pending, Error), (b) the Status column's text, the tokens for a
  kind without one, (c) the texts of the kind's state columns: Status, State, Phase, Sync and Health
  whole, Conditions word by word (a cell drawn in parts part by part), a kind naming its own where
  the default does not fit (an event's Type; a Horizontal Pod Autoscaler's Status, a list of
  condition types), the status tokens by name for a kind without one; chosen (c), because (a)
  merges Running with Ready and cannot tell an Argo CD Application's Sync from its Health, which the
  Dashboard's tiles need, and (b) leaves Deployments, Jobs and Nodes (Conditions) and Argo CD
  Applications (Sync, Health) with tokens only (`KindProfile.States`, `model.DefaultStateColumns`).
  **Confirmed (owner, 2026-10-14 evening, A77):** (c), as built.
- **Cogpit's default (2026-10-14; the owner left it open, so it stands until he asks): a kind with
  several state columns.** Options
  (a) one list of texts, a row shown when it carries any selected text; (b) each state qualified by
  its column's title ("Sync: OutOfSync", "Health: Degraded"), a row shown when it carries a selected
  state of every column the selection names (any of them within a column); chosen (b), because
  both of an Argo CD Application's columns have Unknown, and "Out of sync and Degraded" is the
  question the two columns are for. An unqualified text selected there matches it in any column.
  **Confirmed (owner, 2026-10-14 evening, A78):** (b), as built: AND between the columns, OR within
  one.

Cogpit's defaults for the filter, not the owner's decision (2026-10-14): it belongs to the view it
was set for, as the search does (opening another view clears it; a tab keeps its own, and
`ui.navigate` clears it when it hides the object); the states offered are those of the rows the
namespace filter and the search keep, alphabetical within a column, with each one's count and the
most severe colour its cells draw it in, and the selected ones no row carries listed with 0 so
that they can be unchecked; a state no row carries is not dropped from the filter; the button
reads All states, the one state or "N states", greyed while the rows carry none; on a page (the
Flux and Argo CD dashboards, the Overviews) it sits in the table's header beside the search, since
it filters the table only; at most 100 states of at most 512 bytes each. **Confirmed (owner,
2026-10-14 evening, A79):** the filter belongs to its view and tab, as built. **Decided (A80):** the
button is drawn in the accent colour while a filter is active. Built 2026-10-14 (the button's
high importance, the theme's primary colour; `TestStatusFilterInTheHeader`).

Cogpit's defaults, not the owner's decision: the page's title Flux Dashboard in its header and tab
(the navigator reads Dashboard, as captured), three tiles a row under the navigator's sub-groups'
headings, the table's title Not Ready, and "Every Flux object is ready." when it is empty.

## D-35: What Argo CD's repository and cluster Secrets show

**Decided (owner, 2026-10-14):** option (b), the allowlist of non-credential keys for the window and agents, as built; the night decision below is confirmed.

**The night decision (the lead's, 2026-10-14), confirmed by the owner (2026-10-14).** Argo CD keeps its
registered repositories, credential templates and clusters as Secrets labelled
`argocd.argoproj.io/secret-type` (`repository`, `repo-creds`, `cluster`), and Lens's Argo CD
section shows them as Settings › Repositories (Name, Type, URL, Project, Age) and Settings ›
Clusters (Name, Server, Age) (Lens's documentation, "Using Argo CD", read 2026-10-13; not
captured). Those Secrets hold credentials: a repository's `password`, `username`,
`sshPrivateKey`, `tlsClientCertData` and `tlsClientCertKey`, `githubApp*`,
`gcpServiceAccountKey`, `bearerToken`; a cluster's `config`, its whole client configuration
(bearer token, client key, AWS or exec settings). Options:

| Option | For | Against |
|---|---|---|
| (a) Names only: the Secret's name, namespace and age, no value | Nothing of a Secret's data shown anywhere | Not Lens's views: no URL, type, project or server, which is what a user opens the views for |
| **(b) An allowlist of non-credential keys, for the window and agents (chosen)** | Lens's columns; an agent can answer "which repository does this Application pull from, is it registered, in which project"; a repository URL or a cluster's API server is configuration that Applications and kubeconfigs carry in the clear anyway | A URL can carry a token as user information (`https://user:token@host`), handled below; a list to keep in step with Argo CD's keys |
| (c) The allowlist for the window, nothing for agents | Agents see nothing of these Secrets | Agents lose what the Applications' own specs already tell them (their `repoURL`, their destination's server), for no gain in secrecy |

**Chosen (b)**, because the allowed values are configuration, not credentials, and every other
value stays where the Secrets view already keeps it (masked, revealed in the window only). The
allowlist: repositories `name`, `url`, `type`, `project`; clusters `name`, `server`,
`namespaces`, `clusterResources`, `project` (`kinds.ArgoRepositoryKeys`, `kinds.ArgoClusterKeys`).
**How it is held to:** the views are virtual kinds whose objects Cogpit builds itself
([`kinds/argocd_settings.go`](../../internal/model/kinds/argocd_settings.go), `argoDerive`): from
each labelled Secret only the name, namespace, UID, resourceVersion, creation time, labels, the
Secret type, the allowed keys' values (decoded, at most 2 KiB, control characters dropped) and the
other keys' names. No annotation is copied (kubectl's last-applied annotation repeats the values),
no other value is read, so no row, drawer, YAML, `object_get` result, search, CSV, audit line or
log line can carry one. User information in a URL (`url`, `server`) is replaced with
`<redacted>` (`https://<redacted>@github.com/org`), except an SSH user without a password
(`ssh://git@host`, `git@host:org/repo`), which names no secret (`RedactURLUserinfo`). The other
keys are listed by name ("Values Not Shown"), as the Secrets view lists key names, and a
repository's Authentication column says which method its key names imply (SSH key, username and
password, GitHub App, …) without reading a value. The drawer links to the Secret itself, whose
drawer masks its values as every Secret's does. Not chosen as part of (b): parsing a cluster's
`config` to show its authentication method or `insecure` flag (it would read the credentials to
find the method); Cogpit shows "config" as a key name only.

What stays: the cache holds these Secrets in full while a settings view, its drawer or an agent's
read uses them, as it holds every Secret the Secrets view watches (D-13, [H-66](../security/credentials.md#h-66-argo-cds-credentials-are-in-cogpits-memory-while-its-settings-are-read)).
**Confirmed (owner, 2026-10-14 evening, A32):** they stay whole in the cache, as built; the filter
of the cached Secrets' values in what agents read ([D-44](#d-44-what-agents-read-of-the-docks-logs-and-terminal-tabs))
uses their values.
Tests: `TestArgoSettingsNeverShowCredentials`, `TestArgoSettingsGolden`, `TestRedactURLUserinfo`
(model); `TestArgoCDKindsFollowTheirDefinitions` (engine); `TestArgoSettingsNeverReachAgents`
(commands: `object_get`, `resources_list`, `resources_find`, `view_rows`, the audit log, the
events and the log searched for every credential in every form); on kind `TestKindArgoCD`.

## D-36: How Argo CD's Sync writes the operation

**Decided (owner, 2026-10-14):** as built: Sync writes the Application's `operation` through the Kubernetes API, bound to the UID and resourceVersion; a sync with prune keeps asking twice (the dialog, then the destructive confirmation naming what Argo CD would prune) rather than Lens's one dialog.

**The night decision (the lead's, 2026-10-14), confirmed by the owner (2026-10-14).** Cogpit talks to the
Kubernetes API only, never to Argo CD's own API server (the brief), so Sync
(`argocd.application.sync`, `argocd.application.syncMany`) starts a sync the way Argo CD's API
server does for `argocd app sync`: it writes the Application's `operation`, and Argo CD's
controller runs it. Options for what the write carries and how it guards against a sync someone
else started:

| Option | For | Against |
|---|---|---|
| (a) The minimal operation (`initiatedBy`, `sync.revision`, `sync.prune`), bound to the UID only, refused while the cache shows an operation | Simple | Drops the Application's own sync options (`CreateNamespace=true`, `ServerSideApply=true`) and retry, which `argocd app sync` keeps; an operation started between the cache's last event and the patch is replaced |
| **(b) What `argocd app sync` writes without flags, bound to the UID and the resourceVersion Cogpit checked (chosen)** | The same sync the CLI and Argo CD's UI start: each source's target revision (`revision`, or `revisions` for a multi-source Application), the spec's `syncPolicy.syncOptions` and `retry`, `initiatedBy: cogpit`; a write over a newer version of the Application is refused by the API server (409), so an operation Cogpit has not seen is never replaced | A status update of the controller between check and write refuses the patch: Cogpit then waits for its cache to show the newer version (at most 3 s), checks again that no operation runs and sends the patch again, at most three times in all |
| (c) Argo CD's API server (`/api/v1/applications/<name>/sync`) | Argo CD's own checks (sync windows, its RBAC) | A second credential and endpoint per cluster; against the brief |

**Chosen (b).** Built ([`model/kinds/argocd_ops.go`](../../internal/model/kinds/argocd_ops.go)
`ArgoSyncOperation`, [`cluster/argocd.go`](../../internal/cluster/argocd.go),
[`commands/argocd.go`](../../internal/engine/commands/argocd.go)). Refused, with the reason in the
greyed menu item and the error, while the Application has an operation requested (its `operation`
field set) or running or terminating (`status.operationState.phase`). Not carried over from Argo
CD's API server, and so not checked by Cogpit: sync windows (the controller still honours them:
a denied sync waits or fails in the operation state) and Argo CD's own RBAC (the user's identity
needs `patch` on the Application; what the sync does is Argo CD's,
[H-67](../security/privilege-footprint.md#h-67-a-sync-acts-with-argo-cds-rights-not-the-users)).
Refresh needs no such guard: it sets an annotation the controller removes, bound to the UID only.
**Decided (owner, 2026-10-14 evening, A33):** Sync respects the AppProject's sync windows: inside a
deny window it is refused, or greyed with the reason; Argo CD's RBAC is not emulated. Built
2026-10-14 ([`kinds/argocd_syncwindow.go`](../../internal/model/kinds/argocd_syncwindow.go),
[`commands/argosyncwindow.go`](../../internal/engine/commands/argosyncwindow.go);
`TestArgoSyncBlocked`, `TestArgoSyncRespectsSyncWindows`). **The lead's night decisions for the
owner to confirm:** Cogpit's sync is a manual sync and follows Argo CD's check of one (an open deny
window blocks unless every open deny window allows manual syncs; an open allow window lets it
through; allow windows that match but none open block unless every one of them allows manual
syncs; matching by name, destination namespace and cluster by Argo CD's glob, with `andOperator`;
schedules of five fields in the window's time zone); the project is read through the cache in
Argo CD's own namespace, which the controller writes into the Application's
`status.controllerNamespace` (the Application's own namespace before it has; Enabled greys only
what the cache holds already, the write checks after reading); a project Cogpit may not read
blocks nothing (the controller honours its windows anyway); a window Cogpit cannot read blocks,
with the reason; in a selection a blocked Application is skipped and reported.
**Checked against a live Argo CD controller on a real cluster (2026-10-16, Argo CD 3.5.5 from its
public Helm chart, Applications from a Helm repository, no Git):** Cogpit's operation is taken and
run by the controller (Sync and Sync with prune on a selection to Synced and Healthy, the history
naming `cogpit`; Refresh normal, hard and on a selection reconciled, the annotation removed), and
an operation written inside a deny window waits in the controller as "Sync operation blocked by sync
window" (phase Running), so Cogpit's check before the write is what keeps a user from a sync that
never runs. Found and fixed the same night (the lead's decisions for the owner to confirm), against
Argo CD 3.5.5's source (`SyncWindows.CanSync`, `Server.Sync`, the controller's
`newSyncOperationResult`, read 2026-10-16): (1) the manual-sync exceptions need every open deny
window, and every closed allow window, to allow manual syncs, where Cogpit took any one; (2) a
pattern's `*` matches `/` as in Argo CD's glob (a cluster pattern `*` or `https://*` against a
server URL never matched with Go's `path.Match`), with `?`, classes, `{a,b}` and escapes
(`kinds.ArgoGlobMatch`); (3) a schedule with a descriptor (`@daily`) is a window Cogpit cannot read,
as Argo CD's parser takes none; (4) the end a deny window's reason names follows the openings that
start before it closes, and a window that keeps reopening (every minute for an hour) says its
schedule keeps it open instead of naming a time a minute away; (5) an Application outside Argo CD's
namespace ("Applications in any namespace") found no project in its own namespace, so its deny
window greyed nothing (live: Sync was offered, the controller would have held it) — the project is
now looked up in `status.controllerNamespace`, and the drawer's and the row's project link point
there; (6) the operation carries the source the user confirmed, a copy of the spec's (`source`, or
`sources` for several), as `argocd app sync` writes it since Argo CD 3: the controller syncs that
source, so a change of the spec between the confirmation and the controller's run no longer
changes what the sync applies; a Helm value set to null stays null in the copy. Not carried over,
for the owner to know: the API server also clears the Application's last `operationState` when it
writes the operation, and resolves a Git branch to its commit before it does; Cogpit's patch keeps
the last state until the controller replaces it (a fraction of a second) and lets the controller
resolve the revision, as it does for an automated sync.
Tests: `TestArgoSyncOperation` (model), `TestArgoSyncPatch`, `TestArgoSyncConflicts` (cluster),
`TestArgoSyncWritesTheOperation`, `TestArgoSyncRefusedWhileAnOperationRuns`,
`TestArgoSyncChecksAgainAfterAChange`, `TestArgoActionsBoundToTheConfirmedUID`,
`TestArgoSyncWindowsOfAnAppInAnotherNamespace` (commands); `TestArgoSyncBlocked`,
`TestArgoSyncBlockedAsArgoCD35`, `TestArgoWindowOpenUntil`, `TestArgoGlobMatch`,
`TestArgoV3ResourceHealthAndProject` (model); on kind `TestKindArgoCD`; against a live Argo CD
`TestRealArgoCD` (it runs only with `COGPIT_REAL_KUBECONFIG` and `COGPIT_REAL_ARGOCD_NAMESPACE`,
never in CI; [testing](testing.md)).

## D-39: Pod metrics for a user who may list pods in some namespaces only

**Decided (owner, 2026-10-14): option (b), and a pod-metrics refusal affects only its namespace**
(below). The night build's record (the lead's night decision of 2026-10-13; the owner's open
question, built the same night): options (a) as before: the Pods list, a node's Pods table and
`metrics.get` ask the Metrics API for every pod in all namespaces, which RBAC refuses such a user as
it refuses the pods, so CPU and Memory stay empty and the state reads "forbidden"; (b) one `list` of
`pods.metrics.k8s.io` per namespace the Pods view reads pods in (the accessible namespaces, the
namespace filter's, else the context's own; those whose rules deny pods left out), one per poll,
shared by the Pods list, a node's Pods table and the drawers, only while they show, and the one list
in all namespaces for a user who may list pods there; (c) a `get` of each visible pod's metrics per
poll; chosen (b), because it costs one request per namespace and poll where (c) costs one per pod
(a list of 300 pods: 300 requests a minute instead of one per namespace), it reads what the Pods
view reads, so the columns fill for every row the user sees, and it changes nothing for a user who
may list pods everywhere (budget P12: one list per 60 s).

**Built 2026-10-13** ([`engine/metrics.go`](../../internal/engine/metrics.go) `usageListScopes`,
[`engine/drawerpods.go`](../../internal/engine/drawerpods.go); [architecture §9](architecture.md#9-metrics-internalmetrics)):
the Pods list subscribes the metrics hub to `{pods, <namespace>}` for each namespace its plan reads
(`nsPlan.watch`), to `{pods}` when it reads all namespaces, and to nothing while the plan is
pending; the feed starts once the view's kind has listed, so a read refused in all namespaces (a
RoleBinding in the context's namespace alone, before the plan follows the refusal) polls no metrics
there either. A node's Pods table polls the same scopes (the pods' plan), a namespaced object's its
namespace, as before; `metrics.get` of every pod fetches the same scopes, at most four at once, and
merges them. The hub needed no change: its scopes by namespace existed, and `Hub.Pod` reads a pod
from its namespace's list. Measured at the HTTP level (`TestBudgetNamespaceLimitedUser`, a fake API
server that enforces RBAC): a user whose one RoleBinding is in `team-a` lists `pods.metrics.k8s.io in
team-a` once and nothing in all namespaces; a cluster-admin lists `pods.metrics.k8s.io` once, as
before (`TestPodMetricsPerNamespaceOfThePlan`, `TestPodMetricsClusterWideForAUserWhoMayList`).

**Decided (owner, 2026-10-14): a pod-metrics refusal affects only its namespace.** The Metrics
API's RBAC is not the pods': a namespace where the user may list pods but not their metrics
answers 403 for that namespace's `list pods.metrics.k8s.io`. Until 2026-10-14 the hub counted that
as the Metrics Server failing for a minute (`ProviderRetry`) for every scope, asking Prometheus in
its place where it found one (G12's fallback). Now (built 2026-10-14,
[`metrics/hub.go`](../../internal/metrics/hub.go) `namespaceRefusal`): the refused namespace's
pods get empty cells and the state `forbidden` for that namespace (`Hub.State` of its scope, its
message naming the namespace and the API server's reason); the other namespaces keep the Metrics
Server's values; the Metrics Server's status stays as it is (no failure on the status line, no
`ProviderRetry`), and nothing stands in for it in the refused namespace. Measured at the HTTP level
(`TestBudgetNamespaceLimitedUser`, a fake API server that enforces RBAC: pods in team-a and team-b,
their metrics in team-a only): `list pods.metrics.k8s.io in team-a` once, `in team-b` once (the
refusal), no read of `services` (nothing looks for a Prometheus); before the change the same steps
sent three refused lists in team-b and the status line read the Metrics Server as forbidden; a
cluster-admin's counts are unchanged (one `list pods.metrics.k8s.io` in all namespaces). Tests:
`TestPodMetricsRefusalStaysInItsNamespace`, `TestPodMetricsRefusalWithoutPrometheus`, `TestBudgetNamespaceLimitedUser`.

Cogpit's defaults with it, not the owner's decision:

- **A drawer too.** A pod's drawer in a refused namespace (`get pods.metrics.k8s.io` refused) is
  the same refusal: no live values, the provider untouched, no Prometheus live values in their
  place; Prometheus's history of the pod, which the drawer asks for whatever the Metrics Server
  says, is still shown where there is a Prometheus (it is not limited by namespace, below).
- **A refusal is an answer.** It is kept for the TTL like values (15 s, longer while polled), so an
  agent's `metrics.get` right after the view's poll sends nothing more; the view polls the refused
  namespace at the list interval, as any other, so a RoleBinding added later shows within a poll.
- **Agents see which.** `metrics.get` of every pod, read per namespace, names the refused
  namespaces in `forbidden`; its status is the others' (`available`).
- **What stays provider-wide:** a refusal of every pod (the one list in all namespaces), of the
  nodes, a 401, and a Prometheus's refusal (its service proxy), as before.

What it leaves as it was, for the owner to know:

- **Prometheus is not limited by namespace.** Its queries go through the API server's service
  proxy; RBAC decides whether the user may use that proxy at all, not which namespaces' series come
  back. Cogpit asks Prometheus, while it stands in for the Metrics Server, for the same namespaces
  (the namespace is part of the query), but nothing on the server narrows it
  ([privilege footprint](../security/privilege-footprint.md)).
- **Confirmed (owner, 2026-10-14 evening, A47):** the cluster Overview's Pods history counts the
  whole cluster for a namespace-limited user too (no namespace matcher,
  [`promclusterpods.go`](../../internal/metrics/promclusterpods.go)), as built.
- **Confirmed (owner, 2026-10-14 evening, A48):** a pod's drawer in a refused namespace still
  shows Prometheus's history ("A drawer too" above), as built.

## D-37: The node shell

**Decided (owner, 2026-10-14):** option (b) of the night build's decision, which the owner
confirmed in a decision round: options (a) no node shell
until the owner decides ([H-7](../security/privilege-footprint.md#h-7-node-shell-is-root-on-the-node));
(b) Lens's behaviour: a privileged pod in `kube-system` on the node that shares the host's PID
namespace, entering PID 1's namespaces with `nsenter -t 1 -m -u -i -n -p`, under the rules of
[privilege footprint › Node shell](../security/privilege-footprint.md#node-shell); (c) only a
`kubectl debug node/…`-style pod (the host's namespaces and its root filesystem at `/host`, no
privileged flag, no nsenter); (b), because it is Lens parity (a free Lens feature), every
start is a destructive-level confirmation for every origin, the GUI's included (agents always need
the user's acceptance, D-2), and the pod's life is bounded; (c) stays a later option, as a second
profile, not the default. With it the owner decided two of the build's defaults: the default image
is the newest supported Alpine release, pinned by digest and kept current by Renovate (no longer
Lens's 3.19), and the deadline is a setting with a 2-hour default (both below, built 2026-10-14).
**Built 2026-10-13** (night build;
[`engine/nodeshell.go`](../../internal/engine/nodeshell.go),
[`commands/nodeshell.go`](../../internal/engine/commands/nodeshell.go),
[`cluster/nodeshell.go`](../../internal/cluster/nodeshell.go),
[`terminal/nodeshell.go`](../../internal/terminal/nodeshell.go)); on kind 1.36.1 the shell enters
the node and `hostname` answers its name, and the pod is gone after the session closes
(`TestKindNodeShell`).

What the owner decided on 2026-10-14, built the same day:

- **The deadline: a setting, default 2 h (owner, 2026-10-14).** Of the build's options (a) 1 hour,
  (b) 2 hours, (c) 8 hours, (d) a setting, the owner chose (d) with (b) as its default: Cluster
  Settings › Node Shell › Maximum duration, kept per context in `cluster-settings.json`
  (`nodeShellMaxMinutes`, empty: 2 hours). It is the pod's `activeDeadlineSeconds` and the
  container's `sleep`, so the pod ends by then even if Cogpit dies; the confirmation names it ("ends
  by itself after 1 hour 30 minutes, its maximum duration"), and so do the tab's notice and the
  toast of a failed delete. Only the user sets it, as the image ([D-38](#d-38-who-sets-the-node-shells-image)):
  agents and the CLI may only reset it (`ErrNodeShellGUIOnly`)
  ([`engine/nodeshell_duration.go`](../../internal/engine/nodeshell_duration.go),
  [`fyneui/clustersettings_nodeshell.go`](../../internal/ui/fyneui/clustersettings_nodeshell.go);
  `TestNodeShellMaxDuration`, `TestNodeShellMaxDurationSetting`,
  `TestClusterSettingsNodeShellMaxDuration`; on kind `TestKindNodeShell` sets 30 minutes and finds
  `activeDeadlineSeconds: 1800` and `sleep 1800` on the pod).
- **The default image: the newest supported Alpine release, pinned by the digest of its multi-arch
  index, kept current by Renovate (owner, 2026-10-14).** Alpine's release table
  ([alpinelinux.org/releases](https://alpinelinux.org/releases/), read 2026-10-14) lists v3.24
  (branched 2026-06-09, minor releases 3.24.0 to 3.24.2, supported until 2028-06-01) as the newest
  stable branch, then v3.23 (until 2027-11-01), v3.22 (until 2027-05-01) and v3.21 (until
  2026-11-01); 3.19 is past its end. Chosen: `docker.io/library/alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6`,
  the digest read on 2026-10-14 at 07:45 UTC with `docker buildx imagetools inspect
  docker.io/library/alpine:3.24` (the tags `3.24.2` and `latest` name the same index; its eight
  images, linux/386, amd64, arm/v6, arm/v7, arm64/v8, ppc64le, riscv64 and s390x, are 3.24.2, built
  2026-09-17). `nsenter` and `sleep` are BusyBox 1.37.0 applets in it (`/usr/bin/nsenter`,
  `/bin/sleep`), checked with `docker run` on linux/arm64 (the index) and linux/amd64 (its image):
  `nsenter --help` lists `-t`, `-m`, `-u`, `-i`, `-n`, `-p`, and `sleep 1` exits 0; on kind 1.36.1
  (arm64) `TestKindNodeShell` enters the node with it. **Renovate keeps it current:** the constant
  `DefaultNodeShellImage` in [`engine/nodeshell.go`](../../internal/engine/nodeshell.go) carries a
  `// renovate: datasource=docker depName=docker.io/library/alpine` comment that a custom regex
  manager in [`.github/renovate.json`](../../.github/renovate.json) reads (tag and digest), so a
  new Alpine release (3.25, a minor update) moves the tag and a rebuilt 3.24 moves the digest; a
  dry run of Renovate 44.155.8 (`--platform=local --dry-run=lookup`) extracted 3.24 with its digest
  and found nothing newer, and on a copy pinned to 3.23 it proposed 3.24 with the digest above and
  3.23's newest digest. The tests take the image from the constant, so Renovate's pull request
  passes them, and CI's kind job opens a node shell with the new image before it merges. The texts
  name the image from the constant (`alpine:3.24` on the settings page and in the command's
  description); this page names the pin of 2026-10-14, the constant is the current one. Lens's
  default stays the tag `docker.io/alpine:3.19`: Cogpit's default differs from it, the setting is
  Lens's. H-72 changed with it ([privilege footprint](../security/privilege-footprint.md#h-72-the-default-node-shell-image-ages-with-the-build-that-pins-it)).

What the build chose, Cogpit's defaults (not the owner's decision):

- **The maximum duration's bounds: 5 minutes to 24 hours**, in whole minutes, typed as `90m`, `2h`
  or `1h30m` (Go's duration form; the field shows a set value back in that form, `1h30m` for
  `90m`, and a line under it says it in words). Five minutes leave time to pull the image and look
  at something; a day covers a long working session without leaving an idle root pod for days.
  A value outside them, `2 hours`, `90` or `10m30s` is refused under the field; a file holding one
  reads as the default. **Confirmed (owner, 2026-10-14 evening, A35):** the bounds and the 2-hour
  default stay, as built.
- **Renovate's updates of the image merge on their own** once CI passes (a minor release or a new
  digest; a major, Alpine 4, waits for review), as the other pins of the custom regex manager do,
  and they are `build(deps):` commits, which semantic-release does not release: a new default image
  reaches users with the next `feat:` or `fix:` release.
- **A floating tag is accepted** in the setting (options: (a) refuse an image without a digest,
  (b) accept it and say so; chosen (b), as Lens takes any image): the confirmation then says "a
  floating tag, not pinned by digest: what it runs can change without Cogpit knowing", and the
  settings page says so under the field.
- **The pod:** `kube-system`, `nodeName` set, one toleration `operator: Exists`, `hostPID: true`,
  the container `privileged: true`, no service account token, no service links,
  `restartPolicy: Never`, a 1-second grace period, labels `app.kubernetes.io/name:
  cogpit-node-shell` and `app.kubernetes.io/managed-by: cogpit`, a name
  `cogpit-node-shell-<8 random characters>`; an image pull secret named in the setting. The shell
  runs `nsenter -t 1 -m -u -i -n -p -- env TERM=xterm-256color HOME=/root sh -c` with the node's
  bash as a login shell, else its sh.
- **Its end:** the pod is deleted (with its UID as precondition) when the shell ends, when its tab
  closes (`dock.close`, by anyone, or since 2026-10-14 with its context when that leaves Cogpit:
  its kubeconfig source removed, its file gone) and when Cogpit quits (waiting at most 10 seconds); a failed
  delete is a warning toast, and the pod then ends by its deadline. **Since 2026-10-14 (owner,
  [H-71](../security/privilege-footprint.md#h-71-a-crash-leaves-the-node-shells-pod-until-the-next-connect)):**
  every pod is recorded in `nodeshells.json` before its create is sent (context, name, a random id
  the pod carries as `cogpit.io/node-shell-id`, then its UID), and on the next connect to its
  context Cogpit deletes the pods a crash or a failed delete left, by UID, and says so in a
  notification; never a pod it did not create. Cogpit's defaults in that build (not the owner's
  decision): the entry is written before the create, so a crash between the two leaves an entry
  without a UID, whose pod is read by name and recognised by the id (options: (a) before the
  create with the id, (b) after the create only, which a crash between the two would leave
  unrecorded; chosen (a)); a journal that cannot be written refuses the node shell; the cleanup
  follows every connect, asks nothing and is audited as `cogpit.nodeShellCleanup` with origin
  gui; another live Cogpit process's entries are left to it; without a configuration directory
  the journal lives in memory. The start waits for the pod
  with one get and a watch on that pod alone, at most 3 minutes; a pod that cannot pull its image
  or fails ends the session with the reason and is deleted.
- **What is confirmed binds what runs:** the confirmation's detail names the namespace, the node,
  the image, the pull secret and the maximum duration; a change of the image, the pull secret or
  (since 2026-10-14) the maximum duration while the dialog was open, a node created again under its
  name, or another context refuses the start before anything is created.
- **A digest wider than the dialog breaks** (2026-10-14): the confirmation's text breaks a word
  wider than the dialog at the widest part that fits and flows on after it (`wrapFlow`), since the
  3.24 digest no longer fit on one line of the dialog and Renovate changes it.
- **The access check asks in `kube-system`:** a command's declared verb may name its one
  namespace (`Verb.Namespace`, new), so a user who may create pods in `kube-system` only is not
  greyed by a cluster-wide question.

## D-38: Who sets the node shell's image

**Decided (owner, 2026-10-14):** option (b) of the night build's decision: options (a) agents set
the node shell's image and pull secret too, as the other cluster settings (UI commands, D-2); (b)
only the user sets them, in Cogpit's window, and agents and the CLI may only reset them to the
defaults, as a cluster's proxy (`ErrProxyGUIOnly`; the proxy's rule changes with the round of
2026-10-14 evening, an agent's proxy becoming a confirmed write, [D-2](#d-2-agent-interface-scope);
the image's rule stays); (c) no setting, the pinned default only; (b),
because the image is what runs as root on the node at the user's next node shell, and an agent that
could choose it and then ask for a node shell would leave the user one careful reading of the
confirmation away from running the agent's image privileged; Lens has the setting, so (c) would
lose parity. The owner put the node shell's maximum duration under the same rule (2026-10-14,
[D-37](#d-37-the-node-shell)): it decides how long a root pod lives. **Built 2026-10-13**, the
maximum duration 2026-10-14 (`cluster.settings.setNodeShell` refuses a value that is not the
default from every origin but the GUI with `ErrNodeShellGUIOnly`, before anything is written;
`TestNodeShellSettings`, `TestNodeShellMaxDurationSetting`).

## D-40: A Pods table in the workloads' drawers

**Decided (owner, 2026-10-14): option (b), without a Namespace column in the workloads' drawers;
the node drawer keeps it.** A workload's pods share its namespace, so the column said nothing
there. Built 2026-10-14 ([`kinds/nodes.go`](../../internal/model/kinds/nodes.go)
`workloadPodColumns`; `TestWorkloadDrawerPods`, `TestDeploymentDrawerPodsTable`; PNGs re-rendered:
[Deployment dark](../../internal/ui/fyneui/testdata/deployment_pods_drawer_dark.png),
[light](../../internal/ui/fyneui/testdata/deployment_pods_drawer_light.png),
[Stateful Set dark](../../internal/ui/fyneui/testdata/statefulset_pods_drawer_dark.png),
[light](../../internal/ui/fyneui/testdata/statefulset_pods_drawer_light.png); the node drawer's
[PNG](../../internal/ui/fyneui/testdata/node_pods_drawer_dark.png) unchanged). The night build's
record follows.

**Night decision (the lead's, 2026-10-14, now decided by the owner, above):** options (a) no Pods table in
the workloads' drawers, as before; (b) the node drawer's Pods table (Name, warning icon, Node,
Namespace, Ready, CPU, Memory, Status; sortable; CPU and Memory from the metrics hub, polled only
while the table shows) in the drawers of Deployments, Replica Sets, Stateful Sets, Daemon Sets and
Jobs; (c) a "Pods" link that opens the Pods view filtered to the workload; chosen (b), because it is
what a Lens user expects from a workload's details (inferred: Lens's workload drawers are not
captured), and the node drawer's table, its metrics and its sorting already exist, so the cost is
the index of pods by their owner, which only an open drawer holds. **Built 2026-10-13** (night
build; [`kinds/workloadpods.go`](../../internal/model/kinds/workloadpods.go),
[`engine/drawerlimit.go`](../../internal/engine/drawerlimit.go), `KindProfile.DetailNeeds`;
`TestWorkloadDrawerPodsPerKind`, `TestWorkloadDrawerPodsFollowPods`,
`TestWorkloadPodsIndexOnlyWithTheDrawer`, `TestWorkloadDrawerPodsMetrics`,
`TestWorkloadDrawerPodsSort`, `TestDeploymentDrawerPodsTable`; on kind 1.36.1
`TestKindWorkloadDrawerPods`: a Deployment of two replicas lists its two pods over the engine, and a
deleted pod's replacement).

What the build chose, Cogpit's defaults (not the owner's decision):

- **Cron Jobs too, through their jobs.** Options: (a) the five kinds the decision names, (b) Cron
  Jobs as well, the pods of the jobs they control; chosen (b): `object_get`'s `related: true` lists
  a Cron Job's jobs and their pods already, the cost is one more index (jobs by owner) that only the
  drawer holds, and the pods of the last runs (the jobs' history limits keep three succeeded and one
  failed by default) are what a user checks after a run. Replication Controllers are left out, as
  `related: true` leaves them out (confirmed by the owner, 2026-10-14 evening, A109).
- **The node drawer's eight columns, Node included** (superseded 2026-10-14: the owner removed
  Namespace, so a workload's table has seven, Node kept, option (c)). Options: (a) the seven of the
  lead's brief, without Node; (b) the node drawer's eight; (c) seven without Namespace, which a
  workload's pods share; the night build chose (b): Lens's node drawer shows a Node column that is redundant there (capture,
  2026-10-11), which suggests one shared table whatever the drawer (inferred); Node is what tells a
  Daemon Set's pods apart; and one builder, one set of column ids and one sort state ("Pods") serve
  both drawers. The Node cell links to the node (a node's own drawer keeps it plain). The cost: in
  the default drawer width a workload's pod names, which share their prefix, were cut with "…";
  without Namespace (2026-10-14) the Deployment's fixture names fit
  ([PNG](../../internal/ui/fyneui/testdata/deployment_pods_drawer_dark.png)). The column ids, the
  sort state ("Pods") and the builder stay shared; a sort by Namespace set for a node's table
  sorts a workload's by name, which has no such column (`SortDrawerTables` falls back to the
  default order).
- **A Deployment's pods are those of every replica set it controls**, old ones during a rollout
  included. Options: (a) the newest replica set's only, (b) every one it controls; chosen (b): the
  old pods still serve and terminate during a rollout, which is what a user watches; old replica
  sets kept at zero replicas have no pods and add no row.
- **Owned means the controller reference's UID, in the workload's namespace.** Options: (a) the
  owner's kind and name, (b) the controller reference's UID, counted only in the object's own
  namespace; chosen (b): a workload created again under its name owns none of the old one's pods,
  and Kubernetes's garbage collector treats an owner reference across namespaces as absent; the
  workload logs and `related: true` use the same rule (`controllerUID`).
- **The index lives with the drawer** (`DetailNeeds`). Options: (a) in the profiles' `Needs`, which
  makes every workload list rebuild its rows on every pod change (measured: eleven Deployments
  snapshots for ten pod changes and one change of a Deployment, against one without); (b) the
  drawer's own indexes, held while it is open; (c) no index, every drawer build scanning the pods;
  chosen (b), which also wakes the drawer only for its own pods and replica sets
  (`TestWorkloadPodsIndexOnlyWithTheDrawer`). Its cost: an open workload drawer holds a watch on the
  pods (and the replica sets or jobs) as `Backend.Acquire` reads them, all namespaces or the
  namespaces the access reviews allow, shared with every other reader and warm for the grace period
  after the drawer closes; opened alone it is one list and watch more.
- **At most 500 rows, the rest counted** (`engine.MaxDrawerTableRows`), for every sortable drawer
  table, a node's Pods included. Options: (a) no bound, as the node drawer had; (b) 200, the bound
  of `related: true`; (c) 500; chosen (c): a node runs 110 pods by default and at most a few
  hundred, so only a large workload's table is cut; the cut follows the sort (sorted by CPU, the
  busiest pods), and the window says "N more pods not shown" under the rows.
- **The place:** after the kind's own sections (Conditions included), before the Events; a workload
  without pods, or whose pods or owners are still listing, has no Pods section (inferred).
- **`related: true` reads the table like every section**: the pods first, in the table's order,
  each pod's node with the relation "Pods" as a node drawer's namespaces have it, then the replica
  sets or jobs and any pods the table cut. Options: (a) so; (b) leave the workload's own Pods out of
  the section scan, keeping the replica sets before the pods; chosen (a): the documented rule
  ("every object the drawer links to, in its order, then what a workload controls") holds without
  an exception.

## D-41: What survives a restart

**Decided (owner, 2026-10-14):** as built: the tabs with their per-tab state and the tables' column choice and order are kept and restored without connecting anything; column widths are not kept.

**Changed by the owner's decision of 2026-10-16 ([D-69](#d-69-several-clusters-connected-at-once)
item 8), built 2026-10-16:** the tabs come back only while Preferences › App › "Restore the last
session" is on, and then the clusters connected at quit connect again, the one shown then first
([`restore.go`](../../internal/engine/restore.go)); **off, the owner's default, Cogpit starts
without tabs and without connections.** The tables' column layout, the navigator's collapsed
groups, the navigator hidden and the chat column come back either way (the lead's decision for the
owner to confirm, D-69 stage 3: they are the window's arrangement, not the session).
`workspace.json` keeps `connected` and `shown` beside the tabs whatever the preference says, so
that turning it on restores the session of the last quit; its format version stays 1, a file of
v0.17.0 loads (no connections), and v0.17.0 ignores the two fields. The sections below describe
the restore of the tabs as it runs with the preference on.

**Context.** Until 2026-10-13 everything the user arranged in the window but the hotbar, the
preferences, the cluster settings and the panel widths ended with the run: the tab bar's tabs and the
tables' hidden columns and column order ([ui-parity › Layout](../parity/ui-parity.md#layout),
[› Tables](../parity/ui-parity.md#tables), G5, G13). Whether Lens restores its tabs is not captured
(inferred: it does). **Built 2026-10-13** ([`engine/persist.go`](../../internal/engine/persist.go),
[`engine/persist_restore.go`](../../internal/engine/persist_restore.go); `TestWorkspaceRoundTrip`,
`TestWorkspaceNoRequestAtStart`, `TestWorkspaceDropsOnlyWhatIsCertainlyGone`,
`TestWorkspaceUnreadableFileIsKeptAside`, `TestWorkspaceWritesAfterTheChangesStop`).

- **Decided (owner, 2026-10-14, as built; the night decision of 2026-10-13): whether and how the tabs come
  back.** Options (a) not kept, as before; (b) kept in Cogpit's own owner-only file and restored at
  start without connecting anything, a tab connecting its cluster when the user shows it; (c) as (b),
  and the cluster of the tab shown last connected at start, as a browser reloads its tabs; chosen (b),
  because (c) would send requests to a cluster, possibly a production one, that the user has not
  asked for in this run (Never surprise a cluster), and (a) loses what the user arranged. The file is
  `<user config dir>/Cogpit/workspace.json` (0600 in a 0700 directory, replaced atomically,
  `writePrivateFile`), written 1 second after the tabs or the layout stop changing (at least every
  5 seconds while they keep changing) and when Cogpit quits; read once at start.
- **Decided (owner, 2026-10-14, as built; the night decision of 2026-10-13): what each tab keeps.** Options
  (a) its cluster and view only; (b) its cluster, view and the view's title, preview or fixed, its
  place in the bar, when it was last shown, its namespace and API group filters, its search text with
  the match case and regular expression toggles, its status filter (since 2026-10-14, D-34), its
  sort and its history; (c) (b) plus its selection
  and the drawer's object; chosen (b), because it is the tab's own state that the bar keeps per tab
  today, while a selection and a drawer name objects that may be gone at the next start, and a
  drawer restored at connect would read an object the user did not ask for in this run.
- **Decided (owner, 2026-10-14, as built; the night decision of 2026-10-13): the tab shown last.** Options
  (a) no tab marked at start; (b) the tab shown last marked and its cluster named as
  `UIState.Context`, disconnected, the state `cluster.disconnect` leaves (the welcome page shows, the
  tab is marked, showing it connects); (c) marked and connected; chosen (b), because it shows where
  the user left without a request ((c) is (c) above), and it is a state the engine and the window
  handle already. A cluster connected from the navigator, the hotbar or the welcome page shows its most
  recently used tab, as within a run.
- **Decided (owner, 2026-10-14, as built; the night decision of 2026-10-13): tabs of contexts that are
  gone.** Options (a) restored, failing when shown; (b) dropped with a notice; chosen (b), and only on
  a plain fact, as a connection closes only on one (`CloseGoneSourceConnection`): the context's file
  does not exist any more, its source was removed, its file parses and no longer defines it, or the
  connector's list, read without an error, does not have it (the demo cluster, a run without a
  configuration directory). A file that does not parse for a moment keeps its tabs. The notice ("2
  tabs of the last run were not restored: no kubeconfig source offers dev [dev.yaml] any more.") is
  in the notification log (the title bar's bell) and shown as a toast once the window is up
  ([D-42](#d-42-the-start-screen)).
- **Decided (owner, 2026-10-14, as built; the night decision of 2026-10-13): the tables' layout.** Options
  (a) the hidden columns and the column order per view key, the same on every cluster, as
  `UIState` holds them; (b) per cluster and view; chosen (a), because a view's columns are its kind's,
  the same on every cluster, and a user who hides a column on one cluster's Pods wants it hidden on
  the next. **Column widths:** options (a) not kept (the window's, not engine state, as now); (b)
  kept in Fyne's preferences as the panels' widths; (c) engine state in this file; chosen (a) for
  now, because the table records a width whenever a header's laid-out width differs from the one it
  set, which does not yet tell a user's drag from a relayout, and keeping a relayout's width across
  a restart would make it stick; (b) once the drag's end is its own event. A question for the owner.
- **Not restored:** the dock's tabs (logs, terminals, shells, editors, Helm tabs, the Audit Log):
  a stream or a terminal would open at start, and an editor's unsaved text has its own draft,
  offered when its cluster connects ([D-11](#d-11-crash-stance)); a selection and the drawer (above);
  the connection itself, until 2026-10-16: with Restore the last session on, the clusters connected
  at quit connect again at start (D-69 item 8, above); the dock's tabs stay out (D-69 stage 3).
- **Decided (owner, 2026-10-14 evening, A63), built 2026-10-15:** the navigator's Local Kubeconfigs
  section has source groups that collapse (a click, the arrow keys), and which are collapsed is
  kept across restarts: `workspace.json`'s `collapsedSources`, the groups by their headings' labels
  (at most 1,000 of 512 bytes, a repeat dropped on load), restored into `UIState.Navigator.Collapsed`
  ([D-51](#d-51-the-navigators-and-the-command-palettes-state);
  `TestWorkspaceKeepsTheCollapsedSourceGroups`).
- **Decided (owner, 2026-10-15, F7), built 2026-10-15:** the layout survives a restart:
  `workspace.json`'s `navigatorHidden` and `chat`, whether the navigator is hidden and the chat
  column shown, restored into `UIState.NavigatorHidden` and `UIState.Chat`. Not the dock's state:
  its tabs are not restored (above), and a dock without tabs is closed (the lead's decision for the
  owner to confirm, [D-54](#d-54-the-layout-toggles-and-the-update-button);
  `TestWorkspaceKeepsTheLayout`).

Cogpit's defaults, not the owner's decisions: the delays; the bounds of a file read back (200 tabs,
1,000 namespaces, groups or column ids each, 1,000 views, keys of 512 bytes, a search of 1,000 bytes,
titles of 256 bytes; what is beyond them is dropped without costing the rest); a file that does not
load (not JSON, a newer format) is ignored with a warning notice and moved to
`workspace.json.unreadable` before the first write, never overwritten; two Cogpit processes keep a
copy each and the last write wins, as with the hotbar.

## D-42: The start screen

**Decided (owner, 2026-10-14):** option (b), the welcome page at start without the context picker, as built.

**Changed 2026-10-16 ([D-69](#d-69-several-clusters-connected-at-once) item 8, the owner's
decision):** the start shows the last run's tabs over the welcome page only while Preferences › App
› Restore the last session is on; then the cluster shown at quit connects at once and the window
shows it as on any connect (its most recently used tab). Off (the default) the start is the welcome
page without tabs. `--context` still connects its cluster and shows it; with the restore on, the
other clusters of the last session connect behind it.

**Context.** Until 2026-10-13 the context picker opened by itself whenever Cogpit started without a
connection, over the welcome page and its cluster rows ([ui-parity › Layout, Welcome
page](../parity/ui-parity.md#layout): "Differs: the context picker still opens by itself at start,
over the page"). Lens starts on its own page, not on a picker (inferred; its welcome page is not
captured). Since D-41 the start can also show the tabs of the last run. **Built 2026-10-13**
([`fyneui/shell.go`](../../internal/ui/fyneui/shell.go) `start`;
`TestStartShowsRestoredTabsWithoutConnecting`, PNGs
[dark](../../internal/ui/fyneui/testdata/start_restored_dark.png),
[light](../../internal/ui/fyneui/testdata/start_restored_light.png)).

- **Decided (owner, 2026-10-14, as built; the night decision of 2026-10-13):** options (a) as before: the
  picker opens by itself at start whenever no cluster is connected; (b) no picker at start: the
  welcome page shows with its cluster rows (the hotbar's bank, the clusters of the tabs, the others
  the kubeconfigs offer), the picker opens with ⌘P, All Clusters… or the palette's Connect to
  Cluster; (c) the picker only when no tab was restored; chosen (b), because the welcome page
  already offers every cluster to connect with one click and the picker over it hid exactly that,
  because it is what the welcome row names as the difference from Lens, and because (c) would make
  the first screen depend on a file the user does not see. `--context` and `--demo` (a development
  build's only since [D-67](#d-67-the-demo-cluster-stays-out-of-releases)) connect at start as before; a `--context` that fails opened the picker with the error beside the context (Cogpit's
  default); since [D-57](#d-57-choosing-a-cluster) a refused one says why in a toast and shows the
  navigator's section with its search focused, where a login program not allowed yet can be allowed.
- With it (Cogpit's default): a notice the engine gave before the window was up (the tabs of the
  last run not restored, an unreadable `workspace.json`) is shown as a toast once the window is up,
  with the crash notice, rather than only in the notification log.

## D-43: Where the nodes' disk comes from

**Decided (owner, 2026-10-14):** option (b), node-exporter's root filesystem through Prometheus, as built; without node-exporter the Disk column stays empty (no cAdvisor fallback, never `nodes/proxy`).

**The night decision (the lead's, 2026-10-14), confirmed by the owner (2026-10-14):** Lens's Nodes list shows a
Disk usage bar (capture, 2026-10-11); the Metrics Server measures no disk. Options: (a) the
kubelet's summary API (`/api/v1/nodes/<node>/proxy/stats/summary`), which every cluster has; (b)
node-exporter's root filesystem (`node_filesystem_size_bytes` and `node_filesystem_avail_bytes` of
the mountpoint `/`) through the cluster's Prometheus, matched to nodes by their node label or
instance; (c) cAdvisor's filesystem series of the node's root cgroup through Prometheus (what the
node drawer's Filesystem chart reads), which needs no node-exporter; chosen (b), because (a) needs
`get nodes/proxy`, a permission that reaches the kubelet's whole API, running commands in the
node's pods included ([privilege footprint](../security/privilege-footprint.md), Metrics), which
Cogpit must not ask a user to grant for a column; and (c) sums every block device of the node, not
the root filesystem a "Disk" bar means, and kube-prometheus-stack drops those series (G12). The
price of (b): a cluster without Prometheus and node-exporter shows an empty column, whose header's
tooltip says why. **Built 2026-10-14** ([`metrics/disk.go`](../../internal/metrics/disk.go),
[`engine/drawerdisk.go`](../../internal/engine/drawerdisk.go); `TestNodeDisksFromNodeExporter`,
`TestNodeDiskBudget`, `TestNodesListDiskColumn`, `TestNodesListDiskBudget`; architecture §9).

What the build chose, Cogpit's defaults (not the owner's decision):

- **One query per poll, the list's 60 s,** for every node at once: both gauges in one answer,
  marked by a `part` label (`label_replace(…) or label_replace(…)`), `max by (node, instance)`
  so that two jobs scraping one exporter count once. Options: (a) two queries (size, available),
  (b) one; chosen (b), half the requests for the same numbers.
- **Used is the size less the available bytes** (`df`'s Avail: what an unprivileged process may
  still write), so the share counts the blocks reserved for root as used. Options: (a) size less
  available, (b) size less free (`node_filesystem_free_bytes`); chosen (a), the share at which
  workloads stop being able to write.
- **A node is matched by its node label, else its instance's host as the node's name, else as one
  of the node's addresses** (InternalIP, ExternalIP, Hostname). The prometheus chart's service
  discovery adds the node label; an exporter scraped with host networking under its own address is
  found by the node's InternalIP. Checked against the `prometheus` chart's node-exporter on kind
  since 2026-10-14 night (A115): the node label and the InternalIP as the instance, both read
  (`TestKindNodeExporterByAddress`); the Disk column stays empty on kind, whose node's `/` is an
  overlay that node-exporter leaves out.
- **The Nodes list looks for Prometheus while its Disk column shows** (before: only when a drawer
  opened or the Metrics Server failed). The cost on a cluster with Prometheus: the detection's
  probe (once per 10 minutes) and the watch on Services the detection reads, then one query per
  poll; a cluster whose only provider is the Metrics Server alone (the demo cluster) asks nothing.
- **Hidden with the nodes or the filesystem** (Cluster Settings › Metrics), and not polled while the
  user put the column away (the ⋮ column list), unlike CPU and Memory, which only the cluster's
  settings stop: the disk costs Prometheus a query, the usage the Metrics Server a list the Pods
  view shares.
- **A node's drawer shows its disk** as a line under the node's chart ("Disk (/): 41.2 GiB of 58.4
  GiB used (71%)", in the bars' colour at the thresholds), polled while the drawer is open, shared
  with the Nodes list's poll; Lens's drawer has no disk line that is captured (inferred).

**The nodes' CPU and memory history, the same source (owner, 2026-10-14 evening):**

- **Decided (A43):** a node's CPU and memory history comes from node-exporter where the cluster's
  Prometheus has its series (kube-prometheus-stack), else from cAdvisor's root cgroup as built.
  **Built 2026-10-14** ([`metrics/promql.go`](../../internal/metrics/promql.go),
  [`metrics/promrange.go`](../../internal/metrics/promrange.go),
  [`metrics/promcluster.go`](../../internal/metrics/promcluster.go); `TestNodeHistoryFromNodeExporter`,
  `TestClusterHistoryFromNodeExporter`, `TestNodeDrawerHistoryFromNodeExporterByAddress`,
  `TestNodeHistoryOnKubePrometheusStack`; [architecture §9](architecture.md#9-metrics-internalmetrics)):
  the node's drawer and the cluster Overview. Before, nodes read cAdvisor only, and
  kube-prometheus-stack's kubelet ServiceMonitor drops the root cgroup's series, so a node's drawer
  and the Overview there showed no history (measured on kind, 2026-10-12).
- **The owner's finding (F1, in a real window):** no past data is shown for nodes. "Besorge die
  Vergangenheitsdaten, wenn verfügbar, aus Prometheus" (take the history from Prometheus where it
  is available). **Built 2026-10-14** with A43, after checking every chart: the cause and what
  each chart does are in [architecture §9](architecture.md#9-metrics-internalmetrics) ("the nodes'
  history from node-exporter").

What the build chose for A43 and F1, **the lead's decisions for the owner to confirm** (night run,
2026-10-14; nobody could be asked):

- **Matched as the disks are:** the node label, else the instance as the node's name or one of its
  addresses (InternalIP, ExternalIP, Hostname, from the cache: `Hub.SetNodeAddresses`). Required,
  not optional: kube-prometheus-stack 92.3.0's node-exporter has no relabelings by default (its
  values, read 2026-10-14), so its series carry the exporter's host-network address
  (`10.0.0.7:9100`) and no node label. Options: (a) the name only, (b) name and addresses;
  chosen (b), (a) finds nothing on kube-prometheus-stack.
- **CPU:** the cores the node worked, `rate(node_cpu_seconds_total{mode!~"idle|iowait|steal"})`
  summed over its CPUs. **Memory:** `MemTotal − MemFree − Inactive(file)`, which is how the kubelet
  counts the root cgroup's working set (the Metrics Server's node memory) on cgroup v2, so the
  history and the live values after it mean the same. Options for memory: (a) that, (b) `MemTotal −
  MemAvailable` (kube-prometheus's dashboards); chosen (a), (b) counts the active file cache as free
  and would step down at the seam. Inferred from the kubelet's and runc's accounting, not measured
  on a real node: on kind both read different machines (the exporter the Docker VM, the kubelet the
  node's container).
- **Every exporter counted once** (`max by (node, instance)`, as the disks' query: two jobs that
  scrape one exporter and label it alike; two that label it differently, one with a node label and
  one without, count it twice: not seen, not guarded), and **every exporter is a node** in the
  Overview's "every node" sum: a Prometheus that also scrapes node-exporters outside the cluster
  would add them to the workers' history (not guarded; the live values after it do not have them).
  Options: (a) every exporter, (b) only the exporters matched to the cluster's nodes, which names
  every node in the query or filters a per-node answer in Cogpit; chosen (a), the cAdvisor sum's
  rule and a short query on a cluster of thousands of nodes.
- **node-exporter is tried first, cAdvisor when it has no series**, and the answer is remembered for
  the provider's life (one detection, 10 minutes): a Prometheus without node-exporter costs two more
  range queries per node history (four for the Overview's) once per detection, not at every poll; a
  node without an exporter on a Prometheus that has them is asked of cAdvisor. Options: (a) both
  sources asked at once every time, (b) node-exporter, then cAdvisor, nothing remembered, (c) (b)
  remembered per detection; chosen (c), the fewest requests that still prefer node-exporter (A43).
- **Control-plane nodes without an exporter** (a DaemonSet that does not tolerate their taint): the
  Overview's masters have no history and the workers' history is every exporter's. Options: (a)
  that, (b) the masters from cAdvisor and the workers as every exporter less them (two sources in
  one subtraction); chosen (a), a sum of one source.
- **Said where there is none:** a drawer or the Overview whose Prometheus has no CPU or memory series
  of the object writes "Prometheus has no history of this Node; Cogpit shows what it measured."
  under the source line (`ObjectMetrics.HistorySource`, `ClusterOverviewModel.UsageHistory`, also
  for agents). Options: (a) a note, (b) nothing, as before; chosen (a): the owner's finding was a
  chart that started late without saying why. The wording is Cogpit's.
- **Addresses known late:** a node the cache lists only after its drawer opened gets its addresses
  at the next read of the drawer's metrics, and its history is asked once more with them (the chart
  remembers the addresses it asked with). Options: (a) that, (b) wait for the node before the
  drawer's first poll; chosen (a), the history then never waits for the cache.
- **Not changed then:** the network, the filesystem and the lists' instant queries (the Metrics
  Server's fallback for the Pods and Nodes lists and `metrics.get`) still read the root cgroup. The
  network and the filesystem since 2026-10-16 (below); the lists' fallback still does, so the Nodes
  list has no CPU and Memory on kube-prometheus-stack while the Metrics Server is down.

**A node's network and filesystem, the same source (G12, built 2026-10-16 night).** On
kube-prometheus-stack a node's Network and Filesystem charts stayed empty: they read cAdvisor's root
cgroup, which its kubelet ServiceMonitor drops. Now node-exporter's where it runs, matched as CPU
and memory are ([`metrics/promql.go`](../../internal/metrics/promql.go) `nodeInterfaces`,
`nodeBlockDevices`, `exporterFSUsage`; [`metrics/promrange.go`](../../internal/metrics/promrange.go)
`nodePlan`; `TestNodeNetworkAndFilesystemFromNodeExporter`). Checked against a real
Prometheus Operator installation's node-exporter (kube-prometheus, whose kubelet ServiceMonitor
drops the root cgroup as kube-prometheus-stack's does) on a cluster the owner provides for tests
(Cilium with WireGuard; 2026-10-16): an hour of 121 points per series for received, transmitted, the filesystem's
usage and the disks' rates; the received rate the physical interface's alone; and on kind with the
prometheus chart's exporter. The lead's choices, **confirmed (owner, audit of 2026-10-16)** as built (the
filesystem's rule, which offered no options, not asked):

- **Which interfaces are the node's traffic.** Options: (a) every interface but the loopback, (b)
  the physical ones by name (`eth`, `en…`, `bond…`), ★ (c) every interface but the loopback and the
  virtual ones the runtimes, the network plugins and the kernel create (veth, lxc, cali, cilium_,
  flannel, cni, cbr, docker, br-, virbr, vxlan, genev, tunl, tun, wg, kube-, weave, antrea-, ovs-,
  ovn-, the kernel's fallback tunnels, Calico's hashed names). (a) counts the pods' traffic several
  times: on the real cluster one node had 34 interfaces, the pods' traffic passing the lxc end, the
  overlay, WireGuard and the physical interface; (b) misses interfaces named otherwise; (c) counts a
  name it does not know, and a bond with its members twice (not guarded).
- **The filesystem.** Its usage: each block device's size less what is available (the disks' rule
  above), each device once however often it is mounted, as the root cgroup's usage counts block
  devices; the disks' read and write rates where a node-exporter has no filesystem series, without
  device-mapper, software RAID, network block devices (Ceph's rbd, nbd), loop and RAM devices.
- **One query with both sources.** Options: (a) node-exporter, then the root cgroup, tried in turn
  and remembered, as CPU and memory are; ★ (b) one query each, node-exporter's joined to the root
  cgroup's by `or`. The network and the filesystem are asked at every step while a chart shows them,
  and (a)'s discovery would cost queries the window reports nothing of; under (b) a chart that spans
  an exporter's start may step there, where the two count differently. The budget is unchanged:
  measured on the real cluster, a node's drawer at 1 h asks Prometheus once in 90 s with CPU shown
  (the disk's poll) and 8 times with the network shown ([testing](testing.md#performance-budgets-proposed-not-measured), P13).
- **A query format that fixes the source** (Cluster Settings › Metrics, 2026-10-16): "Prometheus
  Operator" asks node-exporter alone, "cAdvisor only" the root cgroup alone, CPU and memory included.

## D-44: What agents read of the dock's logs and terminal tabs

**Decided (owner, 2026-10-14):** option (b), as built: logs tabs freely, a terminal tab only after the user turned on its switch, off for every new tab.

**Added (owner, 2026-10-14, [H-83](../security/agents-and-mcp.md#h-83-an-allowed-terminal-tab-gives-agents-everything-it-shows-from-then-on)):**
a visible marker on the tab while its switch is on, flashing at each agent's read; the switch turns
itself off after 15 minutes without an agent's read and says so in the tab. **Built 2026-10-14**
([`engine/agentread.go`](../../internal/engine/agentread.go),
[`fyneui/dockagentread.go`](../../internal/ui/fyneui/dockagentread.go)): the switch, the last read,
its end and that it turned itself off are UI state (`DockTab.AgentLastRead`, `AgentReadUntil`,
`AgentReadExpired`); a read emits `EventDockRead`, to which the window reacts with the flash; one
timer on the session's clock (`Options.AfterFunc`) ends the earliest switch.
Cogpit's defaults in the build (not the owner's decision): the 15 minutes count from the later of
switching on and the last read; every read the switch allowed counts (an agent's, or the CLI's
forwarded to the window), the user's own never; the marker an eye after the title in the warning
colour, the flash the tab tinted in that colour for 0.7 s, again at a read during a flash; the note
("Turned off after 15 minutes without an agent's read") beside the switch while that tab is the
active one, gone when the user turns it on again, none when the user turns it off by hand; the end
of a switch is not an audit entry (no one's command; `ui_state` shows it).
**Decided (owner, 2026-10-14 evening, A3):** a switch ends at the latest 1 hour after it was turned
on, whatever the reads; the tab says so, and one click turns it on again. Read here as beside the
15 minutes without a read, whichever comes first (open-decisions follow-up 10, still to confirm).
Built 2026-10-14: `AgentReadMax`, `DockTab.AgentReadOn` and `AgentReadCapped`, the strip's "Turned
off one hour after it was turned on" (`TestAgentReadSwitchEndsAfterAnHour`,
`TestAgentReadMarkerFlashes`).

**The night decision (the lead's, 2026-10-14), confirmed by the owner (2026-10-14):** a terminal tab's output can
hold anything the user typed or a program printed, credentials included (a `cat` of a token file,
`env`, `kubectl create token`, a Secret piped through `base64 -d`). Options: (a) agents read terminal
output freely, as they read logs; (b) terminal output only after the user allows it per tab, with a
switch on the tab that is off when the tab opens, logs freely; (c) never terminal output, logs
freely; chosen (b), because a logs tab shows cluster data that the user's access already governs
and that is an agent's to read like events and YAML (D-2: reads run freely), while a terminal is
the user's own workspace on this computer, whose output neither the cache's redaction of Secret
values nor any other rule of Cogpit's can see; (a) would hand all of it to every connected agent,
and through the built-in agent to its LLM endpoint, without the user knowing; (c) loses the case the
feature exists for ("read the error in my terminal") and leaves the user to copy and paste, which
is (b) by hand and less visible. **Built 2026-10-13** (night build, G17;
[`engine/dockoutput.go`](../../internal/engine/dockoutput.go),
[`commands/dockoutput.go`](../../internal/engine/commands/dockoutput.go),
[`fyneui/dockoutput.go`](../../internal/ui/fyneui/dockoutput.go),
[`termview/output.go`](../../internal/ui/fyneui/termview/output.go)): `dock_read` reads a logs tab's
lines as its view shows them, a terminal tab's screen only once the user turned on the tab's
"Agents may read" switch (`dock.allowAgentRead`, GUI-only: no agent can turn it on);
`TestDockReadTerminalNeedsTheUsersSwitch`, `TestDockOutputOfATerminalTab`.

What the build chose, Cogpit's defaults (not the owner's decision):

- **Every terminal tab is a terminal:** the local terminal, a pod shell, attach and the node shell
  all need the switch (a container's shell prints its environment's Secrets as readily as a local
  one). Options: (a) all four, (b) the local terminal only; chosen (a).
- **Off whenever a tab opens**, also one an agent opened (`terminal_open`, `pod_shell`), and not
  remembered across tabs or restarts; closing the tab ends it.
- **No dialog to turn it on** (options: (a) a plain switch, (b) a dialog whose Allow button arms
  after the confirmations' 750 ms, as the allowance of agent writes has; chosen (a)): the switch sits
  in the strip of the tab the user is looking at, no command opens a dialog or moves the pointer
  there, and turning it on exposes one tab, not every write in a cluster. But an agent can make
  another terminal tab the active one (`dock_activate`, a UI command) just before the user's click
  lands, so turning the switch on is ignored for the confirmations' 750 ms after it began to show
  another tab (turning it off never is; `TestAgentReadSwitchArmsAfterATabChange`).
- **What agents read:** a logs tab's lines its view shows (the pod and container it narrows to, its
  filter applied, resource names and timestamps as the tab shows them, without escape sequences);
  a terminal's lines as text (no colours or escape sequences). Until 2026-10-15 the terminal widget
  kept no scrollback, so that was its screen; since then it keeps one, and agents read its lines
  too, a line by a number it keeps while it scrolls (decided by the owner,
  [D-56](#d-56-the-terminals-scrollback) item 7).
- **Bounds:** 200 lines by default, at most 2,000 and 256 KiB a page, each line at most 8 KiB; paging
  by the lines' numbers (`after`, `before`), which stay while the tab holds the lines.
- **A read and the switch:** a read that the user's switching off (or closing the tab) overtakes
  returns nothing; reads are audited like every agent command, with their parameters, never the
  lines.
- **The built-in agent** (built since 2026-10-14, [D-10](#d-10-built-in-agent)) is an agent like the
  others for the switch (`dock.read` with origin `builtInAgent` in
  [`dockoutput_test.go`](../../internal/engine/commands/dockoutput_test.go)); what it then sends
  to its LLM endpoint falls under the endpoint's rules
  ([H-13](../security/agents-and-mcp.md#h-13-cluster-data-leaves-for-the-llm-endpoint)).

**Decided (owner, 2026-10-14), the answer to [H-82](../security/agents-and-mcp.md#h-82-secret-values-printed-into-logs-reach-agents):**
the values of the cached Secrets, from about 8 characters (against false matches, measured over
10,000 log lines), are replaced in what an agent reads (`dock_read`'s log lines, `object_get`'s and
`editor_get`'s YAML) by `«secret <ns>/<name>/<key>»`; the window shows the text as it is. **Built
2026-10-14** ([`engine/secretscrub.go`](../../internal/engine/secretscrub.go),
[`engine/agentview.go`](../../internal/engine/agentview.go),
[`cache/peek.go`](../../internal/cache/peek.go)): one filter in `Dispatch` for every result of an
agent's command; a matcher over 8-byte windows (a bit filter and an index, one pass over the text
whatever the number of values) built from the Secrets the cache holds, again only when they changed;
the measurement and its numbers are in H-82. The build's choices, **the lead's decisions for the
owner to confirm**:

- **The base64 form is replaced too** (options: (a) the decoded value only; (b) the decoded value
  and the value as stored, base64, plus the forms a log prints a value in: JSON-escaped, a PEM
  private key's body line by line, a registry credential's passwords and auth strings; (c) also
  URL-encoded and other encodings): chosen (b). The base64 form is what a Secret's own manifest
  carries, so a copy of one in another object's YAML or annotation
  ([H-23](../security/agents-and-mcp.md#h-23-secret-annotations-other-than-kubectls-reach-agents))
  or a debug dump in a log holds it; it costs one more pattern per value, and the 10 planted leaks
  of a base64 value are caught through it alone. (c) multiplies the patterns for forms no measured
  line held.
- **Minimum 8 bytes**, the owner's figure, in every form; the measurement shows the minimum is not
  what decides the false matches (H-82's table). **Decided (owner, 2026-10-14 evening, A5):** 10
  bytes instead of 8; the round's figures, given with the answer and not measured again for this
  page: 80 of 100 planted leaks caught, no line loses a common word. Built 2026-10-14
  (`MinScrubbedValue` 10, the matcher's window stays 8 bytes; `TestSecretScrubMeasure` gives the same
  80 of 100 and 0 lines altered without a leak at 10).
- **A value under 16 bytes only where it stands alone** (no letter, digit, `_` or `-` touching it):
  "postgres" is not replaced inside "postgresql" or "postgres-0" (options: every match; stand-alone
  for short values, chosen; stand-alone for every value, which would miss a token glued to other
  text). It halved the lines with a name replaced (2,740 to 1,238) and caught the same leaks.
- **The values of name keys are left out**: a key whose last word is user, username, login, host,
  hostname, server, port, database, db, dbname, namespace, schema, region or email. Options: (a)
  every value, as the owner's words say; (b) leave name keys out, chosen; (c) a higher minimum (10
  bytes: 73 lines, but 5 more planted leaks missed). With (a) 1,238 of 10,000 lines (12 %) had a user,
  host or database name replaced, which hides what a line is about from the agent and protects
  nothing a credential protects; (b) brought it to 0 and caught the same leaks. This narrows the
  owner's rule, hence a question for the owner; the same kind of allowlist as D-35's.
  **Confirmed (owner, 2026-10-14 evening, A4):** (b), as built.
- **Also left out:** certificates (public, D-30: the cluster CA that every namespace's
  `kube-root-ca.crt` holds stays readable), a service account token Secret's `namespace` and
  `ca.crt`, a bootstrap token's non-secret keys, Helm's release records (their payload is never
  printed), values over 64 KiB.
- **Every `dock_read` page, a terminal tab's too**, not only logs tabs: a terminal the user allowed
  shows what programs print as a log does
  ([H-83](../security/agents-and-mcp.md#h-83-an-allowed-terminal-tab-gives-agents-everything-it-shows-from-then-on)).
- **Also the drawer's values, table cells and events, an editor's diff, an apply's and a revert's
  text** in what an agent reads, not only the YAML: they carry the same values. The CLI is not
  filtered (the user at a terminal).
- **Which Secrets: decided (owner, 2026-10-14), option (b).** Options: (a) only the Secrets cached
  already (the build's first choice: before the Secrets view was opened nothing was replaced);
  (b) an agent's read that the filter reads takes the Secrets as the Secrets view reads them, waits
  for them and lets them go, the cache keeping them warm for its warm period; (c) the Secrets
  watched whenever the MCP server is on; (d) a GET per read. The lead recommended (b): only then
  does the filter work when an agent reads a log, at the cost of one watch (a read) and the values
  in memory while agents read, which the Secrets view costs anyway; (c) holds them without any
  agent's read; (d) is a request per read. **Built 2026-10-14**
  ([`engine/agentsecrets.go`](../../internal/engine/agentsecrets.go); `TestAgentViewReplacesCachedSecretValues`,
  `TestAgentReadWithoutSecretsAccess`, `TestBudgetAgentReadTakesTheSecrets`: the first agent read
  11 ms and one streaming watch of Secrets on the fake API server, a second within the warm period
  no request). Cogpit's defaults in it (not the owner's decision): the wait is bounded by 10 s, and a
  read whose Secrets were not listed by then is refused (`ErrAgentSecretsPending`, fail closed)
  rather than sent as printed; a user who may not list Secrets, or may list them in none of the
  namespaces Cogpit reads, is not waited for; only results the filter reads take the Secrets
  (`ui_state`, a list, a search take none, P11).
- **An agent's text that carries a placeholder or `<withheld…>` back is refused**
  (`ErrAgentMarkerWrite`) where a Secret's text is (`GuardSecretEditor`, `GuardSecretText`):
  writing it would replace the value in the cluster (never surprise a cluster). Substituting the
  values back was the other option; rejected, since it would let an agent place a Secret's value
  where it chooses without reading it.
- **The placeholder** names one Secret when several hold the same value (the first by namespace,
  name and key), and overlapping values are replaced together.

Other text agents read, decided in the round of 2026-10-14 evening (owner):

- **Decided (A31, [H-86](../security/agents-and-mcp.md#h-86-a-charts-readme-reaches-agents)):** a
  chart's README reaches agents framed as third-party text ("text of the chart's author, not
  instructions"). Built 2026-10-14: `engine.FrameReadme` in `AgentView.Detail` puts a line before
  and after the README section's text for agents (`ReadmeFrameStart`, `ReadmeFrameEnd`); the window
  reads it unframed (`TestHelmChartReadmeForAgents`).
- **Decided (A19):** the notifications that stay in the window today reach agents as text, through
  this filter and with inline Helm values removed ([D-2](#d-2-agent-interface-scope)). Built
  2026-10-14.

## D-46: Chart icons from the hosts that serve them

**Decided (owner, 2026-10-14):** option (b), icons fetched from the hosts that serve them with the built safeguards, "Load chart icons" on by default, as built.

**The night decision (the lead's, 2026-10-14), confirmed by the owner (2026-10-14):** a chart repository's index
names each chart version's icon as a URL on any host (a CDN, GitHub, the project's own site), and
Lens shows icons in its Charts view (inferred; neither its icon column nor its chart drawer is
captured). Fetching one tells whoever serves it that this computer looked at the chart
(its address, the time, the URL), and an index can name any URL, a host in the user's own network
included. Options: (a) no icons: nothing fetched, Lens's look not met; (b) icons fetched from their
URL when the window shows them, https only, without credentials, bounded in size, a PNG or JPEG
decoded under bounds and an SVG checked, kept in Cogpit's cache by the URL's hash for a while, with
a Preferences switch; (c) icons only from the repository's own host, which already learns what the
user looks at from the archives and indexes (most public repositories name icons elsewhere, so most
charts would show no icon); chosen (b) with Preferences › Kubernetes › "Load chart icons" on by
default, because the icon is how a user finds a chart in a long list, and the cost is a request per
icon shown, to a host the chart's author chose, carrying nothing but the URL, which the switch turns
off for whoever minds; (c) costs most of the icons for a privacy gain only against the hosts a
chart's author chose. **Built 2026-10-14** (night build HELM2, G11;
[`helm/icon.go`](../../internal/helm/icon.go), [`engine/helmicons.go`](../../internal/engine/helmicons.go),
[`fyneui/helmcharticons.go`](../../internal/ui/fyneui/helmcharticons.go)): the Charts view's first
column and a chart drawer's header draw the icon, Cogpit's own helm-wheel glyph while it loads, when
there is none, when it failed and while the switch is off; `TestFetchIcon`,
`TestFetchIconRefusesPrivateAddresses`, `TestChartIcons`, `TestChartIconRefusals`,
`TestChartIconsPreference`, `TestChartIconsInTheWindow`, `TestPreferencesLoadChartIcons`, PNGs
[dark](../../internal/ui/fyneui/testdata/helm_chart_icons_dark.png),
[light](../../internal/ui/fyneui/testdata/helm_chart_icons_light.png).

What the build chose, Cogpit's defaults (not the owner's decision):

- **When:** only for a cell or the drawer the window draws, at most 4 downloads at once, one per URL;
  never for agents (no command fetches an icon; `object_get` and the rows carry the URL), never for
  the CLI.
- **How:** https only, a redirect too (at most 5); no credentials (not the repository's, whatever
  `pass_credentials_all` says), no cookies, no referrer, the user agent "Cogpit"; at most 256 KiB
  and 20 seconds; through the Preferences' proxy for non-cluster traffic, as every download.
- **Where to (beyond the option):** a host that resolves to a loopback, private, link-local,
  carrier-grade NAT or otherwise non-public address is refused, so that an index cannot make Cogpit
  send requests into the user's own network; a chart repository's own host (any repository in
  Helm's file) is allowed at any address, so an internal repository's icons beside its charts show.
  Each request dials the address it checked. Through a proxy the proxy resolves the host and the
  check does not apply ([H-87](../security/trust-boundaries.md#h-87-chart-icons-tell-their-hosts-what-the-user-looks-at)).
- **What:** checked as an uploaded cluster icon is (`CheckIcon`): a PNG or JPEG decoded under bounds
  (16 megapixels, 8,192 a side) and kept as a PNG of at most 256 pixels a side; an SVG kept only
  when it is one `<svg>` document without a DOCTYPE, scripts, event handlers or references outside
  itself; anything else (GIF, WebP, ICO, HTML) refused; the window draws an SVG from a PNG of 128
  pixels a side that it rasterises off the main goroutine under a recover, never through Fyne's
  renderer (2026-10-14)
  ([H-88](../security/trust-boundaries.md#h-88-chart-icons-are-decoded-in-cogpits-process)).
- **Kept:** `<user cache dir>/Cogpit/helm/icons/<hash of the URL>.png` or `.svg`, owner-only, used for
  7 days, the last 2,048; in memory the last 512; a failure remembered for an hour.
- **The switch:** `Preferences.LoadChartIcons`, on by default; off, nothing is fetched or read and the
  glyph shows everywhere at once. Agents and the CLI may only turn it off (`ErrChartIconsGUIOnly`):
  turning it on decides whether Cogpit contacts the icons' hosts.

## D-47: Deep links (`cogpit://`)

**Decided in part (owner, 2026-10-14 evening, A7,
[H-89](../security/trust-boundaries.md#h-89-a-link-switches-the-cluster-and-the-view-under-the-user)):**
a link that names another cluster than the connected one asks once, naming the context, before it
switches; navigation within the connected cluster stays immediate. Built 2026-10-14
(`askLinkConnect`, `TestDeepLinkAsksBeforeItConnects`, `TestDeepLinksOpenInTheWindow`); it replaces
the default "A link may switch the connected cluster without asking" below. **The lead's night
decision, confirmed (owner, 2026-10-15):** a link asks before every connect it would start, also while
no cluster is connected (options: only when another cluster is connected), so that a page opening
links in a loop raises questions rather than connections (H-89's cost of a connect). The round
answered the switch only; the rest of the record stayed the lead's decision for the owner to
confirm until 2026-10-15.

**Confirmed (owner, 2026-10-15 afternoon), as a whole:** (a) navigation only, as built, with the
build's defaults: a login waiting for approval stops the link (no approval dialog from a link); the
window raised and a toast for every link; one audit line per link; the ask before every connect
above. Options weighed: (b) plus commands with a confirmation (consent phishing); (c) no deep links
(loses Lens's `lens://` parity).

**Apart from the switch above: the lead's decision of 2026-10-14, confirmed (owner, 2026-10-15);
built 2026-10-14.** Lens Desktop opens `lens://` links (the row in [ui-parity](../parity/ui-parity.md#appearance-and-application)
since 2026-10-11, from Lens's documentation; not captured, so what a Lens link can name is not
verified). Cogpit's app bundle registers the scheme `cogpit` (`CFBundleURLTypes`, the
[`Makefile`](../../Makefile)'s `app` target), and macOS hands Cogpit every `cogpit://` link that a
web page, a mail, a chat or another app opens, whether Cogpit runs or not. A link is therefore
untrusted input: its author is anyone. Options for what a link may do:

- (a) **navigation only**: connect a context of the user's kubeconfig sources, open a view, set the
  namespace filter, open an object's drawer, through the `ui.*` commands agents run freely (D-2);
- (b) (a) plus commands with a confirmation, as an agent's are confirmed (a link that asks the user
  to restart a Deployment);
- (c) no deep links.

**Chosen (a)**, because it gives what links are for (a runbook, a chat message or an alert pointing
at an object) at the cost of nothing the user's identity can write: (b) turns every web page into a
prompt the user must read with care before a click acts on a cluster with their identity, the
consent-phishing pattern; (c) loses Lens's links. Built ([`engine/deeplink.go`](../../internal/engine/deeplink.go),
[`cmd/cogpit/deeplink_darwin.m`](../../cmd/cogpit/deeplink_darwin.m),
[`fyneui/deeplink.go`](../../internal/ui/fyneui/deeplink.go); the grammar in
[usage](usage.md#deep-links-cogpit); the boundary in
[trust boundaries](../security/trust-boundaries.md#boundaries-and-required-checks)). What the build
chose, Cogpit's defaults (the build's choices, not the owner's, until the owner confirmed them on
2026-10-15 with D-47 as a whole):

- **The grammar:** `cogpit://open?context=<context id>&view=<view key>&namespace=<namespace>&name=<name>`,
  every parameter optional but one; read strictly (the scheme, the action `open`, no user, port,
  path or fragment, each parameter once, not empty, without control or format characters, a view
  key's, a DNS label's or an object name's form, at most 2,048 bytes in all); anything else is
  refused whole, never half followed (`TestParseDeepLink`, `TestParseDeepLinkRefusals`,
  `FuzzParseDeepLink`).
- **Which contexts:** one of the kubeconfig sources' contexts, by its id or by a kubeconfig context
  name one file alone defines, or a development build's demo cluster ([D-67](#d-67-the-demo-cluster-stays-out-of-releases)); never one a connector without sources offers
  (it would run the kubeconfig's credential plugins unasked), never a file path.
- **A login that waits for the user's approval stops the link:** a context whose credential plugin
  or token file the user has not allowed, or that changed since, is not connected, and the toast
  says to connect from the navigator's clusters (⌘P; the context picker until D-57), where the user
  reviews it. Weighed: showing the approval
  dialog from the link (the brief allowed either); not chosen, because a dialog that a web page
  raises at a moment of its choosing invites allowing a program the user did not ask for. An
  approved plugin runs, as when the user connects.
- **A link may switch the connected cluster without asking** (replaced by the owner's answer of
  2026-10-14 evening above, built 2026-10-14), as the hotbar does; Cogpit holds one
  connection, the other cluster's tabs stay, and a write the user confirmed for the other cluster
  refuses to run (`CheckConfirmedContext`). A settings page that is open closes, or it would hide
  what the link opened ([H-89](../security/trust-boundaries.md#h-89-a-link-switches-the-cluster-and-the-view-under-the-user)).
- **Origin `deeplink`**, neither the user in the window nor an agent. `Dispatch` refuses it every
  command but the five navigation commands (`DeepLinkCommands`: `settings.close`,
  `cluster.connect`, `ui.setNamespace`, `ui.openView`, `ui.navigate`), and any of those that is not
  a UI or read command, may escalate, is GUI-only or declares a verb other than get, list and
  watch, before its parameters are decoded; and once more after `Escalate`, any invocation that is
  not UI or read. So a write is unreachable by construction, whatever a later change to
  `OpenDeepLink` dispatches (`TestDeepLinkCommandsOnlyNavigate` over the whole registry,
  `TestDeepLinkGuardsAnAlteredCommand`). `NeedsConfirmation` is not changed: it never sees a link's
  write, because none gets that far. Every `Refuse` and GUI-only mark that admits only the GUI
  treats a link as not the user.
- **What the user sees:** the window comes to the front for every link, and a toast says what it
  opened ("Opened from a link: Pods › team-a/web-0 on kind-dev [config]") or why it did not; both
  are in the notification log. The audit log gets one line per link, `deeplink.open` with origin
  `deeplink`, the link's parameters (or, for a link that does not parse, at most 256 bytes of it,
  escaped) and the outcome ([H-90](../security/trust-boundaries.md#h-90-a-links-text-reaches-agents-through-the-logs)).
- **Delivery:** the Apple event handler (`kAEGetURL`) is installed before the app finishes
  launching and again when AppKit is about to, independent of GLFW's app delegate; links wait in a
  queue (at most 8; one longer than 2,048 bytes dropped) until the window is up, then open one after
  the other (`TestDeepLinkAppleEvents` in a process of its own, `TestDeepLinkQueue`,
  `TestDeepLinksOpenInTheWindow`).
- **Not on Windows and Linux:** there are no packages to register the scheme in yet (G22).

## D-48: Start at login

**Decided (owner, 2026-10-14 evening, A8): as built**, the lead's decision of 2026-10-14 below:
option (a), off by default, only the user turns it on. The details listed as Cogpit's defaults
stay Cogpit's.

**The lead's decision of 2026-10-14, built 2026-10-14.** Lens
Desktop has "Automatic start-up" among its application preferences (inferred from its
[preferences](https://docs.lenshq.io/k8slens/using-lens/preferences/) documentation, not captured;
its default not verified). Options:

- (a) **the macOS app's own login item** (`SMAppService.mainAppService`, macOS 13 and later,
  Cogpit's minimum), which System Settings › General › Login Items lists and the user can switch
  there too; Cogpit keeps no copy and shows what macOS reports;
- (b) a LaunchAgent property list Cogpit writes into `~/Library/LaunchAgents`: works without a
  signature, but is a file outside Cogpit's own folders that System Settings shows as a background
  item, and Apple's way since macOS 13 is (a);
- (c) a preference kept in `preferences.json` that Cogpit applies again at every start: it would
  undo the user's switch in System Settings.

**Chosen (a)**, **off by default**, and **only the user turns it on** (agents and the CLI may turn it
off, as with chart icons, D-46), because it makes Cogpit run after every login without being
opened, its MCP server included unless the user switched that off. Built
([`engine/loginitem.go`](../../internal/engine/loginitem.go),
[`commands/loginitem.go`](../../internal/engine/commands/loginitem.go),
[`cmd/cogpit/loginitem_darwin.m`](../../cmd/cogpit/loginitem_darwin.m),
[`fyneui/preferences_loginitem.go`](../../internal/ui/fyneui/preferences_loginitem.go)): Preferences ›
App › "Start Cogpit at login", a check with a line saying what macOS reports (off, on, waiting for
the user's approval in System Settings with a button to Login Items, not found), read when the
section opens and after every change; `preferences.loginItem`, `preferences.setStartAtLogin`
(turning it on refused for every origin but the GUI) and `preferences.openLoginItems` (GUI-only);
`UIState.LoginItem`. Cogpit's defaults: the row's place (App) and wording; a Cogpit that does not
run from its app bundle (`bin/cogpit`, `go run`) reports not found and refuses to register without
asking macOS.

**What needs the signed app.** "Apps that use SMAppService APIs must be code signed"
(ServiceManagement's `SMAppService.h`). The releases' app carries the Developer ID signature since
v0.6.0 (D-4); a local build is ad-hoc signed. What macOS does with either is **not verified**: it
may register it, report it not found, or refuse with an error, which the row shows under the check.
Nothing was registered on a development machine to find out, and no signed app installed in
Applications has been tried yet; that is the check. **Not offered** on Windows and Linux (a Run key, an XDG autostart file
come with their packages, G22), rather than a switch that does nothing.

## D-49: The external MCP server's switch, honoured by `cogpit-cli mcp`

**Decided (owner, 2026-10-14 evening, A15,
[H-28](../security/agents-and-mcp.md#h-28-the-status-bars-switch-does-not-reach-a-headless-bridge)); built 2026-10-14.**
"The MCP off switch is persisted and honoured by cogpit-cli mcp: with MCP off the bridge refuses
to serve (headless too), with a clear message to the agent." The switch itself is the status bar's
item of [D-23](#d-23-a-link-to-the-owner-in-the-status-bar); the night run asked (#117) for a
record of its own once its reach was decided, and this is it.

**Before (verified in this repository, until 2026-10-14):** the window kept the user's choice in Fyne's preferences
(`mcp.enabled`, absent means on, written only by a click; [`fyneui/mcp.go`](../../internal/ui/fyneui/mcp.go))
and reads it at the app's next start; `--no-mcp` wins for a run. Switching it off stops the app's
HTTP endpoint and removes the token and endpoint files. `cogpit-cli mcp` never reads the choice:
with no running app to forward to (no endpoint file, or nothing listening) or with `--headless`, it
serves an engine of its own ([`cogpit-cli/mcp.go`](../../cmd/cogpit-cli/mcp.go),
[architecture §7](architecture.md#7-mcp-and-the-built-in-agent)), which connects the contexts the
user's kubeconfig sources and approvals allow, runs read and UI commands and refuses those that
need a confirmation. So an agent host that starts the bridge reaches the clusters while the user
has MCP switched off.

**The options (#31):**

| Option | For | Against |
|---|---|---|
| (a) As built: the switch stops the app's endpoint only | Nothing to build | "Off" does not mean off for an agent host that starts the bridge |
| **(b) The switch persisted where the bridge reads it, the bridge refusing to serve while it is off, headless too (decided)** | One switch for every way in; the agent is told why | The bridge depends on a setting of the app's; a user who wants only the headless bridge must switch MCP on |

**Built 2026-10-14** ([`mcp/switch.go`](../../internal/mcp/switch.go),
[`cogpit-cli/mcp.go`](../../cmd/cogpit-cli/mcp.go), [`fyneui/mcp.go`](../../internal/ui/fyneui/mcp.go),
[`cogpit/mcpcontrol.go`](../../cmd/cogpit/mcpcontrol.go); `TestSwitchFile`, `TestServeSwitchedOff`,
`TestMCPSwitchedOffRefuses`, `TestMCPSwitchStore`). **The lead's night decisions for the owner to
confirm** on what the decision left to the build:

- *Where the choice is kept:* ★ a file `mcp-switch` ("on" or "off") beside the token file in
  Cogpit's configuration directory, owner-only and atomic like the token, written by the window's
  status bar item only (`fyneui.MCPSwitchStore`, implemented by the app's `mcpControl`); no file
  means on. Options: Fyne's preferences (cogpit-cli has no Fyne), the engine's `preferences.json`
  (then `preferences.set` would carry it as a command parameter agents could reach). A switch set
  before this build, in Fyne's preferences, is copied into the file at the next launch. A file that
  holds anything else, or is a symlink, reads as off: the switch fails closed.
- *What the agent is told:* ★ the bridge serves an MCP server whose instructions say the server is
  switched off in Cogpit's window and to ask the user, with one tool, `cogpit_mcp_off`, that
  returns the same as an error; it reaches no engine. Options: exit with an error on stderr (an
  agent host shows only "server failed").
- *A bridge forwarding when the user switches off:* ★ unchanged: the app removes its endpoint file
  (until F6 its token file too; the token is kept since,
  [D-53](#d-53-the-mcp-details-and-a-token-kept-until-the-user-regenerates-it)), the bridge's next
  call fails ("switched its MCP server off") and it does not fall back to a headless engine (its
  mode is chosen at its start).
- *`--no-mcp`* (the round's follow-up 7): ★ a flag of one run binds nothing: the stored switch
  stays as the user set it, so a bridge started during such a run serves headless if the switch
  says on. Options: a run marker the bridge honours (a crashed run would leave it behind).

D-2's copied registration command is offered only while the external server runs, as before.

## D-50: kubectl's signature check

**Decided (owner, 2026-10-14 evening, A25,
[H-35](../security/trust-boundaries.md#h-35-kubectls-signature-check-rests-on-an-embedded-trusted-root-and-rekor-v1s-search));
built 2026-10-15 (night run), with the lead's decisions below for the owner to confirm.**
"kubectl's signature checked with sigstore-go including the Rekor transparency-log proof (instead of
the standard library); record as a D-entry." [D-3](#d-3-helm-and-kubectl-binaries) decides where a
terminal's kubectl comes from; this record decides how a downloaded one is verified.

**Until 2026-10-15 (built 2026-10-13, D-3):** the release's `.cert` had to chain to the Fulcio
certificate authority Cogpit embedded and name the Kubernetes release's identity and issuer, and its
key had to have signed the published SHA-256, all with Go's standard library; sigstore's
transparency log was not consulted. What that left (H-35): a signature made with a leaked key, or
under a certificate Fulcio issued by mistake, passed.

**The options (#97):**

| Option | For | Against |
|---|---|---|
| (a) The standard library, as built until 2026-10-15 | No dependency; offline | No transparency log |
| **(b) sigstore-go with Rekor's transparency-log proof (decided, built)** | Sigstore's own verification, the log included | A new dependency in a security-critical path (measured below); one more outgoing request |

**What a download does since 2026-10-15.**

[`kubectl/download.go`](../../internal/kubectl/download.go), [`verify.go`](../../internal/kubectl/verify.go),
[`rekor.go`](../../internal/kubectl/rekor.go); still GUI-only, still through the Preferences' proxy
for non-cluster traffic, each step refusing the download with its reason:

1. `.sha256`, `.sig` and `.cert` from `dl.k8s.io` or the mirror, 16 KiB each at most (unchanged).
2. Offline (`ParseSigned`): the `.cert` parses, and the `.sig` was made over the published SHA-256
   by the certificate's key. Files that do not fit together never reach the log.
3. Online (`lookupRekor`): one `POST https://rekor.sigstore.dev/api/v1/log/entries/retrieve` with
   the hashedrekord entry of that digest, signature and certificate, as `cosign verify-blob` searches
   when it is given `--signature` and `--certificate`; through the same client and proxy, 30 s and
   64 KiB at most, at most four entries. No answer, a 429 or a 5xx is "sigstore's transparency log
   could not be asked" (`ErrLogUnreachable`); an empty answer is "sigstore's transparency log has no
   entry for it" (`ErrNotLogged`).
4. sigstore-go (`Verifier.Verify`): the three files and the entry as a sigstore bundle (v0.3), checked
   with `verify.WithTransparencyLog(1)`, `WithObserverTimestamps(1)` and
   `WithSignedCertificateTimestamps(1)` against the embedded trusted root, the policy the published
   digest and the identity `krel-staging@k8s-releng-prod.iam.gserviceaccount.com` with the issuer
   `https://accounts.google.com`, both exact: the entry's inclusion proof up to a checkpoint the log
   signed, its signed entry timestamp, the entry's digest, signature and certificate equal to the
   files', its time inside the certificate's ten minutes, the chain to Fulcio at that time, one
   signed certificate timestamp of a CT log the trusted root names, the identity and issuer, and the
   signature over the digest.
5. The binary, at most 128 MiB, must have the published digest; 0755, renamed into place
   (unchanged).

The trusted root is [`sigstore_trusted_root.json`](../../internal/kubectl/sigstore_trusted_root.json),
sigstore's public-good `trusted_root.json` (two Fulcio authorities, the Rekor v1 log and the Rekor v2
log `log2025-1`, two CT logs, the timestamp authority); `sigstore_fulcio.pem` is gone, its root and
intermediate are in the trusted root (`TestEmbeddedTrustedRoot` checks their fingerprints). It is
refreshed by `make sigstore-root` ([`hack/sigstoreroot`](../../hack/sigstoreroot/main.go)): fetched
through sigstore's TUF repository, verified from the TUF root sigstore-go embeds, written only when
it parses and names a log, an authority and a CT log. Renovate cannot see it
([releasing › Renovate](releasing.md#renovate)).

**Research (2026-10-15).**

Verified:

- `dl.k8s.io` publishes beside kubectl 1.37.1 (darwin/arm64) `.sha256`, `.sha512`, `.sig` and
  `.cert`, and no `.sigstore.json`, `.bundle`, `.sigstore`, `.intoto.jsonl` or `.pem` (404): no
  offline proof, so the log is asked per download.
- `rekor.sigstore.dev` has the hashedrekord entries of kubectl 1.36.1 (log index 1519700826) and
  1.37.1 (2924600063, integrated 2026-09-23T19:08:57Z, inside the certificate's 19:08:57–19:18:57),
  each with a signed entry timestamp and an inclusion proof whose checkpoint origin is
  `rekor.sigstore.dev - 1193050959916656506`; the log ID is the trusted root's Rekor v1 key
  (`c0d23d6a…801d`). The search by content answers 5,263 bytes for one entry, `[]` for a valid
  signature it never logged, and 400 for an entry whose signature does not fit its digest.
- The certificates carry one signed certificate timestamp (the CT log `ctfe.sigstore.dev/2022`).
- sigstore-go v1.3.0 (2026-07-30, the newest; Apache-2.0) verifies a message signature with a
  certificate and a Rekor v1 entry built into a bundle; it checks the entry's digest against the
  bundle's (`VerifyTlogEntry`). Its trusted root comes from TUF at run time (`root.FetchTrustedRoot`)
  or from a file or bytes (`NewTrustedRootFromJSON`); `pkg/root` links its TUF client either way.
- `TestLiveRelease` (`COGPIT_SIGSTORE_LIVE=1`, never in CI) verified the current stable release,
  1.37.1 for darwin/arm64, against `rekor.sigstore.dev` on 2026-10-15; with
  `COGPIT_SIGSTORE_LIVE=download` it downloaded and installed it through `Downloader.Download`.

Not verified: when, or whether, the Kubernetes release moves its signatures to Rekor v2 or
publishes bundles (H-35's remaining case); whether govulncheck's module-level advisory below was
reported on the tree before this change (inferred: the module's version did not change).

**Measured (2026-10-15, darwin/arm64, Go 1.27.2, `make build`'s flags).**

| What | Before | After |
|---|---|---|
| `go.mod` | 175 lines | 219 lines: 4 direct requirements (sigstore-go v1.3.0 and protobuf-specs v0.5.1 for `internal/kubectl`; rekor v1.5.3 and sigstore v1.10.8 for the test instance `internal/testkit/sigstoretest`) and 40 indirect; no requirement's version changed |
| `go.sum` | 505 lines | 740 lines |
| Module graph (`go list -m all`; `go mod graph`) | 333 modules; 1,906 edges | 560 modules; 3,195 edges (most of them never built: sigstore's KMS providers and servers) |
| Modules linked (`go version -m`) | `cogpit-cli` 130, `cogpit` 151 | 173 and 194: the same 43 more in both, among them sigstore-go, rekor, rekor-tiles, sigstore, timestamp-authority, protobuf-specs, certificate-transparency-go, go-tuf, in-toto, go-openapi (9), gRPC, OpenTelemetry (5); no cloud SDK |
| Packages in `cogpit-cli` (`go list -deps`) | 1,259 | 1,502 |
| `cogpit` | 138.2 MiB | 148.2 MiB (+10.0 MiB, +7.2 %) |
| `cogpit-cli` | 94.5 MiB | 103.1 MiB (+8.5 MiB, +9.0 %) |

Licences of the 43 modules: 33 Apache-2.0, 6 MIT, 4 BSD (read from each module's licence file); all
compatible with Cogpit's Apache-2.0. govulncheck v1.9.0 on `./...`: no vulnerability in the code
Cogpit calls; one module-level advisory, GO-2026-5932 (`golang.org/x/crypto/openpgp`, no fix), in
`golang.org/x/crypto` v0.58.0, which `go.mod` required at that version before.

**The lead's decisions in the night run, for the owner to confirm.**

1. **Where the entry comes from.** `dl.k8s.io` has no bundle (verified). Options: (a) search by the
   entry's content, `POST /api/v1/log/entries/retrieve`, one request that names exactly this
   signature; (b) search by the digest (`/api/v1/index/retrieve`), then one request per UUID: more
   requests, and anyone can log entries under kubectl's digest. **Chosen: (a)**, as cosign does.
2. **Where the trusted root comes from.** Options: (a) embedded, refreshed by `make sigstore-root`
   through TUF; (b) TUF at run time (`tuf-repo-cdn.sigstore.dev`, a cache folder, always current).
   **Chosen: (a)**: one new host instead of two, offline-reviewable diffs, nothing written on the
   user's disk; the cost is that a key rotation by sigstore stops new releases' downloads until a
   Cogpit release carries the new root (fails closed, H-35).
3. **No fallback.** When the log cannot be asked, the download is refused; never the old check
   instead (the lead's default). Options: refuse; fall back to the standard-library check with a
   warning; a preference that turns the log off. **Chosen: refuse.** Consequence: a mirror inside a
   network without access to `rekor.sigstore.dev` no longer downloads through Cogpit; the user puts a
   kubectl on `PATH` or sets the path to kubectl ([usage](usage.md)).
4. **A signed certificate timestamp is required** (one, from a CT log the trusted root names), as
   `cosign verify-blob` requires by default. Options: require; leave it out. **Chosen: require**:
   it narrows H-35's misissued certificate; both real releases carry one.
5. **The inclusion proof is required** besides the signed entry timestamp (bundle v0.3 needs it), so
   a log answer with the timestamp alone is refused. Chosen over accepting the timestamp alone, which
   sigstore-go would allow.
6. **The offline check comes first** (step 2), so a `.sig` that does not fit the published digest is
   refused with that reason, before any request to the log, rather than with the log's 400.
7. **The log's URL is a constant** (`DefaultRekor`, `https://rekor.sigstore.dev`), not a
   preference; tests set `Downloader.Rekor`. A preference would serve a private sigstore, which the
   embedded trusted root does not know anyway.
8. **Bounds and error classes**: 30 s, 64 KiB, four entries; the log's own reason at most 200 bytes
   and sigstore-go's at most 300, each on one line (a proof's hashes otherwise span lines in the
   terminal's line); 429, 5xx and no answer are "could not be asked", any other refusal a signature
   that does not verify.
9. **Renovate**: sigstore-go and its modules update as every Go module does (minor and patch merge on
   their own once CI passes); CI's offline tests with two real releases' files and log entries
   (`TestRealReleaseSignature`, `TestRealReleaseRefused`) are the gate. The option: never merge on
   its own, as Sparkle; not chosen, since the refusals are tested as well as the pass.
10. **Not stopped for the size.** The run's brief was to stop if sigstore-go needed "a very large
    dependency tree". The 43 modules and 8.5–10.0 MiB above are sigstore-go's own verifier's
    (`pkg/verify`, `pkg/tlog`, `pkg/bundle`, `pkg/root`), no cloud SDK or KMS client among them; the
    owner decided for sigstore-go knowing it is large (H-35 said so). Undoing it is one commit.
11. **The test instance** [`internal/testkit/sigstoretest`](../../internal/testkit/sigstoretest/sigstoretest.go):
    a Fulcio-shaped authority and a Rekor-v1-shaped log of the test's own, its search answered as
    Rekor answers it; without a CT log, so the tests that use it ask for no signed certificate
    timestamp (`NewVerifier(…, 0)`), which the real releases' tests cover. The live test is opt-in
    (`COGPIT_SIGSTORE_LIVE=1`, or `download` for the whole download).

## D-51: The navigator's and the command palette's state

**Decided (owner, 2026-10-14 evening, A51, A54); built 2026-10-15.** "The navigator's search text and
keyboard selection become engine state (UI state, visible and settable for agents)"; "the command
palette's typed text becomes engine state, as A51". This replaces Cogpit's default of 2026-10-13,
under which the navigator's search, keyboard row and open groups were the window's
([architecture §4.3](architecture.md#43-ui-state)), and extends `UIState.Palette`, which said only
whether the palette is open.

**The options (#27, #34):**

| Option | For | Against |
|---|---|---|
| (a) The window's, as built | They change which views are listed, not what a view shows; an agent lists the views with `kinds_list` | An agent cannot see or set what the user typed or where the keyboard stands |
| **(b) Engine state, like the view's search (decided)** | Golden rule 2 ("UI state is engine state") without an exception; an agent sees and drives what the user sees | More state to keep in step with the window |

**As built** ([`engine/navigator.go`](../../internal/engine/navigator.go),
[`engine/palettetext.go`](../../internal/engine/palettetext.go),
[`fyneui/navigatorstate.go`](../../internal/ui/fyneui/navigatorstate.go),
[`fyneui/palette.go`](../../internal/ui/fyneui/palette.go); `TestNavigatorStateFollowsTheScopeAndTheView`,
`TestNavigatorAndPaletteStateCommands`, `TestNavigatorReportsAndShowsItsState`,
`TestCommandPaletteAgentTextNeverRunsAChange`): `UIState.Navigator` holds the search, the keyboard's
row and the collapsed source groups (A63); `UIState.PaletteText` the palette's text and
`PaletteTextOrigin` who typed it last. `ui.setNavigatorSearch`, `ui.setNavigatorRow`,
`ui.setNavigatorGroup` and `ui.palette.setText` set them, UI commands that agents run freely
(D-2). The window reports every change a pause after the last (150 ms, as the view's search) and
shows an agent's value unless a report of its own is still on its way, so the user's typing is
never overwritten by an older echo; an agent's value never takes the keyboard focus.

**What the build settled (the lead's night decisions, 2026-10-14, for the owner to confirm; on
2026-10-15 evening the owner confirmed the palette's ↵ below, and the fields, the row's form and the
bounds were not asked, being technical, so they stand as built):**

- **The fields:** `navigator` (`search`, `row`, `collapsed`), `paletteText`, `paletteTextOrigin`.
- **The keyboard's row held as a row of the level shown, not as a view key alone.** Options (a) a
  view key only; (b) the level's own rows: `root`, at a cluster a view key, `group:<path>` or
  `alias:<kind key>`, at the section `context:<context id>` or `source:<label>`; chosen (b),
  because the keyboard stands on groups, the root and the section's clusters as often as on
  views, and (a) could not say where. The engine checks the form against the scope; whether a row
  exists only the window knows (the tree is built from `kinds.list`), so a row it does not show
  leaves its keyboard where it is, a row in a closed group opens the group, a cluster in a
  collapsed source group puts the keyboard on the group's heading. The engine keeps the row in
  step as the window moves its keyboard: a change of level puts it on the level's root (into a
  cluster) or on the cluster it came from (up to the section), a view opened at a cluster on the
  view's row.
- **The bounds:** the search and the palette's text one line of at most 1,000 bytes (the view's
  search's bound); a row at most a context id's 4,096 bytes; at most 1,000 collapsed groups.
- **One per window, not per tab.** Options (a) per tab, as a tab's search; (b) one per window;
  chosen (b), because the navigator is one per window and its scope already decides which tabs
  show; with [D-52](#d-52-several-windows) each window would hold its own.
- **What `workspace.json` keeps** ([D-41](#d-41-what-survives-a-restart)): options (a) nothing of
  it; (b) the collapsed groups (A63 asks for them); (c) the search and the row too; chosen (b),
  because the navigator starts at the section, where a search or a row of the last run means little.
- **An agent's palette text and the user's ↵ (A54 against D-2).** An agent must not choose the entry
  the user's next ↵ runs: the text decides which entry is highlighted first. Options (a) agents may
  not set the text (A54 says they may); (b) an agent may set it, and ↵ on a text another origin
  typed last runs only a read or a UI entry, a change (a mutating or destructive entry) needing the
  user's arrow key or click first; (c) as (b) for every entry, ↵ running nothing on an agent's text;
  chosen (b), because it keeps A54 and the reason for it (the agent shows the user what to run),
  and a read or a switch of view run on ↵ changes nothing the user did not see; (c) would make an
  agent's "Switch to Pods" a two-step chore. The hint line under the list says so, in the warning
  colour after a ↵ it refused; typing makes the text the user's; the window's own report never
  opens a palette the user closed meanwhile. A palette an agent opens does not take the keyboard
  focus either (since 2026-10-13). **Confirmed (owner, 2026-10-15 evening, A54):** (b) as built;
  weighed: (c) ↵ runs nothing on an agent's text; (a) agents set no text.

The navigator's other answers of the same round, details of the window
([parity](../parity/README.md)):

- **Decided (A52), built 2026-10-15:** the scope dropdown lists the hotbar's clusters and the
  connected one only, then All Clusters… (⌘P, the picker) for every other context; capping it
  (#86) is moot. **Which bank** (the lead's night decision for the owner to confirm): options (a)
  the shown bank's tiles in their order; (b) every bank's; (c) every bank's under its name; chosen
  (a), because the dropdown then lists what the hotbar beside it shows, and every other cluster is
  one ⌘P away. A tile whose context no kubeconfig source offers any more is left out
  ([`navigator_header.go`](../../internal/ui/fyneui/navigator_header.go) `scopeClusters`;
  `TestNavigatorSectionDropdown`). **The bank moot (owner, 2026-10-15 evening):** in an interview
  that evening the owner decided that the scope dropdown goes (one list of clusters, the navigator's
  section); built 2026-10-15 ([D-57](#d-57-choosing-a-cluster)): the dropdown, its bank and its All
  Clusters… are gone, `scopeClusters` and that test with them.
- **Decided (A53), built 2026-10-15:** the focus ring shows on keyboard navigation only, not after a
  mouse click: a click moves the keyboard's row without the ring, the next key (or Tab into the
  navigator) shows it (`TestNavigatorRingFollowsTheKeyboardOnly`).
- **Confirmed (A62, as built):** every connect drills into the cluster, a click on another
  cluster's tab and the connected cluster's hotbar tile included (architecture §4.3 called these
  rules Cogpit's defaults).
- **Decided (A63), built 2026-10-15:** the section's source groups collapse (a click on the heading,
  ↵ or Space toggle it, ← collapses an open one, → opens a collapsed one), kept across restarts
  ([D-41](#d-41-what-survives-a-restart)); a collapsed heading counts the clusters it hides; a search
  shows every group open, as the tree's groups (Cogpit's default)
  (`TestNavigatorSectionGroupsCollapse`, `TestWorkspaceKeepsTheCollapsedSourceGroups`).

## D-52: Several windows

**Open (the owner's requirement, 2026-10-14 evening, R3); designed 2026-10-15 in the night run;
postponed (owner, 2026-10-15).** Asked the questions below in their order, the owner answered the
first, the model: **(a), one session with a window's state and a connection per window**, as the
direction. He put the rest back ("not that important now"): he answers it in a separate, larger
interview once the rest is done. Nothing is built; the questions 2 to 7 below stay open.
"Dragging a hotbar tile out of the window and dropping it opens another Cogpit window with that
cluster." With it the owner asked that hotbar tiles can be reordered by drag and drop (R2, a detail
of the window, [parity](../parity/README.md)). R2 is built (2026-10-15, `hotbardrag.go`): a tile
dropped on the bar's tiles moves through `hotbar.moveCluster`, and a drop anywhere else does
nothing, the seam where R3's drop out of the window stays with this decision. Lens's Window menu has several windows, missing in
Cogpit ([ui-parity](../parity/ui-parity.md), "Several windows"); dragging a tile out of the window
is beyond Lens ([beyond Lens](../parity/beyond-lens.md)).

### What the requirement touches

Cogpit is one window today. Each point was checked against the code on 2026-10-15.

- **UI state is engine state** (golden rule 2): a session holds one `UIState` (`Session.ui`,
  [`engine.go`](../../internal/engine/engine.go); [architecture §4.3](architecture.md#43-ui-state)),
  and `UpdateUI` emits it whole in `EventUIState`. It mixes a window's state (the context and its
  connection, the view with its filters, sort, selection and drawer, the tabs, the dock, the
  navigator and its scope, the palette, the chat column, the open settings page, the welcome page)
  with the app's (the hotbar, the preferences, the cluster settings, the notification log, the
  tables' hidden columns and column order, the time format). The window keeps a copy of all of it
  (`shell.ui`, [`shell.go`](../../internal/ui/fyneui/shell.go)).
- **One connection, by construction** (as designed against on 2026-10-15; since 2026-10-16 the
  engine holds a pool of backends by context with one shown per session,
  [D-69](#d-69-several-clusters-connected-at-once) stage 1, which is this option's first stage).
  The engine held one backend (`state.backend`,
  [`views.go`](../../internal/engine/views.go)). `Session.Connect` releases the previous backend
  before it connects the next, "so that UIState.Context always names the cluster the engine acts
  on". `Session.Backend()` hands that one backend to 67 call sites in 43 files (52 in 33 engine
  files, 12 in 9 command files, 3 in `cogpit-cli`; counted without tests). Also single: `UIState.Context`
  and `Connection`; the end of a job when "the connected cluster changes" (`followJobs`,
  [`jobs.go`](../../internal/engine/jobs.go)); the diagnostics' counters; and the Secret values
  replaced in what agents read (`AgentView` from `s.Backend().secretMatcher()`,
  [`agentview.go`](../../internal/engine/agentview.go)). Already per context: the cache (a
  `cache.New` per connect, [`connector.go`](../../internal/engine/connector.go)), the access
  reviews, the allowance of agent writes (`Session.agentWrites`) and the plugin approvals. So
  [architecture §1.1](architecture.md#11-rules-adopted-from-day-one)'s rule ("one connection and
  one cache per kubeconfig context; nothing global assumes one active cluster") holds for the cache
  and not for the engine around it. The tab bar's connect on select, the navigator's scope that
  follows the connection (A62) and [D-47](#d-47-deep-links-cogpit)'s switch question rest on the one
  backend. G5 lists it as a difference from Lens, which keeps several clusters connected
  ([gaps › G5](../parity/gaps.md#g5-app-shell-navigation-tables-and-drawer)).
- **Agents** ([D-2](#d-2-agent-interface-scope)): every command is an MCP tool, 190 in all
  ([commands](commands.md)); 34 are `ui.*`, and 89 command definitions in
  [`internal/engine/commands`](../../internal/engine/commands) declare `DangerUI`. None names a
  window: they act on the one `UIState`. The built-in agent gets a compact `ui_state` with every
  message ([D-10](#d-10-built-in-agent)) and dispatches in-process. The MCP server is one per
  process, its endpoint and token in one file each (`mcp-endpoint`, `mcp-token`,
  [`mcp/server.go`](../../internal/mcp/server.go)).
- **The window:** `fyneui.Run` makes one `fyne.App`, one window and one `shell` for it
  ([`fyneui.go`](../../internal/ui/fyneui/fyneui.go)). The engine takes one confirmer and one dock
  reader (`SetConfirmer`, `SetDockOutputReader`), and `aboutShell` assumes "the app has one window"
  ([`appmenu_darwin.go`](../../internal/ui/fyneui/appmenu_darwin.go)). The shell itself is already
  per window (it holds its `fyne.Window`), the title bar's bridge already keeps several
  (`titlebars`, [`titlebar_darwin.go`](../../internal/ui/fyneui/titlebar_darwin.go)), and the perf
  mode opens a second window in the same app ([`perf.go`](../../internal/ui/fyneui/perf.go)).
- **The hotbar's tiles** (`hotbarTile`, [`hotbar.go`](../../internal/ui/fyneui/hotbar.go)): a tap,
  a secondary tap and hover; when this was designed no drag (Move Up and Move Down stood in for
  Lens's drag to reorder); R2's drag within the bar was built the same night
  ([`hotbardrag.go`](../../internal/ui/fyneui/hotbardrag.go)).
  R2 and R3 are one gesture: a tile dropped inside the hotbar moves (`hotbar.moveCluster` exists),
  dropped outside the window opens a window. Cogpit has drags of its own already: a column header
  ([`columnorder.go`](../../internal/ui/fyneui/columnorder.go)), the panels' edges
  ([`panelwidths.go`](../../internal/ui/fyneui/panelwidths.go)).
- **The window list and what survives a restart:** [D-41](#d-41-what-survives-a-restart)'s
  `workspace.json` keeps one tab bar (`workspaceDoc`: the tabs, the tab shown last, the columns'
  choice and order, the collapsed source groups; [`persist.go`](../../internal/engine/persist.go)).
- **Fyne's start of a window:** the spike measured a stall of 0.42–0.56 s while the window opens,
  0.31 s of it Fyne's GL set-up (`gl.painter.Init`), and 151–193 ms more for the first fill of a
  10,000-row table (S1, passed with a reservation, [spike › results](spike-fyne.md#results)); G15
  says to avoid second windows ([gaps](../parity/gaps.md#g15-performance-and-api-load-at-scale)).
  Fyne 2.8.1 runs every window of a process in one loop on the main goroutine (`runGL`,
  `drawSingleFrame` over `windowList`, `internal/driver/glfw/loop.go`) and sets up a new window's GL
  there (`window.create`, `Painter().Init()`). So a window that opens in the same process stalls
  every other window of it too. A second window was not measured.

### What Fyne 2.8.1 offers for the drag out

Read from Fyne's and GLFW's source in the module cache (`fyne.io/fyne/v2@v2.8.1`,
`github.com/go-gl/glfw/v3.4/glfw@v0.1.0-pre.2`); not tried in a real window.

- **The events.** `fyne.Draggable` is `Dragged(*DragEvent)` and `DragEnd()`. The end carries no
  position, so the tile has to keep the last `Dragged` position itself. A drag starts after 2 points
  of movement (`dragMoveThreshold`).
- **Positions beyond the window come through.** `processMouseMoved` gives `Dragged` the cursor's
  position as GLFW reports it, unclamped, and the driver has no cursor-leave handling that would
  end a drag (`internal/driver/glfw/window.go`). GLFW keeps reporting the cursor while the button is
  held outside the window: on macOS `mouseDragged:` calls `mouseMoved:`, which reports
  `locationInWindow` (`cocoa_window.m`), and AppKit sends the dragged events to the window the press
  began in (AppKit's behaviour as the lead knows it, not checked here and not seen in a running
  Cogpit); on Windows GLFW captures the mouse on the first button down (`SetCapture`,
  `win32_window.c`); on X11 the server's implicit grab does the same (not checked in the source).
- **So "dropped outside" can be decided:** at `DragEnd` the last position lies outside the canvas
  (below 0, or beyond its width or height). That is a workaround, not an operating-system drag: no
  picture of the tile follows the pointer outside the window (Fyne draws inside its windows only),
  and nothing is a drop target outside, neither another app nor another Cogpit window.
- **Where the new window goes.** Fyne 2.8 has `desktop.Window.RequestPosition(x, y int)`, which
  "may be ignored (for example Linux Wayland)" and has "no contract of what these values represent"
  (`driver/desktop/window.go`), and no getter for a window's position on the screen. The pointer's
  screen position at the drop is therefore known on macOS only, through AppKit (`[NSEvent
  mouseLocation]`) over cgo, the way Cogpit's title bar reaches its `NSWindow`
  ([`titlebar_darwin.go`](../../internal/ui/fyneui/titlebar_darwin.go)). Elsewhere the system
  places the window.
- **The native alternative:** an AppKit drag session (`NSDraggingSession`) with the tile's picture,
  ended at a screen point; macOS only, through cgo, outside Fyne's events.
- **macOS quirks.** A window in full screen, or one that covers its screen, has no outside on that
  screen. The hotbar is 64 points wide at the window's left edge (`sizeHotbarWidth`,
  [`theme.go`](../../internal/ui/fyneui/theme.go)), so a short drag to the left already leaves the
  window: a drop needs a margin beyond the edge. Cogpit's own title bar starts AppKit's window drag
  on the top bar (`performWindowDragWithEvent`, [`titlebar_darwin.m`](../../internal/ui/fyneui/titlebar_darwin.m)),
  which the hotbar is not part of.
- **Several windows in one app.** Each window has its own main menu; on macOS Fyne rebuilds the
  menu bar from the focused window's (`processFocused`, `setupNativeMenu`). The app quits when its
  last window closes (`loop.go`); a master window would quit it on its own close, and Cogpit sets
  none (`SetMaster` is not called).

### The options for the model

Three options. Sizes are estimates in agent-hours with PhotoCraft's calibration (8–20 h per
subsystem, [parity rules](../parity/README.md#rules-for-every-progress-page)), not measured.

**(a) One process, one session, several windows; a window's state and a connection per window.**

- **The model.** `UIState` splits in two. The app's part, one per session: the hotbar, the
  preferences, the login item, the cluster settings, the time format, the notification log, the
  tables' hidden columns and column order, the drawer tables' order. A window's part, one per
  window: the context and its connection, the view, namespaces, groups, search and its toggles,
  status filter, sort, selection, drawer, the tabs, the dock, the navigator and its scope, the
  palette, the chat column, the open settings page, the welcome page, the Overview's range. The
  engine's view state (`state` in `views.go`: the view worker, the pages' workers, the drawer's
  watch) becomes one per window. The backends move into a pool keyed by context that counts the
  windows using each: two windows on one cluster share one backend and its cache, and a backend no
  window uses closes as `Disconnect` closes it today. §1.1's rule then holds for the engine too.
  A connect supersedes only its own window's. A window has an id (`w1`, `w2`), as tabs have
  (`tab-1`).
- **What changes.** `internal/engine`: the split, the pool, connect and disconnect per window, the
  67 `Backend()` calls through the command's window (carried in its context as `withInvocation`
  carries the origin, [`readslikeuser.go`](../../internal/engine/readslikeuser.go)), events that
  name their window, a confirmer and a dock reader per window. `internal/engine/commands`:
  `window.open {context?}`, `window.close`, `window.focus` (UI), and an optional `window` on every
  command that reads or sets a window's state. `internal/mcp`: nothing of its own; the tools follow
  the registry. `internal/ui/fyneui`: a shell per window from one `Run`, About for the focused
  window, the tile's drag. `workspace.json` gets a format with windows. Estimated: the engine
  20–40 h, the commands and MCP 8–15 h, the window side with the drag 10–20 h, the workspace 3–6 h;
  40–80 h in all, tests included.
- **What agents see.** One MCP server and one audit log for every window. `ui_state {window?}`
  returns that window's state with the app's part, and `windows`: each window's id, context,
  connection state, title and whether it has the focus. Every window-scoped tool takes an optional
  `window`; without it a call addresses the window the user focused last, so an agent that knows
  nothing of windows works as today, on what the user looks at. The built-in agent's calls address
  the window its chat column is in. An audit line names the window and the context.
- **What survives a restart (D-41).** `workspace.json` keeps each window's tabs, the tab it showed
  last and its frame. They come back without connecting anything, each window on its welcome page
  with its tab marked, as the one window does today. The columns' choice and order stay one set for
  every window (D-41's choice (a)).
- **Crash stance (D-11).** One process: a crash takes every window down, as it takes the one window
  today. Cluster operations stay crash-safe through their journals, unchanged; the next start
  brings the windows back.
- **Performance.** One main goroutine for all windows. Opening a window stalls every open window
  for its GL set-up (0.31 s of the 0.42–0.56 s measured for the first window), and its first
  10,000-row fill (151–193 ms) comes again. Two windows on one cluster cost no extra watch (the
  backend is shared); a window on another cluster costs that cluster's cache, as a switch does
  today, without releasing the first. Memory per window not measured.
- **Security.** The privilege footprint per connection is unchanged: each connection is one
  context's, with the same verbs and the same confirmations naming objects, namespace and context;
  Lens keeps several clusters connected too ([architecture §4.3](architecture.md#43-ui-state)).
  New, per case:
  (1) an agent's write whose parameters name no context goes to the addressed window's cluster; if
  the user moves the focus between an agent's read and its write, the default window changes. A
  write the user confirms names its context, and one confirmed for a context refuses to run on
  another (`CheckConfirmedContext`, [`confirm.go`](../../internal/engine/confirm.go)).
  (2) An agent may drive a window the user does not look at (behind another, minimised). UI
  commands write nothing to a cluster, and a destructive action the user later starts there is
  confirmed with its objects named.
  (3) A window left open keeps its cluster connected and watched with the user's identity, as the
  one connection does today; `ui_state`'s `windows` and the Window menu name them.
  (4) The Secret values replaced in what agents read become the union of every connected backend's
  (today the one connected cluster's): a log line of one cluster that prints a Secret value of
  another connected cluster is then replaced too.
- **The drag out.** The tile keeps its last `Dragged` position. At `DragEnd`, outside the window by
  more than a margin it dispatches `window.open {context}`; over the hotbar it dispatches
  `hotbar.moveCluster` (R2); anywhere else nothing. On macOS the window opens at the pointer; on
  the other systems where they place a new window.

**(b) A second process per window.**

- **The model.** The dropped tile starts Cogpit again with `--context <id>` (the flag exists:
  "connect to this context at start", [`main.go`](../../cmd/cogpit/main.go)); on macOS through
  `open -n` on the bundle or the binary itself (`os.Executable`), how LaunchServices treats either
  not verified. Each process has its own session, engine, caches, MCP server and window. The
  engine does not change.
- **What breaks.** `cmd/cogpit` has no single-instance guard (none found), and the processes share
  Cogpit's configuration directory:

  | File (in `<user config dir>/Cogpit`) | Shared today as | With a window per process |
  |---|---|---|
  | `workspace.json` | a copy per process, the last write wins (D-41) | the second process restores the first's tabs as well; the window closed last decides what the next start shows |
  | `hotbar.json` | a copy per process, the file not watched, the last change wins ([`hotbar.go`](../../internal/engine/hotbar.go)) | a tile moved (R2) or added in one window does not show in the other, whose next change overwrites it: the hotbar the gesture starts from diverges |
  | `preferences.json`, `cluster-settings.json` | as the hotbar ([`preferences.go`](../../internal/engine/preferences.go), [`clustersettings.go`](../../internal/engine/clustersettings.go)) | a theme or a cluster's name changed in one window only |
  | `kubeconfigs.json`, `exec-approvals.json` | approvals read from the file at every check (`loadApprovals`); the files not locked, "two Cogpit processes changing the files at once may lose one change" ([`kubeconfigs.go`](../../internal/engine/kubeconfigs.go)) | checks agree; a source or an approval written by both at once can be lost, and a lost approval asks again |
  | the drain, Helm and node shell journals, the drafts | under the journals' lock, an owner per process (H-22, [`journallock.go`](../../internal/engine/journallock.go)) | safe as built |
  | `audit.log` | appended by every process, read back from the file ([`auditlist.go`](../../internal/engine/auditlist.go)) | one log for both windows |
  | `crash.log` | one name ([`crash.go`](../../cmd/cogpit/crash.go)) | the second start removes the first's empty report file, which the first keeps writing into, and the first's clean exit removes the second's: a crash of either then leaves no report |
  | `mcp-token`, `mcp-port`, `mcp-endpoint` | one name each; the token and the port kept for every start (since 2026-10-15, [D-53](#d-53-the-mcp-details-and-a-token-kept-until-the-user-regenerates-it)), the endpoint removed only by the process whose URL it holds (`removeIfHolds`, [`mcp/token.go`](../../internal/mcp/token.go)) | the second start serves the same token, finds the kept port taken, listens on another, keeps that one and overwrites the endpoint: `cogpit-cli mcp` forwards to the window started last; an HTTP client registered with the first's port reaches the first only |
  | `mcp-switch` (D-49) | the status bar's switch stops its own process's listener (`Server.Stop`) | switching MCP off in one window leaves the other window's server on |
  | `llm-connections.json`, `chat-conversations.json` (the built-in agent's) | saved whole from memory under an in-process lock ([`agent/store.go`](../../internal/agent/store.go)) | the window that saves last replaces the other's conversations |

  Not checked: which process macOS hands a `cogpit://` link to; Sparkle in two processes of one
  app; the kubectl download folder used by two.
- **What changes to make it sound.** The spawn and the drag: 5–10 h. Then the table: a workspace
  per process (or the first process's only), file watches for the hotbar, the preferences and the
  cluster settings, a crash file per process, an MCP endpoint per process with the bridge choosing,
  the switch across processes, a lock for the conversations: 15–30 h. Estimated 20–40 h, the link
  and updater questions still open.
- **What agents see.** Two MCP servers, one window each; `ui_state` needs no window name, but no
  agent sees both windows through one server. The bridge reaches one of them; the token is kept
  across starts and the port while it is free (D-53). Each window's chat is its own.
- **What survives a restart.** Only the tabs of the process that writes last; the second window
  does not come back unless the processes agree on a window list (not designed).
- **Crash stance.** Its strong point: a crash of one window leaves the other running. The crash
  reports are lost (above).
- **Performance.** The new window's GL set-up stalls only the new process; the open window keeps
  drawing. Each process carries its own Go runtime, Fyne, engine and caches (not measured). Two
  windows on the same cluster watch everything twice, against §1.1's "one watch per (cluster, kind,
  namespace scope), shared by every consumer".
- **Security.** The privilege footprint per connection is unchanged. New: the MCP switch and the
  endpoint split per process (the table); an agent registered with one process does not see the
  other window, and the user's off switch in one window does not reach the other's server.
- **The drag out.** As (a); the drop starts a process instead of `window.open`. Placing the window
  at the pointer needs a new flag.

**(c) One window; the tile dropped outside opens the cluster in a new tab.** Not the requirement;
the cheap alternative. `ui.tab.open` opens a view of the connected cluster only, so it needs a
`context`; and since Cogpit holds one connection, showing that tab switches the window's cluster,
which a click on the tile already does. Estimated 3–6 h (the drag and the parameter). Agents,
restart, crash stance, performance and security: unchanged.

| | (a) one session, a window's state | (b) a process per window | (c) a tab |
|---|---|---|---|
| Meets R3 | yes | yes | no |
| Estimated size | 40–80 h | 20–40 h | 3–6 h |
| Hotbar, stores, MCP server, audit log | one for all windows | one per process, the files shared (the table) | one |
| Agents | one server, `window` optional, the focused window by default | one server per window | unchanged |
| A window's opening stall | in every window | in the new one only | none |
| A crash | ends every window | ends one window | ends the window |
| Same cluster twice | one cache | every watch twice | – |

### The lead's recommendation

**(a), in four stages.** It is the only option in which what the gesture starts from stays one:
one hotbar, one set of stores, one MCP server and one audit log for every window. (b) splits each
of them per process (in the table the workspace, the hotbar, the preferences, the MCP endpoint and
its switch and the conversations diverge, and crash reports get lost), and an agent would see two
Cogpits; (c) is not the requirement. The cost of (a), the engine's single backend behind 67 call
sites, is the assumption [architecture §1.1](architecture.md#11-rules-adopted-from-day-one) already
rules out; removing it also prepares Lens's several connected clusters in one window (G5's
difference), as a later decision of its own. Its performance cost, a window's opening stalling
every window once, is the kind of once-per-window stall that S1's reservation accepted.

1. **The engine with one window:** the pool of backends by context and the window's state; no
   visible change, the existing tests the gate, a test per command with two windows on two
   clusters added.
2. **Several windows:** `window.open`, `window.close`, `window.focus`, `ui_state`'s `windows`, a
   shell per window, a Window menu that lists them, and "Open in New Window" in the tile's menu.
3. **The drag:** R2's reorder and R3's drop out of the window, one gesture.
4. **`workspace.json` with windows.**

### Questions for the owner, in the order to ask them

One decision each; a later question assumes the recommended answer to the earlier ones.

1. **The model.** (a) one session, a window's state and a connection per window; (b) a process
   per window; (c) a tab instead. Recommended (a), for the reason above.
2. **A connection per window or per tab.** (i) Per window: a window shows one cluster at a time,
   and a tab of another cluster in the same window switches it, as today; (ii) per tab: every tab's
   cluster stays connected, in any window, as in Lens. Recommended (i): A62's "every connect drills
   in", the navigator's scope and D-47's question then stay true per window unchanged; (ii) is Lens
   parity of its own (G5) and can follow on the same pool.
3. **Which window an agent's call addresses.** (i) An optional `window`, by default the window the
   user focused last; (ii) `window` required while more than one is open; (iii) the first window
   unless named. Recommended (i): an agent that knows nothing of windows keeps working on what the
   user looks at, which is what "this pod" means (D-10); (ii) breaks every agent that works with one
   window today; (iii) drives a window the user may not look at.
4. **Whether agents open and close windows.** (i) Yes, as UI commands, as they run
   `cluster.connect` and `dock.close` today (D-2), the opened window in front, the last window never
   closed by an agent; (ii) open yes, close by the user only; (iii) neither, the user only.
   Recommended (i): D-2's "UI operations run without asking" holds, an opened window is in plain
   view, and closing a window does no more than closing its dock tabs, which agents may do now;
   refusing the last window keeps the app and its MCP server running.
5. **What survives a restart.** (i) Every window with its tabs, the tab it showed last and its
   frame, none connected; (ii) one window, every window's tabs merged into its bar; (iii) the first
   window only. Recommended (i): D-41's reason (what the user arranged comes back without a
   request) holds per window. Its cost: every restored window's GL set-up at start, one after the
   other on the main goroutine (0.42–0.56 s measured for one window; several not measured).
6. **How a tile is dropped out.** (i) Beyond the window's edge by a margin (24 points proposed,
   Cogpit's default), the window at the pointer on macOS and where the system puts it elsewhere,
   with "Open in New Window" in the tile's menu for full screen and the keyboard; (ii) a native
   macOS drag session through AppKit, the tile's picture following the pointer (macOS only,
   outside Fyne's events); (iii) the menu item only. Recommended (i): it needs nothing beyond Fyne's
   events and one cgo call of the kind the title bar already makes; (ii) feels better on macOS and
   can follow if (i) feels bare in a real window.
7. **A tile of a cluster another window shows.** (i) A new window all the same, sharing the
   connection; (ii) the window that shows it comes to the front. Recommended (i): the gesture asks
   for a window, a second window on one cluster costs no watch under (a), and two namespaces of one
   cluster side by side is a use of its own.

### What it does not cover, and the risks per case

- **Stage 1, the engine, live while it is built.** Moving 67 call sites off the single backend can
  make a command read or act on the wrong window's cluster. `CheckConfirmedContext` bounds a
  confirmed write only, not a read or a UI command; the tests with one window are the gate, the
  tests with two windows on two clusters are new.
- **Stage 2, agents, dormant until built.** An agent's write whose parameters name no context lands
  on the window focused at dispatch, and the user can move the focus between an agent's read and its
  write. Bounded by the confirmation naming the context; where the user allowed agent writes for
  that context the write runs without asking, but only on a context the user allowed.
- **Stage 2, connections the user does not look at, dormant.** A minimised or hidden window keeps
  its cluster connected and watched with the user's identity; `ui_state` and the Window menu name
  them, nothing closes them by itself.
- **Stage 2, performance, dormant.** Every window's opening stalls the others (one loop); a second
  window's numbers are not measured; G15's reservation grows with each window.
- **Stage 3, the drop, dormant.** The detection rests on GLFW and AppKit reporting the pointer
  outside the window during a drag, read from their source and not tried in a real window; Windows
  and Linux not tried (M11); on Wayland Fyne's own note says the position request is ignored.
- **Deep links ([D-47](#d-47-deep-links-cogpit)), dormant.** Which window a link opens in is not
  designed here. A proposal for stage 2: the window that shows the link's cluster, else the focused
  one with D-47's question.
- **Two processes, live today whatever the choice.** Without a single-instance guard, a user who
  starts Cogpit twice (`open -n`, or the binary run again) meets (b)'s table now: lost crash
  reports, the MCP endpoint of the last start, the hotbar and the workspace of the last write
  (read from the code, not tried).
  Under (a) a second start could hand its `--context` to the running Cogpit through the MCP
  endpoint the bridge already reads; a follow-up, not designed.
- **Not measured:** memory per window in either option; a second window's GL set-up; a second
  process's start.

## D-53: The MCP details, and a token kept until the user regenerates it

**Decided (owner, 2026-10-15, finding F6); built 2026-10-15.** The owner's decisions:

1. The top bar's "MCP" button goes.
2. A gear in the status bar's MCP item opens the MCP connection details and settings: the server
   on and off (the switch of [D-23](#d-23-a-link-to-the-owner-in-the-status-bar)), the endpoint
   URL, the token (masked, with copy), the registration commands with copy (what the button copied,
   the stdio variant's text included), and Regenerate token with one click.
3. Regenerate token warns first: regenerating ends every current MCP connection, and the clients
   need the new token to connect again. Then a new token, the port kept, the old token refused at
   once, the files updated.
4. The token and the port stay the same across restarts, kept owner-only beside `mcp-token`; a new
   token only by the click. If the kept port is taken at start, Cogpit picks another port and says
   so (a notice, and the details show it).

**Before (verified in this repository, until 2026-10-15):** every start of the server, at launch
and at every switch on, made a new token and bound a free port; `Server.Stop` and `Close` removed
the token file; the HTTP registration had to be repeated after every restart, and a leaked token
was dead at the next start. The top bar's MCP button and Help › Copy MCP command copied the
registration and showed it in a dialog, the HTTP one with its token in plain text.

**Built** ([`mcp/server.go`](../../internal/mcp/server.go) `Start`, `listen`, `keepPort`,
`RegenerateToken`, `PortTaken`, `Token`; [`mcp/token.go`](../../internal/mcp/token.go) `loadToken`;
[`mcp/endpoint.go`](../../internal/mcp/endpoint.go) `PortFileName`; [`mcp/http.go`](../../internal/mcp/http.go)
the guard's `rotate`; [`mcp/bridge.go`](../../internal/mcp/bridge.go) `followed`;
[`cogpit/mcpcontrol.go`](../../cmd/cogpit/mcpcontrol.go) `Regenerate`;
[`fyneui/mcpdetails.go`](../../internal/ui/fyneui/mcpdetails.go), the gear in
[`fyneui/mcp.go`](../../internal/ui/fyneui/mcp.go)). The token file `mcp-token` keeps the token
across starts and switches; a new port file `mcp-port` beside it keeps the port; only the endpoint
file `mcp-endpoint` comes and goes with the listener, so it still says whether Cogpit serves MCP.
Security perspective: [agents and MCP › H-98](../security/agents-and-mcp.md#h-98-a-leaked-token-stays-valid-until-the-user-regenerates-it).
Tests: `TestTokenKeptAcrossStarts`, `TestTokenFileNotKept`, `TestPortKeptAcrossStarts`,
`TestPortTakenMovesAndKeepsTheNewOne`, `TestGivenPortIsNotKept`, `TestRegenerateToken`,
`TestRegenerateEndsPendingConfirmation`, `TestRegenerateWhileStopped`, `TestBridgeFollowsARegenerate`,
`TestBridgeFollowsARestart`, `TestMCPControlRegenerate`, `TestTopBarHasNoMCPButton`,
`TestMCPGearOpensTheDetails`, `TestMCPDetailsTokenShownAndCopied`, `TestMCPRegenerateWarnsFirst`,
`TestMCPRegenerateWhileOffAndFailing`, `TestMCPPortTakenNotice`; renders
[details dark](../../internal/ui/fyneui/testdata/mcp_details_dark.png),
[light](../../internal/ui/fyneui/testdata/mcp_details_light.png),
[the warning dark](../../internal/ui/fyneui/testdata/mcp_regenerate_dark.png),
[light](../../internal/ui/fyneui/testdata/mcp_regenerate_light.png).

**The lead's decisions for the owner to confirm** (★ the choice):

- *Switching off and on:* ★ keeps the token and the port, so a client registered once connects
  again by itself. Options: a new token at every switch on (the switch as a panic button, but the
  registration breaks at every switch, which F6 wanted gone; Regenerate token is the panic button).
- *Where the port is kept:* ★ a file `mcp-port` beside the token, written like it (0600, atomic,
  no symlink). Options: the endpoint file kept across stops (then it no longer says that Cogpit
  listens, and the bridge would send the token to whatever holds the port while Cogpit is off);
  Fyne's preferences (cogpit-cli has no Fyne).
- *A port that was taken at start:* ★ the new port is kept, so the registration the details show
  then works at the next start too; the notice says so and names both ports. Options: try the old
  port again at the next start (a registration made before works again once the other program is
  gone, but the one the window showed meanwhile breaks).
- *`--mcp-addr` with a port:* ★ binds that port and keeps nothing; a flag binds one run, as
  `--no-mcp` (D-49). The default `127.0.0.1:0` keeps the port.
- *A token file Cogpit cannot keep using:* ★ a file that holds no token Cogpit wrote, or that other
  users may read (any group or other mode bit), is replaced by a new token at the next start,
  logged without the token; a symlink is refused, as for writing.
- *Showing the token:* ★ masked, with Show and Hide as a Secret's value in the drawer; hidden again
  when the panel closes; the HTTP registration masks it the same way. Options: copy only, never
  on screen (safer against screen sharing, but the user cannot compare it with a client's
  configuration).
- *Copying:* ★ the token and the HTTP registration (it carries the token) as credentials: kept from
  clipboard managers on macOS and cleared after 1 minute, as a Secret's value
  ([D-25](#d-25-copying-values)); the endpoint and the stdio registration as plain text. A copy says
  so in a note at the panel's foot, not a toast: the panel covers the corner the toasts show in.
- *Regenerate token:* ★ the window's alone (the lead's default, kept: no engine command, no MCP
  tool, so no agent can cut the other clients off), one audit log line (`gui.mcpToken`,
  `{"regenerated":true}`, never the token), a warning first with its button armed after the
  confirmations' delay, offered while the server is off too (the next start serves the new token).
  The server keeps listening: the guard takes the new token and ends every request the old one
  admitted, a call waiting for a confirmation included (its command does not run), and idle
  connections close. Options: stop and start the server on the same port (a moment with nothing
  listening, which a bridge reads as "Cogpit is not running", and another program could take the
  port in it).
- *The bridge after a regenerate:* it read the files once per connection and again on a refused
  token (401), then sent the call once more: it followed by itself already. ★ It now also reads
  the endpoint and token files before every call and reconnects when they changed, so a regenerate
  costs no refused request, and the token, now valid across starts, never goes to a port Cogpit
  left (H-98). The 401 path stays for the moment between the file and the guard
  (`TestBridgeFollowsARegenerate`).
- *Help › Copy MCP command:* ★ kept; it copies the command (the HTTP one as a credential) and opens
  the details instead of its dialog, which showed the token in plain text.
- *Where the details show:* ★ a panel above the status bar, its right edge at the gear's, as the
  bell's panel hangs below the bell; a click outside closes it. The pill still switches the server
  on a click (D-23), the gear beside it opens the details.

## D-54: The layout toggles and the Update button

**Decided (owner, 2026-10-15, finding F7); built 2026-10-15.** The owner's finding, with a
screenshot of VS Code's title bar (`plan/captures/vscode-titlebar-layout-toggles-2026-10-15.png`,
local): at the top right of Cogpit's top bar, as VS Code has them,

1. three buttons that show and hide the parts of the layout, in VS Code's order: its primary side
   bar, Cogpit's **navigator** (the hotbar at the left edge stays, as VS Code's activity bar does);
   its panel, the **dock**; its secondary side bar, the agent chat's **column**. Each icon shows
   whether its part is open (VS Code draws the part's region filled while it is open);
2. left of them an **Update** button, shown only while an update is available, whose click opens
   Sparkle's update window (as Help › Check for Updates… does).

The lead's defaults that came with it, kept: each state is engine state with a command
(`ui.toggleNavigator` ⌘B and `ui.toggleDock` ⌘J, both free in the registry, and the chat's
existing `chat.toggle` ⌘I), the dock's collapse moves from the window (`dock.expanded`) into
`UIState`, the layout survives a restart in `workspace.json` ([D-41](#d-41-what-survives-a-restart)),
and the View menu has all three with their shortcuts.

**Built (2026-10-15):** [`engine/layout.go`](../../internal/engine/layout.go),
[`commands/layout.go`](../../internal/engine/commands/layout.go),
[`fyneui/layoutbar.go`](../../internal/ui/fyneui/layoutbar.go), the delegate in
[`update/sparkle_darwin.m`](../../internal/update/sparkle_darwin.m)
([architecture §4.3](architecture.md#43-ui-state), [§11](architecture.md#11-updates-internalupdate)).
`UIState.NavigatorHidden` and `UIState.DockOpen` beside `UIState.Chat`, the two commands (UI
commands with an optional `open`, run freely by agents, D-2), `UIState.Update` from the updater;
in the window the three toggles left of the bell, View › Navigator (⌘B) and View › Dock (⌘J)
before Agent Chat (⌘I), checked while their part is open. Tests: `TestSyncDock`,
`TestWorkspaceKeepsTheLayout`, `TestUpdateNotice`, `TestLayoutToggles`,
`TestLayoutTogglesThroughTheEngine`, `TestLayoutMenuItems`, `TestUpdateButtonShowsOnlyWithAnUpdate`,
`TestUpdateNoticeTellsTheListeners`, `TestSparkleReportsAScheduledUpdate`; renders
`internal/ui/fyneui/testdata/layout_*.png`.

**The lead's decisions for the owner to confirm** (★ built):

- *The dock's toggle while the dock has no tabs:* ★ disabled, its tooltip "Dock: no tabs to show",
  `ui.toggleDock` refused with "the dock has no tabs: open a terminal (⌘⇧T), a pod's logs or a
  Create resource tab first"; `UIState.DockOpen` is false without tabs. Options: an empty dock
  that opens with a hint and its "+" menu (a third of the window for nothing to show); a toggle
  that opens a terminal (a shell started as the side effect of a UI command agents run freely).
  Why: the dock is its tabs, as Lens's is, unlike VS Code's panel, which always holds its views.
- *When the dock opens by itself:* ★ when a tab is added (as the window did) and when another tab
  is made active while the one active before is still there (a click on a tab, a pod's logs
  opened again where their tab exists, an agent's `dock.activate`), never when the active tab
  moves because its tab closed; the dock's own arrow and a click on the shown tab of a collapsed
  dock go through `ui.toggleDock`. Options: a new tab only (logs opened again would not show
  while the dock is collapsed). The rule is `UIState.SyncDock`, which `UpdateUI` runs after every
  change, so it holds whoever changed the dock.
- *What `workspace.json` keeps:* ★ whether the navigator is hidden and whether the chat column is
  shown; not the dock's state, since its tabs are not restored and a dock without tabs is closed.
  Options: the navigator only (the chat column would come back closed after every restart, while
  the owner counts it as one of the three toggles).
- *Sparkle's window for an update a scheduled check finds:* ★ never by itself: the delegate of
  Sparkle's standard user driver takes over showing scheduled updates (Sparkle's "gentle
  reminders", `standardUserDriverShouldHandleShowingScheduledUpdate:andInImmediateFocus:` answers
  NO), so only the Update button shows; a check the user starts (the button, Help › Check for
  Updates…, an agent's `app.checkForUpdates`) shows Sparkle's window as before. Options: Sparkle's
  documented example, its window when it would show it "in immediate focus" (shortly after launch,
  or after the computer was idle) and the button otherwise; Sparkle's default, its window always
  (no button for scheduled finds). Why: a window that pops up over a production cluster's terminal
  while the user types is a surprise, and the owner's model is VS Code's button. Cost: a user who
  never looks at the bar's right end updates later.
- *How long the button shows:* ★ from Sparkle's report that it holds an update
  (`standardUserDriverWillHandleShowingUpdate:forUpdate:state:`, for scheduled and user-started
  checks alike) until its update session ends (`standardUserDriverWillFinishUpdateSession`: the
  update installed, skipped, put off with "Remind Me Later", or failed). Not until the user's first
  attention (`standardUserDriverDidReceiveUserAttentionForUpdate:`, Sparkle's hook for attention
  indicators): the button stays while Sparkle's window holds the update, and a click brings it back
  up. After "Remind Me Later" the button goes until Sparkle's next scheduled check finds the update
  again (an hour later at the most since 2026-10-16, D-4), since Sparkle forgets it then; after "Skip This Version" until a newer release.
- *The button:* ★ "Update" (VS Code's word, and Lens's captured button's), a compact pill in the
  theme's primary colour, "Update to v0.16.0" with the version since 2026-10-16 (the owner, F11;
  a version's own leading v not doubled), "Update" without one and where the bar has no room for
  the version (the version gives way first and never widens the window's minimum), its tooltip "Cogpit <version> is available: show it in the update
  window", a click running `app.checkForUpdates`. The version is the appcast's
  `displayVersionString`, kept when it is at most 64 bytes of letters, digits, spaces and
  `.+-_()`, else the button shows without it; agents read it in `ui_state`'s `update`. Lens's
  captured Update button has a menu whose items are not captured, so Cogpit's opens Sparkle's
  window directly.
- *The toggles' words:* ★ View › Navigator and View › Dock, checked while open, right before Agent
  Chat; tooltips "Hide Navigator (⌘B)" or "Show Navigator (⌘B)", "Collapse Dock (⌘J)" or "Open
  Dock (⌘J)", "Hide Agent Chat (⌘I)" or "Show Agent Chat (⌘I)"; the chat's toggle does what ⌘I
  does (opened by the user, the input takes the focus, D-10). The palette lists Navigator and Dock
  under View.
- *The hidden navigator:* ★ its column and its edge give their room to the tab bar, the views and
  the dock; the hotbar stays; the keyboard leaves it if it was in it; its width stays the user's for
  when it comes back.
- *While a settings page is open:* ★ the toggles keep working; the navigator's and the dock's state
  shows once the page closes (the page replaces both, the chat column stays beside it). Options:
  the two toggles disabled while a page is open.
- *The icons:* ★ Cogpit's own drawings on the 16-point icon grid, so that their one-point lines
  stay sharp on a 1× screen: a window's outline with the part filled while it is open and a line at
  its inner edge while it is closed, in the theme's foreground colour. No VS Code codicon or other
  asset ([D-12](#d-12-parity-sources)).
- *The renders:* ★ the eight new ones and the title bar's history render with the toggles; every
  other full-window render keeps its old top bar until one pass re-renders them after the parallel
  branches are merged (about 50 renders differ in the top bar only; re-rendering them on this
  branch would collide with every branch that touches a render).

**Verified, and how:** with the real Sparkle 2.10.0 in a harness bundle against a feed on a local
server whose item is newer (`TestSparkleReportsAScheduledUpdate`, in `make test-darwin`): the check
Sparkle's scheduler runs (`checkForUpdatesInBackground`) reaches the delegate, Sparkle shows no
window, and the delegate reports the update with its version. The engine and the window are tested
with a fake updater. **Not verified:** the report at the end of an update session (it needs a click
in Sparkle's window); a click on the button bringing Sparkle's window with the held update up in the
running app (Sparkle's documentation says `checkForUpdates:` does); a check by Sparkle's own
scheduler a day after the last; an installed copy with the update key, which no release with files
carried on 2026-10-15 (v0.11.0 to v0.13.0 are built with it; [D-4](#d-4-packaging-and-updates)); the toggles and the button in a real window.

## D-55: A Config Map's values edited in the drawer

**Decided (owner, 2026-10-15, his finding F3):** in the real window, on `cilium-config` of a real
cluster, the drawer's Data section ran long keys under their values: `itemName` wrapped a name wider
than the 128 pt name column (`sizeDrawerLabel`) only at spaces, and `agent-not-ready-taint-key` has
none. Lens's ConfigMap drawer (the owner's capture of 2026-10-15) shows under "Data" each key on a
line of its own and its value below in a code editor: line numbers, colours by the value's type
(strings blue, `true` and `false` ochre), about five lines high even for a one-line value, a resize
handle at its bottom right; a Save at the end of the data is not in the capture (assumed). The
owner decided:

1. In the Data sections (a Config Map's, a Secret's too) the key on a line of its own and the value
   below it at the full width, a Config Map's in a code-editor look of Cogpit's own drawing (D-12).
2. Everywhere else a name without spaces wraps at `-` `.` `_` `/` within the 128 pt column, and by
   characters where it has none of them, so that no name runs into its value; a name that fits is
   drawn as before.
3. A Config Map's values are edited in the drawer, as in Lens, with Save at the end of the data,
   through Cogpit's existing write path: an engine command (an MCP tool, in the audit log), mutating
   with its verbs, the diff shown, a server-side dry run and the resourceVersion precondition, a
   conflict shown and never overwritten silently, D-2 for agents through the registry.
4. A Secret's Data takes the layout; its values stay masked with the reveal (H-8) and the decoded
   certificates of [D-30](#d-30-certificates-in-secrets-decoded) keep working. Editing a Secret's
   values in the drawer is not in this scope: **open for the owner**.

**Built 2026-10-15.** Engine ([`engine/configmapdata.go`](../../internal/engine/configmapdata.go),
[`commands/configmapdata.go`](../../internal/engine/commands/configmapdata.go)):
`configMap.readData` (read) returns the first 100 keys of `data`, sorted, each value whole from the
cache, and the resourceVersion they are of; `configMap.reviewData` (read, verbs `update configmaps`
as `object.dryRun`'s) diffs the Config Map with the given values against the cached object and, when
they change something and are based on the cached version, asks the server for a dry run;
`configMap.saveData` (mutating, verbs `list`, `watch`, `update configmaps`) writes them as `object.apply`
writes an Edit tab: the cached object with the values set, `managedFields` dropped, the
`ObjectWriter`'s `UpdateObject` with `dryRun=All` first, then for real, the given resourceVersion as
precondition. A Config Map whose cached version is not that one is not sent at all; a 409 from the
server waits for the cache to see the newer version; either way a `DataConflictError` says
nothing was saved, names the version the Config Map is at and carries the values against it, and a
save that names that version writes them over it, a second, explicit call. Values only then (keys
too since the owner's audit decision of 2026-10-16, below): a key the data does not hold, binary
data, a Helm release record and an immutable Config Map are refused before anything is sent. An agent's save waits for the user unless agent writes are allowed for the
context (D-2, through the registry, `TestConfigMapDataAgents`); the confirmation shows the diff it
writes (`Spec.Review`); a value that holds a placeholder Cogpit showed in place of a Secret's value
is refused before anyone is asked; values and diff lines an agent reads have the cached Secrets'
values replaced ([H-82](../security/agents-and-mcp.md#h-82-secret-values-printed-into-logs-reach-agents)).
GUI ([`fyneui/configmapdata.go`](../../internal/ui/fyneui/configmapdata.go),
[`fyneui/drawerdata.go`](../../internal/ui/fyneui/drawerdata.go),
[`fyneui/wrapname.go`](../../internal/ui/fyneui/wrapname.go)): the Data section draws each key
muted on its own line and its value in a `yamlview` editor the width of the details; Revert and
Save at the end of the data, the line under them saying what the editors hold; Save opens a review
over the window (the diff, the dry run's verdict) whose Save writes. A Secret's Data and a Config
Map's Binary Data put each key above its value the same way. Tests: `TestConfigMapDataSave`,
`TestConfigMapDataNeverOverwritesUnasked`, `TestConfigMapDataRefusals`,
`TestConfigMapDataAgents`, `TestConfigMapDataAgentsReadNoSecret`,
`TestConfigMapDataDisabledOnTheDemo`, on kind `TestKindConfigMapDataSave` (a real dry run, save and conflict in its own namespace); GUI `TestConfigMapDrawerKeysAboveValues`,
`TestConfigMapDrawerGripResizes`, `TestConfigMapDrawerEditAndSave`,
`TestConfigMapDrawerConflict`, `TestConfigMapDrawerKeepsUnsavedEdits`,
`TestConfigMapDrawerReadOnly`, `TestDrawerNameWrapsAtSeparators`,
`TestDrawerLongNameNeverRunsIntoItsValue`, `TestSecretDrawerKeysAboveValues`; yamlview
`TestPlainStrings`, `TestScrollPast`. PNGs: [drawer dark](../../internal/ui/fyneui/testdata/configmap_drawer_dark.png),
[light](../../internal/ui/fyneui/testdata/configmap_drawer_light.png),
[Save at the end](../../internal/ui/fyneui/testdata/configmap_drawer_footer_dark.png),
[review dark](../../internal/ui/fyneui/testdata/configmap_save_dark.png),
[light](../../internal/ui/fyneui/testdata/configmap_save_light.png),
[conflict](../../internal/ui/fyneui/testdata/configmap_conflict_dark.png),
[a Secret's long keys dark](../../internal/ui/fyneui/testdata/secret_longkeys_dark.png),
[light](../../internal/ui/fyneui/testdata/secret_longkeys_light.png).

**The owner's audit answers (2026-10-15), built the same night:** three of the lead's questions
below answered. (9) **Save rebases by itself:** when the Config Map changed on the cluster since the
version the values are based on, but none of the edited keys changed there, the edits move onto the
newer version, and the diff and the dry run run again against it and are shown to the user before
Save writes; otherwise the conflict and "Save over version N" as built; never overwritten silently.
(10) **Unsaved drawer values as disk drafts**, as an Edit tab's ([D-11](#d-11-crash-stance),
`drafts/`, 0600, offered at the next connect). (2) **A Secret's values in the drawer**, the window's
only: [D-59](#d-59-a-secrets-values-edited-in-the-drawer), built the same night. Built: `configMap.reviewData` takes
`from`, each edited key's value at the version the values are based on; a cached version newer than
that one whose edited keys still hold those values rebases the edit (`rebasedFrom` names the old
version, `baseResourceVersion` the new one, which the save takes), and a dry run that meets a still
newer version on the server reviews once more against it; `configMap.saveData` itself never rebases,
so nothing is written that the user has not seen. The drawer keeps each edited value's base, sends
it with the review, shows "rebased onto version N" with the diff, and after a save that meets a
newer version reviews again instead of writing. `configMap.keepEdits` (GUI-only, [`engine/drafts_drawer.go`](../../internal/engine/drafts_drawer.go))
takes the drawer's edited values 300 ms after the typing stops and at once when the drawer moves on:
the session keeps them for the drawer's next visit (`configMap.readData` hands them back, to the
window only), and the drafts write them 2 s later into `drafts/<id>.json` (type `configMapData`,
the values, their bases and the version), removed when the user saves or reverts; a quit or a crash
leaves the file, offered when the context connects (the drafts banner's row "Unsaved values of
ConfigMap … (drawer)"), and `editor.restoreDraft` opens the Config Maps view and the drawer with the
values in its editors, based on the version they were edited at. Tests: `TestConfigMapDataRebases`,
`TestConfigMapDrawerDrafts`, on kind `TestKindConfigMapDataSave` (a real rebase); GUI
`TestConfigMapDrawerRebases`, `TestConfigMapDrawerConflict`, `TestConfigMapDrawerKeepsUnsavedEdits`,
`TestConfigMapDrawerTakesARestoredDraft`, `TestDraftsBannerOffersAndActs`. PNG: [rebased](../../internal/ui/fyneui/testdata/configmap_rebased_dark.png).

| The lead's decision (night run 2026-10-15/16), confirmed by the owner, audit of 2026-10-16 | Options | Choice | Reason |
|---|---|---|---|
| What "none of the edited keys changed" compares | (a) each edited key's value at the base version, which the window sends (`from`); (b) the engine remembers the values it handed out per version; (c) a three-way merge of the texts | (a) | The window has those values already; the engine keeps no history of versions; (c) merges within a value, which the owner's rule does not ask for |
| A key the cluster changed to exactly the value the user typed | (a) a change of an edited key: the conflict; (b) no conflict, nothing to write | (a) | The owner's rule names the keys, not the outcome; the conflict shows that the two agree, and Save over writes nothing new |
| Whether the save rebases | (a) the review only, the save never; (b) the save too | (a) | "Diff and dry run again against the new version, shown to the user": a save that meets a newer version reviews again and waits for Save |
| After "Save over version N", a further change of another key | (a) the conflict again; (b) a rebase from N | (a) | The window does not know the edited keys' values at N; the conflict shows them, and Save over writes. Conservative, never silent |
| When the window hands its edits to the engine | (a) 300 ms after the typing stops and at once when the drawer moves on; (b) on every keystroke; (c) only when the drawer moves on | (a) | (b) sends up to 1 MiB per key press; (c) loses what a crash finds unsent. The draft's own delay (2 s, as an Edit tab's) follows |
| Who may keep or read unsaved drawer values | (a) the window only (GUI-only command, the values in `configMap.readData` for the window only); (b) agents too | (a) | An agent could plant values the user would take for their own typing; an agent changes values with `configMap.saveData`, which the user sees |
| A restored draft and the user's typing in the same Config Map | (a) the restore refused while the session holds unsaved values of its own for it; (b) merged | (a) | As an Edit tab with unsaved changes refuses a draft |

**The owner's audit decision (2026-10-16, N152 (b)): keys too**, for a Config Map and, consistently,
a Secret ([D-59](#d-59-a-secrets-values-edited-in-the-drawer)): the drawer adds and removes keys, not
only values; a removal is shown prominently in the review before Save. **Built 2026-10-16.** Engine
([`engine/datakeys.go`](../../internal/engine/datakeys.go), [`engine/configmapdata.go`](../../internal/engine/configmapdata.go)):
`configMap.reviewData` and `configMap.saveData` take `add` (keys with their values) and `remove`
(keys) beside `data`, one update for all; `data` is optional now. An added key follows the API
server's rule for ConfigMap and Secret keys (`CheckDataKey`, which defers to apimachinery's
`IsConfigMapKey`: letters, digits, `-`, `_` and `.`, at most 253 characters, not `.` or `..`, not
starting with `..`), is not a key of the binary data, and not one the data holds at the version the
edit is based on; a removed key is one it holds and not binary data; a key named twice is refused;
every refusal names the key and nothing is sent. The review reports `keysAdded`, `keysRemoved` and
the Config Map's `uid` beside `keys`; the save `keysAdded` and `keysRemoved`. A save that removes a
key escalates to **destructive** (`Escalate`): the user confirms it whoever asks, an agent's even
where agent writes are allowed (D-2), the confirmation's detail naming the keys removed and added
(`DataKeysDetail`, "Removes key a and adds key b."), the save acting only on the Config Map with
the UID the confirmation recorded (`checkRemovalTarget`); a save that only adds keys or changes
values stays mutating. The rebase (D-55's (9)) moves an edit onto a newer version only when none of
its keys changed there: each changed and removed key still holds its value at the base (`from`,
which now names removed keys too), each added key is still absent; an edit that only adds keys needs
no `from`. Against a newer version that is not rebased (stale) the review and the conflict show what
a save over it would write: an added key that appeared there is a change of its value, a removed key
gone there is left out. `configMap.keepEdits`, the kept edits and their draft (`drafts/`, 0600) hold
`add` and `remove` too, and a draft's listing names its keys and `keysRemoved`, never a value. GUI
([`fyneui/datakeys.go`](../../internal/ui/fyneui/datakeys.go), [`fyneui/configmapdata.go`](../../internal/ui/fyneui/configmapdata.go)):
where Save is possible, each key's line ends with a remove button (a trash icon, the tooltip "Remove
the key … when you save"), a removed key shows in the error colour with "Removed: Save removes this
key and its value." and Undo; Add Key at the start of the footer opens a row for the new key's name,
checked as the user types (the server's rule, a key the Config Map or the drawer holds already),
whose Add puts the key with an empty editor at the end of the data, marked "new" (the warning colour
when the cluster holds the key by then), with a button that drops it. The review names above the
diff the keys removed, bold in the error colour with the Config Map, its namespace and its context,
what a removal means and "This cannot be undone.", then the keys added; its Save is red for a
removal and arms after the confirmation's delay (750 ms), and is the engine's confirmation of the
removal: the window hands the confirmer the answer for exactly that command, object, UID and keys
(Rollback's dialog answer, [`helmrollback.go`](../../internal/ui/fyneui/helmrollback.go)), anything
else asks with the usual dialog. "Save over version N" after a conflict writes what the review
showed. Tests: `TestConfigMapDataAddsAndRemovesKeys`, `TestConfigMapDataKeyRefusals`,
`TestConfigMapDataKeysRebase`, `TestConfigMapDataAgentRemovalAsks`, `TestConfigMapDrawerDraftsKeepKeys`,
`TestCheckDataKeyIsTheServers`, `TestDataKeysDetail`, `TestDataEditRebase`, on kind
`TestKindConfigMapDataSave` (a real dry run and save of a 253-character key added and a key
removed); GUI `TestConfigMapDrawerAddsAndRemovesKeys`, `TestConfigMapDrawerRemovalArms`,
`TestConfigMapDrawerKeysConflict`, `TestConfigMapDrawerKeepsKeys`. PNGs: [drawer dark](../../internal/ui/fyneui/testdata/configmap_keys_drawer_dark.png),
[light](../../internal/ui/fyneui/testdata/configmap_keys_drawer_light.png), [review dark](../../internal/ui/fyneui/testdata/configmap_keys_review_dark.png),
[light](../../internal/ui/fyneui/testdata/configmap_keys_review_light.png).

| The lead's decision for the owner to confirm (2026-10-16, keys too) | Options | Choice | Reason |
|---|---|---|---|
| How added and removed keys reach the engine | (a) `add` and `remove` beside `data` in the existing commands, one write; (b) `data` alone, a new key in it adding one and `null` removing one; (c) commands of their own | (a) | One update, one new resourceVersion, one review and one confirmation for everything the drawer holds; a key is added only where it is named in `add`, so a typo in `data` still never adds one (the reason values only was safe); (b) makes a typo an addition; (c) splits one Save into writes that meet each other's versions |
| The danger of a removal | (a) a save that removes a key destructive (escalated), one that adds keys or changes values mutating; (b) every save mutating; (c) every save destructive | (a) | A removed key loses its value and changes what a workload reads (an environment variable from it, a file of a mounted volume), as scaling to zero is destructive among Scale's invocations; an added key changes nothing a workload reads until something refers to it. The verbs stay `update configmaps` (`update secrets`): a removal is an update of the object |
| The window's confirmation of a removal | (a) the review is the confirmation: it names the object, the namespace, the context and the keys, its Save arms after the confirmation's delay, and the engine takes its answer for exactly that command, object, UID and keys; (b) the usual confirmation dialog after the review | (a) | Rollback's dialog does the same; a second dialog would repeat what the review shows one click earlier. Bound to the UID and the keys, an answer confirms nothing else; anything that does not match asks with the usual dialog |
| A key the edit adds or removes against the version it is based on | (a) refused: an added key the data holds, a removed key it does not; (b) taken leniently | (a) | The window never sends either; for an agent it is a mistake to be told of, as a key of `data` the data does not hold |
| The same against a newer version (a stale review, the conflict) | (a) shown as a save over it would write: an added key that appeared is a change of its value, a removed key gone is left out, and "Save over version N" sends that; (b) refused | (a) | The user sees exactly what Save over writes; (b) would leave a conflict that no Save can resolve |
| The rebase of a removal or an addition | (a) a removed key must hold its value at the base (`from`), an added key must still be absent, else the conflict; an edit that only adds keys needs no `from`; (b) removals and additions never rebase | (a) | The owner's rule of D-55's (9) with keys: an added key that appeared and a removed key that changed are changes of the edit's keys |
| Where Add Key and the remove action sit | (a) Add Key at the start of the footer, a row for the name above it; a trash icon at the end of each key's line; a removed key in the error colour with Undo; (b) a menu per key; (c) the remove action only in an edit mode | (a) | One click each, at the key it acts on; nothing is removed before Save and its review, and Undo takes it back. Look and words are the lead's: the owner's visual review is open |
| An added key's place | (a) at the end of the data, marked "new", until the save; then the data's order; (b) sorted in at once | (a) | The user sees what they added; after the save the drawer reads the Config Map again |
| A removed key's value | (a) hidden, "Removed: Save removes this key and its value." in its place; (b) the editor kept, read-only | (a) | The key's line says what Save does; Undo brings the editor back with what it held |
| Kept and drafted keys | (a) `add` and `remove` kept and drafted with the values, a draft's listing naming the keys and `keysRemoved`; (b) values only kept | (a) | The owner's "no typing lost unasked" (D-55's (10)) holds for a key typed too |

### The lead's decisions for the owner to confirm

| Question | Options | Choice | Reason |
|---|---|---|---|
| One Save or one per key | (a) one Save at the end for every edited value; (b) a Save per key | (a) | The owner's "Save at the end of the data", as Lens has it; one update is one new resourceVersion, while a save per key would make the next key's save meet a conflict with the version the first one wrote |
| Where the diff is shown | (a) a review over the window: the diff, the dry run's verdict, Cancel and Save; (b) under the data in the drawer; (c) the change opened in an Edit tab | (a) | The Edit tab's two steps (review, then write) without leaving the drawer: the 100 editors and the drawer's scroll stay as they were, and a long diff does not push the data away; (c) would move the user away from the drawer for a one-word change |
| Edited values when the drawer moves to another object or closes | (a) dropped silently; (b) a question before the drawer moves; (c) kept in memory per object, a toast saying so, back in the editors when the Config Map is shown again; (d) a draft on disk ([D-11](#d-11-crash-stance)) | (c), and since the owner's answer of 2026-10-15 (d) too | No typing is lost unasked; (b) cannot be: the drawer moves through `ui.*` commands that agents run too, and a question in the window would hold their command. **Answered (owner, 2026-10-15): (c) and (d)**, the values kept in memory and as a draft on disk, built (above); a Secret's never on disk ([D-59](#d-59-a-secrets-values-edited-in-the-drawer)) |
| The Config Map changes on the cluster while values are edited | (a) the edited editors keep the typing and stay based on the version they were edited at, the others follow the cluster, the line under the data says it changed, Save shows the conflict first and "Save over version N" writes; (b) moved silently onto the newer version when the keys edited did not change there (a three-way merge); (c) the edits dropped | (a), since the owner's answer of 2026-10-15 (b) shown to the user | "Never overwritten silently": the user sees the values against the version they would replace before they replace it, as the Edit tab's apply after a conflict. **Answered (owner, 2026-10-15): (b), not silently**: the edits move onto the newer version when none of the edited keys changed there, the diff and the dry run against it shown before Save writes; (a) otherwise. Built (above) |
| Adding and removing keys | (a) values only; (b) keys too | (a), since the owner's decision of 2026-10-16 (b) | The owner's "probably not"; the Edit tab adds and removes keys; `configMap.saveData` refuses a key the data does not hold, so an agent cannot add one by accident either. **Decided (owner, audit 2026-10-16, N152): (b)**, for a Config Map and a Secret, built (above): a key is added only where `add` names it, so the accident stays impossible |
| Values the drawer model cuts (4 KiB, 40 lines a value; 32 KiB, 400 lines in all) | (a) the editors get every value whole (`configMap.readData`); (b) a cut value read-only with a note | (a) | Saving a cut text would truncate the value; the editors draw only the lines in view, so the caps' reason (a text line per line of a value) no longer holds for them. The drawer model keeps its caps for agents' `object_get`; a Helm release record's payload and binary data stay by size only and are never editable |
| The editor | height: (a) Lens's fixed five lines; (b) as tall as the value; (c) between a minimum and a maximum, with a grip | (c) | `yamlview` with plain scalars coloured as strings (`SetPlainStrings`), `true`/`false`/`null` and numbers in their YAML colours, line numbers; as tall as the value's lines between 3 and 12 (then it scrolls), a grip at its bottom right drags it between 1 and 40 lines: 100 one-line values take half of Lens's height; the wheel scrolls the drawer once the editor is at its end (`OnScrollPast`) |
| A Config Map's Binary Data | (a) keys above their sizes as in Data; (b) the name column as before | (a) | One layout for a Config Map's data, binary or not |
| Where Save is not possible | (a) the editors read-only, Save disabled, the reason under them; (b) the editors hidden | (a) | The values stay readable and copyable; the reason is the command's (the demo cluster, no `update` on the namespace's Config Maps from the access reviews, an immutable Config Map, a Helm release record) |
| The commands' ids | `configMap.readData`, `configMap.reviewData`, `configMap.saveData` | as listed | `<kind>.<action>` as in [extending](extending.md#adding-a-command); Lens's documentation names no id for the drawer's Save |

### What it does not cover

- **A Secret's values in the drawer:** decided by the owner on 2026-10-15 and built, the window's
  only: [D-59](#d-59-a-secrets-values-edited-in-the-drawer).
- **A Config Map whose controller rewrites the edited key every few seconds, live.** The rebase
  carries an edit over changes of other keys only: a write of an edited key between the review and
  Save shows the conflict again, as the Edit tab's apply does.
- **Kept values are lost with less than 300 ms of typing unsent, live.** The window hands its edits to
  the engine 300 ms after the typing stops and the draft follows 2 s later (as an Edit tab's): a
  crash within that time loses the last keystrokes; a quit writes the waiting draft.
- **The values in a draft are plain text on disk, live.** A Config Map's values are no Secret's, but
  a user may keep a credential in one; the file is owner-only (0600) in Cogpit's own folder, as an
  Edit tab's draft ([credentials › H-38](../security/credentials.md#h-38-drafts-keep-typed-text-in-plain-text)).
- **Pointer events over a scrolled-out editor, live, not verified in the window.** Fyne 2.8.1's
  object walk gives a scrolling widget (the editor) its own rectangle as its clip instead of the
  intersection with the drawer's scroll; the window's GL painter intersects them, so nothing is
  drawn outside the drawer, but the hit test uses the walk: a click or hover on an empty part of the
  drawer's header above an editor scrolled under it could reach that editor (the release drawer's
  values editor has the same). Fyne's software painter draws such an editor over the header, which
  is why the footer PNG shows a short Config Map.
- **Lens's Save is assumed**, not captured; nor is what Lens does with an edit when the drawer moves
  on, nor whether Lens's drawer adds or removes keys at all (the owner's decision, inferred).
- **A key removed through the Edit tab is not destructive, live.** `object.apply` escalates only
  for a scale to zero, so a Config Map's key that an agent removes with `object_apply` where the user
  allowed agent writes is written without asking, while the drawer's removal always asks (2026-10-16).
  Whether `object.apply` should escalate for a removed data key too is the owner's to decide.
- **An added key beyond the first 100, live.** The drawer lists the first 100 keys; a name among the
  keys beyond them passes the drawer's check and is refused by the engine when the review runs ("its
  data holds the key … already").

## D-56: The terminal's scrollback

**Decided (owner, 2026-10-15):** a colleague could not scroll back in a terminal tab. The terminal
widget kept no scrollback: fyne-io/terminal's `scrollDown` overwrote the top row. Upstream's draft
[#147](https://github.com/fyne-io/terminal/pull/147) (open since 2026-04-05) adds one by keeping
every row in the grid; tried in the fork, it showed 13 defects, each proven by a test (to be
offered to #147 as fixes, item 10), and the rendering cost measured below. The owner decided:

1. A view scrolled back stays where it is while output arrives; typing returns it to the screen (as
   Terminal.app, iTerm2 and VTE do; xterm and #147 jump to the bottom).
2. A scroll bar while there is scrollback: thin, wide under the pointer, its thumb draggable; a tap
   above or below the thumb moves a screen.
3. No line limit, as Terminal.app has none by default: the screen stays a grid of the screen's rows,
   and the lines that scroll off go into a compact scrollback, drawn only while in view.
4. A bound of 256 MiB per terminal against output without end (container output, which anyone who
   can deploy controls); past it the oldest lines go, and a row above the scrollback says how many.
5. The wheel in a full-screen program (the alternate screen: less, man, vim, k9s) goes to the
   program: as mouse buttons 4 and 5 to one that asked for the mouse, as the arrow keys otherwise.
6. Scrolling by keyboard: not pursued.
7. Agents read the scrollback too, with the tab's switch on ([D-44](#d-44-what-agents-read-of-the-docks-logs-and-terminal-tabs)),
   by stable line numbers: `dock_read`'s `before`, `after` and `limit` ask for exact lines (the two
   above line n: `before: n, limit: 2`).
8. A check in a real window before the release. The lead cannot drive or capture a window on the
   owner's Mac (no Accessibility or Screen Recording permission), so the owner checks it in a build
   with a profile of its own that the lead starts. Done in two rounds on 2026-10-15; what the first
   found is fixed (below), the second passed.
9. Released after the tests and that check.
10. Upstream: a pull request to #147 with all the fixes and their tests, and this model offered as a
    counter-proposal with its measurements; Cogpit goes ahead meanwhile. Every outward step is the
    owner's to approve.
11. A selection reaches over the whole scrollback: a drag past the view's edge scrolls it; a line
    wrapped over rows is copied as one line.
12. A new width wraps the lines again, the scrollback's and the main screen's together, the cursor
    staying on its character; the alternate screen's program draws itself again.
13. `clear` empties the scrollback (erase saved lines, CSI 3 J, which zsh's and bash's `clear`
    send), as xterm and Lens do; the owner compared it with Terminal.app and kept it.

**Measured (2026-10-15, Apple M1 Ultra, Fyne's test driver, not a window):**

| What | Result |
|---|---|
| #147's model, 3,000 lines in 300 batches, a refresh each | 0.29 s without scrollback, 1.46 s with 100 rows, 4.7 s with 300, 16.2 s with 1,000 (56×); with Fyne's software painter 6.4 s against 1 min 59 s |
| Why | `TermGrid` sets `Scroll = None`, so Fyne's TextGrid makes and refreshes a row object for every row it holds, in a loop quadratic in their number (`addRowsIfRequired`) |
| A row in Fyne's grid | 116 bytes per cell (a style object per cell, as output writes them), 9,267 bytes per row of 80 cells, 93 MB per 10,000 rows |
| A line in Cogpit's scrollback | 205 bytes for 80 cells in two colours (`TestScrollback_Size`); 256 MiB hold about 1.3 million such lines (estimated from that) |
| Terminal.app | no limit by default ("Limit number of rows to" is a profile option, unset in the owner's profiles); its binary trims scrollback when the startup disk runs low (read in its strings, not observed) |
| `time seq 1 200000` in a window (a terminal tab whose shell ran it, a profile of its own, `--cpuprofile`) | 24.6 s before the scrollback, 5.1 s with it, 1.9 s with the output's characters going to the main goroutine in batches, 0.87 to 0.90 s with the reader waiting for it at most every 17 ms; the owner's Terminal.app 0.29 s. The profiles showed the time in the hand-overs between the goroutines, not in the widget's code |

**Built 2026-10-15** (`third_party/fyne-terminal`, [PATCHES.md › Scrollback](../../third_party/fyne-terminal/PATCHES.md#scrollback)):
[`history.go`](../../third_party/fyne-terminal/history.go) holds the lines (the text as UTF-8, the
styles as runs, numbers kept, the bound, the rows of each line at a width);
[`scrollback.go`](../../third_party/fyne-terminal/scrollback.go) the view of the rows from
`viewTop`, the wheel, the reflow, the selection by row numbers and `Lines`, `LineTexts`;
[`scrollbar.go`](../../third_party/fyne-terminal/scrollbar.go) the bar. The grid that holds the screen
is never drawn; a second one draws copies of the rows in view. termview's `Output` pages the lines
for `dock_read`; `TestScrollback_*`, `TestOutputNumbersStayWithTheirLines`.

**The owner's check (2026-10-15, two rounds):** the first found seven things, fixed the same day:
the output twenty to thirty times slower than Terminal.app's (above); a modifier alone (⌘, Ctrl)
returned the view to the screen; a selection dragged past the terminal's edge could not be copied
(Fyne unfocuses on a release outside the focused widget; the terminal takes the focus back); the
prompts after a resize stood in the middle of rows (zsh's PROMPT_SP padding taken for a wrap); lines
out of order after resizes (the cursor's shortened line moved the cursor below the screen; found
again with a random test); the wheel did nothing in man and vim (the arrow keys in the wrong form
for their mode); and the window frozen while resized with the mouse, which is Fyne's (see Open). The
second round passed.

The lead's decisions for the owner to confirm:

- A carriage return on a row that holds only blanks takes back the wrap into it, so that zsh's
  PROMPT_SP padding is no part of the line before; a program that pads a wrapped line with blanks
  and returns to its start would lose that wrap too.
- A drag that ends outside the terminal gives it the focus back while it has a selection.
- The reader waits for the main goroutine at most every 17 ms (`maxRefreshInterval`), which keeps
  the queue bounded under a flood (upstream's #148) without a round trip per read.
- Erase in display (CSI 2 J) does not save the screen into the scrollback, and rows go into the
  scrollback only from a scroll region that starts at the screen's top (xterm).
- A line keeps no trailing blanks in the scrollback; a cell never written is a space there.
- A line longer than a quarter of the bound (output without line feeds) ends there, so that the
  bound can drop it.
- Repeat (CSI b) past a screenful and a line leaves the skipped lines out of the scrollback (#156's
  bound against a count near MaxInt).
- The wheel's arrow keys are the normal ones (`ESC [ A`), also in the application cursor-key mode,
  which the widget does not track; less and man take both.
- A reflow leaves out the rows below the cursor and below the last row with text, so shrinking the
  screen drops empty rows before it pushes lines into the scrollback.
- A selection that lies in the scrollback stays while output arrives; one that reaches the screen
  goes, as before.
- A drag past the view's edge scrolls a row per mouse movement, not on a timer.
- The timer of blinking text stops when the terminal's renderer goes (a closed or hidden tab) and
  starts again at its next refresh; it ran on after its terminal was gone.

Open: search in the scrollback; wide characters (CJK) take one cell, the widget's limit; output
stops while the window is resized with the mouse on macOS, as in every Fyne app (Fyne's GLFW loop
runs queued functions only between its event polls, and the poll stays in Cocoa's resize loop;
read in Fyne 2.8.1's `loop.go`, not patched).

## D-57: Choosing a cluster

**Decided (owner, 2026-10-15 evening, an interview):** "We have far too many ways to choose a
cluster." The ways counted that day: the top bar's "Switch context ▾"; Cluster › Switch Context…
and ⌘P; the palette's Connect to Cluster; the navigator's scope dropdown (the section, the shown
hotbar bank's tiles, All Clusters…); the cluster root's ‹; the hotbar's tiles and F1–F12; the
welcome page's rows; another cluster's tab (A62); a `--context` that fails at start. Four of them
opened the context picker, a modal that listed every context a second time beside the navigator's
section (Local Kubeconfigs), the list the owner likes. The section already had the picker's list,
its connect with the login approval, its row menu and Add Kubeconfigs… (on the root's
right-click); only the picker had a visible Add button and the error text beside a context (the
section: a red dot). The owner decided:

1. **I1, the context picker goes.** Every way that opened it shows the navigator's section
   instead: the Cluster menu's item, ⌘P, the palette's `cluster.connect` entry, a `--context` that
   fails at start; the navigator is shown first if ⌘B hid it (`ui.toggleNavigator`,
   [D-54](#d-54-the-layout-toggles-and-the-update-button)).
2. **I2, the top bar's "Switch context ▾" goes;** the dot, the context's name, its file and the
   state stay, a display of the connected cluster.
3. **I3, the navigator's scope dropdown goes.** At a cluster the header shows a button "‹ All
   Clusters" in its place (the search button as before), and the cluster's root row loses its ‹
   (one way up); at the section the header shows the title "All Clusters" alone. The dropdown's
   listing of the hotbar's bank (A52, [D-51](#d-51-the-navigators-and-the-command-palettes-state)'s
   bank question) and its All Clusters… go with it.
4. **I4, ⌘P** shows the section with its search open and focused; ↵ drills into the first match (as
   before); **Escape on an empty search returns to the tree of the cluster it came from**.
5. **I5, the palette's Connect to Cluster** (`cluster.connect`) stays and does what ⌘P does. The MCP
   tool `cluster_connect`, with which agents connect by name, is unchanged.
6. **I6, one name in menu and palette: Connect to Cluster** (the registry's label): the Cluster
   menu's "Switch Context…" reads "Connect to Cluster…"; the navigator's button stays "‹ All
   Clusters".
7. **I7, the welcome page keeps its cluster groups** (the hotbar's shown bank, Recent up to 6, up to
   8 others); only its All Clusters… now shows the section, as ⌘P does.
8. **I8, a visible Add Kubeconfigs in the section:** a "+" in the navigator's header beside the
   search (tooltip "Add Kubeconfigs…"), at the section only; it replaces the picker's visible
   button. The Cluster menu's, the welcome page's and the root's right-click stay.
9. **I9, names:** the header at the section reads "All Clusters"; the root row below it keeps
   "Local Kubeconfigs".

Not asked, since a home exists: the error text after a failed connect, which only the picker showed
beside the context; the welcome page and an open view say "Could not connect to <name>: …", the
hotbar toasts it, and the section keeps its red dot. Unchanged: the hotbar and F1–F12, another
cluster's tab (A62), `--context`, deep links, agents' `cluster.connect` and `ui.setNavigatorScope`.

**Options weighed in the interview:**

| | Chosen | Weighed |
|---|---|---|
| I1 | the picker goes, every way leads to the section | a ⌘P-only quick switcher built like the section (two lists to keep in step); the picker kept, only the top bar's button gone |
| I2 | the button goes, the display stays | the name clickable into the section (a hidden way); the button kept, leading to the section (two buttons, one target) |
| I3 | ‹ All Clusters at a cluster, the title at the section, the root without its ‹ | a button "All Clusters" on both levels with the root's ‹ kept (two ways); the cluster's name as the button (does not say where it leads) |
| I4 | the search focused, Escape on the empty search goes back | Escape only clearing (one key more after a slip); the section without the search (a key more to search) |
| I5 | the palette's entry does what ⌘P does | removed from the palette (a gap in "every command is in the palette"); the palette listing every cluster (a second list) |
| I6 | Connect to Cluster in menu and palette, chosen over the lead's recommendation | "All Clusters" everywhere (the lead's recommendation); "Switch Context…" kept |
| I7 | the welcome page keeps its groups, chosen over the lead's recommendation | Recent only and a button (the lead's recommendation); no clusters on the page |
| I8 | a "+" beside the search, at the section | a row at the list's end (scrolls away); nothing new |
| I9 | "All Clusters" in the header, "Local Kubeconfigs" on the root | "All Clusters" everywhere (a name needed again once cloud sections come, G18); "Local Kubeconfigs" everywhere (would relabel I3's button) |

**Why:** one list instead of two copies of it. The picker and the section listed the same contexts
from the same sources, with the same connect and the same menu; every change had to be made twice,
and the user had to learn which of eight ways led where. The section stays the one place that
chooses a cluster, and every way to it is a way to that list.

**Built 2026-10-15 (night run).** [`fyneui/findcluster.go`](../../internal/ui/fyneui/findcluster.go)
(`findCluster`: an open settings page closed through `settings.close`, the navigator shown through
`ui.toggleNavigator`, the section through `ui.setNavigatorScope`, then the search focused once the
window shows the engine's answer;
`connectMenuItem`), [`navigator_header.go`](../../internal/ui/fyneui/navigator_header.go) (the
title, "‹ All Clusters", the "+", Escape), [`navigator_sections.go`](../../internal/ui/fyneui/navigator_sections.go)
(the row's tooltip lines, Show Connection Updates…), [`navigator.go`](../../internal/ui/fyneui/navigator.go)
(the root without its ‹), [`bars.go`](../../internal/ui/fyneui/bars.go) (the top bar without the
button), [`shell.go`](../../internal/ui/fyneui/shell.go), [`palette.go`](../../internal/ui/fyneui/palette.go),
[`welcome.go`](../../internal/ui/fyneui/welcome.go) and [`tableview.go`](../../internal/ui/fyneui/tableview.go)
(Connect to Cluster…), [`clusterrow.go`](../../internal/ui/fyneui/clusterrow.go) (the welcome page's
rows, the picker's rows until then); `picker.go` removed. The registry's `cluster.connect` has the
menu path `Cluster/Connect to Cluster`; the engine's refusals of a login not allowed yet say
"connect from the navigator's clusters in Cogpit's window (⌘P)". Tests:
`TestConnectToClusterShowsTheSection`, `TestConnectToClusterShowsAHiddenNavigator`,
`TestConnectToClusterOverThePalette`, `TestConnectToClusterClosesASettingsPage`,
`TestConnectToClusterOverTheEngine`,
`TestWelcomeConnectToClusterButton`, `TestAutoConnectRefusedShowsTheSection`,
`TestCommandPaletteOpensTheWindowsDialogs`, `TestNavigatorHeaderAddsKubeconfigs`,
`TestNavigatorSectionTooltipLines`, `TestNavigatorSectionRowMenu`,
`TestSectionMarksAFailedConnectAndOpensItsLog`, `TestClusterRowsMarkAnInsecureContext`; renders
`navigator_find_*.png`, `navigator_back_*.png`, `welcome_insecure_*.png`, the picker's removed.

**The lead's decisions** (night run, 2026-10-15; ★ built). **Confirmed (owner, audit of 2026-10-16)** as built,
every one but the registry's menu path, which he changed, and the welcome page's button, which he
wants changed; the renders were not asked:

- *What is engine state (golden rule 2):* ★ the navigator shown, the section and the search's text
  are engine state through `ui.toggleNavigator`, `ui.setNavigatorScope` and `ui.setNavigatorSearch`,
  as before; the search field's keyboard focus and the cluster ⌘P came from stay the window's, as
  the palette's focus does. Options: a field such as `navigator.from` and commands for ⌘P and for
  Escape's way back. Why: both belong to a keystroke; an agent shows the user the clusters with the
  three commands, and the cluster to go back to is always the connected one (Cogpit holds one
  connection), which `ui_state` names.
- *Escape on a text in ⌘P's search:* ★ clears the text and keeps the search; the next Escape, on the
  empty search, goes back. Any other search: Escape clears and closes it, as before. Options: clear
  and close (the next Escape would land in the list, no longer "on an empty search"). Why: a slip in
  the text costs one Escape more, never the way back.
- *⌘P pressed at the section:* ★ Escape on its empty search closes the search, the section stays.
  Options: the connected cluster's tree (a surprise: the user was at the section).
- *⌘P over a settings page:* ★ closes the page first (`settings.close`, as Escape does), since the
  page takes the navigator's place; the context picker, an overlay, showed over it. Options: the
  section behind the page, its search focused only once the user closed the page (⌘P would seem to
  do nothing).
- *Back only to the cluster still connected:* ★ disconnected meanwhile, Escape only closes the
  search; another cluster connected drills into it and ends the search anyway.
- *← on the tree's root:* ★ still goes up to the section, the keyboard's way; I3's "one way up" read
  as the one visible way, the header's button, which takes no keyboard focus. Options: ← on the root
  doing nothing (a keyboard user would have no way up).
- *The palette's label:* ★ "Connect to Cluster", the registry's label as the palette shows every
  label; the Cluster menu adds "…" to a label whose item opens something, as for Remove from
  Cogpit…. Options: "Connect to Cluster…" in the palette too.
- *The registry's menu path:* ★ `Cluster/Connect to Cluster` (the lead's default: the item stays in
  the Cluster menu; the window did not use `File/Connect to Cluster`); the palette's category reads
  Cluster. The other cluster commands' paths still said `File/`, an inconsistency left as it was.
  **Resolved by the owner's decision in the audit of 2026-10-16 (N121, built the same day):** every
  cluster and kubeconfig command is under `Cluster/`, the window's Cluster menu that holds them:
  `cluster.disconnect`, `cluster.reconnect`, `cluster.login`, `cluster.remove`,
  `kubeconfig.sources`, `kubeconfig.addFile`, `kubeconfig.addFolder` and `kubeconfig.remove` moved
  from `File/`, so the palette files them under Cluster; no command names `File/`, a menu the
  window does not have. Metadata only: the window builds its menus itself, and the MCP tools'
  names do not depend on the path (`TestClusterCommandsUnderTheClusterMenu`).
- *The welcome page's button:* ★ "Connect to Cluster…" (the lead's default) with the search icon,
  its hint line keeping "⌘P finds a cluster by name"; the workspace's button while no cluster is
  connected ("Connect to a cluster…", which opened the picker) reads the same and does the same.
  **The owner wants this changed (audit of 2026-10-16); how, not said yet.**
- *A `--context` refused at start:* ★ a toast "Could not connect to <name>: <error>" and the section
  with its search focused, as ⌘P. Only a refusal before connecting (no such context) gets here; a
  connect that fails ends in the state error, which the welcome page shows, as before. Options: the
  section alone (the refusal's text, which only the picker showed, would be lost).
- *What the picker's row said that the section's row did not:* ★ a line each in the row's tooltip:
  "Could not connect: <error>" for the cluster that just failed (the lead's default),
  "The last connect failed: Show Connection Updates… in its menu says why" for an earlier failure,
  "Login program not allowed yet: Cogpit shows it before it connects" (or the token file or the
  unchecked connection, not allowed yet or changed), and the line naming the server the credential
  of an `insecure-skip-tls-verify` context goes to; and Show Connection Updates… in the menu of a row
  whose last connect failed, the connect's log ([G2](../parity/gaps.md)) that the picker's warning
  icon opened. Options: the picker's pills and icon on the section's rows (the owner likes the
  section as it is); nothing (the log reachable only from the welcome page's rows, which list a
  part of the clusters). The welcome page's rows keep the pills, the warning line and the icon.
- *The empty section's hint:* ★ "No clusters yet: add a kubeconfig with the + above." (was:
  right-click Local Kubeconfigs).
- *The renders:* ★ every render made again (the top bar without Switch context, the navigator's
  header; they also pick up the logo of [D-24](#d-24-logo) that older renders still showed).
  The README's pictures (`docs/images`) still show the old top bar and dropdown until
  `make showcase` runs.

**What it does not cover:** a context whose kubeconfig skips the certificate check is marked in the
section by a warning icon on its row and a line in the row's tooltip (the icon added 2026-10-16 by
the lead, for the owner to confirm), where the picker showed a pill and a line
([trust boundaries](../security/trust-boundaries.md)); the approval before its first connect and the
status bar's "TLS not checked" while connected are unchanged. Not verified: the section, ⌘P and
Escape in a real window (Fyne's test driver only).

## D-58: Standby and lost connections

**The problem (owner, 2026-10-15):** with a cluster connected, closing the MacBook lets it sleep
and drops every connection; opened again, Cogpit's window hung. Nothing detected an interrupted
connection (the state stayed `connected` whatever the watches did), nothing reconnected, and no
request but the connect's check had a timeout. The owner's `cogpit.log` on his Mac Studio (no
sleep) showed `failed to list services: Unauthorized` every ~5 minutes for two hours, for Services
only (once CRDs) while pods and namespaces kept working, and four kinds' watch streams breaking
together ("unable to decode an event from the watch stream").

**Decided by the owner (interview, 2026-10-15 evening), S1 to S8:**

1. **S1, find the hang's cause by reproducing it:** freeze a kind cluster's control plane
   (`docker pause` of its node, a peer gone silent as after sleep), drive the window offscreen and
   take a goroutine dump while it hangs; fix the cause, then build the rest.
2. **S2, detection by three signals:** macOS's wake notification checks at once; while connected a
   heartbeat `GET /version` every 15 s with 5 s to answer, two failures meaning interrupted; a watch
   stream that breaks checks at once. The heartbeat is no poll of cache data (the lead's reading of
   "never poll what can be watched") and goes through the cluster layer.
3. **S3, what the user sees:** the top bar's dot amber and "Reconnecting…"; a thin banner over the
   view saying what is interrupted, since when, the next attempt, and **Reconnect now**; the window
   stays usable. Amended by the owner on 2026-10-16 (F13, below): the dot blinks green, no word.
4. **S4, data and writes while interrupted:** the views keep the cache's last state (the banner says
   since when); every mutating command is disabled with the reason "connection interrupted", for the
   window and for agents, so that no write with an unknown outcome can start.
5. **S5, reconnecting:** automatic and without end while the window is open: at once on wake and on
   a network change, then after 1, 2, 4, 8, 16, at most 30 s; Reconnect now resets the backoff.
6. **S6, a lasting 401 (login expired):** at the first lasting 401 (two heartbeats) and at wake the
   approved credential plugin runs once by itself (D-26: an approved plugin only, one whose program
   changed asks first), so silent plugins renew without the user; if that does not help (401 again,
   or the plugin waits longer than 30 s, as for a browser login), the state is **"Login expired"**
   with a banner and **Log in again**; never a plugin run in the backoff, never a pile of browser
   tabs; writes stay disabled.
7. **S7, open streams by their kind:** logs resume by themselves from the last line's time with a
   marker line "— interrupted 14:03–14:07 —"; port forwards keep their backoff but do not dial while
   interrupted and dial at once after the reconnect; pod shells, the node shell and attach end with
   a note in the tab and a **Reconnect** button that opens a new shell in the same tab (a process
   cannot be resumed); local terminals are not affected.
8. **S8, a multi-step operation at the break** (drain; Helm install, upgrade, rollback): handled as
   after a crash ([D-11](#d-11-crash-stance)): it stops at the interruption and stays a pending entry
   of its journal; after the reconnect the existing banner offers Finish or Dismiss against the
   cluster as it is then. Nothing continues by itself.

**S3 amended by the owner (2026-10-16, F13):** the connection's state shows as a dot, never a
word ("Nur Punkt, nie Wort"), in the top bar and, by the same rule, the status bar, for every
state; the dot's tooltip names the state with the engine's reason, since when and the next
attempt (no credential: those fields hold none). Connected is the Running containers' green
(`colorContainerRunning`, #3dd68c dark, #177f4a light since 2026-10-16, #1a8f53 before), in place of the status green (olive);
Reconnecting blinks slowly in that green, 500 ms full and 500 ms dimmed, for at least three
intervals (3 s) even when the reconnect succeeds at once, so that the user has a chance to see it,
and longer while it lasts; Login expired and Connection failed stop the blink at once and show
red, Not connected stays grey; the amber goes. Every dot of that cluster blinks in step (the top
bar's, the status bar's, its hotbar tile's, its navigator rows', its welcome-page row's), the other
clusters' do not. A click on any dot of a cluster opens its connection log
([D-66](#d-66-the-connection-log)), and not the row or tile it sits in. The banner
("The connection to … is interrupted · since … · Reconnect now") stays as built. **Built
2026-10-16** ([`conndot.go`](../../internal/ui/fyneui/conndot.go): one blinker on the main
goroutine, a ticker of 500 ms that runs only while a dot blinks; macOS's Reduce motion through
[`reducemotion_darwin.m`](../../internal/ui/fyneui/reducemotion_darwin.m); `TestDotsBlinkWhileReconnecting`,
`TestBlinkLastsThreeIntervalsAfterAQuickReconnect`, `TestLoginExpiredStopsTheBlink`,
`TestReduceMotionKeepsTheDotSteady`, `TestDotTooltipNamesTheState`,
`TestDotClickOpensTheConnectionLog`, `TestSectionDotOpensTheLogNotTheRow`; renders
`connection_interrupted_*.png` (the full phase), `connection_interrupted_dim_*.png`,
`connection_reduce_motion_dark.png`, `navigator_reconnecting_*.png`). Measured, the green dot's
contrast (WCAG): dark 8.9:1 on the navigator, 5.5:1 on its selected row, 7.5:1 on a hovered one;
light 3.4:1 on the navigator, 3.0:1 on a hovered row and 2.5:1 on the selected row, under the 3:1
of non-text contrast there (the olive it replaces read 1.9:1). **Decided (owner, 2026-10-16):** the
light Running green one step darker everywhere, #177f4a for dots and the containers' squares alike:
3.1:1 on the selected row, 4.1:1 on the navigator, 3.5:1 on the header's background (computed).

**The lead's decisions of F13, for the owner to confirm:**

| Question | Options weighed | Chosen | Why |
|---|---|---|---|
| Connecting | steady green; blinking as Reconnecting | blinks as Reconnecting | Dots only: "being established" reads the same everywhere |
| The dimmed phase | the dot gone; dimmed | 30 % alpha | The dot keeps its place and the layout does not move |
| Where the 3 s minimum lives | the engine's state; the window | the window: from the moment it first saw the (re)connect, its dots blink 3 s even when the engine says Connected again | Agents read the true state in `ui_state`; the minimum is for the eye |
| macOS's Reduce motion | ignore it; a steady dot | no blinking, a steady half-alpha green while (re)connecting; read when a blink starts, so a change shows with the next one | The setting asks for no blinking; a property read per blink costs nothing, an observer would |
| The welcome page's row | keep the state pill ("connected", "reconnecting"); a dot | the dot, also while connecting | The owner's rule: a dot, not a word, for the cluster's state |
| The status bar | "demo · Connected"; the context alone | the context alone beside its dot | The same rule as the top bar's |
| A hotbar tile under the pointer on its dot | lit; unlit | unlit while the pointer is on the dot: the dot is a target of its own, with its own tooltip | A tile and its dot open different things |

The options the owner weighed per item were in the interview's brief: for S2 the heartbeat's
cadence and whether a broken watch alone interrupts (no: it only checks); for S3 a modal dialog
(no: the window stays usable) or the status bar alone (no: too easy to miss); for S4 writes queued
until the reconnect or sent anyway (no: an unknown outcome); for S5 reconnecting only on a click
(no) or giving up after a while (no: without end while the window is open); for S6 running the
plugin at every 401 (client-go's behaviour, a pile of browser tabs) or never by itself (no: silent
plugins should renew); for S7 one rule for every stream (no: by their kind); for S8 resuming the
operation after the reconnect (no: the cluster may have changed; D-11's road).

**The lead's defaults (2026-10-15, built, the lead's decisions for the owner to confirm):** (a) a
request that is not a stream ends after 30 s; a write in flight when the connection drops ends with
"outcome unknown: … the view shows the object's state once reconnected"; watches, logs, exec,
attach and port forwards are not bounded. (b) Agents see it in `ui_state`'s `connection.state`
(`interrupted`, `loginExpired`, with `since`, `nextAttempt` and `reason`); their reads come from the
cache with a note naming the time the data is from, their writes are refused at once with S4's
reason. (c) The built-in agent's turn reads on, its writes are refused. (d) Local terminals and the
MCP server (127.0.0.1) are not affected. (e) A test freezes the control plane and asserts: the state
interrupted within the heartbeat window, the window never blocks (the main goroutine's work within
100 ms throughout), writes disabled, the reconnect after the unfreeze, the log resumed with its
marker. Once the cluster answers, the informers relist or watch on by themselves; the cache, the
tabs and the view stay.

**S1, measured (2026-10-15/16; a kind cluster of its own, `cogpit-standby`, deleted afterwards;
[`standby_freeze_test.go`](../../internal/ui/fyneui/standby_freeze_test.go), tag `standby`):**

| Case | What happened | The main goroutine | Found |
|---|---|---|---|
| The node paused 120 s, the window driven offscreen (the drawer, a log, view changes every 5 s, a pod shell opened while frozen) | The state said `connected` throughout; the watch streams ended together after about 40 s ("http2: client connection lost", HTTP/2's health check: the owner's log line); every new request waited 10 s for a TLS handshake; the pod shell's WebSocket dial hung in its TLS handshake for as long as the freeze lasted; after the unpause the view synced again within 80 ms | worst 148 ms per step, a full 1440×900 render included; `UIState()` from another goroutine worst 13 ms | No hang of the window. The goroutine dump at the freeze's end: no goroutine waiting on an engine lock; two watches waiting for a connection, the shell's dial in `crypto/tls.(*Conn).clientHandshake` for a minute |
| A credential plugin (the kind certificate, valid 20 s) that waits once its credential expired, as a browser login nobody answers | From the credential's expiry every request of the connection waited behind client-go's exec authenticator, whose lock the plugin's run holds (`refreshCredsLocked` → `os/exec.(*Cmd).Run`, ten informers' watches in `getCreds`): the views opened from then on stayed empty, without an error, the state `connected`, for as long as the plugin ran | worst 231 ms per step | The likely hang the owner saw after sleep: a login whose token expired during the night, a plugin waiting (a browser tab, MFA), every view and action silently waiting behind it, nothing saying so |

Measured again with the build (2026-10-16, the same cluster and test): the paused node made the
connection `interrupted` 14 s after the pause, and after the unpause the view synced within 76 ms;
the waiting plugin made the login `loginExpired` once its run had waited 30 s ("the login program
has not answered for 30s"), and when it answered the connection came back by itself, the views
synced.

Not reproduced offscreen: Fyne and GLFW after a real sleep (the offscreen driver has no display to
lose); the owner's check in a real window is open. What the measurements proved from the code and
the dumps is fixed: the plugin's wait (the gate holds the informers' requests in front of the
plugin, a heartbeat that has not returned after 30 s makes the login expired and says so, the
login guard stops client-go from running the plugin again on every 401), requests on dead
connections (the request timeout, `ResetConnections`), and the shells' dials (ended with the
interruption).

**The owner's log (Services 401 every ~5 minutes), read in the code, not verified against his
cluster:** every client of a connection shares one HTTP client and one exec authenticator (client-go
keys it by the plugin's configuration: Helm's and the shells' copies of the configuration share it),
so no informer holds a credential of its own. A watch that stays open is never authenticated again,
so pods and namespaces "working" can be long-lived watches still delivering events, while the
Services scope listed again (its watch ended, or the Prometheus detection acquired it anew for each
drawer it opened, kept warm 5 minutes) and met the rejected credential: client-go re-runs the
plugin on a 401, and a plugin that hands back its cached, rejected token (a token cache whose own
clock still finds it valid, clock skew) keeps every new request failing. Since D-58 the
authenticated heartbeat meets that 401 too and makes it "Login expired" with its banner instead of
one kind's silent error.

**Built 2026-10-15/16:**

- The cluster side ([`cluster/health.go`](../../internal/cluster/health.go), [architecture §6](architecture.md#6-cluster-connections-internalcluster)):
  the request timeout, the gate (outer, in front of the plugin; inner, in every transport made from
  the configuration), the login guard, the tracked dialer (`ResetConnections`), the probes
  `Heartbeat` and `Reachable`.
- The engine's connection watch ([`engine/connwatch.go`](../../internal/engine/connwatch.go),
  [`connwatch_streams.go`](../../internal/engine/connwatch_streams.go), [`logsresume.go`](../../internal/engine/logsresume.go),
  [architecture §4.3](architecture.md#the-connections-states-d-58)): the states `interrupted` and
  `loginExpired` with `since`, `nextAttempt`, `reason`, `loggingIn` and `needsApproval`; the three
  signals (`Session.NoteWake`, `NoteNetworkChange`, `cache.OnStreamError`) and the heartbeat; the
  backoff; `cluster.reconnect`, `cluster.login` (GUI only), `dock.reconnect`; writes disabled
  (`Command.Enabled`, and again after a confirmation); a pod's followed log and a workload's
  resuming with the marker; port forwards waiting; pod shells, attach and node shells ended with
  `DockTab.Interrupted`; drains and Helm actions stopped (Helm's through `Session.OperationContext`;
  a drain, since it runs as a job the same night, through its job's stop, `errJobInterrupted`, D-63); agents' reads
  noted (`Session.StaleNote`).
- The window ([`connbanner.go`](../../internal/ui/fyneui/connbanner.go), [`dockreconnect.go`](../../internal/ui/fyneui/dockreconnect.go),
  [`connhints_darwin.m`](../../internal/ui/fyneui/connhints_darwin.m)): the dot and the state text in
  the top bar and the status bar (since 2026-10-16 the dot alone, S3 amended above), the banner with Reconnect now or Log in again, a shell tab's row
  with Reconnect, macOS's wake notification and network path monitor.
- Tests: `TestTwoFailedHeartbeatsInterruptTheConnection`, `TestReconnectBacksOffAndComesBack`,
  `TestAWakeChecksAtOnce`, `TestALastingUnauthorizedExpiresTheLogin`,
  `TestAPluginThatWaitsExpiresTheLogin`, `TestAPodLogResumesAfterTheInterruption`,
  `TestAShellEndsWithTheConnection`, `TestUIStateStaysAcrossAnInterruption`,
  `TestConnectionCommands`; on the real client-go stack against the fake API server behind
  `testkit.FreezableProxy`: `TestStandbyFrozenControlPlane`, `TestStandbyRevokedLogin` and, with the
  window, `TestStandbyWindowStaysUsable` (interrupted within the window, the main goroutine's work
  at worst 1 ms, 8 ms under the race detector, the log resumed after the marker with the line
  written meanwhile, none twice); renders `connection_interrupted_dark.png`, `…_light.png`,
  `connection_login_expired_dark.png`.

**The lead's decisions of the build, confirmed (owner, audit of 2026-10-16)** as built, every row but the
reasons' wording, which the audit did not ask:

| Question | Options weighed | Chosen | Why |
|---|---|---|---|
| When the second check runs after a failed one | 15 s later (the heartbeat's cadence, up to 50 s to detect); at once; after 2 s | after 2 s; at wake one failed check interrupts at once | One lost packet is no interruption, and 15 s more of a silent window is what the owner met |
| How a network change is noticed | wake and heartbeat only; SCNetworkReachability (deprecated); Network.framework's `nw_path_monitor` | `nw_path_monitor` beside the wake notification, one each per process ([`connhints_darwin.m`](../../internal/ui/fyneui/connhints_darwin.m)) | About 30 lines; a VPN or Wi-Fi that comes back reconnects at once instead of within the backoff. A change checks at once but one failed check does not interrupt (path updates come in bursts) |
| What a reconnect attempt sends | the authenticated heartbeat; an anonymous `GET /version` | anonymous (`Reachable`, no credentials, a 401 or 403 counts as an answer); one authenticated heartbeat once the server answers | An authenticated attempt would run an expired login's plugin at every step of the backoff: S6's "never a plugin run in the backoff" |
| What the connection's requests do while interrupted | fail at once; wait; hold the informers' and refuse the rest | hold the informers' lists and watches at a gate in front of the plugin, refuse every other request at once | Failed lists put client-go's informers into a backoff of up to a minute after the reconnect; held ones go out at once. Nothing else waits on a dead connection, and no held request makes the plugin run |
| How "one automatic run" of the plugin is kept | Cogpit runs the plugin itself (client-go's exec authenticator rewritten, certificate plugins included); a guard below client-go's authenticator; nothing | the login guard: the first 401 after a working request reaches the plugin, later ones end as `ErrLoginRejected` until a request works again or the user logs in | Small and on client-go's own path; the rewrite would re-do D-26's binding for every plugin form. A credential that expires by its time is still renewed by client-go on the next request (once per expiry) |
| What Log in again does | a full new connect (the owner's wording); renew on the open connection | renew on the open connection: the guard armed, one heartbeat that lets the plugin run with its browser for as long as the user takes; a full connect only when the plugin's approval is needed again (its program changed: the picker shows it, D-26) | Same effect for the user (the plugin runs, with its browser), and the cache, the tabs, the view and the dock stay; a full connect would reset the view and close the port forwards |
| While the login is expired | probe on; probe nothing | no probe at all until Log in again; a plugin run still waiting from before that ends with a working login reconnects by itself | Every authenticated probe could run the plugin |
| The colours | amber for both; amber and red | interrupted amber (Cogpit handles it), login expired red (it needs the user); replaced 2026-10-16 by the owner (S3 amended): blinking green, login expired red | |
| The reasons' wording | "Connection interrupted"; lowercase | "connection interrupted", "login expired: log in again first" (Go's error convention; the menus show "Delete — connection interrupted") | |
| A node shell at the interruption | its pod deleted with the session; kept | kept: dock.reconnect opens a new root shell in the same pod while it runs (it ends by its deadline or with its tab); the pod's deletion after a session's end waits for the connection's verdict (at most 15 s) | No second privileged pod. dock.reconnect is mutating as pod.shell, destructive for a node shell (D-37), so agents ask |
| What the views show while interrupted | the informers' errors; nothing | the cache's rows without an error | The banner says what is wrong; "use of closed network connection" in every view says nothing more |
| Which connects the journals and banners follow | every change of state; real connects only | drains and Helm actions offer again after the reconnect (S8); the drafts banner and the node shell sweep take an interruption's end as no connect | The drafts were offered already; the sweep's toast speaks of "an earlier run" |
| CI's test of (e) | `docker pause` on CI's kind cluster; an in-process proxy | the fake API server behind `testkit.FreezableProxy` (CI); the `docker pause` test only by hand, build tag `standby`, on a cluster of its own, never in parallel | CI's kind job shares one cluster with other tests |
| Agents' note | in every result; read commands only | a text content of the MCP result and a line after the built-in agent's tool result, for read commands | A UI command's result is the UI state, which says it already |

**A cluster the window does not show (the owner's F16, [D-69](#d-69-several-clusters-connected-at-once)
item 4, built 2026-10-16):** each connected cluster has its own watch and renews its login as above
(S6) by default. Its cluster setting "Renew the login while the cluster is not shown" off, it holds
its credential plugin while another cluster is shown: a login that expires by its time, or that the
API server rejects, is not renewed and **lapses** (`UIState.Connections[].loginLapsed`; the
connection stays `connected`, not red: whether a renewal would work is not known); its heartbeat
pauses, writes are disabled with "login lapsed: show the cluster to renew it", a log that has to open
again waits with a notice line. Showing the cluster renews it as S6 renews the shown cluster's (one
run of the approved plugin, Login expired when that does not help within 30 s), and Log in again
(`cluster.login` with the context) renews it where it is. How the plugin is held, and the lead's
decisions: [D-69 › Stage 3](#d-69-several-clusters-connected-at-once).

**The selection bar's Delete while the connection is interrupted (the lead, 2026-10-16 night;
changed by the owner, audit of 2026-10-16).** The header's selection bar greys its Delete while the
shown cluster's connection is not `connected` and gives it back with the connection, instead of the
engine's refusal after a click. Options: (a) active, refused after the click; (b) greyed, the
tooltip unchanged (built the night of 2026-10-16); (c) greyed, the tooltip naming the reason. The
owner chose (c): greyed, its tooltip "Delete — connection interrupted", as the row menus say it.
**Built 2026-10-16:** while greyed, the bar asks the engine off the main goroutine why
`object.deleteMany` cannot run on the selection (its `Enabled`, the write gate's reason) and its
tooltip says it in the row menus' form (`disabledLabel`, shared with them): "Delete — connection
interrupted", "Delete — login expired: log in again first", and for a cluster that is not
connected whatever the engine answers for it, as the selection's row menu shows it; with the
connection back it says "Delete the selected objects…" again, and a tooltip on screen follows
(`tipButton.setTip`). The engine's reason for `object.deleteMany` while interrupted is checked
in `TestConnectionCommands`. The build's choices, for the owner to
confirm: the engine is asked once per context and state, not per click or row change; a shown
cluster whose login lapsed greys Delete too, "Delete — login lapsed: show the cluster to renew it"
(D-69 item 4: the engine refuses its writes until the renewal), the bar following the connected
clusters' list for it; Delete stays active for every other reason it may be refused (permissions,
the 500 objects), as before, the engine refusing after the click
([`tablecheck.go`](../../internal/ui/fyneui/tablecheck.go) `selectionBar.followConnection`;
`TestSelectionBarDeleteNamesTheReason`, `TestRowCheckboxes`; PNGs
[dark](../../internal/ui/fyneui/testdata/selection_interrupted_dark.png),
[light](../../internal/ui/fyneui/testdata/selection_interrupted_light.png)).

**Open:** the owner's check in a real window after a real sleep (Fyne and GLFW after sleep were
not reproducible offscreen); Windows and Linux have no wake or network signal (the heartbeat and
the broken streams find an interruption there); the security gaps
[H-101](../security/trust-boundaries.md#h-101-a-credential-plugin-runs-again-during-a-connection-without-being-checked-again)
and [H-102](../security/agents-and-mcp.md#h-102-an-agent-may-act-after-the-reconnect-on-what-it-read-during-the-interruption).

## D-59: A Secret's values edited in the drawer

**Decided (owner, 2026-10-15, his audit answers, (2)):** a Secret's values are editable in the
drawer as a Config Map's are ([D-55](#d-55-a-config-maps-values-edited-in-the-drawer)), in Cogpit's
window only, GUI-only as Create Secret ([D-19](#d-19-editing-and-creating-secrets-from-yaml)): no
engine command path an agent, the CLI or an MCP client reaches writes a Secret's value. The editor
per key comes after Show (the drawer's Show and Hide of a Secret's values); Save shows a **masked
diff** (the keys changed, added or removed and their sizes, never a value) and a dry run; no value
in logs, toasts, the audit log (its line names the Secret and the keys only) or a crash report. The
layout is the Config Map's (the key on its own line, the editor below, one Save for all edits,
conflicts as built), with Save rebasing by itself when none of the edited keys changed underneath
(D-55's (9)); a Secret's unsaved values are never on disk, memory only (D-55's (10)). Lens's Secret
drawer editing is not captured (inferred).

**Built 2026-10-15 (night run).** Engine ([`engine/secretdataedit.go`](../../internal/engine/secretdataedit.go),
[`commands/secretdataedit.go`](../../internal/engine/commands/secretdataedit.go)):
`secret.reveal` takes `whole` (the value uncut, for the editor; GUI-only as before);
`secret.reviewData` (read, verbs `list`, `watch`, `update secrets`) returns the masked diff
(`SecretKeyChange`: key, `changed`/`added`/`removed`, bytes before and after) of the Secret with the
given values against the cached one and, when something changes and the values are based on the
cached version, the server's dry run; given `from` (each edited key's value at the base version,
compared base64-encoded, nothing decoded) it rebases onto a newer version none of those keys
changed in, as `configMap.reviewData`; `secret.saveData` (mutating) writes the cached Secret with
the values set base64-encoded in `data`, `managedFields` dropped, a dry run, then the update with
the base version as precondition; a newer version is a `DataConflictError` with the masked changes
(`Changes`) against it, never a value, written over only by a save that names it. Both are marked
GUI-only (`ErrSecretDataGUIOnly`, which `errors.Is` `ErrSecretGUIOnly`): `Dispatch` refuses every
other origin before `Enabled`, `Refuse` and `Run` refuse again, and the engine functions check the
origin themselves; a save refuses a Secret the drawer does not show, a key the data does not hold, a
binary value, an immutable Secret and a Helm release record (`helm.sh/release.v1`) before anything
is sent, naming keys only; an API server's message that quotes a value, old or new, is scrubbed
(`scrubSecretWrite`). The audit line of an agent's refused call keeps the Secret, the version, the
keys of `data` and `from` with each value `<redacted>`, and the names of any other parameters
(`auditSecretDataParams`). The toast says "Secret <namespace>/<name> saved". GUI
([`fyneui/secretdata.go`](../../internal/ui/fyneui/secretdata.go)): the Data section keeps each key
above its masked value with Show; where `secret.saveData` is enabled, Show reads the whole value
into the Config Map's editor row (yamlview, plain scalars as strings, 3 to 12 lines, the grip) with
Hide at the key's end, the editor's ⌘C and ⌘X going to the clipboard for Secret values (yamlview's
`OnCopy`; concealed, cleared after 60 seconds); Revert and Save at the end of the data once a value
is in an editor; Save opens a review over the window: the masked diff, one line per key ("changed,
28 B → 27 B"), the dry run's verdict, Save or "Save over version N"; a rebase says so. Tests:
`TestSecretDrawerDataSave`, `TestSecretDrawerDataIsGUIOnly`, `TestSecretDrawerDataConflictAndRebase`,
`TestSecretDrawerDataRefusals`, `TestSecretDrawerDataDisabledOnTheDemo`, on kind
`TestKindSecretDataSave` (a real dry run, save, rebase, conflict and the agent's refusal); GUI
`TestSecretDrawerEditAndSave`, `TestSecretDrawerHide`, `TestSecretDrawerKeepsUnsavedInMemory`,
`TestSecretDrawerRebaseAndConflict`, `TestSecretDrawerReadOnly`, `TestSecretEditorCopiesAsASecret`;
yamlview `TestOnCopyTakesTheCopy`. PNGs: [drawer dark](../../internal/ui/fyneui/testdata/secret_edit_dark.png),
[light](../../internal/ui/fyneui/testdata/secret_edit_light.png), [masked diff dark](../../internal/ui/fyneui/testdata/secret_save_dark.png),
[light](../../internal/ui/fyneui/testdata/secret_save_light.png).

**The owner's audit decision (2026-10-16, N152 (b)): keys too**, as for a Config Map
([D-55](#d-55-a-config-maps-values-edited-in-the-drawer), where the engine's rules, the danger, the
rebase and the window's look are recorded). **Built 2026-10-16.** `secret.reviewData` and
`secret.saveData` take `add` (keys with their values as text, saved base64-encoded) and `remove`
beside `data`, still GUI-only for every parameter: an agent, the CLI and an MCP client are refused
before anything is read, asked or sent, and the audit line of a refused call keeps the keys of `add`
with each value `<redacted>` and the keys of `remove` (`auditSecretDataParams`). The masked diff
lists an added key ("added, 27 B") and a removed one ("removed, 38 B", in the error colour), never a
value; the review's block above it names the keys removed with the Secret, its namespace and its
context; a save that removes a key is destructive and the review is its confirmation, its detail
"Removes key a and adds key b. The values are not shown." (`SecretSaveDetail`). A key's value need
not be revealed to remove it; a binary value cannot be edited but can be removed. An added key's
editor is the revealed value's (yamlview, Hide masks it, copies go to the clipboard as a Secret's
value); added and removed keys the drawer moves away from stay in the window's memory only
(`drawer.secretUnsaved`), never on disk, and come back masked. Errors, toasts, results and the log
name keys only. Tests: `TestSecretDrawerDataAddsAndRemovesKeys`, `TestSecretDrawerDataKeysConflict`,
`TestSecretDrawerDataKeysAudit`, on kind `TestKindSecretDataSave` (a key added and one removed
through a real dry run and save, an added key that appeared meanwhile shown as the conflict); GUI
`TestSecretDrawerAddsAndRemovesKeys`. PNGs: [drawer dark](../../internal/ui/fyneui/testdata/secret_keys_drawer_dark.png),
[light](../../internal/ui/fyneui/testdata/secret_keys_drawer_light.png), [review dark](../../internal/ui/fyneui/testdata/secret_keys_review_dark.png),
[light](../../internal/ui/fyneui/testdata/secret_keys_review_light.png).

| The lead's decision for the owner to confirm (2026-10-16, keys too) | Options | Choice | Reason |
|---|---|---|---|
| Who adds and removes a Secret's keys from the drawer | (a) the window only, as its values (`add` and `remove` on the GUI-only commands); (b) agents through the registry | (a) | The owner's "the window's only" for a Secret's values: an added key carries a value, and a removal is a loss an agent must not cause unseen |
| Removing a key whose value was never shown | (a) allowed, the review naming key and size; (b) only after Show | (a) | Removing needs no value; Show would put a value on the screen for no reason |
| The rebase of a removal | (a) only with the removed key's value at the base, which the window has when the value was shown; else any change of the Secret is the conflict; (b) by size; (c) never | (a) | The rebase compares the encoded texts and decodes nothing; a size match says nothing about a value. A removal of a value never shown is therefore conservative: the conflict, then "Save over version N" |
| Removing a binary value | (a) allowed; (b) refused as its editing is | (a) | A removal needs no text; the masked diff names its size |
| The footer | (a) Add Key wherever Save is possible, Revert and Save once a value is in an editor or a key is added or removed; (b) as before (nothing until a value is in an editor) | (a) | Adding a key needs an entry point before any value is shown; the drawer of a Secret nobody edits shows one low-importance button more than before |

**The owner's audit decision (2026-10-16, the copy from the editor, (c)): a copy icon over the
editor. Built 2026-10-16.** Every value in an editor, an added key's included, has D-25's copy icon
(the chip-coloured plate, the checkmark for 1.5 s after a copy) in its key's line, between Hide and
the remove button: right above the editor's right end, so it covers none of the editor's text. A
click copies the editor's whole value as typed so far through the path ⌘C and ⌘X take
(`secData.copied`, `shell.copyText(…, secret=true)`: concealed for clipboard managers, cleared after
60 seconds if the clipboard still holds it), never the plain clipboard, and the toast says "Copied.
The clipboard is cleared in 60 seconds." (a copy's toast, outside the notification history). A
masked value has no icon; a Config Map's editors have none. The build's choices, for the owner to
confirm: the icon is always shown with its editor (not only under the pointer, as the drawer's
values have it: an editor takes the pointer for its text), its tooltip "Copy the whole value"; an
empty value copies nothing ("The value is empty: nothing was copied."), so that the click does not
empty the clipboard. ([`secretdata.go`](../../internal/ui/fyneui/secretdata.go) `copyIcon`,
[`copy.go`](../../internal/ui/fyneui/copy.go) `newPinnedCopyIcon`; `TestSecretEditorCopyIcon`,
`TestConfigMapEditorsHaveNoCopyIcon`; PNGs [drawer dark](../../internal/ui/fyneui/testdata/secret_edit_dark.png),
[light](../../internal/ui/fyneui/testdata/secret_edit_light.png), an added key's
[dark](../../internal/ui/fyneui/testdata/secret_keys_drawer_dark.png),
[light](../../internal/ui/fyneui/testdata/secret_keys_drawer_light.png).)

### The lead's decisions (night run 2026-10-15/16)

**Confirmed (owner, audit of 2026-10-16)** as built, every row but two he changed: adding and removing keys,
and copying from the editor (its row).

| Question | Options | Choice | Reason |
|---|---|---|---|
| The GUI-only path | (a) two commands of their own, `secret.reviewData` and `secret.saveData`, marked GUI-only (refused in `Dispatch` before `Enabled`), refusing again in `Refuse`, `Run` and the engine; (b) no command, the window calling the engine directly; (c) the Secret's Edit tab | (a) | Create Secret's and Show's pattern: everything is a command (AGENTS.md rule 1), the MCP tool exists and tells an agent to ask the user, a refused call is audited; (b) would be the one write outside the registry; (c) leaves the drawer |
| Where the editor's whole value comes from | (a) `secret.reveal` with `whole`; (b) a command of its own; (c) the drawer model | (a) | One GUI-only read of a value; the cut (4 KiB, 40 lines) stays for the read-only text, whose widget wraps every character; (c) would put values into the model agents read |
| Show where Save cannot work (the demo cluster, no `update secrets`, immutable, a Helm release record) | (a) the value as before, read-only text with its copy icon, the reason under the data; (b) a read-only editor | (a) | A value that cannot change keeps D-25's copy icon and its wrapping; the editor appears where typing can lead to a save. The existing renders and tests of Show stay as they are |
| Adding and removing keys | (a) values only; (b) keys too | (a), since the owner's decision of 2026-10-16 (b) | Not trivial: a key row with the API server's key rules, a removal is a loss the review would have to weigh; the Edit tab adds and removes keys. The masked diff's model names added and removed keys for when it does. **Decided (owner, audit 2026-10-16, N152): (b)**, built (above) |
| Hide on an edited value | (a) masks it, the typing kept until Save or Revert; (b) Hide disabled while edited; (c) the edit dropped | (a) | The user can mask a value while it waits, and loses nothing; the masked line says "(edited, not saved)" |
| Unsaved values when the drawer moves | (a) the window's memory until Cogpit quits, a toast, back masked when the Secret is shown again; (b) dropped | (a) | The owner's "memory only"; back masked, so no value shows without a click |
| An unedited value revealed at an older version when the Secret changes | (a) masked again, gone from the window's memory; (b) kept | (a) | The rule of 2026-10-12 (R-6): a rotated credential's old value must not stay to be copied. An edited value stays, based on the version it was edited at |
| What the masked diff shows | (a) the key, what changes, the sizes before and after; (b) a hash of each value too | (a) | The owner's "keys and sizes"; a hash of a short password can be guessed offline |
| Copying from the editor | (a) ⌘C and ⌘X through the clipboard for Secret values (concealed, cleared after 60 seconds), no copy icon over the editor; (b) the plain clipboard; (c) as (a), plus a copy icon over the editor that copies the whole value | (a), since the owner's audit (c) | D-25's rule for every copy of a Secret's value; the Config Map's editors have no copy icon either. **Changed (owner, audit of 2026-10-16): (c)**, the copy icon over the editor beside ⌘C and ⌘X, every copy through the clipboard rule for Secret values; built 2026-10-16 (above) |
| An audit line for the window's own save | (a) none, as for every write of the user's (a Secret's Edit tab included); (b) a line naming the Secret and the keys | (a) | The audit log is agents' (D-2); an agent's refused call has its line, with the Secret and the keys only. (b) is one `AuditUser` call if wanted |
| A save of a Secret the drawer does not show | (a) refused; (b) allowed | (a) | Defence in depth beside the GUI-only mark: a value is written only from the drawer that shows its Secret |
| The footer | (a) Revert and Save once a value is in an editor; (b) always | (a); since 2026-10-16 Add Key wherever Save is possible (above) | The drawer of a Secret nobody edits looks as before |

### What it does not cover

- **kubectl's last-applied annotation is dropped by a save, live:**
  [H-103](../security/credentials.md#h-103-a-secret-saved-from-cogpit-loses-kubectls-last-applied-annotation).
- **The values stay in the window's memory until the garbage collector takes them, live.** A
  revealed value, the editor's text, its undo and the base values are Go strings; the decoded bytes
  the engine handles are zeroed, the strings cannot be (as for Show since 2026-10-12, H-8).
- **A crash loses a Secret's unsaved values, by design:** never on disk; the keys added and removed
  too.
- **A removal of a value never shown never rebases, by design:** any change of the Secret since the
  removal began is the conflict, resolved by "Save over version N" after the review against it.
- **Pointer events over a scrolled-out editor**, as D-55 records for a Config Map's.
- **Lens's Secret drawer editing is inferred**, not captured; adding and removing keys there too.

## D-61: Cogpit Metrics, the bundled metrics stack

**Withdrawn by the owner (2026-10-16), removed the same day.** On seeing Cluster Settings › Cogpit
Metrics (his words, translated): "Remove it completely. I never intended that: we use charts that are already installed,
try to find them and make their information about the cluster useful to the user. We do not
install them!" The rule is in [AGENTS.md](../../AGENTS.md#never-surprise-a-cluster-outranks-feature-work):
Cogpit never installs anything into a cluster for its own features (metrics, agents, exporters);
it finds and uses what is installed, and installing is the user's own action on their charts.

**What it was** (the lead's design of the night run of 2026-10-15/16, built that night, released in
v0.16.0 and v0.17.0): Lens's Lens Metrics page as Cluster Settings › Cogpit Metrics, three switches
(Prometheus, kube-state-metrics, node-exporter) and Apply, which installed prometheus-community's
prometheus chart 29.36.1, embedded in the binary with its SHA-256, as one Helm release per switch
in a namespace `cogpit-metrics` Cogpit created, with the charts' ClusterRoles and
ClusterRoleBindings; detection ranked that Prometheus first.

**Removed:** the section and its window (`fyneui/clustersettings_metricsstack.go`, its four
renders), the commands `cluster.metricsStack` and `cluster.setMetricsStack`
(`commands/metricsstack.go`), the engine's state and plan (`engine/metricsstack.go`), the embedded
chart archive and its loader (`helm/metricsstack.go`, `helm/bundle/`, 91,710 bytes), `make
metrics-bundle` with `hack/metricsbundle`, the Renovate rules for the chart's version, its
attribution, the kind and real-cluster trials, and detection's preference for Cogpit's own
Prometheus (score 110). The night's decisions about the stack (N183 to N192: names, one release
per switch, the embedded chart, storage, values, the window's confirmation, the namespace,
detection first, Pod Security, the rights check) are moot. Closed with it:
[H-105](../security/privilege-footprint.md#nothing-installed-for-cogpits-own-features), the stack's
cluster-wide reads gathered in one namespace.

**What stays** (the night's work on reading an installed Prometheus better, N193 to N197, in
[D-43](#d-43-where-the-nodes-disk-comes-from) and [architecture §9](architecture.md#9-metrics-internalmetrics)):
the provider's query format (auto, Prometheus Operator, cAdvisor only), the empty-container
filter, HTTPS through the service proxy for the setting and for detection's candidates, a node's
network and filesystem from an installed node-exporter with the denylist of virtual interfaces,
and the two sources in one query joined by `or`. Detection finds the prometheus chart's server
under any release's name since the removal (a Service named `<release>-prometheus-server`, score
80, as `prometheus-server` before; `TestPrometheusChartServerOfAnyRelease`), so a stack v0.16.0 or
v0.17.0 installed is read like any other Prometheus; it stays in the cluster as the user's own
releases, which the Helm Releases view lists and uninstalls. Where Cogpit finds no provider, the
charts say so, as before.

## D-63: Long commands as background jobs

**The question (G4, the night run of 2026-10-15/16):** the actions on a selection ran their loop
inside the command's call: Delete of 500 objects took about 100 seconds at the cluster
connection's 5 requests per second, with nothing shown until all were done and no way to stop
them but quitting, and an agent's call waited as long. Reconcile with Source already ran as a
background job in a dock tab of its own (A91, the owner's decision of 2026-10-14 evening,
[D-33](#d-33-the-flux-actions)); the night's scope was to run every long command that way, for
every origin, with progress in the dock and Cancel, the confirmations unchanged.

**Built 2026-10-16 (night run; the lead's decisions below, confirmed by the owner in the audit of 2026-10-16
where it asked).** Every
action on a selection that loops with one request per object runs as a job when it acts on more
than ten objects (`engine.DirectSelection`): `object.deleteMany`, `workload.restartMany`,
`flux.reconcileMany`, `flux.suspendMany`, `flux.resumeMany`, `flux.resetMany`,
`argocd.application.syncMany` and `argocd.application.refreshMany`; Reconcile with Source stays a
job for every size, and so does Drain since later the same night ([below](#drain-as-a-job-built-2026-10-16-night-run)). The confirmation is asked in `Dispatch` before the command runs, exactly as
before (destructive commands for every origin, D-2 for agents); then the command starts the job
and returns at once with its dock tab (`ManyJobResult`: `job`, `targets`). The job's steps are the
objects, in the confirmed order; each step goes from pending to running to done ("Deleted",
"Restarted", "Reconcile requested", …) or failed with the object's error, and the others go on;
nothing is retried. The job keeps the command's context values (`JobSpec.Parent`): its origin and
the confirmation with the UIDs it bound, so each step acts on exactly the object the user
confirmed. Its outcome counts what was done, what failed and what was not attempted ("Deleted: 11
of 12 objects; 1 failed: the job's steps say why."), its steps' counts are engine state with the
steps (`JobTab.Counts`), and a toast says how it ended when it ended by itself.

Cancel: `job.cancel` (a UI command, an MCP tool) and the job tab's Cancel button stop a job and keep
its tab; closing the tab (`dock.close`) stops it too, as before. A request under way finishes; the
steps after it are marked stopped, "Not attempted: cancelled", and the outcome begins with why the
job stopped ("Stopped: cancelled.", "Stopped: cancelled by an agent.", "Stopped: the connected
cluster changed.") and goes on with what was done and what was not. `job.cancel` waits until the
tab shows the end, at most 10 seconds, and returns the tab. A job an agent started gets a third
audit line when it ends: phase `ended`, ok when it is done, else its outcome as the error, after
the command's `requested` and `done` lines (`done` saying the job started); the Audit Log tab shows
it as "job ended: …".

A crash ends a job like a stop (D-11): nothing is journaled, and what it wrote stays. Each
object's action is one request (Argo CD's Sync at most three of the same write), so no object is
left half-done, only objects done and objects not attempted, which the lists show; each command's
comment says why that is harmless for it ([`commands/manyjob.go`](../../internal/engine/commands/manyjob.go),
[`selection.go`](../../internal/engine/commands/selection.go), [`restartmany.go`](../../internal/engine/commands/restartmany.go),
[`flux.go`](../../internal/engine/commands/flux.go), [`argocd.go`](../../internal/engine/commands/argocd.go)).
A step's request that does not end (a dead connection) holds its step until the request
timeouts of the night's connection work (D-58) end it with an error, which the step shows; the
loop knows nothing else of the connection.

In the window the job tab ([`fyneui/jobtab.go`](../../internal/ui/fyneui/jobtab.go)) shows, while
the job runs, its progress from the counts ("9 of 24 done, 1 failed") and Cancel at the header's
right end, and its list follows the step under way until the user scrolls it. Engine:
[`engine/jobs.go`](../../internal/engine/jobs.go) (`JobSpec.Parent`, `JobCounts`,
`Session.CancelJob`, `JobCancellable`, the `ended` line), [`commands/jobs.go`](../../internal/engine/commands/jobs.go)
(`job.cancel`). Tests: `TestJobCancelKeepsItsTab`, `TestJobEnd`, `TestJobKeepsItsCommandsContext`
(engine, with `-race`), `TestDeleteManyAsAJob`, `TestDeleteManyJobCancelled`,
`TestRestartManyDirectOrJob`, `TestFluxManyAsAJob`, `TestArgoManyAsAJob`; GUI `TestJobTabCancel`,
[PNG dark](../../internal/ui/fyneui/testdata/job_tab_delete_dark.png),
[light](../../internal/ui/fyneui/testdata/job_tab_delete_light.png); on kind 1.36.1
`TestKindDeleteSelectionAsAJob` (30 ConfigMaps, cancelled after 13 deletes: the API server holds
exactly the 17 the job did not attempt; the rest deleted by a second Delete).

### The lead's decisions

**Confirmed (owner, audit of 2026-10-16)** as built: which commands, small selections, who may cancel, a step
under way, what `job.cancel` returns, the audit log and the bound of 500. Not asked in the audit,
for the owner to confirm: the tab after Cancel, steps not attempted, the toast at the end, the
tab's list on a long job and the wording.

| Question | Options | Choice | Reason |
|---|---|---|---|
| Which commands run as jobs | (a) the actions on a selection that loop with a request per object, and Reconcile with Source as before; (b) Drain too; (c) the documents of a multi-document Apply or Create too | (a) | They are the commands whose time grows with a selection of up to 500. Drain is one node with its own crash journal and progress (`node.pendingDrains`, D-11), its steps known only once the node's pods are listed, and its kind test cordons a node of the shared cluster, which the night's run could not; a manifest's documents are few and belong to the editor's tab. Both stayed open; Drain was built later the same night ([Drain as a job](#drain-as-a-job-built-2026-10-16-night-run)) |
| Small selections | (a) a job for every size; (b) a job from two objects on; (c) a job above ten objects, ten or fewer within the call as before; (d) a setting | (c) | Ten is the connection's burst (`cluster.DefaultBurst`): up to ten requests go out at once and end within about a second, so a toast says enough and a dock tab per small Delete would pile up; beyond ten the 5 requests per second spread them over seconds to minutes, which a tab shows step by step. An agent gets the result in the call for the common small case and a job only when it takes seconds; both results say what they are (the descriptions say it too) |
| Who may cancel a job | (a) every origin, any job, a UI command; (b) an agent only the jobs it started; (c) the user only | (a) | Cancel writes nothing and undoes nothing; agents may already stop any job by closing its tab (`dock.close`, a UI command), so (b) and (c) would guard nothing. The outcome says when an agent cancelled ("Stopped: cancelled by an agent."), and the agent's `job_cancel` is in the audit log |
| A step under way when the job is stopped | (a) its request finishes and the step says what it did; (b) its request is cut and the step says the object may or may not be done | (a) | The record of what was deleted stays exact; the wait is one request, bounded by D-58's request timeouts. Reconcile with Source's waits for a controller still end at once |
| What `job.cancel` returns | (a) the tab once it shows the end, at most 10 seconds later; (b) at once | (a) | An agent reads the outcome in the result instead of polling `ui_state` |
| The tab after Cancel | (a) it stays with the outcome; (b) it closes | (a) | Closing is `dock.close`; Cancel exists to stop the job and keep its record |
| Steps not attempted | (a) marked stopped, "Not attempted: <why>"; (b) left waiting | (a) | "Waiting" after the job's end would read as still to come. Reconcile with Source keeps A91's rule: its steps stay where they stopped |
| The audit log | (a) the command's two lines as before, `done` meaning the job started; (b) a third line, `ended`, with the job's outcome; (c) a line per object | (b) | The audit log is where the user reviews what an agent did: with (a) an agent's Delete of 500 objects of which three failed would read "ok"; (c) would flood it while the parameters already name every object |
| The bound of 500 (`MaxSelection`) | (a) kept; (b) raised to the performance target's 10,000 rows; (c) none | (a) | The confirmation lists every object and an agent's call carries them all; the owner confirmed 500 for Delete (A72). With progress and Cancel the bound's reason is weaker, so raising it is the owner's call |
| The toast at the end | (a) when the job ends by itself, done or failed, none when stopped; (b) always; (c) never | (a) | As Reconcile with Source's (A91); a stop was asked for, and the tab says what it did |
| The tab's list on a long job | (a) it follows the step under way until the user scrolls it; (b) it stays at the top | (a) | Only about eight steps fit in the dock; the header counts all of them, the list shows where the job is |
| Wording | the title "Delete: 24 objects", step details "Deleted", "Not attempted: cancelled", the outcome "Deleted: 11 of 12 objects; 1 failed: the job's steps say why." | as built | Cogpit's own; Lens has no such tab |

### Drain as a job (built 2026-10-16, night run)

**Built (the lead's decisions below, confirmed by the owner in the audit of 2026-10-16, every row but the
tests', which it did not ask).** `node.drain` asks for its
confirmation as before (destructive, for every origin, D-2), then records the drain in its journal
(before anything is written, as before), starts a job and returns at once with the job's dock tab
(`DrainResult`: `ref`, `action` "job started", `job`, [`commands/drainjob.go`](../../internal/engine/commands/drainjob.go)).
The job's steps are the cordon ("Cordoned; ignoring DaemonSet-managed Pods: …") and then one per
pod the drain evicts: the cache's pods on the node while the drain has not listed them, when the
cache holds the cluster's pods at that moment (`Session.CachedNodePods`, which starts no list),
filtered as the drain filters (no mirror pods, no running DaemonSet pods while they are left
running); the drain's own list replaces them once it has listed the node (`cluster.DrainProgress.Pods`),
and a pod it reports that no step names gets one. A pod's step runs while its eviction is asked for
("Evicting"), refused for now ("Waiting: the PodDisruptionBudget web allows no disruption now
(…); retrying", kubectl's library retrying every 5 s until the drain's timeout) and accepted
("Evicted; waiting for the pod to end", `cluster.DrainAccepted`), and ends done ("Evicted", "Gone
already"), failed with the API server's error (`cluster.DrainPodFailed`) or stopped. The outcome
says "Drained node worker-1: 5 pods evicted." and what the drain left running, or counts the pods
evicted, failed and not attempted, adds the drain's error and says that the node stays cordoned and
partly drained, with Finish drain, Uncordon and Dismiss.

The job keeps the call's values but not its end (`JobSpec.Parent`): an MCP client that gives up on
its call, and the end of the call itself ([`mcp/tools.go`](../../internal/mcp/tools.go)'s
`callContext`), no longer stop a drain the user confirmed (`TestDrainJobOutlivesItsCall`; on kind
the last drain's caller gives up at once). `job.cancel`, the tab's Cancel, closing the tab, another
cluster, an interrupted connection (D-58: the job's own `errJobInterrupted` replaces
`Session.OperationContext` for drains) and Cogpit closing stop it: the drain's context ends, the
library stops at its next request or wait (a budget's retry pause of 5 s runs out first; `job.cancel`
waits up to 10 s for the end), and the journal keeps the drain as `stopped`
(`DrainStopped`, the cause as its error), which the banner and `node.pendingDrains` offer as after
a crash (D-11, S8). A crash is handled exactly as before: the entry, written before the job starts,
stays `running` and is offered as interrupted at the next connect; a job that does not start
removes it, since nothing was written. `node.pendingDrains` names the job's tab of a drain running
in this Cogpit (`job`).

In the window the job tab is the drain's progress and its Cancel; the drain banner no longer has a
row for a drain running in this window and keeps the drains that wait for the user (interrupted,
failed, stopped) and those another Cogpit window runs ([`fyneui/drains.go`](../../internal/ui/fyneui/drains.go)).
The progress toasts are gone; the end's toast stays when the job ends by itself (D-63's rule).
Tests: `TestDrainNodeReportsEachPod`, `TestDrainNodeCancelledReportsNoPodFailure` (package
`cluster`, client-go's fake API server: a budget blocking then allowing, a pod gone, a refusal);
`TestDrainJobBudgetBlocksThenAllows`, `TestDrainJobOutlivesItsCall`,
`TestDrainJobCancelledLeavesAPendingDrain`, `TestDrainJobStopsWhenInterrupted`,
`TestDrainJobPodFailures` and the existing drain tests (engine, with `-race`); GUI
`TestDrainJobTabWithoutABanner`, [PNG dark](../../internal/ui/fyneui/testdata/job_tab_drain_dark.png),
[light](../../internal/ui/fyneui/testdata/job_tab_drain_light.png), the banner's stopped row in
[`drains_dark.png`](../../internal/ui/fyneui/testdata/drains_dark.png); on kind 1.36.1 (the drain
cluster, `make kind-drain-up`) `TestKindDrainCrashFinishUncordon`: the crash while the budget holds
the guarded pod, a finish that times out naming the budget, a Cancel while it holds the pod (the
drain `stopped`, the node cordoned, the pod in place), and a finish whose caller gives up at once.

| Question | Options | Choice | Reason |
|---|---|---|---|
| Where the pod steps come from at the start | (a) the cache's pods on the node when the cache holds them, replaced by the drain's own list; (b) the drain's list only, the steps appearing once it listed (within a second); (c) a list started for the job | (a) ★ | The tab and an agent's result name the pods at once from what the Nodes view already holds; the drain's list is the one it evicts, so it wins; (c) would list every pod of the cluster for a preview |
| A small drain within the call (D-63's ten) | (a) always a job; (b) within the call for a few pods | (a) ★ | A drain waits for pods to end (their grace periods) and for budgets up to its timeout, minutes even for one pod; D-63's ten was about requests that end within a second |
| Does a stop uncordon? | (a) no, the node stays cordoned and the banner offers Uncordon; (b) Cancel uncordons; (c) Cancel asks | (a) ★ | A stop is not a decision to give the drain up (Finish drain runs it again); an uncordon is a write, which an agent's `job_cancel` (a UI command) must not make without the user; after a crash Cogpit cannot uncordon either, and the same banner serves both |
| An eviction request under way at the stop | (a) cut with the drain's context; (b) let the evictions under way finish, as D-63's steps do | (a) ★ | kubectl's library sends the evictions in parallel and retries them in its own loop, so there is no single request to wait for; a cut eviction may or may not have reached the API server, which Finish drain sorts out, and the pod's step says it stopped before the answer |
| The journal state of a stopped drain | (a) `stopped` with the cause; (b) failed with the cause; (c) left running, as a crash leaves it | (a) ★ | The banner says "was stopped · cancelled by an agent" instead of "failed"; a Cogpit older than the state reads it as interrupted, which offers the same actions |
| The banner and the job tab | (a) a running drain in its job tab only, the banner for what waits for the user and for drains of another window; (b) both; (c) the banner only | (a) ★ | Not the same progress twice in one window; the tab has the steps and Cancel, the banner the actions for afterwards |
| A pod whose eviction was accepted when the drain ended | (a) done, "the pod was still shutting down"; (b) failed | (a) ★ | The API server accepted it, the budget allowed it, and nothing stops the pod's deletion; Finish drain finds it gone |
| A pod a budget held when the drain was stopped | (a) counted with the steps not attempted, as the job tab counts every stopped step, its detail "Stopped while waiting: …"; (b) a fourth count | (a) ★ | The job tab's header counts by state for every job (D-63); the step says what happened |
| The tests "with the fake API server" | (a) the engine with the fake writer's drain reporting what `cluster.DrainNode` reports, `cluster`'s tests on client-go's fake API server, the kind test; (b) testkit's `FakeAPIServer` with writes | (a) ★ | `FakeAPIServer` only reads; giving it node patches, evictions, field selectors and the eviction subresource in discovery would touch the file tonight's discovery-cache work changes |

### What it does not cover

- **Drain, built as a job (above); open around it.** No test interrupts a real connection while a
  drain runs: the engine test sets the state, and D-58's freeze test (`standby`) does not drain. An
  agent may stop a drain the user confirmed with `job_cancel` or `dock_close`, which leaves the node
  cordoned and partly drained ([H-111](../security/agents-and-mcp.md#h-111-an-agent-can-stop-a-drain-the-user-confirmed)).
  Lens's own drain progress is not captured.
- **A multi-document Apply or Create, live.** Its documents are written within the call; a long
  manifest shows nothing until it is done.
- **Helm's actions, live.** Install, upgrade, rollback and uninstall wait for the chart's hooks
  within the call (`HookOnlyStrategy`, at most 5 minutes, [`helm/actions.go`](../../internal/helm/actions.go));
  each is one release with its own journal (D-11) and no Cancel; how far the Install and Upgrade
  tabs show such a wait was not checked in this run.
- **D-58's interrupted connection (decided at the merge, 2026-10-16, the lead's decision,
  confirmed by the owner in the audit of 2026-10-16).** A job of the cluster stops when the connection goes `interrupted` or
  `loginExpired`, with its own reason: "Stopped: the connection was interrupted; Cogpit does not
  continue the job after the reconnect." (`errJobInterrupted`, `TestJobStopsWhenInterrupted`); a
  disconnect or another cluster still ends it as "the connected cluster changed". Options: hold the
  job until the connection is back and go on (a delete or a restart of the rest of a selection
  after a night's sleep, unasked); stop it as a cluster change (a wrong reason). Why: D-58's S8, as
  after a crash, nothing continues by itself; writes are refused while interrupted anyway, and the
  step under way may or may not have reached the cluster, which the outcome's counts say.
- **500 rows in the job tab, not measured.** The tab draws a row per step (a box, not a recycling
  list) and updates them as the steps move: two UI state changes per object, about ten a second.

## D-64: envtest, a real API server in CI without Docker

**The need ([G1](../parity/gaps.md#g1-workspace-ci-and-progress-generators), 2026-10-16):** the
tests ran against testkit's fake API server or the shared kind cluster. D-58's standby case needs a
real API server whose process can be stopped: `docker pause` of a kind node does it only on a
cluster of its own, by hand (tag `standby`, [D-58](#d-58-standby-and-lost-connections)), and CI's
kind job shares its one cluster with every other kind test.

**Built 2026-10-16 (night run; the lead's decisions, confirmed by the owner in the audit of 2026-10-16, but the
Kubernetes version, Windows, the CI job and the test's intervals, which it did not ask):**
[`internal/testkit/envtest`](../../internal/testkit/envtest/envtest.go) starts etcd and
kube-apiserver through controller-runtime's envtest with a cluster-admin's kubeconfig
(`envtest.Start`, `FreezeAPIServer`, `ThawAPIServer`); `make test-envtest` runs the tests tagged
`envtest` (`TestEnvtest*`) with the race detector after `make envtest-bin` fetched the pinned
binaries ([`hack/envtest/fetch.sh`](../../hack/envtest/fetch.sh)); CI runs it as the job
Integration Tests (envtest) ([releasing › CI](releasing.md#ci)). The first test,
[`TestEnvtestStandbyAPIServerStopped`](../../internal/testkit/integration/standby_envtest_test.go),
stops the API server with SIGSTOP and resumes it with SIGCONT.

| Question | Options weighed | Chosen | Why |
|---|---|---|---|
| What starts the control plane | (a) controller-runtime's envtest; (b) a starter of our own around setup-envtest's binaries | (a), `sigs.k8s.io/controller-runtime` v0.25.2 | Already in the module graph through Helm, and it asks for k8s.io v0.37.0 where Cogpit has v0.37.1: no k8s.io module moved and no module was added (go.mod: controller-runtime and `go-logr/logr` from indirect to direct). (b) would rebuild the certificates, the client-certificate login and kube-apiserver's flags, about 300 lines that follow every release |
| Where the binaries come from | (a) setup-envtest, a tool that reads the release index at `HEAD` at run time and checks the archive against that same index; (b) envtest's own download (`DownloadBinaryAssets`, the same index); (c) the release archive fetched by a script, pinned by version and SHA-512 in the `Makefile` | (c): `envtest-v1.37.0` of controller-tools' release store for darwin-arm64 and linux-amd64, kept in `.cache/envtest/1.37.0` (never committed), etcd and kube-apiserver only | As oxipng and Sparkle are pinned: a changed archive fails the fetch instead of being trusted from an index that changes with it; no tool to install. Both SHA-512 agree with the index and with the downloads, whose SHA-256 agree with GitHub's digests of the assets (2026-10-16) |
| The Kubernetes version | 1.37.0; several minors | 1.37.0, the newest envtest release | kind's Kubernetes in CI, and client-go v0.37.1 is the 1.37 release; more minors are the platform row "every minor version the Kubernetes project supports", not this step |
| How the API server goes silent | (a) SIGSTOP of its process; (b) `testkit.FreezableProxy` in front of it; (c) `docker pause` | (a), the process id from a wrapper script that records it and execs the binary | The kernel's own silent peer: the port accepts connections and takes requests, nothing answers, the open HTTP/2 connections stall; (b) is the fake API server's test already, (c) needs Docker. envtest keeps its process unexported; `exec` keeps the wrapper's id |
| Windows | build the package there; Unix only | Unix only (`//go:build unix`) | controller-runtime v0.25.2's envtest does not compile for Windows (its process package declares `signalProcess` twice there), and the freeze needs SIGSTOP; Compile (windows) passes over the package |
| The CI job | a step of the kind job; a job of its own | a job of its own, race detector on, its coverage profile in the combined report, in Semantic Release's `needs` | One check per job ([releasing › CI](releasing.md#ci)); it needs no Docker and waits for no cluster |
| The test's intervals | the owner's (a heartbeat every 15 s, 5 s to answer, the second check after 2 s: about 25 s per freeze); shortened | a heartbeat every 1.5 s, 1 s to answer, the second check after 0.2 s, the backoff from 0.1 to 0.8 s | The same code path as `TestStandbyFrozenControlPlane`, the window computed from the options; 1 s to answer keeps a slow answer on a loaded runner from counting as a failed check |
| Without `COGPIT_ENVTEST_ASSETS` | fail; skip | skip; fail when it names a folder without the binaries | As the kind tests skip without `COGPIT_KIND_KUBECONFIG`; `make test-envtest` always sets it after the verified fetch |

**Measured (2026-10-16, Apple M1 Ultra, race detector on, five runs, two of them under a load
average of 10 to 14):** interrupted 3.68 s after SIGSTOP (the window: 5.7 s), connected again 0.40
to 0.41 s after SIGCONT, a pod created then in the view 7 to 25 ms later; 9.3 to 11.2 s per test,
11.5 to 14.1 s for the package, `make test-envtest` 34 s with its race build. The CI job is not
observed yet (the branch was not pushed in the night run); expected about 5 to 8 minutes, most of
it the Go setup and the race build with `-coverpkg=./...`.

### What it does not cover

- **No kubelet, scheduler or controller manager:** logs, exec, attach, port forwards and pods that
  run stay the kind tests'; pods created on envtest stay Pending.
- **What changed during the freeze:** the stopped API server is the only way to etcd, so nothing
  changes meanwhile; that the cache catches up with such changes is
  `TestStandbyFrozenControlPlane`'s, against the fake API server.
- **One Kubernetes version**, 1.37.0, and a new one pinned by hand: Renovate does not see the
  archive's pin ([updates › H-18](../security/updates.md#h-18-dependencies)).
- **The login-expired case** stays the fake API server's (`TestStandbyRevokedLogin`): envtest's
  client certificate cannot be revoked.

## D-62: Checks against a real cluster

**The lead's decisions (night run, 2026-10-16), confirmed by the owner (audit of 2026-10-16), every row as
built.** The owner provided a real
cluster for tests for one night (Kubernetes 1.35, four nodes, Rook-Ceph, Flux with 64
Kustomizations and 48 Helm releases, cert-manager, Kyverno, Cilium, 232 custom resource
definitions). Its name, kubeconfig and objects stay out of the repository: the docs say "a real
cluster the owner provides for tests", and every fixture made from what it showed is synthetic.
What was checked there and what it found is in [resource parity](../parity/resource-parity.md),
[gaps › G6, G7, G23](../parity/gaps.md) and [testing › Real cluster](testing.md#tiers).

| Question | Options | Choice | Reason |
|---|---|---|---|
| How the writes run against a real cluster | (a) `cogpit-cli`, headless; (b) Go tests that build the engine with origin GUI and a confirming hook, as the kind tests do; (c) by hand in the window | (b) | A headless `cogpit-cli` gives no confirmation, so every destructive action is refused there (D-2), and the lead cannot drive the window on the owner's Mac. (b) runs the same commands the window runs, the confirmation included, and repeats |
| Where those tests live | (a) a build tag of their own (`real`); (b) untagged, skipped by an environment variable; (c) the `kind` tag, named `TestReal*` and skipped without `COGPIT_REAL_KUBECONFIG` and `COGPIT_REAL_CONTEXT` | (c) | (a) is compiled by nothing in CI and would rot; (b) cannot use the kind tests' helpers (namespaces, confirmers, echo servers); (c) is compiled by CI's kind job and never run by it (`make test-kind` runs `^TestKind`); `make test-real` runs them with a HOME of their own ([`real_kind_test.go`](../../internal/testkit/integration/real_kind_test.go)) |
| What they may touch | (a) whatever the test needs; (b) only namespaces of their own (`cogpit-real-*`, deleted at the end) and what an environment variable names: the node to cordon, to drain, to open a shell on, the Helm repository, the Flux objects | (b) | A real cluster may be someone's production; a node, a release or a Flux object is acted on only when the person running the test named it, and every test undoes what it did (uncordon, resume, uninstall) whatever happens |
| P11 on a real API server: the drawer's first `SelfSubjectRulesReview` of a namespace | (a) the budget counts every request, so it is not met: prefetch the rules of every namespace at connect; (b) the budget counts reads of objects; one rules review per namespace and session is the access reviews' own cost, which P15 budgets | (b) | (a) would cost one review per namespace up front for drawers that may never open; the review is what lets the drawer grey out what the user may not do ("Discover, don't assume") |
| An OCI HelmRepository without a status | (a) Unknown, as before; (b) Ready unless suspended, with an empty Message; (c) Ready with the flux CLI's "Helm repository is Ready" | (b) | No controller reconciles one since Flux 2.1, and `flux get` calls it Ready; Cogpit's Message column is the Ready condition's message, so (c) would show a text the cluster never wrote |
| How an event's involved custom resource links to its view | (a) an index over the definitions by group and Kind (`CustomKindsByGroupKind`) the Events profile names in `Needs`; (b) the backend's custom kinds, asked while the rows are built | (a) | The definitions' scope is the one the custom kinds already watch, so no request; and the related-objects machinery rebuilds the rows when the definitions change, where (b) would leave rows built before the definitions were listed without links until each event changed |
| An owner that is a custom resource (a pod's Challenge, a StatefulSet's Prometheus, a Deployment's CephCluster) | (a) plain text, as before, since Lens's form for owners is not captured; (b) linked through the same index: the Pods list names it in `Needs`, the engine adds it to every drawer's needs on a cluster with custom kinds | (b) | One rule for every reference to a custom resource; the index costs no request (the definitions' scope is the custom kinds' own), and the drawer's set holds it only while the drawer is open; a virtual or local kind's drawer does not take it |
| A drain that did not finish | (a) "cordoned and partly drained", as before; (b) what it left: "this drain evicted no pod", or the count | (b) | On the real cluster the default drain refused the node at once for its pods with emptyDir data, and "partly drained" was wrong. The unfinished-drains banner keeps the old text: the journal records no count (below) |
| A confirmation's sentence named objects by kind key ("Drain nodes worker-1", "Delete helmreleases web") | (a) leave it; (b) the kind as the dialog's Objects row names it ("Drain Node worker-1", "Delete Release web"), the drawer's singular title or the view's title in the singular | (b) | The dialog said "Delete pods web-0" above "Objects: Pod web-0"; `ConfirmRequest.Kinds` carries the names, and the dialog's Objects row reads them too. Lens's dialogs are not captured |

**Fixed at the merge (2026-10-16, the lead):** the Pods and Events views wait for their indexes, and
for a namespace-limited user the definitions' list is refused, a cluster-scoped kind with no
namespace to read instead: the views never synced (CI's `TestKindNamespaceLimitedUser` and
`TestKindAccessibleNamespaces`). A related-objects set no longer takes the definitions once their
watch was dropped after a refusal (`customKinds.refused`, the view's `relatedKey` changes and the
view starts again without them), and a refused cluster-scoped kind does not hold a set's `synced`;
its lookups stay empty, so a custom owner or an event's custom object shows as plain text for that
user.

### What it does not cover

- **The unfinished-drains banner, live.** For every failed drain it says "the node is cordoned and
  partly drained", also when no pod was evicted; the journal would need the count.
- **The README's showcase picture of an agent's confirmation, live until `make showcase` runs.** It
  still reads "Restart deployments …"; the test that draws it now passes the kind.
- **Owners that are custom resources, linked since 2026-10-16, inferred.** Every drawer's and the
  Pods list's Controlled By link a Prometheus, a CephCluster or cert-manager's Challenge as the Events
  list links custom kinds (the same index); Lens's capture shows custom kinds linked in the Events
  list, for owners it is not captured. The workloads' and the node's Pods tables in the drawers keep
  such an owner plain.
- **Lens's own behaviour on such a cluster** is not observed: the checks compare Cogpit with the API
  server's tables, `kubectl` and the `flux` CLI, not with a running Lens.

## D-65: Discovery cached on disk

**The need (G3, [architecture §5](architecture.md#5-cache)):** connecting asks the API server which
groups and resources it serves (the built-in kinds a cluster offers, the curated kinds of a user
who may not list the custom resource definitions, the REST mapper of the object writes). The answer
was kept in memory only, so every connect, every restart of the window and every run of
`cogpit-cli` asked again; §5 planned it on disk with a time-to-live, refreshed when the definitions
change. It is not the cache's objects: [D-13](#d-13-cache) keeps those in memory only, and this
answer holds no object, only which kinds the server serves.

**Decided by the lead (night run, 2026-10-16), built the same night; confirmed by the owner
(audit of 2026-10-16) but the table's last three rows, which the audit did not ask**
([`cluster/discocache.go`](../../internal/cluster/discocache.go),
[`engine/customkinds_digest.go`](../../internal/engine/customkinds_digest.go)):

| Question | Options weighed | Chosen | Why |
|---|---|---|---|
| Whose cache | client-go's `disk.NewCachedDiscoveryClientForConfig` (kubectl's: a file per group version and an HTTP cache that revalidates by ETag, valid by the files' modification time, files 0660 in 0750 directories, and a client of its own built from the configuration: a second connection pool, outside the gate D-58 puts in front of the credential plugin); a small layer of Cogpit's under client-go's memory cache, over the connection's own discovery client | Cogpit's layer: one file per server and identity, read and written whole | The connection's own client keeps one connection, D-58's gates, the request counter and discovery's own budget; owner-only files; the server's version and the definitions checked, which client-go's cache does not do |
| How long an answer is reused | 10 minutes (the answer in memory before); 6 hours flat (kubectl's default, verified in cli-runtime v0.37.1's `config_flags.go`); 6 hours once the definitions confirmed it, 10 minutes otherwise | 6 hours once the definitions confirmed it, 10 minutes before that or for a user who may not list them; a partial answer (an aggregated API that did not answer) a minute and never written | What changes discovery in practice is caught otherwise: a new server version at each connect, the definitions' digest once their watch listed them, a kind the REST mapper does not know. What is left (an aggregated API, or an API switched on or off without a new version) feeds no view. Without the definitions nothing confirms the answer: the 10 minutes the curated kinds had |
| What names the file | the server URL alone (kubectl's `host_port`); the server and the credential (a hash of a token or certificate); the server URL, its TLS server name, the kubeconfig files, the context's cluster and user entry names and the impersonated user | the last, as a SHA-256 | A proxy that serves several clusters under one URL by the client's identity (Teleport) gets a file per identity; no credential, not even a hash of one, names a file |
| Where | Cogpit's configuration directory; its cache directory | `<user cache dir>/Cogpit/discovery` (macOS `~/Library/Caches/Cogpit/discovery`), shared by the app and `cogpit-cli` | Data Cogpit can fetch again, beside the log and Helm's indexes; a run of `cogpit-cli` reuses the window's answer |
| When the definitions change | drop the answer at every change of the custom kinds (as the answer in memory was); compare a digest of the definitions | a SHA-256 over each definition's name, generation, established and being deleted, compared with the one the answer was checked against | The first build of the custom kinds after a connect is a change: dropping there would drop every answer the file gave |
| What a dropped answer costs | ask again at once; at its next use | at its next use; the file is removed at once | Nothing reads discovery until a connect, the curated kinds or a write needs it |
| An answer fetched in this connection, at the definitions' first digest | drop it; take the digest | take the digest and write it again | It was asked moments before the definitions were listed. A definition created in between is the case left open below |
| A file nobody checked (written before the definitions were listed, then the window quit) | trust it for 6 hours; 10 minutes, dropped at the first digest | 10 minutes, dropped at the first digest | Nothing says which definitions it saw |

**Measured on a real cluster the owner provides for tests** (2026-10-16, Kubernetes 1.35, 232
custom resource definitions, aggregated discovery: 57 groups, 340 resources, a cache file of
141 KiB; the API server on the local network, `TestRealDiscoveryCacheConnectTime`, 7 rounds):
the engine's connect (`ClusterConnector.Connect`: `GET /version`, discovery, the backend) took
17.8 ms (median) from the server with the file written, 11.2 ms from the file, 14.6 ms without a
cache directory; the server was asked 3 discovery requests, then 1 (`GET /version`, which Connect
makes anyway). A whole run of `cogpit-cli get pods`, process start to exit, 223 ms against 212 ms
(medians of 7, the difference within the runs' spread); the session's connect to the Pods view's
first synced rows 237 ms against 242 ms (5 rounds each: no difference measured). The saving is
the two discovery round trips and their decoding, so it grows with the distance to the server: a
cluster 50 ms away saves about 100 ms per connect (estimated, not measured).

**What a stale answer can cause, until one of the checks drops it (harmless, named):** a
built-in view the server stopped serving without a new version (an API group switched off) shows
the server's refusal of its list; a kind the server began to serve without a new version or
definition (an aggregated API) is missing from the navigator for up to six hours; a write of a
manifest whose kind the answer does not know asks the server again (`MapKind`'s reset), one whose
kind it still knows after the server dropped it gets the server's 404. The custom kinds come from
the definitions' watch, not from discovery, so a definition added or removed shows at once either
way.

**Open:** the owner's confirmation of the table's last three rows; a definition the API server serves only a moment after its
watch event can be missing from an answer stamped with that digest until the next change of the
definitions or the TTL (a write of that kind maps it again); the folder is never cleaned (a file
of about 140 KiB per server and identity once used; deleting it is harmless); not run on Windows
or Linux (the path is `os.UserCacheDir`'s there, untested).

## D-66: The connection log

**The need (G2, the owner's request of 2026-10-16):** every connection being set up should be
traceable, the successful ones too: a click on a cluster's status indicator shows a view that
documents the set-up in detail and with times, and on a failure an extensive picture of the error.
The top bar loses its connection words at the same time (another part of the same request, F13):
the dot alone says the state, and every dot of a cluster opens this log. Until then the connect
log ([architecture §6](architecture.md#6-cluster-connections-internalcluster)) kept the last attempt of a context in memory only, a
few lines ("Connecting to", the error, "Connected"), and was reachable only for a context whose
connect had failed; a credential plugin's standard error went to Cogpit's own standard error.

**Decided by the owner (2026-10-16, in a question round with the lead):**

1. **The view:** the existing "Connection Updates: <context>" dialog, extended, opened from every
   status dot of a cluster (connected or not, failed or not; a context without a log says so),
   following the log live while it is open, with a Copy button (all shown attempts as plain text,
   for bug reports).
2. **Retention:** on disk, rotated: one file per context, attempts older than 7 days and beyond
   1 MiB per context dropped, oldest first; owner-only like the audit log; the file not named by
   the raw context id; written line by line, so it survives a restart and a force-quit; kept in
   attempts (connect, reconnect, login renewal, disconnect), each with its kind, start, outcome and
   duration.
3. **Detail:** a successful connect documents its steps with times and durations; a failure the
   step that failed and the error picture (the error chain, the HTTP status and the API server's
   message, the class: DNS, TCP refused or timed out, TLS with the presented certificate's public
   facts, the credential plugin, unauthorized or forbidden; timeouts with their limits). No line
   holds a credential.
4. **The credential plugin's standard error:** "only on failure, filtered": captured on a failed
   login only, scrubbed, cut to 4 KiB, written to the log; its standard output (the credential)
   never captured.
5. **Agents:** `cluster.connectLog` keeps working, its result extended with the attempts; a read,
   no confirmation.

**Built 2026-10-16** ([`engine/connectlog.go`](../../internal/engine/connectlog.go) the attempts,
[`engine/connectlog_files.go`](../../internal/engine/connectlog_files.go) the files and their
rotation, [`engine/connectlog_steps.go`](../../internal/engine/connectlog_steps.go) and
[`engine/connwatch_log.go`](../../internal/engine/connwatch_log.go) the engine's steps,
[`cluster/connecttrace.go`](../../internal/cluster/connecttrace.go) the cluster layer's steps and
the error picture, [`cluster/pluginstderr.go`](../../internal/cluster/pluginstderr.go) the plugin's
standard error, [`fyneui/connectlog.go`](../../internal/ui/fyneui/connectlog.go) the dialog;
`showConnectLog(contextID)` kept as the dots' entry point). What a connect writes, from what the
code does: the kubeconfig file and context (its cluster and user entries); the API server's URL
without user information, query or fragment; the login (a client certificate's subject CN and
expiry date, "bearer token … (not shown)", a credential plugin's program name and whether the user
allowed it under D-26, basic authentication without the password, an auth provider's name, an
impersonated user); how the server's certificate is checked (the kubeconfig's CA with its CN and
expiry, a CA file, the system's roots, `tls-server-name`, or not at all, A29); the proxy (Cogpit's
own as before with its password as •••, else the kubeconfig's `proxy-url` or the environment's,
redacted the same way, else none); the first answer (`GET /version`, its duration and the server's
version, or its failure with the limit and the error picture); discovery (groups, resources, those
that can be listed and watched, from the disk cache with its time or from the server, D-65, its
duration); the cache (the kinds offered; their watches start with the views, cache first); the
heartbeat (D-58's period and timeout); and, after "Connected", the access reviews' first answers
with their duration. A reconnect writes the interruption with the failed check's error picture,
each try whose reason changed, and its end; a login the API server rejects, the user's Log in
again, and an expired login write a login attempt; a disconnect writes one of its own (the user's,
or Cogpit's when the kubeconfig file went away, an error as before).

Tests: `TestConnectTraceRecordsTheSteps`, `TestConnectTraceNamesTheClientCertificate`,
`TestConnectTraceErrorPicture`, `TestErrorPictureClasses`, `TestPluginStderrOnFailureIsScrubbed`,
`TestPluginStderrOfASuccessfulRunIsNotKept`, `TestScrubPluginOutput` (cluster);
`TestConnectLogFilesKeepAttemptsAcrossARestart`, `TestConnectLogFileToleratesACutLineAndForeignRecords`,
`TestConnectLogRotation`, `TestConnectLogUnwritableIsSaidOnce`, `TestConnectLogStepsOfARealConnect`,
`TestConnectLogSessionWritesAndAnnounces`, `TestConnectLogIsBoundedAndEscaped`,
`TestConnectLogOfAFailedConnect`, `TestTheWatchWritesReconnectAndLoginAttempts`,
`TestAFailedPluginsStandardErrorIsInTheReconnect` (engine); `TestConnectLogDialogFollowsTheLog`,
`TestConnectLogDialogCopy`, `TestConnectLogDialogEmpty` and the PNGs
[success](../../internal/ui/fyneui/testdata/connection_updates_success_dark.png) and
[failure](../../internal/ui/fyneui/testdata/connection_updates_failure_dark.png) (window).

**The lead's decisions for the owner to confirm** (2026-10-16):

| Question | Options weighed | Chosen | Why |
|---|---|---|---|
| The dialog's order | oldest first, scrolled to the newest; newest first | the newest attempt first, each attempt's lines in the order they happened | The latest outcome is what the dialog opens on without scrolling, and an attempt still reads top down like a log; the live lines of a running attempt appear at its foot, at the top of the dialog |
| Where the files live | Cogpit's cache directory, beside `cogpit.log`; its configuration directory, beside the audit log | `<user config dir>/Cogpit/connection-logs` (`app.Dirs.ConnectLogDir`; macOS `~/Library/Application Support/Cogpit/connection-logs`), 0700, files 0600, shared with `cogpit-cli` | A record of what happened is nothing Cogpit can fetch again; a connect of `cogpit-cli get` is in the window's dialog too |
| The file's name and form | the context id escaped; a hash | the first 32 hex digits of the SHA-256 of the context id and `.jsonl`; JSON lines appended one write each (`O_APPEND`), the context id in each attempt's first record, an attempt naming another context not read | A context id holds any character; a cut last line after a crash is skipped as a line that does not decode, and the next run's first attempt rewrites the file without it |
| When the rotation runs | at every write; at an attempt's begin | at a context's first attempt of a run, and at an attempt's begin once the file holds more than 1 MiB less the room one attempt may take (256 KiB); whole attempts go, oldest first, the newest stays; files untouched for 7 days are deleted at a run's first write | A write stays one append; an attempt holds at most 200 lines of 600 bytes, so a file never passes 1 MiB |
| How the plugin's standard error is captured | a public option (client-go v0.37.1 has none); replacing the process's `os.Stderr` (every write of the process mixed in, the plugin's runs not told apart); a wrapper program in the plugin's place (changes the command D-26 approved); Cogpit's own exec authenticator (client certificates from a plugin need client-go's TLS callback); setting the unexported field of client-go's own Authenticator | the field, through reflection, under the Authenticator's own lock, once per Authenticator; the output still goes on to Cogpit's standard error as before, a copy of the newest 64 KiB held in memory and dropped after a request that worked | The least invasive way that is correct: the same Authenticator client-go's transport uses (its cache), the plugin's runs told apart by request, the command unchanged; a client-go that renames the field leaves the plugin's standard error where it went before (logged once) and fails `TestPluginStderrOnFailureIsScrubbed` |
| The scrubber | the engine's Secret filter (H-82; its values are the connected cluster's Secrets, so it knows no plugin's token and refuses without a connection, which a failed login is); `helm.ScrubValues` (a chart's values); the cluster layer's replacement of known secrets | the cluster layer's replacement (`replaceSecrets`, shared with the proxy's password in errors) for the user entry's own values (its token, password, the plugin's environment values and `--flag=value` arguments, the proxy's password), plus the shapes a credential has (a JWT, a PEM block, a value after `Bearer`/`Basic`, a value after a key that names a secret, a run of at least 32 token characters with letters and digits), every URL without user information, query and fragment, then the cut at 4 KiB | The lead's brief asked to reuse the existing scrubber; none of them can know a token the plugin made, so the shapes are new, in the cluster layer, the only place the text exists unscrubbed ([H-112](../security/credentials.md#h-112-a-credential-plugins-standard-error-in-the-connection-log)) |
| A reconnect that waits all night | a line per try; a line per change of reason | a line when a try's reason differs from the last one written | A try every 30 seconds for hours would push everything else out of the 1 MiB |
| Levels | ERROR for every failure | ERROR for a failed connect and for a connection Cogpit closed, as before; WARN for an interruption, its tries and an expired login | A context is marked (`ConnectProblem`, the navigator's red dot) by a failed connect, as before; an interruption has its own state and banner (D-58) |
| Bounds | — | 200 lines per attempt (the rest counted and named), 30 attempts per context and 64 contexts in memory, the dialog the newest 50 attempts; `EventConnectLog` coalesced to one per 100 ms, the dialog reading the log again (memory and the file, at most 1 MiB) one read at a time | The dialog follows a running connect without polling anything, and the file stays the record |
| A file that cannot be written | fail the connect; say it | a toast and a log line once per run; the session keeps the log in memory | The log must never cost a connection |

**Open:** the owner's confirmation; [H-112](../security/credentials.md#h-112-a-credential-plugins-standard-error-in-the-connection-log)
(the scrubber can miss a secret a plugin prints, and the file keeps it for 7 days); a record that
another Cogpit process (`cogpit-cli`) appends while this one rewrites the file can be lost; the
status dots that open the dialog are F13's other half, built the same day (`TestDotClickOpensTheConnectionLog`); not tried with a real credential
plugin's failure, nor on Windows or Linux.

## D-69: Several clusters connected at once

**The need (the owner, 2026-10-16 morning, F16):** "Aktuell unterstützen wir nur die Verbindung zu
einem Cluster … passt nicht zum Alltag eines K8S Admins." Today the engine holds one backend:
`Session.Connect` closes the old one before it connects the next ([`views.go`](../../internal/engine/views.go)),
`UIState.Context` and `Connection` are single, and 67 call sites take `Session.Backend()`
([D-52](#d-52-several-windows) counted them); the cache is already one per backend. Lens keeps
several clusters connected and shows one.

**Decided by the owner (interview of 2026-10-16, one question at a time, each with the lead's
recommendation; the owner took every recommendation but two, marked):**

1. **The model:** several connections stay open, a window shows one of them, and the hotbar, the
   navigator and the tabs switch to another without reconnecting, the view there at once (as
   Lens). Several windows (R3, D-52's model (a)) later give each window its own shown cluster from
   the same pool of backends, so this is the engine work R3 needs first.
2. **A cluster that is connected but not shown:** its views stay warm for the cache's warm period
   (5 minutes, `cache.DefaultWarmPeriod`), so switching back within it is instant; then their lists
   and watches are released and listed again on the way back (seconds). It keeps its heartbeat,
   discovery, access reviews and the dock's streams (logs, shells, port forwards) all along.
3. **No limit:** a cluster stays connected until Disconnect or quit; Diagnostics shows memory and
   requests per cluster, so the cost stays visible.
4. **A background cluster's login** (the owner's own answer): by default renewed as the shown
   cluster's is (D-58: one renewal, then Login expired), and **configurable**; the lead proposes
   per cluster, in Cluster Settings ("Renew the login while the cluster is not shown", on by
   default), since a browser login is a property of the cluster's credential plugin. **Not red**
   when Cogpit did not renew it: whether a renewal would work is not known, and red says the user
   must act.
5. **Marks that inform without asking for action** (the owner asked the lead for a proposal and
   took it): the dot's shape says live or resting, its colour the state.

   | Mark | State |
   |---|---|
   | filled green | live: shown, or warm |
   | filled green, blinking | connecting or reconnecting (D-58 S3 as amended by F13) |
   | green ring | connected, cold: its views released, switching lists them again |
   | grey ring | its login lapsed and was not renewed (the setting of 4 off); renewed when the cluster is shown or by Log in again |
   | filled red | needs the user: a renewal failed, a connect failed |

   The tooltip names the state; no banner and no toast for the rings.
6. **Agents** work on any connected cluster: every cluster command takes an optional `context`,
   the shown cluster by default; `ui_state` lists every connection with its state; D-2 is
   unchanged (writes allowed per context, destructive commands always confirmed, the confirmation
   naming the context).
7. **A background cluster's interruption** (D-58) shows as the shown one's does: its dots blink
   green, at least three intervals; each connection has its own heartbeat and reconnect, writes are
   disabled per cluster while it is interrupted; the banner is the shown cluster's only. After
   sleep or a network change every cluster reconnects at once, so several tiles blink briefly.
8. **Restart** (the owner's own answer): a global preference, **restore the last session**: on,
   Cogpit connects again to the clusters connected at quit and restores their tabs; off, it starts
   without tabs and without connections. **Off by default.** This changes D-41, whose tabs are
   restored today without a connection, and D-42.
9. **The dock per cluster:** the dock shows the shown cluster's tabs; switching back shows the
   others with what ran meanwhile (as Lens), so a terminal never stands beside another cluster's
   log.

**How it will be built (the lead's plan, estimates in agent-hours with PhotoCraft's calibration,
not measured):** (1) the engine: a pool of backends keyed by context with one shown per session,
`Connect` adding and showing, Disconnect per context, every `Session.Backend()` caller choosing the
shown backend or its reference's, the optional `context` on commands, `UIState.Connections`, a
connection watch per backend (12–25 h); (2) the window: switching without reconnecting, the
marks of 5, the dock per cluster, the navigator and the hotbar (8–20 h); (3) the settings of 4 and
8 and the restore at start (4–10 h).

**Stage 1, the engine, built 2026-10-16** (the lead's run in a worktree of its own; the window part
is stage 2):

- **The pool** ([`pool.go`](../../internal/engine/pool.go), [architecture §6](architecture.md#several-clusters-connected-the-pool-of-backends-d-69-built-2026-10-16)):
  one backend per connected context (`state.pool`), the shown one a field of the window's view state
  (`state.shown`, beside the view workers, so that R3 gives each window its own). `cluster.connect`
  of a connected context only shows it: no request, no new cache, the views from the warm cache;
  the cluster shown before stays connected. `Session.Backend()` is the shown backend;
  `BackendOf(context)` and `TargetBackend(context)` any connected one's.
- **Leaving a cluster** (item 2) releases the window's view handles to the cache's warm period;
  its heartbeat and reconnect, discovery, access reviews, custom kinds watch, port forwards, the
  dock's logs and shells and its jobs go on (`followJobs` ends a job when its own cluster is
  interrupted or disconnected, not when another one is shown). **Cold**
  (`UIState.Connections[].cold`): a timer at the end of the cache's last warm period asks the cache
  (`Cache.WarmUntil`) and marks the cluster once its views' informers are gone; no request, no
  polling.
- **`cluster.disconnect {context?}`** closes one cluster, the shown one by default as before; another
  one's tabs close and its streams end, the window stays.
- **Every command on its cluster** ([`target.go`](../../internal/engine/target.go)): a reference's
  context, a dock tab's, or an optional `context` parameter on the commands that read or act on a
  cluster without naming an object (item 6; the shown cluster by default); a context that is not
  connected is an error, never a connect. 80 call sites of `Session.Backend()` went through: 20 keep
  the shown cluster (the window's view, its banners, the headless CLI), 24 take their reference's,
  dock tab's or draft's context, 19 a `context` parameter, 17 the pool itself (a backend still
  connected, every connected backend). Dispatch takes the command's cluster for `writeBlocked`,
  the access reviews, the allowance of agent writes (D-2) and the confirmation's context; references
  of two clusters in one command are refused.
- **The connection watch per cluster** (item 7): each connected cluster has its own heartbeat and
  reconnect; its state is `Connection` while shown, its entry of `UIState.Connections` otherwise
  (`connected`, `interrupted`, `loginExpired`, `connecting` for the shown one's connect, with since,
  next attempt and reason); writes are blocked per cluster; the banner is the shown cluster's; a wake
  or a network change checks every cluster. A background cluster's login renewal follows D-58 as the
  shown one's; item 4's setting is stage 3.
- **Agents** (item 6): the tools carry the new `context` parameters; `ui_state` lists
  `connections`; the stale-data note follows the read cluster (`StaleNoteFor`); the confirmation
  names the command's cluster; the Secret values replaced in what agents read are every connected
  cluster's cached ones (D-52's case 4).
- **Diagnostics** (item 3): `diagnostics.get {context?}` and the tab's report carry `clusters`, the
  requests, watches, objects and memory of every connected cluster.
- **Tests:** `TestConnectKeepsTheShownClusterConnected`, `TestSwitchingBackWithinTheWarmPeriodListsNothing`,
  `TestReadsOfAnotherConnectedCluster`, `TestDisconnectingOneLeavesTheOther`,
  `TestABackgroundInterruptionBlocksItsWritesOnly`, `TestJobsSurviveASwitch`,
  `TestConnectionsFollowTheConnects`, `TestDiagnosticsShowEveryCluster`; on client-go against two
  fake API servers, `TestSwitchingClustersOnARealStack` (showing a cluster again within the warm
  period sends it no list; once cold, its pods are listed again); in the commands,
  `TestCommandOnABackgroundClusterActsThere` (an agent's write on the cluster not shown goes there
  and its confirmation names it), `TestContextParameterReadsAnyConnectedCluster`,
  `TestBackgroundInterruptionDisablesItsWrites`, `TestDisconnectWithContext`; `TestWarmUntilFollowsTheWarmScopes`
  in the cache. The tests that assumed "a connect replaces the connection" now disconnect first or
  expect the cluster to stay.

**The lead's decisions of stage 1, for the owner to confirm:**

| Decision | Options weighed | Chosen | Why |
|---|---|---|---|
| A connect of a connected context | connect anew; only show it | only show it, except a login whose plugin changed (D-26), which connects anew | Item 1: no reconnect, the views at once. A changed plugin must be shown to the user first, which only a new connect does |
| A cluster's proxy or approval changed while it is connected | reconnect on the next connect; apply after a Disconnect | after Disconnect and Connect (the Cluster Settings note says so) | A show must never drop the cluster's streams by itself; the connection open keeps its way, as the note said before |
| A connect under way to another cluster | goes on in the background; superseded | superseded by the next connect or show, as before | The window shows one cluster; a background connect would run without a state the user sees |
| A failed connect | the previous cluster shown again; the error shown | the error shown (the cluster tried is the shown one, `ConnectError`), the previous cluster stays connected behind it | The user asked for that cluster; the other is one click away and loses nothing |
| `Connections` entries | connected clusters only; also the connect under way | also the connect under way (`connecting`, shown) | Stage 2's blinking mark needs it; a failed one leaves none |
| How cold is known | the time since leaving; ask the cache | ask the cache at the end of its last warm period (`Cache.WarmUntil`), again if a release moved it | Cold means the informers are gone, which only the cache knows; one timer per left cluster, nothing polls |
| Commands without a reference | a required context; an optional one | an optional `context` (`engine.OnCluster`), the shown cluster by default | Item 6; agents that know one cluster keep working |
| A command whose objects are in two clusters | act per object; refuse | refused before anything is asked | One confirmation names one context (D-2) |
| The window's namespace filter for another cluster | apply it; ignore it | ignored: a read, a create or a dashboard of another cluster takes every namespace or the context's default | The filter is the shown cluster's |
| `ui.navigate` to an object of another connected cluster | refuse; show the cluster first | show it first (`Session.Show`, never a connect) | An agent can show the user any connected cluster's object; a context that is not connected is refused |
| The notices of unfinished drains, Helm actions and drafts | the cluster connected last; the shown one | the shown one's, offered again when another connected cluster is shown | They offer actions on what the window shows |
| The Secrets kept out of what agents read | the read cluster's; every connected cluster's | the read cluster's taken first, every connected cluster's cached ones replaced | D-52's case 4: a log of one cluster that prints another's Secret value |
| The hotbar's tiles | the shown cluster's state only; each connected cluster's | each connected cluster's state (`HotbarTile.Connection`) | Agents read `hotbar_list`; the window's tile dot shows it already, stage 2 adds the rings |
| The Create tab's create | the shown cluster; the tab's | the tab's cluster (the window passes it) | A Create tab of a cluster not shown must not create in the shown one |
| Background jobs | end when another cluster is shown; end with their cluster | end with their cluster's interruption or disconnect | Item 2: a drain goes on while the user looks at another cluster |

**What stage 1 leaves to stage 2 (the window):** the marks of item 5 (the hotbar, the navigator's
section and the welcome page read `UIState.Connections`: filled for live, a ring for `cold`, the
blink for a background reconnect), the dock per cluster (item 9: today every cluster's dock tab is
listed), Disconnect of a cluster that is not shown in its menus, and Diagnostics' per-cluster sums
in the tab. Stage 3, item 4's setting and item 8's restore, below. **Not verified:** a real credential
plugin's renewal on a background cluster, several clusters on a real computer's sleep, memory with
several clusters connected for hours.

**Stage 2, the window, built 2026-10-16** (the lead's run in a worktree of its own, beside stage 3's;
[architecture §4.3](architecture.md#several-clusters-connected-d-69)):

- **Switching without reconnecting** (item 1), checked at every entry point: a hotbar tile and its
  F-key (`hotbar.openCluster`), a row of the navigator's section, its › and ↵
  (`ui.setNavigatorScope`), a row of the welcome page (`cluster.connect`), a tab of another cluster
  (`ui.tab.select`) and the row and tile menus show a connected cluster at once: the engine only
  shows it, and the window asks for no login program's approval first, not even for one that
  changed since its connect (`switchable`, [`clustermarks.go`](../../internal/ui/fyneui/clustermarks.go)).
  A cluster that is not connected connects and is shown; the others stay connected. A cluster shown
  while it reconnects loads its kinds for the navigator too (`connectionChanged`: every live state,
  not only Connected).
- **The marks of item 5 on every dot of a cluster**: its hotbar tile, its row in the navigator's
  section and the tree's root, its row on the welcome page (since stage 2 every connected cluster's
  row has a dot, not only the shown one's), the top bar and the status bar for the shown one.
  `markOf` derives a cluster's mark from `UIState` alone: the shown cluster's `Connection`, another
  connected cluster's entry of `Connections`, else a failed last connect
  (`ContextInfo.ConnectProblem`, read again after every connect that ends). Filled green: the shown
  cluster connected, or a connected one whose views are warm ("Connected, in the background: its
  views kept warm, showing it is instant"); filled green blinking: (re)connecting; a green ring:
  `cold` ("…its views released, showing it lists them again"); a grey ring: `loginLapsed` ("Connected,
  its login lapsed: Cogpit did not renew it while the cluster was not shown; showing it or Log in
  again renews it"); filled red: a failed connect, a failed renewal (Login expired); filled grey: not
  connected. A ring is the dot's own 8-point circle drawn as a 1.5-point line
  ([`conndot.go`](../../internal/ui/fyneui/conndot.go), `ringStroke`), so no layout moves; a
  blinking dot is drawn filled. The tooltip names the state; no banner, no toast. A click on any
  dot still opens its cluster's connection log (D-66). The shown cluster's hotbar tile alone is
  raised; another connected cluster's tile has its dot.
- **Each cluster blinks for itself** (item 7): the blinker keeps a blink per cluster
  (`blinker.blinks`), follows every cluster's state (`observeAll`, the shown one's and each entry of
  `Connections`), and a cluster shown or left no longer ends another's blink; at least three
  intervals each, Reduce motion's steady half-alpha dot as before. One ticker serves every blink.
- **The dock per cluster** (item 9), engine state
  ([`dockscope.go`](../../internal/engine/dockscope.go)): the dock shows the shown cluster's tabs and
  those of no cluster (the Audit Log, the Diagnostics; `UIState.DockInScope`); `ActiveDock` and
  `DockOpen` are the shown cluster's, and the engine keeps every other cluster's active tab and open
  state while it is not shown (`dockMemo`, run by `SyncDock` after every change). Every tab stays in
  `Dock`, its stream running (`dock_read` finds a hidden tab by its id); the window keeps every
  tab's content and shows the strip's (`dock.sync`). A tab an agent opens on a cluster the window
  does not show goes to that cluster's dock, active and open for when it is shown; `dock.activate`
  of such a tab shows its cluster first (`Session.Show`, no connect); the active tab closing gives
  way to a tab the dock shows, never to a hidden one; a local terminal belongs to the cluster it was
  opened for (`DockTab.Context`). `ui.toggleDock` is greyed while the shown cluster's dock has no tab.
- **The tab bar** follows the navigator's scope as before (`TabsInScope`), and greys a tab only
  while its cluster is not connected: a tab of another connected cluster is drawn as the shown
  one's.
- **Disconnect** in a connected cluster's tile, section row and welcome row menus passes its context
  (`cluster.disconnect {context}`, `disconnectOther`); the shown cluster's Disconnect is the
  window's as before. A connected cluster's first menu item says **Show**, not Connect.
- **Diagnostics** (item 3): a line per connected cluster under the summary, the shown one first,
  while more than one is connected: its API requests, open and warm watches, objects and memory
  (`DiagnosticsReport.Clusters`).
- **A failed connect while another cluster was shown** (the lead's item 7 of stage 2): the window
  shows the cluster it showed before again (`cluster.connect` of it, which only shows it,
  `followSwitch`), one toast says why ("Could not connect to <name>: …"; the hotbar's own toast stays
  away then), and the failed cluster's dots turn red. Without another connected cluster the failed
  connect stays shown with its error, as before.
- **`ConnectedCluster.LoginLapsed`** (JSON `loginLapsed`) added in [`connections.go`](../../internal/engine/connections.go)
  for the grey ring; stage 3 sets it.
- **Tests:** in the engine `TestDockFollowsTheShownCluster`, `TestAHiddenClustersTabStaysInItsDock`,
  `TestDockMemoIsCloned`, and `TestDockActivateShowsTheTabsCluster` in the commands; in the window
  `TestDotsShowEachClustersMark`, `TestABackgroundReconnectBlinksItsDots`,
  `TestAConnectedClusterIsShownWithoutAReview`, `TestDisconnectInAClustersMenuNamesIt`,
  `TestTabsGreyOnlyWithoutAConnection`, `TestAFailedConnectShowsTheClusterBefore`,
  `TestAFailedConnectWithoutAnotherClusterStays`, `TestWelcomeRowsShowTheConnectedClusters`,
  `TestDiagnosticsListsEachCluster`, `TestDockShowsTheShownClustersTabs`. PNGs: the hotbar's marks
  ([dark](../../internal/ui/fyneui/testdata/hotbar_marks_dark.png),
  [light](../../internal/ui/fyneui/testdata/hotbar_marks_light.png)), the navigator's section
  ([dark](../../internal/ui/fyneui/testdata/navigator_section_marks_dark.png),
  [light](../../internal/ui/fyneui/testdata/navigator_section_marks_light.png)), the dock before and
  after a switch ([before](../../internal/ui/fyneui/testdata/dock_cluster_before_dark.png),
  [after](../../internal/ui/fyneui/testdata/dock_cluster_after_dark.png), light
  [before](../../internal/ui/fyneui/testdata/dock_cluster_before_light.png),
  [after](../../internal/ui/fyneui/testdata/dock_cluster_after_light.png)), the Diagnostics with two
  clusters ([dark](../../internal/ui/fyneui/testdata/diagnostics_clusters_dark.png)).

**The lead's decisions of stage 2, for the owner to confirm:**

| Decision | Options weighed | Chosen | Why |
|---|---|---|---|
| The tab bar with several clusters connected | every connected cluster's tabs; the navigator's scope's | the scope's, as before: at a cluster its tabs, at the section every tab | One cluster shown is the owner's model, and Lens's bar filters to the cluster drilled into; the section still reaches every tab |
| A tab of another connected cluster | greyed as before; drawn as the shown one's | drawn as the shown one's; greyed only while its cluster is not connected | Grey said "a click connects"; a connected cluster's tab shows at once |
| Where the dock per cluster lives | the window; the engine | the engine (`dockscope.go`) | UI state is engine state: `ui_state`'s `activeDock` and `dockOpen` are what the dock shows, and `dock_read` still finds every tab |
| Tabs of no cluster (Audit Log, Diagnostics) | one cluster's dock; every cluster's | every cluster's | They show Cogpit's own state, not a cluster's |
| A cluster shown the first time | its last tab, open; collapsed | its last tab, collapsed; a tab of no cluster that was active stays, open as it was | Showing a cluster does not open a panel the user did not ask for |
| A tab an agent opens on a cluster not shown | show that cluster; that cluster's dock | that cluster's dock, active and open for when it is shown; the shown dock keeps its tab | Item 6: agents work on any connected cluster without moving the user's window |
| `dock.activate` of a hidden cluster's tab | refuse; show its cluster | show its cluster first, never a connect | As `ui.navigate` shows the cluster of an object (stage 1) |
| The dock's height | per cluster; the window's | the window's | A layout preference like the panels' widths, not engine state |
| The dock before any cluster is shown | its own tabs only; every tab | every tab | Nothing to scope by; only the Audit Log can be there then |
| A failed connect while another cluster was shown | the error shown (stage 1's choice); the cluster before shown again | shown again by the window, one toast, the failed cluster red | The lead's item 7 of stage 2 ("the shown cluster stays"); `cluster_connect`'s result still says `error`, so an agent learns the failure |
| Which hotbar tile is raised | every connected one; the shown one | the shown one; the others carry their mark | The raised tile says where the window is, the dots what is connected |
| A connected cluster's menu item | Connect; Show | Show | It connects nothing |
| A failed last connect on a hotbar tile | the shown cluster's only; every tile | every tile red while the connect log says the last attempt failed | Item 5's red "a connect failed" on every dot; it stays red once the window went back to the cluster before |
| The grey ring's colour | the muted text colour; the not-connected grey | the not-connected grey (`status.unknown`) as a line | Grey says resting, the ring that the connection stands |
| The ring's line | 1, 1.5 or 2 points | 1.5 points inside the 8-point dot | Readable at 1×; the dot's size unchanged |
| Several states at once | — | red over the grey ring over blinking over the green ring over filled | What needs the user first; a cold cluster's reconnect blinks (item 7) |
| Diagnostics with one cluster | a line for it; none | none: the summary says it; a line per cluster from two | No repetition |

**Not verified:** a real credential plugin and real clusters in the window (the tests drive a fake
engine; the engine's dock rules run against the pool's fake connectors); the marks on a Retina
screen (the renders are at 1×); the window under a wake that reconnects many clusters at once (each
change of `Connections` reads the hotbar again, not measured); the grey ring with stage 3's
`loginLapsed` on a real lapsed login.

**Stage 3, the two settings, built 2026-10-16** (the lead's run in a worktree of its own, beside
stage 2's):

- **Item 4, a background cluster's login** ([`connwatch_lapse.go`](../../internal/engine/connwatch_lapse.go),
  [`cluster/loginhold.go`](../../internal/cluster/loginhold.go)): Cluster Settings › General ›
  "Renew the login while the cluster is not shown", **on by default** (kept as its negation
  `holdBackgroundLogin` in `cluster-settings.json`, so that the default is the zero value;
  `cluster.settings.set {renewLoginInBackground}`, `cluster.settings.get`'s
  `renewLoginInBackground`). Off, a connected cluster the window does not show holds its credential
  plugin (`cluster.Conn.HoldLogin`; `Session.applyLoginHolds` after every show, connect and change
  of the setting). **How:** client-go's exec authenticator starts the plugin by itself in
  `refreshCredsLocked` (a credential expired by its time, none cached, a 401, a client
  certificate's handshake), inside its own round-tripper and in the exec, attach, port-forward and
  Helm transports, in front of which Cogpit has no round-tripper; `refreshCredsLocked` asks the
  Authenticator's unexported `interactiveFunc` first and runs nothing when it fails. Cogpit wraps
  that field once per Authenticator under the Authenticator's own lock, as D-66 sets its `stderr`:
  while every connection using the Authenticator holds it, the wrapper fails and **no plugin process
  starts**, and the request ends as `cluster.ErrLoginLapsed`; the login guard ends a held
  connection's 401 the same way instead of handing it to the plugin. The watch makes the login
  **lapsed**: `UIState.Connections[].loginLapsed` (the connection `connected` with `since` and the
  reason "the login lapsed; it is renewed when the cluster is shown"; not red), the gate holds the
  informers' lists and watches and refuses every other request with the lapsed login's error, the
  heartbeat pauses (a wake or a network change asks nothing of it), writes are disabled with "login
  lapsed: show the cluster to renew it", a log that has to open again writes "[cogpit] the login
  lapsed; it is renewed when the cluster is shown" and waits, port forwards do not dial, a running
  drain or Helm action stops as at an interruption. **Shown again**, the login is renewed as D-58
  renews the shown cluster's: the approved plugin runs once (one whose program changed asks first,
  D-26), Login expired (red, Log in again) when that does not help within 30 s. **Log in again**
  (`cluster.login {context?}`, GUI only, with a context since stage 3) renews it where it is, the
  hold let go for that login only; turning the setting on renews it in the background.
- **Item 8, restoring the last session** ([`restore.go`](../../internal/engine/restore.go),
  [`connect_background.go`](../../internal/engine/connect_background.go)): Preferences › App ›
  "Restore the last session", **off by default** (`preferences.set {restoreSession}`).
  `workspace.json` keeps `connected` (every context of `UIState.Connections`) and `shown` beside the
  tabs (D-41 changed above). At start with it on, the tabs come back as D-41 restored them, and
  `Session.RestoreLastSession` (called by `cmd/cogpit` before the window runs) connects the cluster
  shown at quit with `Connect`, which the window shows as any connect, then the others one after
  another with `ConnectBackground`, a connect that shows nothing (the cluster joins the pool cold,
  its entry `connecting` while it connects; a user's connect of it meanwhile waits for it and shows
  it). Each is a connect of the user's: an approved credential plugin runs as on any connect, one not
  allowed yet or changed since is refused with its reason (D-26). `--context` shows its own cluster
  and the restore connects the others behind it; a context no kubeconfig source offers any more is
  left out with a notice; a background connect that fails says why in a toast. Off: no tab, no
  connect. `cogpit-cli` reads no workspace and is not affected.
- **Agents** (D-2): both settings are UI commands, and each one's "on" makes Cogpit run credential
  plugins without a click, so agents and the CLI may turn each **off** only
  (`ErrRenewLoginGUIOnly`, `ErrRestoreSessionGUIOnly`); `cluster.login` stays GUI-only. An agent's
  read of a lapsed cluster comes from the cache with a note (`lapsedNote`), one that needs the
  cluster and every write get the lapsed login's error; no agent command renews it but a show
  (`cluster.connect`, `ui.navigate`), which the window shows.
- **Tests:** on client-go with a credential plugin script that notes each run,
  `TestHeldPluginDoesNotRun` (a held, expired login: `ErrLoginLapsed`, no process; let go, one run),
  `TestHeldLoginKeepsA401FromThePlugin`, `TestSharedLoginIsHeldOnlyWhenEveryConnectionHoldsIt`,
  `TestHoldWithoutAPluginChangesNothing`, and in the engine `TestHeldPluginOnARealStack` (two fake
  API servers: the background cluster's plugin login lapses without a run, showing it runs the
  plugin once); `TestABackgroundLoginLapsesWhileHeld`, `TestLogInAgainRenewsALapsedBackgroundLogin`,
  `TestTurningTheRenewalOnRenewsALapsedLogin`, `TestABackgroundLoginIsRenewedByDefault`,
  `TestALogOfALapsedClusterWaitsWithANotice`, `TestRenewLoginInBackgroundIsOnByDefaultAndKept`,
  `TestRestoreSessionIsOffByDefault`, `TestRestoreOffStartsWithoutTabsAndConnections`,
  `TestWorkspaceOfV0170Loads`, `TestWorkspaceKeepsTheConnections`,
  `TestRestoreOnConnectsTheLastSession` and `TestRestoreBehindAnExplicitContext` (two fake API
  servers), `TestRestoreOffOnARealStack`, `TestConnectWaitsForTheBackgroundConnect`;
  `TestSessionSettingsAgentsMayOnlyTurnThemOff`; in the window `TestPreferencesRestoreSession`,
  `TestClusterSettingsRenewLogin`, `TestStartWithoutTheRestoreShowsNoTabs`, renders
  `preferences_restore_session_{dark,light}.png` and `cluster_settings_login_{dark,light}.png`. The
  workspace tests that assumed the tabs come back by default turn the preference on.

**The lead's decisions of stage 3, for the owner to confirm:**

| Decision | Options weighed | Chosen | Why |
|---|---|---|---|
| Where the plugin is held | Cogpit runs the plugin itself (client-go's exec authenticator re-done); refuse every request of a background cluster; wrap the Authenticator's `interactiveFunc` | wrap `interactiveFunc`; the login guard for the 401 | client-go starts the plugin inside its own round-tripper and transports Cogpit cannot wrap; `refreshCredsLocked` asks `interactiveFunc` before it builds the command. The reflection is D-66's, and a test fails on a client-go that renames the field. Refusing every request would also stop a working login's streams; re-doing the authenticator was rejected in D-58 |
| Two contexts with the same plugin configuration | held per connection; per Authenticator | held only while every connection using it holds it | client-go keeps one login per plugin configuration: the shown cluster's renewal serves both, and holding it for the shown one would break the cluster the user looks at |
| What renews a lapsed login | showing the cluster; Log in again; an agent's command on it | showing it (an agent's `cluster.connect` or `ui.navigate` too, which the window shows), Log in again, turning the setting on | The lead's suggestion: an agent's read or write gets the lapsed login's error, never a silent plugin run |
| Streams of a lapsed cluster | stop them; let the open ones run and hold the ones that must open again | the open ones run (the API server checked the login when they opened); a log that must open again writes the notice line and waits; port forwards do not dial; a shell is not ended | Stopping a working stream loses output for nothing; a shell cannot be resumed |
| A drain or Helm action when the login lapses | go on; stop as at an interruption | stop (`errJobLoginLapsed`, `OperationContext`), finishable afterwards (D-11) | Its next request would fail anyway |
| The heartbeat while lapsed | on; paused | paused; a wake and a network change ask nothing either | Every request would fail; reachability is checked when the cluster is shown |
| The renewal setting's agents' rule | free both ways; the user only; agents may turn it off | agents and the CLI may turn it off only | Off runs less; on lets a browser login pop up while the user works elsewhere |
| The restore's connects | all at once; the shown first, the others concurrently; the shown first, then one after another | the shown first, then one after another in the background | The lead's suggestion: the window is usable at once, and never two plugins (two browser logins) ask at the same time; a plugin that waits holds the next ones back at most for its connect's limit |
| What the file keeps with the preference off | nothing of the session; the session as at quit | the session as at quit (`connected`, `shown`, the tabs) | Turning the preference on later restores the session of the last quit; the file held the tabs' context ids before |
| What comes back with the preference off | nothing; the window's arrangement | the column layout, the collapsed groups, the navigator hidden, the chat column | They are the window's arrangement; the owner's "without tabs and without connections" names the session |
| The dock's tabs | restored where they can be reopened (logs, the Audit Log); none, as D-41 | none, as D-41 | A log would open a stream and a shell run `exec` (a write) at start; an editor's text has its draft; the dock per cluster is stage 2's |
| The restore's agents' rule | free; the user only; agents may turn it off | agents and the CLI may turn it off only | The lead's suggestion: on, the next start connects clusters and runs their plugins without a click |
| `--context` with the restore on | skip the restore; restore behind it | its cluster shown, the others (the one shown at quit among them) connected in the background | The explicit wish wins the window; the rest of the session still comes back |
| A plugin not allowed at the restore | a dialog at start; refused with the reason | refused with its reason (a toast, or the shown cluster's error on the welcome page) | D-57's road for `--context`: allowed in the navigator, never a dialog that pops up at start |

**Not verified in stage 3:** a real credential plugin (kubelogin with a browser) held on a real
cluster and renewed when shown; the restore on the owner's computer with his clusters; a real lapsed
login in the window (stage 2's grey ring reads `loginLapsed`; rendered from a test state only).

## D-70: The disk image's window

**The need (the owner, 2026-10-16 morning, F15):** the disk image opened in Finder's default
window, small icons on white ("absolut hässlich"). Wanted: big icons, Cogpit on the left, an arrow,
the Applications folder on the right, and under the folder Kubernetes's helm wheel, to say "Cogpit
goes into your Kubernetes lifecycle". He decided that the look and the trademark question are
settled with him interactively, in a visual review, never by the lead.

**The review (2026-10-16):** a lab (a claude.ai artifact, its link in
[`packaging/dmg/lab.json`](../../packaging/dmg/lab.json)) drew the window in light and dark Finder
side by side, with presets and controls for the background, the arrow, the wheel, the icons and
the window, and the names' contrast measured on the background under them. **The owner's pick:**
a 600 × 340 window on the app's plate (the logo's navy gradient and halo), the icons at 124 points
at 27 % and 68 % of the width, a straight light-blue arrow, Cogpit's own wheel (the logo's wheel
without the C) at 240 points and 22 % opacity behind the Applications folder, the names at 14 points
on grey pills (about 4.4:1 against both black and white, so they read in both Finder themes).

**The trademark (read 2026-10-16, not legal advice):** the Kubernetes logo falls under the Linux
Foundation's trademark guidelines ([usage guidelines](https://github.com/kubernetes/kubernetes/blob/master/logo/usage_guidelines.md),
[LF trademark usage](https://www.linuxfoundation.org/legal/trademark-usage)): modifying it is allowed
for non-commercial items such as t-shirts and stickers; a mark is not to be incorporated into one's
own logos or designs, used on marketing material, recoloured or overlaid, and must not imply
endorsement. An installer's background is the product's own design, so the official logo was not
offered without the Foundation's written permission; a seven-spoke heptagon in Kubernetes blue was
shown for comparison only and not recommended (confusingly similar). The wheel in the disk image is
Cogpit's own drawing.

**How it is built (the owner's choice of three, 2026-10-16):** dmgbuild (pure Python, MIT, pinned
by version and hash) writes Finder's view settings itself, without Finder or AppleScript, so it runs
on a CI runner; but it is code from PyPI, and until now the signing job ran only Apple's tools and
bash. Options weighed: (a) a template made in the app's job, which holds no secret, filled in by
the signing job with Apple's tools after checking its contents; (b) a job of its own building the
whole image around the signed app, the signing job comparing the app's code directory hash (a job
and an artifact more, about five minutes per release, estimated); (c) dmgbuild in the signing job
with its environment emptied and the keychain locked (third-party code on the same machine as the
identity, which a left-behind process could use once the keychain is unlocked for the disk image).
**Chosen: (a)**, the lead's recommendation ([releasing › the disk image's window](releasing.md#the-disk-images-window)).

The lead's decisions of the build, for the owner to confirm:

| Decision | Chosen | Why |
|---|---|---|
| The tool | dmgbuild 1.6.7 with ds_store 1.3.3 and mac_alias 2.2.3, wheels only, `--require-hashes` | `create-dmg` drives Finder through AppleScript and needs a login session; a `.DS_Store` made once by hand breaks with the volume's name, which carries the version |
| Python on CI | `actions/setup-python@v6`, 3.13 | dmgbuild needs 3.10 or newer, the system's Python is 3.9 |
| The background | the lab's own drawing code rendered at 1× and 2× (headless Chrome), committed as PNGs, joined into one TIFF at build time | the window the owner saw is the one that ships, pixel for pixel |
| The check of the template | a UDRW image, the volume's name, exactly `.DS_Store`, `.background.tiff`, `.VolumeIcon.icns` and `Applications` → `/Applications` | anything else in the template would ship signed and notarized; the files' bytes are not checked ([updates › H-18](../security/updates.md#h-18-dependencies)) |
| Hiding `.app` | no Finder flag | `SetFile -a E` puts `com.apple.FinderInfo` on the bundle and `codesign --verify --strict` refuses it; Finder hides an app's extension anyway |
| Renovate | proposes updates of the three packages, never merges them on its own | their output ships in every disk image |

**Built 2026-10-16:** [`dmg-template.sh`](../../hack/macos/dmg-template.sh),
[`dmg_settings.py`](../../hack/macos/dmg_settings.py),
[`dmgbuild-requirements.txt`](../../hack/macos/dmgbuild-requirements.txt), `build_dmg` and
`check_dmg` in [`release.sh`](../../hack/macos/release.sh), the step in
[`build.yml`](../../.github/workflows/build.yml)'s app job, `make dist-dmg-template`. **Checked**
locally (`make dist-app`, `dist-dmg-template`, `dist-macos MACOS_SIGN=adhoc`, VERSION 0.0.99).
**Not checked yet:** the jobs on CI (the next release runs them); the names on their pills in a real
Finder window on the owner's Mac (the lab placed them 6 points under the icons, Finder's own
placement may differ); a Retina and a non-Retina screen.

## D-67: The demo cluster stays out of releases

**Decided by the owner (2026-10-16, task F14):** "Der Demo-Cluster ist für die Entwicklung, er soll
nicht über den Release ausgeliefert werden." Asked how, he chose: compiled out of release builds by
a build tag that only the release pipeline sets. A release then contains neither package
`internal/demo` nor the flags `--demo`, `--demo-pods` and `--demo-events`, and lists no context
`demo`; development builds, `make run-demo`, the tests and the README's pictures (`make showcase`)
keep the demo. Releases up to v0.16.0 carry it.

**Built 2026-10-16.** The choices of the build, for the owner to confirm:

| Question | Options weighed | Chosen | Why |
|---|---|---|---|
| The tag's name | `release` (what the build is); `nodemo` (what it leaves out); a positive `demo` that development builds would set | `nodemo` | A `release` tag would invite other release-only behaviour behind the same switch, code that no development build, test or CI job runs. A negative tag keeps every build without tags as it was (`go test`, `go run`, gopls, golangci-lint, `make showcase`); only the pipeline opts out. No dependency reacts to it: none of the 1,614 package directories both binaries build from has a file constrained by `nodemo` (checked 2026-10-16) |
| Where the tag cuts | a pair of tagged files in every package that names the demo (the UI, the commands, `cogpit-cli`); a pair only where the demo is wired in, and everything else from data or one constant | the second: [`engine/connector_demo.go`](../../internal/engine/connector_demo.go) (`//go:build !nodemo`: `DemoBuilt = true`, `DemoOptions` = `demo.Options`, the demo's backend) against [`connector_nodemo.go`](../../internal/engine/connector_nodemo.go) (`DemoBuilt = false`, an empty `DemoOptions`), and [`cmd/cogpit/demo.go`](../../cmd/cogpit/demo.go) (the three flags) against [`nodemo.go`](../../cmd/cogpit/nodemo.go) | The window shows the demo only where the connector says so (`ContextInfo.Demo`, `Connection.Demo`): with no such context, the welcome page, the navigator, the hotbar and the status bar have nothing to show. The constant lets the compiler drop the remaining branches |
| `ClusterConnector.IncludeDemo` in a release | remove the field; keep it without effect | kept, ignored (`DemoBuilt && IncludeDemo`) | `cogpit-cli` and the tests set it unconditionally; one rule in one place |
| A kubeconfig context called `demo` in a release | shadowed by a demo that is not there; one like any other | one like any other: listed, resolved and connected through its kubeconfig; Cluster Settings and Remove from Cogpit treat it as any context (`engine.IsDemoContext` is false) | Only a run without a configuration directory gives a context its bare name as its id, but there the old comparison with `"demo"` would have shown the demo's notes for a real cluster; with sources, a bare `demo` (`--context`, a link, an agent) now resolves to the one file that defines it |
| Texts that named the demo (the tools `cluster_contexts`, `cluster_connect` and `cluster_remove`; `cogpit-cli contexts --help`, `get`'s `--context` and its usage example; a Prometheus setting's refusal) | worded per build; worded for both | for both | The demo's id is "a context id as `cluster_contexts` lists it"; the generated [commands.md](commands.md), from a development build, stays true for a release. The Prometheus refusal names the demo only when the connection is the demo |
| What guards it | the release's own check; a Go test of the Makefile; a check in CI | all three: `make dist-app` passes `DIST_TAGS` and fails unless `go version -m` shows `-tags=nodemo` for both binaries; [`TestReleaseBuildsLeaveTheDemoOut`](../../internal/engine/connector_test.go) checks that `dist-app` passes `DIST_TAGS`, that `build` gives both binaries `GO_TAGS`, and that `DIST_TAGS` name the tag of `connector_nodemo.go`; CI's lint job runs `make check-nodemo` | A tag no file knows is ignored without a word, and the release would ship the demo. `check-nodemo` proves the result: `go list -deps` has `internal/demo` in both binaries without the tag and in neither with it, `./cmd/...` and `./internal/engine/...` lint with it, and `TestNoDemoInARelease` runs in `internal/engine` and `cmd/cogpit-cli` |
| Which job runs `check-nodemo` | a new job; the macOS job; the lint job | the lint job | Linux, 3.2 minutes, Fyne's headers installed; about a minute more locally (not measured on CI yet) |
| Tests that need the demo | compiled with the tag and failing; left out with the tag | left out where the tag is vetted: `engine/connector_demo_test.go` (the demo tests of `detail_test.go` and `applications_test.go` moved there) and `hidemetrics_test.go` carry `//go:build !nodemo`, and `cogpit-cli`'s tests size the demo in `demo_test.go` (`nodemo_test.go` for the variant) | The packages `check-nodemo` lints compile with the tag, tests included. The UI's, the agent's and the kind tests drive the demo and are not compiled with the tag |

**What it saves** (2026-10-16, darwin/arm64, `go build -ldflags '-s -w'` as `DIST_LDFLAGS`): the
app's `cogpit` 108,649,938 bytes without the tag, 108,341,170 with it (308,768 less); `cogpit-cli`
75,175,330 against 74,904,306 (271,024 less), about 0.3 % each. The release binaries carry no
string of the flags (`demo-pods`, their help text) and no symbol of `internal/demo` (checked with
`grep -a`). The point is the owner's: a release ships no development fixture, not the bytes.

**After an update from v0.16.0 or earlier** (read in the code, not tried): a restored tab of the
context `demo` is dropped as a context no longer offered (`goneContexts`: not offered, while the
list of contexts read without an error), and a hotbar tile of it shows as missing.

**Open:** the owner's confirmation of the build's choices; `check-nodemo`'s time on CI; Windows and
Linux releases (M11) must build through `DIST_TAGS` too.

## D-68: One short name for a cluster, and the user Cogpit is logged in as

**Decided by the owner (2026-10-16 morning, task F17)**, his answers of the same morning. He looked
at the navigator's root row: its badge, the cluster's name (the Cluster Settings' name, else the
context's) and as its subtitle the kubeconfig file, "Synthetic demo cluster" or a connection word.

1. **One short name for a cluster, everywhere** ("überall gleich"): a context name with an "@"
   shows only the part after the last "@" ("kubernetes-admin@prod-eu" → "prod-eu"), else the whole
   name; a name the user set in Cluster Settings › General still wins. The window names a context so
   wherever it names one: the navigator's root and section rows, the hotbar's tiles, the top bar,
   the status bar, the tab bar, the command palette and the welcome page. The full context name
   stays in each tooltip and in Cluster Settings. The navigator's search still finds a context by
   its full name. Agents keep the full context ids in every command and in `ui_state`.
2. **The user Cogpit is logged in as, in the root row's subtitle**, fetched live at every connect
   as `kubectl auth whoami` asks: a `SelfSubjectReview` in package `cluster`, no kubectl process; a
   failure never fails the connect; renewed when the login is renewed and on every reconnect that
   re-authenticates. Shown: the username, and beside it a small box with an icon whose tooltip lists
   the groups, the groups only (no UID, no extra fields); no groups, no box.
3. **Fallback:** where the API server does not know the review (404, older than 1.27) or refuses
   it (403), the kubeconfig's user entry's name with the mark "(kubeconfig)" and the tooltip "not
   confirmed by the API server"; the reason in the connect log's line.
4. **Not connected:** the last known user of the context, dimmed, its tooltip saying when it was
   last seen ("last seen 2026-10-16 09:12", in the user's date and time format, D-31). Only the
   username is kept, per context, where Cogpit keeps its per-cluster state, owner-only. Never
   connected: the kubeconfig file as before. The demo cluster keeps "Synthetic demo cluster".
5. **Agents:** `ui_state` names the shown cluster's username and groups, and whether the API server
   confirmed them or the kubeconfig stands in. A read without secrets; that an agent (and a built-in
   agent's LLM endpoint) can learn the organisation's structure from the groups is the owner's
   choice ([agents and MCP › H-113](../security/agents-and-mcp.md#h-113-the-users-name-and-groups-reach-every-agent-that-reads-ui_state)).

**Built 2026-10-16.** Where:

| What | Code |
|---|---|
| The short name | [`engine.ShortContextName`](../../internal/engine/displayname.go) (the part after the last "@", trimmed; nothing after it keeps the whole name), used by [`engine.ClusterName`](../../internal/engine/clustersettings.go) (the display name, else the short name of the context's name without its file), which the window's `sh.identity` and every place below read: the tabs' titles (`tabTitle`), the palette's tile entries, the hotbar's tiles (`HotbarTile.DisplayName`, new beside `label`), `cluster.settings.get`'s `name` |
| The review | [`cluster.Conn.ReviewUser`](../../internal/cluster/whoami.go): `authentication.k8s.io/v1` `SelfSubjectReviews().Create`, on a 404 `v1beta1` once more (client-go 0.37.1 has both typed clients); the username and the deduplicated groups kept on the connection (`Conn.User`), UID and extra dropped there; 404, 403 and every other failure give the kubeconfig's user entry's name, read again from the connection's files, with the reason |
| At the connect | [`engine/whoami.go`](../../internal/engine/whoami.go): `Session.learnUser`, one line in `Session.Connect` after the connector returned and before the connection is published, bounded by 5 s (`DefaultProbeTimeout`); its line in the connect log ("Logged in as … (confirmed by the API server); groups: …", or "The API server did not confirm the user: <reason>; Cogpit shows the kubeconfig's user entry …") |
| The renewal | `Session.renewUser`, one line in the connection watch's `reconnect` (connwatch.go): after an interruption and after Log in again (`cluster.login`), in the background; "Logged in again as …" in the connect log |
| The UI state | `UIState.User` (`ClusterUser`: `context`, `username`, `groups`, `source` `apiServer`, `kubeconfig` or `lastSeen`, `reason`, `lastSeen`), derived after every change (`syncUser`) from the session's book of answers and last known users |
| The last known users | [`engine/whoami_store.go`](../../internal/engine/whoami_store.go): `<user config dir>/Cogpit/cluster-users.json` beside `cluster-settings.json`, owner-only, the username and when it was last seen per context id |
| The window | [`fyneui/clusteruser.go`](../../internal/ui/fyneui/clusteruser.go) (the root row's subtitle, the groups' box, its tooltips), [`fyneui/clustername.go`](../../internal/ui/fyneui/clustername.go) (the bars' name with the full name as the tooltip) |
| Tests | `TestShortContextName`, `TestClusterNameIsTheShortName`, `TestConnectLearnsTheUser`, `TestUserFallsBackToTheKubeconfig`, `TestLoginRenewsTheUser`, `TestSyncUserFollowsTheContext`, `TestLastUsersFileUnreadable`, `TestPaletteNamesTilesByTheShortName`, `TestHotbarTilesCarryTheDisplayName` (engine); `TestReviewUser*` (cluster, against the fake API server's `FakeIdentity`: v1, v1beta1 only, 404, 403, 500); `TestRootRow*`, `TestShortClusterNameEverywhere` and the renders `navigator_user*` (window); `TestKindWhoAmI` (kind: a ServiceAccount without permissions of its own, its username and its three groups confirmed, one review) |

**The lead's decisions, for the owner to confirm:**

| Question | Options weighed | Chosen | Why |
|---|---|---|---|
| When the connect's review runs | before the connection shows Connected; in the background after it | before, bounded by 5 s | The root row never shows a connected cluster without its user, and `ui_state` names the user the moment `connection.state` says connected. The cost is one round trip on the connect (P1 counts it; measured on the fake API server: within the connect's 25–34 ms); a server that does not answer the review delays the connect by 5 s at most, then the fallback stands |
| Kubernetes 1.26's `v1alpha1` | ask it after `v1beta1`; not | not | Behind a feature gate that is off by default; a third request on every old cluster for almost none |
| What renews it | only after Log in again; every reconnect | every reconnect (after an interruption and after Log in again) | Cogpit cannot tell whether the credential plugin ran while the connection was away (its token may be another identity's); one request per reconnect |
| What "last seen" is | the connect's time; the time the connection ended | both: the last confirmation (connect, renewal), and the moment a connection that showed it ended (a disconnect, another cluster, a failed reconnect) | Tells the user when that identity last worked, not only when it began |
| What is kept | every answer; only the API server's | only usernames the API server confirmed, never the fallback's label, never the groups | The fallback's name says nothing of who the server sees; the groups were the owner's "not stored" |
| Where it is kept | in `cluster-settings.json`; a file of its own | `cluster-users.json` beside it | It is no setting: `cluster.settings.get` and the settings' `IsZero` would change with every connect |
| The fallback's and the last known user's tooltips | the owner's words alone; with a second line | the owner's words, then a line: "The name of the context's user entry in its kubeconfig; its connection updates say why", "The user the API server last confirmed for this cluster" | The reason itself stays in the connect log and in `ui_state`'s `reason` |
| The groups' box | an icon; an icon and a count | an icon (two people, drawn for Cogpit) in a small outlined box | The owner's words; the tooltip lists them |
| Its tooltip | the box a hoverable widget of its own; the row choosing its tooltip by where the pointer is | the row (`tipAt`) | The row keeps its hover while the pointer crosses the box and the name |
| The short name's edge cases | | "@prod" → "prod"; "admin@", "@" and blanks after the last "@" keep the whole name | Nothing after the "@" would leave no name |
| The full name where nothing has a tooltip | — | the palette's tile entry: "Open prod-eu", the full context name as its category (shown at the row's end and searched); the welcome page's row: the short name as its title, "context <full name>" in its second line, as for a display name | The palette's and the welcome page's rows have no tooltips |
| The tooltips that carry the full name | | the section row's first line (the full context name before its source; until now the display name), a tab's (its title, then the context id), the top bar's name (the context's full name; the file stands beside it), the status bar's (the context id with its file), the hotbar tile's and the root row's (the context id, as before) | |
| What keeps the full id | | the engine's connect log and toasts ("Connecting to <id>", "Reconnected to <id>"), Remove from Cogpit's list of the contexts that leave with a file, the navigator items' "<item> - <context>" tooltips | Logs and a confirmation name exactly what they mean; the hotbar's own toasts use the short name |
| Where the root row shows a cluster that is not connected | | while it connects (the navigator shows the section after a failed connect or a disconnect, D-57); its dot says the state, the subtitle the last known user | |

**Open:** the owner's confirmation of the lead's decisions; Lens's root row is not captured for a
user, so the row is Cogpit's own ([UI parity](../parity/ui-parity.md)).

## Revision history

| Date | Change | Summary |
|---|---|---|
| 2026-10-16 | minor | D-58: the owner's audit decision of 2026-10-16 on the selection bar's greyed Delete, (c), built: its tooltip names the engine's reason in the row menus' form ("Delete — connection interrupted", the login expired, a cluster not connected) and says what it does again with the connection back; a shown cluster whose login lapsed greys it too; the build's choices for the owner to confirm; the index row |
| 2026-10-16 | minor | D-59: the owner's audit decision of 2026-10-16 on the copy from a Secret's editor, (c), built: D-25's copy icon in the key's line above each editor (an added key's too), copying the whole value as typed through the clipboard rule for Secret values, never the plain clipboard; the build's choices for the owner to confirm (always shown, its tooltip, an empty value copies nothing); the table row, D-25's Secret paragraph and both index rows |
| 2026-10-16 | minor | D-61 withdrawn by the owner and removed: Cogpit Metrics (the bundled chart, `cluster.metricsStack`, `cluster.setMetricsStack`, the section, `make metrics-bundle`, the Renovate rules, detection's preference for Cogpit's own Prometheus) gone; N183 to N192 moot; H-105 closed; the reading improvements (N193 to N197) stay; detection finds the prometheus chart's server under any release's name |
| 2026-10-16 | minor | The owner's audit of the night decisions of 2026-10-15/16 (2026-10-16 afternoon) recorded: confirmed as built in D-10 (send all tools, the plain-http warning and the CA file's path and reading, editing a sent message), D-11 (a stopped drain), D-15 (the scorecard's main choices), D-43 (a node's network and filesystem), D-55 (the rebase and the drafts), D-57 (but the menu path and the welcome page's button), D-58 (the build's table), D-59 (but adding and removing keys and the copy), D-62, D-63 and its drain, D-64 and D-65 (the rows it asked); changed: D-59's copy, an icon over the editor, and D-58's greyed Delete, its tooltip naming the reason, both decided, not built; D-57's welcome page button to be changed, how not said; his four other changes left to the changes that build them; the index's duplicate rows of D-43, D-62 and D-63 removed |
| 2026-10-16 | minor | D-10 (6): a chat connection's CA file replaces the system's roots for that connection instead of adding to them (`x509.NewCertPool`), a public certificate failing with "this connection trusts only its CA file" (the owner's decision in the audit of 2026-10-16, N141, built the same day); H-104's link |
| 2026-10-16 | minor | D-57: the eight cluster and kubeconfig commands still under `File/` moved to `Cluster/` (the owner's decision in the audit of 2026-10-16, N121, built the same day); its index row |
| 2026-10-16 | minor | D-55 and D-59: the owner's audit decision of 2026-10-16 (N152 (b)) built: a Config Map's and a Secret's drawer add and remove keys too, `add` and `remove` beside `data` on the review and save commands (one write), the key checked as the API server checks it, a save that removes a key destructive (escalated; the review its confirmation in the window, bound to the command, object, UID and keys; an agent's Config Map removal always asks), the rebase with an added key absent and a removed key unchanged, kept edits and drafts with the keys (a Secret's in memory only); the lead's decisions for the owner to confirm in two tables; two points in D-55's "What it does not cover" (the Edit tab's removal not destructive, a key beyond the first 100); the index rows |
| 2026-10-16 | minor | D-69 stage 2 built (the window): switching without reconnecting, the marks of item 5, each cluster blinking for itself, the dock per cluster (`dockscope.go`), Disconnect per cluster, Diagnostics per cluster, a failed connect showing the cluster before again; `ConnectedCluster.LoginLapsed` for stage 3; the lead's decisions for the owner to confirm; its index row |
| 2026-10-16 | minor | D-69 stage 3 built: a background cluster's credential plugin held while its setting "Renew the login while the cluster is not shown" is off (client-go's `interactiveFunc` wrapped, the login guard for the 401), its login lapsing (`loginLapsed`, not red) until shown or logged in again; Preferences › App › Restore the last session (off by default; the shown cluster first, the others one after another in the background); D-41 and D-42 changed (no tabs at start without the preference), D-58's background renewal; the lead's decisions for the owner to confirm |
| 2026-10-16 | minor | D-68 added (the owner's F17 of 2026-10-16) and built: one short name for a cluster everywhere (the part of its context name after the last "@", a display name still first, the full name in each tooltip and in Cluster Settings); the user Cogpit is logged in as in the navigator's root row, from a SelfSubjectReview at every connect and reconnect (v1, else v1beta1), the groups in a box's tooltip, the kubeconfig's user entry marked "(kubeconfig)" on a 404 or 403, the last known user dimmed with when it was last seen, kept in `cluster-users.json`; `ui_state`'s `user`; the lead's decisions for the owner to confirm; H-113 opened |
| 2026-10-16 | minor | D-58's S3 amended by the owner (F13): the connection as a dot, never a word, in the top bar and the status bar; Connected in the Running green; blinking 500 ms / 500 ms while (re)connecting, at least 3 s; Login expired and Connection failed red at once; every dot of the cluster in step; a click opens the connection log (D-66); the lead's choices (Connecting blinks, 30 % alpha, the window's minimum, Reduce motion, the welcome row's dot, the status bar's word, the tile's hover) for the owner to confirm; D-22 and D-23 follow |
| 2026-10-16 | minor | F11 (owner, 2026-10-16): D-4's scheduled update check every hour (`SUScheduledCheckInterval` 3600, Sparkle's least; its default was a day); D-54's Update button says "Update to v<version>", "Update" without a version or room, and sits on the bar's centre line, the line under the bar now directly under it |
| 2026-10-16 | minor | D-67 added (the owner's decision, task F14): the demo cluster compiled out of releases by the tag `nodemo`, which only the release pipeline sets (`DIST_TAGS`); built: the tagged files in `internal/engine` and `cmd/cogpit`, `IncludeDemo` without effect in a release, a context `demo` one like any other there, the texts that named the demo worded for both builds, `make dist-app`'s check, `TestReleaseBuildsLeaveTheDemoOut`, `make check-nodemo` in CI's lint job; the sizes saved; the build's choices for the owner to confirm. D-42 and D-47 follow (`--demo` and the demo's link a development build's) |
| 2026-10-16 | minor | D-66 (the connection log, F13) decided by the owner and built: the Connection Updates dialog for every context, live, Copy; attempts on disk per context, 7 days and 1 MiB; the steps and the error picture; a credential plugin's standard error on a failed login, scrubbed and capped; the lead's choices for the owner to confirm; H-112 opened; its index row |
| 2026-10-16 | minor | D-15: the scorecard built in the night run: `make scorecard` (`hack/scorecard`), `testkit/scorecard`'s `Record` in nine budget tests, five benchmarks' ns/op, docs/parity/scorecard.md (13 budgets measured, all within their limits; P3, P6, P7 not measured); the lead's choices for the owner to confirm; not in CI |
| 2026-10-16 | minor | D-64 (envtest) recorded and built in the night run: controller-runtime's envtest v0.25.2 (no k8s.io module moved), etcd and kube-apiserver of envtest-v1.37.0 pinned by SHA-512 (`make envtest-bin`), `TestEnvtestStandbyAPIServerStopped` stopping the API server with SIGSTOP, the job Integration Tests (envtest); the lead's decisions for the owner to confirm, measured locally, the CI job not observed yet |
| 2026-10-16 | minor | D-65 (G3's discovery cached on disk) decided by the lead in the night run for the owner to confirm, built the same night: Cogpit's own layer under client-go's memory cache, one owner-only file per server and identity, the server version and the definitions' digest checked, 6 hours once confirmed, else 10 minutes; connect times measured on a real cluster; D-13 says the answer holds no object |
| 2026-10-15 | minor | D-57 (the owner's interview of 2026-10-15 evening) built in the night run: the context picker, the top bar's "Switch context ▾" and the navigator's scope dropdown gone; ⌘P, the Cluster menu's Connect to Cluster… (was Switch Context…), the palette's Connect to Cluster, the welcome page's and the workspace's button and a refused `--context` show the navigator's section with its search focused, the navigator shown first if ⌘B hid it; Escape on that empty search goes back to the cluster's tree; at a cluster the header's "‹ All Clusters", the root without its ‹; at the section the title "All Clusters" with a "+" (Add Kubeconfigs…); the section's row tooltip and menu take over what the picker's row said (the error, a login waiting, an unchecked certificate, Show Connection Updates…); `cluster.connect`'s menu path `Cluster/Connect to Cluster`; the lead's decisions for the owner to confirm listed. D-42 (a refused `--context`), D-47 (the link's refusal names the navigator) and D-51 (the dropdown's bank gone with it) follow |
| 2026-10-15 | minor | D-10 F9 built in the night run: a sent message edited and sent again (`ChatAgent.Resend`), everything after it cut from the window, the model's history and the kept file; seven decisions of the lead for the owner to confirm (editing while a turn runs, where editing starts, the keys, when the input could not send, the first message's title, a private conversation, when the file is rewritten); H-100 opened; the index row: the chat's audit answers built |
| 2026-10-15 | minor | D-10 (6) built in the night run: the warning for `http://` to another machine (Preferences › Agent, the list's mark, the connection test) and a CA file per connection added to the system's roots, Go 1.27.2's verifier checked in its source and recorded; five decisions of the lead for the owner to confirm (the loopback exemption, where it warns, a CA file over http, the path, when it is read); item 10 revised; H-104 opened |
| 2026-10-15 | minor | D-10: the owner's audit answers of 2026-10-15 for the chat recorded ((4) send all tools per connection, (6) a warning for `http://` to another machine and a CA file per connection, F9 a sent message edited and sent again; F8 postponed); (4) built in the night run (`ChatConnection.AllTools`, 179 tools and about 56,000 tokens measured), with the lead's decision for the owner to confirm (no meta-tools with all tools on); item 2 and the index row say so |
| 2026-10-15 | minor | D-59 (the owner's audit answer of 2026-10-15) built (night run): a Secret's values edited in the drawer after Show, in the window only: `secret.reveal`'s `whole`, `secret.reviewData` (a masked diff and a dry run, the rebase of D-55) and `secret.saveData`, GUI-only, the audit line of a refused call with the Secret and its keys only; unsaved values in the window's memory only; the lead's decisions for the owner to confirm in a table; H-103 (the last-applied annotation a save drops) opened; index row; D-55's index row and open point point to it |
| 2026-10-15 | minor | D-55: the owner's audit answers of 2026-10-15 built (night run): `configMap.reviewData` takes the edited keys' values at the base version (`from`) and rebases the edit onto a newer version none of those keys changed in, the diff and the dry run against it shown before Save, which never rebases; `configMap.keepEdits` (GUI-only) keeps the drawer's unsaved values in memory and as a draft of type `configMapData` (`drafts/`, 0600), offered at the next connect and restored into the drawer by `editor.restoreDraft`; two of the lead's questions marked answered; the lead's decisions for the owner to confirm of the night listed; "What it does not cover" brought to the new state; a Secret's values in the drawer decided (owner, 2026-10-15) |
| 2026-10-16 | minor | D-58 (the owner's decisions S1–S8 of 2026-10-15 evening) recorded and built: S1 measured on a kind cluster of its own (no hang of the window offscreen; a waiting credential plugin stalls every request of the connection behind client-go's authenticator), the connection watch (wake, network change, heartbeat, broken watch), the states `interrupted` and `loginExpired`, the banner, writes disabled, the backoff, the login guard and Log in again, logs and port forwards resuming, shells ended with Reconnect, drains and Helm actions stopped as after a crash, agents' reads noted; the lead's defaults and the build's decisions for the owner to confirm in a table; H-101 and H-102 opened |
| 2026-10-16 | minor | D-63: Drain built as a background job (night run): the cordon and a step per pod (the cache's pods at the start, the drain's own list once it listed), the PodDisruptionBudget's wait in the pod's step, Cancel and every other stop leaving the drain `stopped` in its journal and offered as after a crash, nothing uncordoned by itself, an MCP client that gives up no longer stopping it, the banner without a row for a drain running in the window; nine decisions of the lead for the owner to confirm; D-11 and D-58 follow; H-111 opened |
| 2026-10-16 | minor | D-63 (G4, night run) built: the actions on a selection of more than ten objects as background jobs for every origin, progress in the dock, `job.cancel` and the tab's Cancel, the `ended` audit line; the lead's decisions for the owner to confirm (ten and fewer within the call, any origin cancels any job, a request under way finishes, the bound of 500 kept); Drain, a multi-document Apply and D-58's interrupted state left open |
| 2026-10-16 | minor | D-61 added (night run, the lead's decision for the owner to confirm): Cogpit Metrics, Lens's Lens Metrics page as Cluster Settings › Cogpit Metrics, eleven choices with their options (names, one release per switch, the chart embedded and pinned, the version, the values and the storage, the command and the window's dialog, the namespace, detection, order and crash safety, node-exporter's host access, rights); built and tried on kind and a real cluster. D-43: a node's network and filesystem from node-exporter, with the lead's choices for the owner to confirm (the interfaces counted, the disks, one query with both sources) |
| 2026-10-16 | minor | D-62 added: checks against a real cluster the owner provides for tests, the lead's decisions for the owner to confirm (the TestReal* tier, P11's reading, Flux's OCI Helm repositories, events' custom kinds, a failed drain's wording, confirmations naming kinds) |
| 2026-10-16 | minor | D-36 checked against a live Argo CD 3.5.5 controller on a real cluster (night run): Cogpit's operation runs, a sync written in a deny window waits in the controller; fixed (the lead's decisions for the owner to confirm): the manual-sync exceptions need every matching window to allow them, Argo CD's glob (`*` over `/`), no cron descriptors, a deny window's end followed through its reopenings, the project looked up in `status.controllerNamespace` (an Application in another namespace was not held to its project's windows), the operation carries the confirmed source as `argocd app sync` writes it; the index row |
| 2026-10-15 | minor | Decided by the owner and recorded in D-9, the entry that holds Cogpit's scope against Lens (the alpha gate itself lives in the roadmap, which no D-entry decides): verifying the credential plugins against EKS, GKE and AKS is out of scope until long after parity, no longer an alpha-gate step or part of M1's definition of done; index row D-9. The docs brought to the code (the owner's rule 10): the index row of D-10 said "being built (M12)" for what the code shows built (the engine side, the chat column, Preferences › Agent, the private connection, the key file); the index lacked D-14 to D-32, now one row each, its status from the entry and checked against the code; D-1: S7's re-check since CI (the macOS app packaged, Windows compiled and linked, no Windows or Linux package); D-2: "the rest is not built yet" over answers each marked built 2026-10-14, and the token's place (`<user config dir>/Cogpit/mcp-token`, kept since D-53); D-4: Sparkle said not built (built 2026-10-14, key 2026-10-15); D-44: the built-in agent said not built; D-48: "the release's app is ad-hoc signed today" (Developer ID since v0.6.0, the login item still not tried with it); D-16: "not verified until the first signed release" (signed releases with the bundled `cogpit-cli` since v0.7.0, checked by download on 2026-10-14; a run from a quarantined copy not tried); the decision round's intro: the night run built every answer that asked for a build but R3 and A10 |
| 2026-10-15 | minor | D-4: Sparkle's update key set up by the owner (the environment `sparkle` with its rules and Pages by the lead with the owner's go-ahead; the secret and the variable by the owner), in v0.11.0 to v0.13.0, no signed appcast yet (`appcast.xml` 404: v0.9.0 to v0.12.0 got no files, v0.13.0 notarizing; checked with `gh` ~19:00 UTC); v0.6.0 to v0.8.0 carry the signed disk image; the notarization's 330-minute job noted at A9. A10 decided further by the owner: the semantic-release App the tag ruleset's only bypass, nobody tags by hand, an emergency tag by an admin switching the ruleset off; the settings not made (no ruleset, `release`'s admin bypass on). Confirmed by the owner: D-47 as a whole (navigation only with the build's defaults; the A7 ask verified built); D-10 item 12's repaired key file (a log warning, no bell notification, against the lead's recommendation) and no keychain migration, its third choice not asked; D-51's ↵ on an agent's palette text (b); D-51's dropdown bank moot (the scope dropdown to go, not built). The index rows of D-2, D-33, D-34, D-36, D-41 and D-44 said "not built" for answers of 2026-10-14 evening that the code shows built in the night run (A1, A2, A16, A18, A20, A65; A34 and A91; A80; A33; A63; A3, A5, A31): corrected |
| 2026-10-15 | minor | D-24 revised (owner): the glowing C helm of the second icon lab as the app icon and the logo everywhere (the header, the welcome page, About, the README's banner, the social preview), the same in both themes; every version of the logo kept and drawable (`logo.Versions`, `hack/icon -version`, `assets/logo/`), D-24's own drawing matching its shipped icon pixel for pixel |
| 2026-10-15 | minor | D-56 (the owner's decisions of 2026-10-15) built: the terminal's scrollback in the fork, a compact store (205 bytes for a line of 80 cells against 9,267 in the grid) without a line limit, bounded at 256 MiB, the view drawing only the rows in view (#147's model measured at 56× the refresh cost with 1,000 rows), a draggable bar, the view kept while scrolled back, reflow of the scrollback and the main screen, the wheel to full-screen programs, a selection over the scrollback, `dock_read` by stable line numbers (D-44 follows); the lead's decisions for the owner to confirm listed |
| 2026-10-15 | minor | D-10 item 12 replaced by the owner (2026-10-15, "übertrieben, erzeugt Reibung": each self-built binary made macOS ask for the login password): no keychain, the API keys in the owner-only `llm-keys.json` on every platform; built the same day, the keychain code removed; the lead's decisions for the owner to confirm (a key file others could access repaired, no migration of a keychain key, a file that does not decode replaced); H-99 opened in credentials |
| 2026-10-15 | minor | D-55 (the owner's finding F3, decided 2026-10-15) built: a Config Map's Data with each key on a line of its own and its value in an editor (yamlview, plain scalars as strings, 3 to 12 lines, a grip), Revert and Save at the end; `configMap.readData`, `configMap.reviewData` and `configMap.saveData` (mutating, `update configmaps`) write as `object.apply` does: dry run, then update with the resourceVersion the values are based on, a `DataConflictError` with the values against the newer version and nothing written when it changed; values only, no binary data, Helm records or immutable Config Maps; agents through D-2 with the diff in the confirmation; names without spaces wrap at - . _ / in the name column; a Secret's Data and a Config Map's Binary Data keys above their values; the lead's decisions for the owner to confirm in a table; a Secret's values in the drawer open |
| 2026-10-15 | minor | D-54 (F7, the owner's finding of 2026-10-15) built: the navigator (⌘B, `ui.toggleNavigator`), the dock (⌘J, `ui.toggleDock`, its collapse the engine's now) and the chat column (⌘I) shown and hidden from three toggles at the top bar's right end and the View menu, the navigator's and the chat's state kept in `workspace.json` (D-41); an Update button left of them while Sparkle holds an update (its gentle reminders, D-4), checked against Sparkle 2.10.0 in a harness; the lead's decisions for the owner to confirm: the dock's toggle disabled without tabs, when the dock opens by itself, what the file keeps, never Sparkle's window by itself for a scheduled find, the button until the update session ends, the words, the icons, the renders |
| 2026-10-15 | minor | D-53 (F6, the owner's decisions of 2026-10-15) built: the top bar's MCP button gone; the status bar's MCP item has a gear that opens the MCP details (the switch, the endpoint, the token masked with Show and copy, the stdio and HTTP registrations with copy, Regenerate token after a warning); the token and the port kept across starts and off and on (`mcp-port` new), a new token only by the click, a taken port moved, kept and said; the bridge reads the files before every call; the lead's decisions for the owner to confirm listed; D-2, D-23, D-49 and D-52 follow it; H-98 |
| 2026-10-15 | minor | D-10 item 20 built: a conversation in which a turn ran over a private connection is not kept (memory only); the rule recorded (private from that turn's start, for good, also after a switch to a connection not marked private; no turn, not private); files of v0.8.0 to v0.10.0 rewritten without such conversations at the next start; a hint under Preferences › Agent's private mark, "not saved" in the chat's history; H-96 closed for the file |
| 2026-10-15 | minor | D-4: GitHub Pages serves a project site case-sensitively under the repository's name; the owner renamed the repository back to `guided-traffic/cogpit`, so the feed stays `https://guided-traffic.github.io/cogpit/appcast.xml` (for an hour it read `/Cogpit/`, no build used it); H-97 the feed moves with a rename |
| 2026-10-15 | minor | D-10 (H-96): a private connection's conversations are not kept, decided by the owner, not built; D-52: the owner chose the model (a) as the direction and postponed the rest until the rest is done, to a separate interview |
| 2026-10-15 | minor | R2 built (night run): a hotbar tile dragged within the bar and dropped into the gap between two tiles moves there through `hotbar.moveCluster`, as the tile's Move Up and Move Down do (`hotbardrag.go`, `TestHotbarTileDragMovesIt`, `TestHotbarTileDragCancels`); the lead's decisions for the owner to confirm: an accent line in the gap and the tile drawn as an outline where it was, no copy following the pointer; a drop on the "+" or the empty bar under the tiles puts the tile last (a bank has no empty cells); a drop outside the bar's tiles does nothing, the seam for R3 (D-52); a drag shorter than eight points opens the cluster as the click that slipped. D-52's context notes R2 built |
| 2026-10-15 | minor | D-52 designed (night run, R3), the lead's proposal for the owner to decide, nothing decided or built: what several windows touch, checked against the code (one `UIState`, one backend behind 67 `Session.Backend()` call sites, one MCP server, one `workspace.json`, Fyne's one loop for all windows); what Fyne 2.8.1 and GLFW offer for a drop outside the window (positions beyond the canvas, no position on `DragEnd`, `RequestPosition` without a getter); options (a) one session with a window's state and a connection per window, (b) a process per window with the shared files it breaks, (c) a tab instead, each with its size, agents, restart, crash stance, performance, security and drag; (a) recommended in four stages; seven questions in order; the risks per stage; the index and R3's line updated |
| 2026-10-15 | minor | D-25: A69 built (the lists' copy icon over the cell's end, the text keeping the cell's width) and A71 built (the drawer's Forward… right after the port again, the icon over the port's end); the lead's readings for the owner to confirm: the drawer's lines and links keep shortening before the icon, a link without buttons keeps the icon after its text; A73 built (a plain click on a row opens its drawer and keeps the checked rows: `ui.select`'s mode `open`), its rules in architecture §4.3 |
| 2026-10-15 | minor | D-14 (A116) built: the terminal fork's four shell tests under the race detector in CI (`make test-fork` without `-skip`); seven races in `RunLocalShell`, its directory check and `ExitCode` patched in the fork, shown by `TestRunLocalShell_MainLoop` on a main-loop stand-in; two decisions of the lead in the night run for the owner to confirm: the patches upstream candidates offered with the input patches at the end (A86), the fork's tests in the Linux race job only |
| 2026-10-15 | minor | A110, A111, A112, A114 built: the README's pictures in the dark and the light theme through `<picture>`, its banner, `make showcase` with the pinned oxipng, and `demo` the connected context in every picture; the lead's decisions for the owner to confirm in [the README's pictures](#the-readmes-pictures-a110-a111-a112-a114) |
| 2026-10-15 | minor | D-50 built (night run, A25): kubectl's signature, certificate and Rekor entry checked with sigstore-go v1.3.0 against an embedded sigstore trusted root (`make sigstore-root`), the entry asked for at `rekor.sigstore.dev` per download, no fallback; the research (no bundle on `dl.k8s.io`, the real entries of 1.36.1 and 1.37.1), the measured cost (43 modules more in both binaries, `cogpit` +10.0 MiB, `cogpit-cli` +8.5 MiB) and eleven decisions of the lead for the owner to confirm; D-3's index row and its signature bullet |
| 2026-10-15 | minor | D-51 built (A51, A52, A53, A54, A63): the navigator's search, keyboard row and collapsed source groups and the palette's typed text as engine state; the lead's night decisions for the owner to confirm, each with its options: the fields, the row's form, the bounds, one per window, `workspace.json` keeping the collapsed groups only, the dropdown's shown bank, an agent's palette text never running a change on ↵; D-41: the collapsed source groups kept; the index's answers A51–A55, A61, A63, A64, A67 and A113 marked built |
| 2026-10-15 | minor | The owner's Helm decisions of 2026-10-14 evening built: A39 (a release in both drivers notes the other as older, the tie's wording), A17 (nothing offered after an interrupted install; the journal still records installs, Show release in the banner), A36 (dotted repository names, the chart id doubling the dot, nothing saved to migrate), A14 (http and insecure-TLS origins marked in the Charts view, the drawer, both Helm tabs, the preview and an agent's confirmation); D-11 and D-29 record the lead's decisions for the owner to confirm |
| 2026-10-14 | minor | D-10: the private connection honoured by the engine (M12 (c)): `Invocation.ReadsLikeUser` only with origin `builtInAgent`, every read path that withheld from agents by origin asks it, no write rule does, the audit lines marked; the core set re-measured (29,746 bytes); six decisions of the lead in the night run for the owner to confirm (15–20): a Secret's Edit tab's text, a Secret in `object_get`, the window's own `object_get`, the chart preview, exec plugin arguments, the kept conversation (H-96) |
| 2026-10-14 | minor | D-10: the built-in agent's engine side built (M12 (a)); the endpoints checked against their documentation (verified and not verified per kind; Open WebUI forwards a caller's tools, from its documentation); fourteen decisions of the lead in the night run for the owner to confirm: the client, the tools and their measured sizes, a result's bound, thinking, the per-message `ui_state`, a cluster switched during a turn, what is kept, the connection test, `stream_options`, where the key goes (H-95), errors, the keychain, the window's update rate, a connection without a model |
| 2026-10-14 | minor | D-10: the window's chat built (M12 (b), night run); the lead's decisions for the owner to confirm, each with its options: the column beside a settings page, ⌘I, ⌘. from any focus and Escape in the input, the pinned message, tables as a grid, what was sent in a dialog, the chip of the view and selection, when conversations are created and shown, the model menu, Preferences › Agent, the thinking line |
| 2026-10-14 | minor | D-10: the owner's requirements and decisions of 2026-10-14 late evening for the agent chat (M12): the column at the right edge (⌘I), several connections and a connection test by layer, the private-connection mark ("reads like the user"), kept scrubbed conversations, the limits per turn and the loop guard, the kill switch; the engine's contract (`engine/chat.go`, `chat.toggle`) built |
| 2026-10-14 | minor | A60 and F2 built: About Cogpit in macOS's application menu with Cogpit's own dialog, which shows the version the binary reports instead of Fyne's default 0.0.1; D-16: the version a build reports, one variable (`internal/buildinfo`) for both binaries, development builds saying `dev` and their commit, the options weighed |
| 2026-10-14 | minor | A46 built in the night run: charts of one object whose ranges lie far apart get query sets of their own (architecture §9, with the lead's decisions for the owner to confirm) |
| 2026-10-14 | minor | D-43 (A43, F1, A115) built in the night run: a node's and the cluster Overview's CPU and memory history from node-exporter where the Prometheus has it, matched by the node label or the node's name or addresses, cAdvisor's root cgroup otherwise; where Prometheus has neither, the drawer and the Overview say so; node-exporter and kube-state-metrics on in the kind cluster's Prometheus values; the build's choices listed as the lead's decisions for the owner to confirm |
| 2026-10-14 | minor | The owner's decision round of 2026-10-14 evening recorded, nothing of it built: a section with one line per answer (A1–A117, F1, F2, R1–R3, the moot and housekeeping notes) naming where it lives. Decided, not built: D-2 (every RBAC kind and an agent's Helm install and upgrade always ask; `cluster.setPrometheus` and a proxy confirmed writes for agents; notifications' read state GUI-only; window-only notifications to agents filtered; removing a bank with tiles asks; a grant's confirmation text), D-3 (bash and fish hand-over, an enforcing setting), D-11 (no offer after an interrupted install), D-14 (the fork's shell tests under `-race`), D-22 (Sync the default theme, a free cluster colour), D-25 (the icon over a value's end, Forward… after the value), D-26 (the approval binds Cogpit's proxy), D-28 (unlink a sync folder), D-29 (http and insecure origins marked, dotted repository names; A39 found largely built), D-30 (combined PEMs, encrypted keys, R1 the kind of CA), D-33 (Reconcile with Source waits, the controller's identity named), D-34 (the filter button's accent), D-36 (sync windows), D-41 (collapsed source groups), D-43 (node history from node-exporter, F1), D-44 (a 1-hour end, 10 bytes, the README framed), D-47 (a link to another cluster asks). Confirmed as built: D-4 (A9, A11, A12; A10 a tag ruleset to set up; REL's two bullets moot), D-48, and defaults in D-3, D-14, D-22, D-25, D-29, D-34, D-35, D-37, D-39, D-40, D-44. New: D-49 (the MCP switch the bridge honours), D-50 (kubectl's signature with Rekor), D-51 (the navigator's and the palette's state), D-52 (several windows, open) |
| 2026-10-14 | minor | D-16 revised by the owner: only the complete product per release, today `Cogpit-<v>-macos-arm64.dmg` and `SHA256SUMS`; `cogpit-cli` inside `Cogpit.app`, the CLI archives for six platforms revoked; built (agent CLIBUNDLE). For the owner to confirm: `Contents/MacOS` as its place, the identifier `io.guided-traffic.cogpit.cli`, the bundled binary from `make build`, `make dist-cli` removed, the CLI run in the build job and never in the signing job. D-2 (the lead's default, for the owner to confirm): the window copies the stdio registration of the bundled `cogpit-cli`, HTTP without one; built |
| 2026-10-14 | minor | D-44 decided by the owner (H-82, option b): an agent's read that the filter reads takes the Secrets as the Secrets view reads them and waits for them; built (`engine/agentsecrets.go`) |
| 2026-10-14 | minor | D-4 built, the third step (agent SPARKLE): Sparkle 2.10.0 in `Cogpit.app`, pinned by version and SHA-256, loaded at run time (`internal/update`), `CFBundleVersion` the release's version; Preferences › App › Check for updates automatically and Update channel, Help › Check for Updates…; Sparkle's parts signed inside-out; the appcast drafted, signed (Sparkle 2.9's signed feeds) in the environment `sparkle` and deployed to Pages, in three jobs; no update key yet. For the owner to confirm: the signed feed, the preference as the source of truth, agents may turn the checks on but not off, any origin checks, Beta refused until prereleases, no XPC services and no entitlements, the three jobs, the history from the signed feed on Pages, the embedded release notes, the menu item's place and label |
| 2026-10-14 | minor | D-47 (the lead's decision, for the owner to confirm): `cogpit://open?context=&view=&namespace=&name=` links navigate only, through five `ui.*` commands with origin `deeplink`, which `Dispatch` refuses every other command and any write; a login waiting for approval stops the link; built, with Cogpit's defaults listed |
| 2026-10-14 | minor | D-48 (the lead's decision, for the owner to confirm): start at login as the macOS app's own login item (SMAppService), read from macOS, off by default, turned on by the user only; built; what the ad-hoc signed app does not verified |
| 2026-10-14 | minor | D-44 (owner, H-82): the values of the cached Secrets replaced by `«secret <ns>/<name>/<key>»` in what agents read; built, minimum 8 bytes; the base64 form, the stand-alone rule under 16 bytes, name keys left out, terminal tabs and every agent result, only cached Secrets and the refusal of placeholders in agents' texts the lead's decisions for the owner to confirm. H-63 (owner): Flux's credential fields withheld from the YAML agents read, built |
| 2026-10-14 | minor | D-29 (owner): a repository's credentials over plain http asked once per repository, bound to its name, URL and user, then used as helm uses it; nothing downloaded before (H-40 narrowed); built, the file, the binding without the password, the token and the banner Cogpit's defaults |
| 2026-10-14 | minor | D-37 (owner, H-71): every node shell pod recorded in `nodeshells.json` before its create, and the pods a crash left deleted by UID at the next connect to their context, with a notification; built, the details Cogpit's defaults |
| 2026-10-14 | minor | D-44 (owner, H-83): a terminal tab agents may read carries a marker that flashes at each agent's read, and its switch turns itself off after 15 minutes without one, which the strip then says; built, the details Cogpit's defaults |
| 2026-10-14 | minor | D-2 (owner): a role binding an agent creates or changes always waits for the user, also where agent writes are allowed: `roleBinding.create`, and `object.create` and `object.apply` when the text or the Edit tab holds a binding (a command's declared `AgentsAsk`); built. Helm charts and the role's own rules are not covered (H-27) |
| 2026-10-14 | minor | D-4 decided by the owner: Sparkle's feed on GitHub Pages (`https://guided-traffic.github.io/cogpit/appcast.xml`), signed in the environment `sparkle` with the owner as required reviewer |
| 2026-10-14 | minor | D-4 built, the second step: the release pipeline signs the app and `cogpit-cli` for macOS with the Developer ID (hardened runtime, no entitlements), ships the app in `Cogpit-<v>-macos-arm64.dmg`, notarizes both and staples the disk image's ticket, in a job of its own; no tag run yet. The lead's decision for the owner to confirm: a failure fails that job and the release gets no files until it is re-run. Agent REL's, for the owner to confirm: signing in its own job, the disk image instead of the zip, `cogpit-cli` notarized but not stapled, only the disk image submitted |
| 2026-10-14 | minor | The night decisions' bodies brought in line with the owner's answers of 2026-10-14: D-34's state reading and page, D-41's five and D-42's choice marked decided as built; D-35, D-36, D-43, D-44, D-46 confirmed; D-33's two rules (Reconcile with Source does not wait for the source; the whole chain checked before the first patch) and D-34's two filter rules (a row's states, several state columns) marked Cogpit's defaults, standing until the owner asks |
| 2026-10-14 | minor | D-4: the owner's Apple Developer ID certificate and a team API key for notarization set up in the environment `release`; the pipeline's signing steps not built yet |
| 2026-10-14 | minor | D-40 decided by the owner: option (b), the workloads' drawers' Pods tables without a Namespace column (the node drawer keeps it); built, PNGs re-rendered |
| 2026-10-14 | minor | D-39 decided by the owner: option (b), and a refusal of one namespace's pod metrics affects that namespace only (empty cells and the state forbidden there, the other namespaces' values and the Metrics Server's status kept, no Prometheus in its place); built, with Cogpit's defaults listed (a drawer too, the refusal kept for the TTL, `metrics.get`'s `forbidden`) |
| 2026-10-14 | minor | D-33 decided by the owner: Reconcile, Suspend and Resume as built, plus Reconcile with Source (the source's reconcile request first, then the object's; a Helm release's chart source and HelmChart) and Reset (resetAt with requestedAt's token) as the flux CLI does them, mutating, on selections too; Force stays an agent's parameter; built. Night decisions for the owner to confirm: Reconcile with Source does not wait for the source, and reads every object of the chain before the first patch |
| 2026-10-14 | minor | D-34 decided by the owner: a dashboard's state opens its kind's view filtered, through a status filter every view has (option a), the Argo CD Dashboard's tiles too; built, with two decisions for the owner to confirm (a row's states: the kind's state columns, tokens without one; several columns qualified and each a condition) and Cogpit's defaults listed; D-41: the tab's status filter kept |
| 2026-10-14 | minor | D-37 and D-38 decided by the owner (option b each): the default image the newest supported Alpine release, 3.24 pinned by its index digest and kept current by Renovate (no longer Lens's 3.19), the deadline a setting, Maximum duration, default 2 hours, set by the user only; built, the bounds (5 minutes to 24 hours) and the automerge of Renovate's updates Cogpit's defaults |
| 2026-10-14 | minor | D-35, D-36, D-41, D-42, D-43, D-44, D-46 decided by the owner in the morning's decision round, each as built (D-33, D-34, D-37 to D-40 follow with their builds) |
| 2026-10-14 | minor | D-43 (night decision, for the owner to confirm): the Nodes list's Disk and a node drawer's disk from node-exporter's root filesystem through Prometheus, never the kubelet's nodes/proxy; one query per 60 s poll, used as size less available, nodes matched by node label, instance or address, the Nodes list looking for Prometheus while the column shows, Cogpit's defaults; built |
| 2026-10-14 | minor | D-46 (night decision, for the owner to confirm): chart icons fetched from the hosts that serve them when the window shows them, with Preferences › Load chart icons on by default (option b); a repository's own host allowed at a private address, other private addresses refused; built |
| 2026-10-14 | minor | D-11, D-37 (the night build's seams): a dock tab whose context leaves Cogpit closes with it; an editor's draft is kept, a node shell's pod deleted (Cogpit's defaults) |
| 2026-10-13 | minor | D-44 (night decision, for the owner to confirm): agents read what the dock's logs tabs show freely, a terminal tab's screen only once the user turned on the tab's switch (off when a tab opens, GUI-only); built with Cogpit's defaults (every terminal type, no dialog, the bounds, no scrollback in the widget) |
| 2026-10-13 | minor | D-42 (night decision, for the owner to confirm): no context picker at start, the welcome page with its cluster rows; `--context` and `--demo` as before; the engine's notices from before the window shown as toasts once it is up; built |
| 2026-10-13 | minor | D-41 (night decision, for the owner to confirm): the tab bar's tabs and the tables' hidden columns and column order kept in `workspace.json` and restored at start without connecting anything, the tab shown last marked, the tabs of contexts that are gone dropped with a notice; column widths not kept (a question for the owner); built |
| 2026-10-13 | minor | D-40 (night decision, for the owner to confirm): the node drawer's Pods table in the drawers of Deployments, Replica Sets, Stateful Sets, Daemon Sets, Jobs and Cron Jobs, from indexes only the open drawer holds; Cron Jobs, the eight columns, every replica set's pods, the owner's UID in its namespace, the 500-row bound and the order of `related: true` Cogpit's defaults; built |
| 2026-10-13 | minor | D-39 (night decision, for the owner to confirm): pod metrics listed per namespace of the Pods view's plan for a user who may not list pods everywhere, built |
| 2026-10-13 | minor | D-36 (night decision, for the owner to confirm): Argo CD's Sync writes the operation `argocd app sync` writes without flags, bound to the Application's UID and resourceVersion, refused while an operation is requested or running |
| 2026-10-13 | minor | D-35 (night decision, for the owner to confirm): Argo CD's repository and cluster Secrets show an allowlist of non-credential keys to the window and agents, user information in URLs redacted, the other keys by name only |
| 2026-10-13 | minor | D-33 and D-34 (night decisions, for the owner to confirm): the Flux actions (Reconcile, Suspend, Resume, as the flux CLI patches them, mutating, on selections too) and the Flux dashboard (the states after kstatus, counts per kind, the objects that are not ready); built (G23) |
| 2026-10-13 | minor | D-37 and D-38 (night decisions, for the owner to confirm): the node shell as Lens's privileged pod in kube-system with nsenter, its 2-hour deadline, the pinned Alpine 3.19 default and the other defaults of the build; its image and pull secret set by the user only; both built |
| 2026-10-13 | minor | D-19: Create Secret built (night build, G9): `secret.create` refuses agents and the CLI before anything is asked or sent; its values go base64-encoded in `data` (Cogpit's default) |
| 2026-10-13 | minor | D-4 (owner): the environment `release` for the Apple credentials created; Sparkle's EdDSA key in an environment with the owner as required reviewer |
| 2026-10-13 | minor | D-16 (owner): dependency updates committed as `build(deps)`, no release of their own |
| 2026-10-13 | minor | D-16 revised (owner): releases live: semantic-release tags and releases after every gate on `main` with the GitHub App's token, the release pipeline attaches `cogpit-cli` and the unsigned app; a breaking change makes a minor release under v0. D-4: what is built of it (the unsigned app zip and the CLI archives; no signature, notarization, disk image, cask or Sparkle yet) |
| 2026-10-13 | minor | D-32 (owner): every byte amount in binary units with one decimal ("15.6 GiB"); built 2026-10-13 with one formatter in `model` (`FormatBytes`, `FormatByteQuantity`, `FormatResource`): lists, drawers, charts, data sizes, the kubectl download and the messages that state a size; the space before the unit, the decimal point whatever the region, milli-bytes as written, a size beside its limit not named and the places listed Cogpit's defaults; ten places not converted named for the owner |
| 2026-10-13 | minor | D-31 (owner): every timestamp in the local time zone and the region's format ("10.09.2026 17:41:01"); built 2026-10-13 from macOS's region (CoreFoundation), else the environment's locale and a CLDR table, one formatter in the engine; the format as a setting not built; the conversion, Windows reading the environment, logs with milliseconds and the list of places Cogpit's defaults |
| 2026-10-13 | minor | D-25 (owner, in a real window): the drawer's copy icon right after every kind of value, a status pill's included (it sat at the drawer's right edge), a long line shortened before it, a value with action buttons owning the room for it; built |
| 2026-10-13 | minor | D-30, the owner's answers: agents and the built-in agent's LLM endpoint read the decoded certificates, never a key (H-45 closed by the decision, removed from agents-and-mcp.md); `tls.crt` checked against `tls.key` (the key parsed in memory, the outcome yes, no or a reason) and DER certificates decoded, both built; the pairing rule, the chain check, the 64 KiB key bound and the names that count as private keys Cogpit's defaults |
| 2026-10-13 | minor | D-30 (owner): certificates in Secrets decoded into separate fields in the drawer and for agents (`object_get`'s `certificates`), an exception to "Secret values never reach an agent"; built 2026-10-13, which keys, the bounds, the fields, the thresholds and the layout Cogpit's defaults; no chain or key-pair check |
| 2026-10-13 | minor | D-25 (owner, after seeing it): the lists' copy icon not on every cell but outside a blacklist of columns (by type: numbers, ages, times, metrics, warnings; by id: state words, booleans, counts and ratios in text, policy words: Cogpit's reading of the owner's principle), and right after the text instead of at the column's end; built |
| 2026-10-13 | minor | D-29 built: Helm's repositories.yaml read every 2 s (the helm CLI's resolution, one place), credentials in memory only and to the repository's own host, indexes in Cogpit's own cache keyed by name and URL (decided at the build), `helm.repo.add` and `helm.repo.remove` removed; Cogpit's defaults listed |
| 2026-10-13 | minor | D-25: copy icons in the lists built (one icon per list on the text cell under the pointer, the drawer's checkmark and toast); every text cell, the icon inside the cell's right end with the text shortened before it, a click that never selects, and no values in a Secret's list are Cogpit's defaults |
| 2026-10-13 | minor | D-29 (owner): repositories are added with the helm CLI, Cogpit only reads Helm's file and writes nothing into it |
| 2026-10-13 | minor | D-29 (owner): Helm chart repositories read from Helm's repositories.yaml, changes within 5 seconds, credentials included; D-25 (owner): names in the lists copyable with the drawer's copy icon; neither built |
| 2026-10-13 | minor | D-11: drafts of unsaved editor text built (night run; the delays, the offer on connect and closing as a discard are that run's defaults, not decided); D-19: no draft of a Secret's text |
| 2026-10-13 | minor | D-3 built (night run): the version check when a terminal opens, the GUI-only verified download (SHA-256 and sigstore signature, no transparency log: H-35), the release first on the terminal's PATH (zsh after its profile; H-36), the Preferences; the differences listed under D-3 |
| 2026-10-13 | minor | D-14: the right mouse button's action and ⌃Ins and ⇧Ins on the clipboard, patches of the fork and upstream candidates (night run), not offered |
| 2026-10-13 | minor | D-22 amended (Cogpit's default): the theme is a preference too, `--theme` overrides it for a run; the clusters' icon colours are Solarized's accents |
| 2026-10-13 | minor | D-11: Helm install built on the journal's road as the lead's default (an interrupted install offered as an uninstall); the decision stays the owner's |
| 2026-10-12 | minor | D-11: crash reports kept locally (built), and what they found first |
| 2026-10-12 | minor | D-14: the upstream pull requests opened (#155–#158, owner's OK) |
| 2026-10-12 | minor | D-28 (owner, built, scope C4): one approval dialog per file and identical plugin, listing every context with its server and trust; the approvals stay per context with their own fingerprints; approveExec takes the list and refuses all if one changed |
| 2026-10-12 | minor | D-28 built (scope C3): every context from a source file read alone, `~/.kube/config` a file of the default folder, `$KUBECONFIG` and `--kubeconfig` files session sources, bare names resolved; the default kubeconfig no longer merged |
| 2026-10-12 | minor | D-28 (owner): the default kubeconfig as sources, as Lens (config an ordinary synced file, `$KUBECONFIG`'s files sources of their own); C1's Rollback on the current revision does what helm does (owner) |
| 2026-10-12 | minor | D-25 (owner, built): a copied Secret value marked concealed for clipboard managers and kept from Universal Clipboard on macOS (H-25) |
| 2026-10-12 | minor | D-26 (owner): a credential plugin's approval binds the program's content when it lies in the kubeconfig's or sync folder; D-27 (owner, built): an agent's inline Helm values shown in the confirmation |
| 2026-10-12 | minor | D-22 to D-25 built: Solarized theme tokens (contrast departures listed), the status bar link, the C helm logo (`internal/ui/logo`), copying values from the drawer |
| 2026-10-12 | minor | D-19 built: Secrets edited and created from YAML in the GUI only; agents and the CLI refused in the engine, values scrubbed from errors |
| 2026-10-12 | minor | D-17 built: a token file outside an added kubeconfig's folder is asked for in the plugin dialog; the approval binds the content of a program or CA file inside the folder (R-5, D-26) and the dialog shows proxy, insecure and own CA (R-11) |
| 2026-10-12 | minor | Owner's decisions on look and use: D-22 Solarized colours, D-23 the status bar link, D-24 the C helm logo, D-25 copying values |
| 2026-10-12 | minor | D-21 (owner): no adding of kubeconfigs from pasted text; the paste feature built the same day is removed (pasted sources of earlier builds are no longer listed, their files stay) |
| 2026-10-12 | minor | D-20 (owner): a port forward fails at once on permanent errors (option a); built the same day |
| 2026-10-12 | minor | D-14 (owner): the upstream pull requests grouped, opened by the lead after the owner's OK on the series |
| 2026-10-12 | minor | D-19 (owner): Secrets editable and creatable from YAML in the GUI only (option a), not built |
| 2026-10-12 | minor | D-18 (owner): Secret annotations stay visible to agents; H-23 documented and accepted (option d) |
| 2026-10-12 | minor | D-17 (owner): token files of added kubeconfigs outside their folder are asked for like credential plugins (H-24, option a) |
| 2026-10-12 | minor | D-11: Helm upgrade decided and built (journal, offered as a rollback), journals locked between processes |
| 2026-10-12 | minor | D-14 built: the two panics patched, the widget fuzzed alone, further crashes, hangs and races fixed; the upstream series prepared, not sent |
| 2026-10-12 | minor | D-13 as revised built (fourth run): the Secrets view on a full watch, the drawer from the cache, Helm's releases through covering scopes |
| 2026-10-12 | minor | D-15 built (`make parity`); D-14: the copy is on upstream's head, whose code builds on Fyne 2.8.1 |
| 2026-10-12 | minor | Owner's decisions after the third run: D-13 revised to a full watch on Secrets (option a); D-14 terminal widget (keep the fork, upstream, fuzz); D-15 the parity checker before the generators; D-16 dry runs until the alpha's release |
| 2026-10-11 | minor | D-11: Helm rollback and uninstall are crash-safe through a journal and an offer on the next connect (third run); D-13: the Secrets view built, the Type column found empty too, the Table format open for the owner |
| 2026-10-11 | minor | D-13: Secrets in the cache decided (owner): metadata only for the list, a GET for an opened Secret, a label-selected watch for Helm's releases (H-8, option b) |
| 2026-10-11 | minor | D-1 decided (owner): Fyne after the spike gate, with re-checks of S7 at M0's CI and S5 at M11, each reopening D-1 only for its platform |
| 2026-10-11 | minor | D-1: the spike ran as the local MVP; results recorded, gate decision pending |
| 2026-10-11 | minor | D-7: macOS first, Windows and Linux after the macOS beta; Cogpit is a desktop client, never a web UI |
| 2026-10-11 | minor | D-5 decided (owner): extensions are built into the code; no external extension mechanism for now |
| 2026-10-11 | minor | D-9 decided (owner): Flux CD and Argo CD views in the beta, other paid features after it, vendor features never |
| 2026-10-11 | minor | D-4 for macOS decided (owner): the app in a signed, notarized disk image as the standard install, a Homebrew cask from the same image, Sparkle 2 for updates |
| 2026-10-11 | minor | D-4: macOS release artifact is the zipped app bundle, no dmg (owner); Sparkle 2 recommended as updater |
| 2026-10-11 | minor | D-3: `kubectl` from `PATH`, verified matching download on one click (owner) |
| 2026-10-11 | minor | D-13: cache held in memory only, not persisted across restarts (owner) |
| 2026-10-11 | minor | D-13 cache: decided requirement, open persistence question |
| 2026-10-11 | major | Go instead of Rust (D-7); Fyne behind a spike gate (D-1); client-go, kubectl libraries, Helm SDK (D-6, D-3); MCP control of UI and cluster (D-2); built-in agent with an OpenAI-compatible endpoint (D-10); one-click updates (D-4); crash stance (D-11); parity sources (D-12) |
| 2026-10-11 | major | First version: D-1 to D-9 |
