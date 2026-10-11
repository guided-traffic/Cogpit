# Testing

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** minor (the cluster tier in CI, the drain cluster, the second run's tests) · **Target:** Lens Desktop 2026.10.51501

The MVP (2026-10-11) brought the first tiers; the table says for each whether it exists, the
`make` target and where it lives. Tiers without code are the plan.

## Tiers

| Tier | Runs where | Needs | What it proves | State |
|---|---|---|---|---|
| Unit | `go test ./...` everywhere | nothing | logic in isolation; `model` and `kubeconfig` are fully testable without I/O | exists: `make test-unit` |
| Race | `go test -race ./...` | nothing | no data races; a concurrent map write would end the process | exists: `make test-unit-race` (in CI on macOS as `make test-unit-coverage`, the same run with a coverage profile, and on Linux inside `make test-kind`; [releasing › CI](releasing.md#ci)) |
| Kind-profile goldens | `internal/model` | recorded objects in `testdata/` | an object becomes the row and detail model Lens would show | exists: [`kinds/golden_test.go`](../../internal/model/kinds/golden_test.go) (`TestGolden`, 16 built-in kinds) and `TestCustomGolden` (definitions and custom resources), 119 synthetic objects; a kind's own `<kind>_test.go` holds its status texts (`checkStatusTexts`) and fuzz target; `-update` regenerates |
| Cache | `internal/cache`, `internal/testkit/integration` | an in-memory source; a fake API server over HTTPS | watches, warm period, error state, transforms, and the API requests each scenario costs | exists; envtest not used yet ([`budget_test.go`](../../internal/testkit/integration/budget_test.go)) |
| Engine | `internal/engine` | the real cache over an in-memory source | commands, snapshots, the drawer, confirmation per origin and what it binds (context, objects, UIDs), the audit log's redaction, custom kinds following their definitions, the drain journal across a simulated crash | exists |
| Command fuzzing | `go test -fuzz` on the registry | the in-memory source | every command, fed adversarial params, returns an error instead of panicking | exists: `FuzzDispatch` |
| MCP | `internal/mcp`, `internal/testkit/integration` | the SDK's in-memory transports and loopback HTTP | tools match the registry; the showcase; the HTTP endpoint rejects missing tokens and foreign hosts and origins | exists ([`showcase_test.go`](../../internal/testkit/integration/showcase_test.go)) |
| Agent | `internal/agent` | a fake OpenAI-compatible server | the tool loop, its limits, redaction of what is sent | not built |
| Cluster | locally and in CI (the Cluster Tests job on Linux, Kubernetes 1.37.0; locally 1.36.1) | Docker and `kind` | views, logs, exec, `kubectl` in the terminal, edits, custom resources and workload actions against a real API server and kubelet | exists: `make test-kind` (build tag `kind`), [`engine_kind_test.go`](../../internal/testkit/integration/engine_kind_test.go), `exec_kind_test.go`, `kind_test.go`, [`edit_kind_test.go`](../../internal/testkit/integration/edit_kind_test.go) (create, edit, conflict and delete in namespace `cogpit-edit`; every template dry-run), `TestKindCustomResources` (a test definition in `cogpit-crd`), `TestKindNodeAndWorkloadActions` (scale, trigger, suspend, resume, evict in `cogpit-actions`), `TestKindStorageAndNetworkViews` (a claim, its pod, volume, Service, network policy and replication controller in `cogpit-d`: the network and storage views and the related objects); attach, port-forward, node shell and Helm not yet |
| Drain cluster | locally, on request | Docker and `kind`: a separate three-node cluster `cogpit-drain` with its own kubeconfig (`make kind-drain-up`) | cordon, a drain blocked by a PodDisruptionBudget, a simulated crash, the offer on the next session, finish, uncordon, on real nodes; never on the shared single-node cluster | exists: `make test-kind-drain` (`TestKindDrainCrashFinishUncordon`, skipped unless `COGPIT_KIND_DRAIN_KUBECONFIG` is set); passed 2026-10-11; not in CI |
| UI | everywhere with cgo | Fyne's `test` package (offscreen) | views render from snapshots; images for visual review | exists: `internal/ui/...`, PNGs in `testdata/` (`-update` rewrites them); widgets that reach the main goroutine take the shell's `do`, so tests drive it through their queue (Fyne's test driver runs `fyne.DoAndWait` inline on the caller's goroutine) |
| Fuzz | locally | `go test -fuzz` | kubeconfig parsing, kind profiles, YAML tokenizer and edits, terminal escape filter, manifests for edit and create (`FuzzParseManifests`, `FuzzEditYAML`) | targets exist; no nightly run |
| Performance | locally | Go benchmarks and `cogpit --perf` | the budgets below | exists: `make bench`; no fixed runner |

## Fixtures

- **Recorded objects** are real API responses with names, namespaces, IPs and UIDs replaced by
  synthetic values. Never commit objects from a real cluster unscrubbed: they carry internal
  hostnames, registries and sometimes credentials in annotations or environment variables.
- **Clusters** are created by the tests (`kind`, envtest), never borrowed from a developer's
  kubeconfig. Tests must not read `~/.kube/config`; `kind` writes its kubeconfig to a temporary
  file (`--kubeconfig`), never into the user's.
- **API request counting:** tests wrap the client's transport in a counting round-tripper and
  assert the requests a scenario may cost.

## Performance budgets (proposed, not measured)

Targets to validate against Lens Desktop on the same machine before they are enforced. The heading
keeps its name so that links survive; the "Measured" column holds what Cogpit reached on
2026-10-11 on an Apple M1 Ultra (macOS 26.6.2), without a Lens comparison yet.

| Id | Scenario | Proposed target | Measured (2026-10-11) |
|---|---|---|---|
| P1 | Open a context to the first rows of the Pods list, 1,000-pod cluster | ≤ 1 s | not measured |
| P2 | Sort a 10,000-pod list by a column | ≤ 50 ms | 10.2 ms sort, 11.6 ms command to new snapshot (`BenchmarkSortChange10k`, `BenchmarkSortCommand10k`); 41 µs to apply it to the table without paint |
| P3 | Scroll a 10,000-pod list | every frame ≤ 16 ms | real window: after the first fill at most 3 stalls of 16–25 ms in 9 s of scrolling 3 rows per frame; the first fill of the table stalls 151–193 ms once per window ([S1](spike-fyne.md#results)) |
| P4 | Apply a burst of 1,000 watch events per second to an open list | no frame over 16 ms | engine side: 29 snapshots in 3 s at 1,000 events/s (`TestDemoClusterThroughTheEngine`), a 10,000-row snapshot in 2.1–4.6 ms; frames: the P3 runs had the churn on; view switches under the churn at most 42 ms, p99 6.3–7.4 ms |
| P5 | Follow a log stream at 10,000 lines per second | no frame over 16 ms; memory bounded by the scrollback limit | offscreen: 0.15 ms median, 2.3 ms max of work per 100 ms batch at ~9,800 lines/s; 1.5 MiB for 10,000 kept lines (termview logs test); no GPU frames measured |
| P6 | Resident memory with three clusters open on their Pods lists | track only until measured against Lens | the demo cluster with 10,000 pods keeps ~98 MB of its own heap; Cogpit's not measured |
| P7 | Cold start to an interactive window | ≤ 1 s | 0.44–0.58 s from the shell's start to the first rows of the demo cluster, of which 0.42–0.56 s is the window's GL set-up; the time before `fyneui.Run` (logger, engine, MCP) is not measured |
| P8 | Type into a 5 MB YAML document in the editor, highlighting on | keystroke to repaint ≤ 16 ms | real window, one keystroke per frame: widget work p50 0.09 ms, max 2.1 ms; one frame of 19 ms in 645 keystrokes |
| P9 | A view stays open on an idle cluster for 10 minutes | 0 list requests; only the view's watches | 0 requests in 2 s idle (fake API server) and 3 s idle (kind); 1 streaming watch per kind (`TestBudgetIdleViewAndWarmReopen`, `TestKindIdleViewsCostNoLists`); 10 minutes not run |
| P10 | Reopen a view within the warm period | 0 list requests; rows shown ≤ 100 ms | 0 requests, rows in ~90 µs in the engine (fake API server); in the real window 48–51 ms from the switch to the rows on screen; a view opened for the first time on the demo cluster shows rows after 109–149 ms |
| P11 | An agent asks `ui_state`, `resources_find` and `ui_navigate` for an object in a warm cache | 0 API requests | 0 lists and 0 watches on the source (`TestShowcaseServiceByClusterIP`) |

## Before a pull request

```sh
make lint
make test-unit-race
make test-kind      # when you touched cluster, cache, terminal or engine streams; needs make kind-up
make test-kind-drain  # when you touched cordon or drain; needs make kind-drain-up (a separate cluster)
```
